#!/usr/bin/env python3
"""Task-local runtime. No action executes without the explicit final go-ahead.

prepare -> http-check seed-legacy -> upgrade -> http-check check -> vite -> CUA
-> close. Never operates on the user's database or signals their processes.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import secrets
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import time
import urllib.request

ROOT = Path('/Users/bakhromachilov/startups/justixauto')
TASK = 'warehouse-bulk-move-20261013'
IMAGE = 'docker.io/library/postgres:18.6-alpine3.24@sha256:d3e1620b530c944afa6e887d22eb899824da68e19c52024bf98f5220c88a65b2'
DB = 'warehouse_bulk_move_e2e'
API = 'http://127.0.0.1:8091'
UI = 'http://localhost:5192'
GO_AHEAD = 'warehouse-bulk-move-final'
MIGRATION46_HASH = 'af845b1de029f60e84b05a92afd9c5027d923e4867befbe9070a271c47e0e188'
NODE = ROOT / 'docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node'
PRIVATE_DIR = None


def require(ok, message):
    if not ok:
        raise RuntimeError(message)


def private_write(path, value):
    path = Path(path)
    raw = value if isinstance(value, str) else json.dumps(value, indent=2) + '\n'
    with os.fdopen(os.open(path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC, 0o600), 'w') as f:
        f.write(raw)
    os.chmod(path, 0o600)


def command(argv, *, cwd=ROOT, env=None, stdin=None, timeout=60):
    try:
        p = subprocess.run([str(a) for a in argv], cwd=cwd, env=env, input=stdin,
                           text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout)
    except subprocess.TimeoutExpired as exc:
        if PRIVATE_DIR:
            private_write(PRIVATE_DIR / 'command-failure.json', {'argv': [str(a) for a in argv],
                'timeout': timeout, 'stdout': str(exc.stdout or ''), 'stderr': str(exc.stderr or '')})
        raise RuntimeError('Bounded runtime command timed out; inspect private command-failure.json') from None
    if p.returncode != 0 and PRIVATE_DIR:
        private_write(PRIVATE_DIR / 'command-failure.json', {'argv': [str(a) for a in argv],
            'exit': p.returncode, 'stdout': p.stdout, 'stderr': p.stderr})
    require(p.returncode == 0, f'{Path(str(argv[0])).name} failed (exit {p.returncode}); details retained privately by execution owner')
    return p.stdout


def basic_env():
    # Preserve the original HOME for tool caches; never import arbitrary runtime secrets.
    return {'HOME': os.environ['HOME'], 'PATH': '/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin',
            'LANG': 'en_US.UTF-8', 'GOTOOLCHAIN': 'local', 'GOPROXY': 'off', 'GOSUMDB': 'off'}


def runtime_env(state):
    env = basic_env()
    env.update(DATABASE_URL=f"postgres://postgres:{state['dbPassword']}@127.0.0.1:55443/{DB}?sslmode=disable",
               HTTP_ADDR='127.0.0.1:8091', COOKIE_SECURE='false',
               ALLOWED_ORIGINS=f'{UI},{API}', FILE_STORAGE='local',
               DOCUMENTS_DIR=str(Path(state['directory']) / 'documents'), WEB_DIR='',
               API_DOCS='false', OTEL_EXPORTER_OTLP_ENDPOINT='', SHUTDOWN_DRAIN='0s')
    return env


def source_hash():
    digest = hashlib.sha256()
    suffixes = {'.go', '.sql', '.ts', '.tsx', '.css', '.json', '.html', '.mjs'}
    paths = []
    for folder in ('cmd', 'internal', 'migrations', 'web/apps', 'web/packages'):
        paths.extend(p for p in (ROOT / folder).rglob('*')
                     if p.is_file() and p.suffix in suffixes and not {'node_modules', 'dist', '.vite'}.intersection(p.parts))
    paths.extend(ROOT / p for p in ('go.mod', 'go.sum', 'web/package.json', 'web/package-lock.json') if (ROOT / p).exists())
    for path in sorted(set(paths)):
        digest.update(str(path.relative_to(ROOT)).encode() + b'\0' + path.read_bytes() + b'\0')
    return digest.hexdigest()


def save(state):
    private_write(Path(state['directory']) / 'state.json', state)


def load(path):
    global PRIVATE_DIR
    path = Path(path).resolve()
    require(path.name == 'state.json' and path.parent.parent == Path('/private/tmp') and
            path.parent.name.startswith('justixauto-warehouse-e2e-'), 'State must be this task private /private/tmp/state.json')
    require(path.stat().st_uid == os.getuid() and path.stat().st_mode & 0o077 == 0, 'State is not private to this user')
    require(path.parent.stat().st_mode & 0o077 == 0, 'Runtime directory must be mode 700')
    state = json.loads(path.read_text())
    require(state['task'] == TASK and Path(state['directory']).resolve() == path.parent, 'State task/directory mismatch')
    require(state['api'] == API and state['ui'] == UI and state['database'] == DB, 'Refusing non-fixture target')
    PRIVATE_DIR = path.parent
    return state


def check_source(state):
    require(source_hash() == state['sourceHash'], 'Source changed since runtime preparation; stop and request affected rebuild')


def container(state):
    require(re.fullmatch(r'[0-9a-f]{64}', state.get('containerId', '')) is not None, 'Missing exact fixture container ID')
    # Read only non-secret metadata; never print or inspect Config.Env.
    raw = command(['docker', 'inspect', state['containerId'], '--format',
                   '{{json .Id}} {{json .Name}} {{json (index .Config.Labels "justixauto.task")}}'])
    require(raw.strip() == f'"{state["containerId"]}" "/{state["containerName"]}" "{TASK}"', 'Fixture container identity mismatch')
    return state['containerId']


def sql(state, statement):
    cid = container(state)
    return command(['docker', 'exec', '-i', cid, 'psql', '-X', '-qAt', '-v', 'ON_ERROR_STOP=1',
                    '-U', 'postgres', '-d', DB], stdin=statement, timeout=20).strip()


def signature(pid):
    p = subprocess.run(['ps', '-p', str(pid), '-o', 'lstart=,comm='], text=True, capture_output=True)
    return p.stdout.strip() if p.returncode == 0 else ''


def start(state, key, argv, env, cwd):
    require(key not in state.get('processes', {}), f'{key} already recorded; do not create duplicate listener')
    log_path = Path(state['directory']) / f'{key}.log'
    with os.fdopen(os.open(log_path, os.O_WRONLY | os.O_CREAT | os.O_APPEND, 0o600), 'wb') as log:
        proc = subprocess.Popen([str(a) for a in argv], cwd=cwd, env=env, stdout=log, stderr=log,
                                stdin=subprocess.DEVNULL, start_new_session=True)
    state.setdefault('processes', {})[key] = {'pid': proc.pid, 'signature': signature(proc.pid), 'log': str(log_path)}
    save(state)


def stop(state, key):
    entry = state.get('processes', {}).get(key)
    if not entry:
        return
    sig = signature(entry['pid'])
    if sig:
        require(bool(entry['signature']) and sig == entry['signature'], f'{key} PID identity changed; no signal sent')
        os.kill(entry['pid'], signal.SIGTERM)
        deadline = time.monotonic() + 15
        while time.monotonic() < deadline and signature(entry['pid']) == sig:
            time.sleep(0.2)
        require(signature(entry['pid']) != sig, f'{key} did not drain; no forced termination attempted')
    del state['processes'][key]
    save(state)


def free_ports():
    for host, port in [('127.0.0.1', 55443), ('127.0.0.1', 8091), ('127.0.0.1', 5192)]:
        with socket.socket() as sock:
            sock.bind((host, port))


def ready(url, expected):
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    request = urllib.request.Request(url, headers={'Accept': 'text/html' if expected == 200 else 'application/json'})
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        try:
            with opener.open(request, timeout=1) as response:
                if response.status == expected:
                    return
        except (OSError, urllib.error.URLError):
            pass
        time.sleep(0.25)
    raise RuntimeError(f'Expected readiness unavailable at {url}; inspect private service log')


MIGRATOR = '''package main
import (
 "errors"
 "fmt"
 "os"
 "github.com/golang-migrate/migrate/v4"
 "justixauto/internal/pkg/database"
)
func main() {
 m, err := database.NewMigrator(os.Getenv("DATABASE_URL")); if err != nil { panic(err) }; defer m.Close()
 switch os.Args[1] {
 case "45": err = m.Migrate(45)
 case "up": err = m.Up()
 case "version": var v uint; var d bool; v,d,err=m.Version(); fmt.Printf("%d %t\\n",v,d)
 default: panic("unsupported task migration action")
 }
 if err != nil && !errors.Is(err,migrate.ErrNoChange) { panic(err) }
}
'''


def prepare(args):
    global PRIVATE_DIR
    require(command(['git', 'rev-parse', '--show-toplevel']).strip() == str(ROOT), 'Wrong Git root')
    require(command(['git', 'rev-parse', 'HEAD']).strip() == args.expected_sha, 'Unexpected source checkpoint')
    require((ROOT / 'migrations/000046_inventory_receipt_locations.up.sql').is_file(), 'Final migration46 absent')
    require(hashlib.sha256((ROOT / 'migrations/000046_inventory_receipt_locations.up.sql').read_bytes()).hexdigest() == MIGRATION46_HASH,
            'Migration46 differs from application-approved SQL; obtain updated acceptance')
    versions = sorted(int(p.name.split('_', 1)[0]) for p in (ROOT / 'migrations').glob('*.up.sql'))
    require(versions[-1] == 46, 'Unexpected migration set; obtain a revised packet')
    watcher = command(['ps', '-p', '31138', '-o', 'stat=,comm=']).strip()
    require(watcher.split()[0].startswith('T') and watcher.endswith('/air'), 'Project watcher is not the paused Air process')
    require('n' + str(ROOT) in command(['lsof', '-a', '-p', '31138', '-d', 'cwd', '-Fn']).splitlines(), 'Air cwd changed')
    require('p37081' in command(['lsof', '-nP', '-iTCP:8090', '-sTCP:LISTEN', '-Fp']).splitlines(), 'Pre-change API identity changed')
    require(command(['ps', '-p', '37081', '-o', 'ppid=']).strip() == '31138', 'Pre-change API parent changed')
    free_ports()
    directory = Path(tempfile.mkdtemp(prefix='justixauto-warehouse-e2e-', dir='/private/tmp'))
    os.chmod(directory, 0o700)
    PRIVATE_DIR = directory
    state = {'task': TASK, 'directory': str(directory), 'stage': 'preparing', 'sourceSHA': args.expected_sha,
             'sourceHash': source_hash(), 'api': API, 'ui': UI, 'database': DB,
             'dbPassword': secrets.token_urlsafe(24), 'accounts': {'admin': {'login': 'warehouse-e2e-admin',
             'password': secrets.token_urlsafe(24)}}, 'processes': {},
             'containerName': 'justixauto-warehouse-e2e-' + secrets.token_hex(5)}
    save(state)
    print(f"private_state={directory / 'state.json'}", flush=True)
    shutil.copy2(ROOT / 'var/bin/dev-api', directory / 'api-before')
    state['legacyBinaryHash'] = hashlib.sha256((directory / 'api-before').read_bytes()).hexdigest()
    save(state)
    private_write(directory / 'migrate-main.go', MIGRATOR)
    private_write(directory / 'overlay.json', {'Replace': {str(ROOT / 'cmd/migrate/main.go'): str(directory / 'migrate-main.go')}})
    env = basic_env()
    for name, pkg in [('api', './cmd/api'), ('bootstrap-admin', './cmd/bootstrap-admin'), ('migrate', './cmd/migrate')]:
        argv = ['bash', 'tools/go.sh', 'build']
        if name == 'migrate':
            argv += ['-overlay', directory / 'overlay.json']
        argv += ['-o', directory / name, pkg]
        result = subprocess.run([str(a) for a in argv], cwd=ROOT, env=env, capture_output=True, timeout=180)
        private_write(directory / f'build-{name}.log', (result.stdout + result.stderr).decode(errors='replace'))
        require(result.returncode == 0, f'{name} runtime compilation failed; inspect private build log')
    check_source(state)
    denv = basic_env() | {'POSTGRES_PASSWORD': state['dbPassword']}
    state['containerId'] = command(['docker', 'run', '--detach', '--rm', '--pull=never', '--name', state['containerName'],
        '--label', f'justixauto.task={TASK}', '--publish', '127.0.0.1:55443:5432', '--tmpfs', '/var/lib/postgresql:rw',
        '--env', f'POSTGRES_DB={DB}', '--env', 'POSTGRES_USER=postgres', '--env', 'POSTGRES_PASSWORD', IMAGE], env=denv).strip()
    save(state)
    deadline = time.monotonic() + 45
    while True:
        p = subprocess.run(['docker', 'exec', state['containerId'], 'pg_isready', '-U', 'postgres', '-d', DB], capture_output=True)
        if p.returncode == 0:
            break
        require(time.monotonic() < deadline, 'Disposable PostgreSQL did not become ready')
        time.sleep(0.25)
    renv = runtime_env(state)
    command([directory / 'migrate', '45'], cwd=directory, env=renv)
    require(command([directory / 'migrate', 'version'], cwd=directory, env=renv).strip() == '45 false', 'Expected clean schema45')
    admin = state['accounts']['admin']
    command([directory / 'bootstrap-admin', '-login', admin['login'], '-email', 'admin@warehouse-e2e.test', '-name', 'Warehouse E2E Admin'],
            cwd=directory, env=renv, stdin=admin['password'] + '\n')
    start(state, 'api', [directory / 'api-before'], renv, directory)
    ready(API + '/readyz', 204)
    state['stage'] = 'legacy_ready'
    save(state)
    print('stage=legacy_ready schema=45; next: http-check.py seed-legacy')


def upgrade(state):
    require(state['stage'] == 'legacy_seeded', 'Seed legacy fixture before upgrade')
    check_source(state)
    stop(state, 'api')  # Old writers must be drained before backfill.
    before = sql(state, 'SELECT COALESCE(jsonb_agg(to_jsonb(b) ORDER BY id),\'[]\'::jsonb) FROM inventory_receipt_batches b;')
    private_write(Path(state['directory']) / 'batches-before-migration.json', before)
    directory = Path(state['directory'])
    command([directory / 'migrate', 'up'], cwd=directory, env=runtime_env(state))
    require(command([directory / 'migrate', 'version'], cwd=directory, env=runtime_env(state)).strip() == '46 false', 'Expected clean schema46')
    after = sql(state, 'SELECT COALESCE(jsonb_agg(to_jsonb(b) ORDER BY id),\'[]\'::jsonb) FROM inventory_receipt_batches b;')
    require(before == after, 'Migration changed original receipt rows')
    mismatch = sql(state, '''SELECT count(*) FROM inventory_receipt_batches b LEFT JOIN inventory_receipt_locations l
       ON l.receipt_batch_id=b.id AND l.warehouse_id=b.warehouse_id
       WHERE l.receipt_batch_id IS NULL OR l.unidentified_count<>b.unidentified_count;''')
    require(mismatch == '0', 'Migration backfill mismatch')
    state['migrationEvidence'] = {'from': 45, 'to': 46, 'originalReceiptRowsUnchanged': True,
                                 'allOriginalBalancesMatch': True, 'oldWriterDrained': True}
    start(state, 'api', [directory / 'api'], runtime_env(state), directory)
    ready(API + '/readyz', 204)
    state['stage'] = 'current_ready'
    save(state)
    print('stage=current_ready schema=46 backfill=passed; next: http-check.py check')


def vite(state):
    require(state['stage'] in ('current_ready', 'http_passed'), 'Final API is not ready')
    check_source(state)
    directory = Path(state['directory'])
    script = "import {createServer} from " + json.dumps(str(ROOT / 'web/node_modules/vite/dist/node/index.js')) + ";\n"
    script += 'const server=await createServer(' + json.dumps({'configFile': str(ROOT / 'web/apps/portal/vite.config.ts'),
                'cacheDir': str(directory / 'vite-cache'), 'server': {'host': '127.0.0.1', 'port': 5192, 'strictPort': True}}) + ');\nawait server.listen();\n'
    private_write(directory / 'vite.mjs', script)
    env = basic_env() | {'JUSTIX_API': API, 'HTTP_ADDR': '127.0.0.1:8091'}
    start(state, 'vite', [NODE, directory / 'vite.mjs'], env, ROOT / 'web/apps/portal')
    ready(UI + '/', 200)
    print(f'CUA_only_url={UI}/; synthetic sign-in stays in private state; user services unchanged')


def close(state):
    stop(state, 'vite')
    stop(state, 'api')
    if state.get('containerId'):
        command(['docker', 'stop', '--time', '10', container(state)], timeout=20)
        state['stoppedContainerId'] = state.pop('containerId')
    state['stage'] = 'closed'
    save(state)
    print('isolated_resources_closed; private evidence retained; user services and Air state were not changed by cleanup')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('action', choices=['prepare', 'upgrade', 'vite', 'close'])
    parser.add_argument('--go-ahead', required=True)
    parser.add_argument('--state')
    parser.add_argument('--expected-sha')
    args = parser.parse_args()
    require(args.go_ahead == GO_AHEAD, 'Explicit final execution go-ahead required')
    os.umask(0o077)
    if args.action == 'prepare':
        require(bool(args.expected_sha) and not args.state, 'Prepare needs exact accepted SHA and no existing state')
        prepare(args)
    else:
        require(bool(args.state), 'Exact private state path required')
        state = load(args.state)
        {'upgrade': upgrade, 'vite': vite, 'close': close}[args.action](state)


if __name__ == '__main__':
    try:
        main()
    except Exception as exc:
        print(f'BLOCKED: {type(exc).__name__}: {exc}', file=sys.stderr)
        sys.exit(1)
