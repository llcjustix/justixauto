#!/usr/bin/env python3
"""One accepted user-local cutover: drain old API, backup, resume Air once.

PREPARATION ARTIFACT. Invocation requires primary acceptance of finished source,
HTTP and CUA evidence. No secret-file reads, direct SQL migration, fixture writes,
automatic recovery, production/Git operations, or Vite/PostgreSQL stop commands.
"""
import argparse
import hashlib
import importlib.util
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import urllib.error
import urllib.request

sys.dont_write_bytecode = True
_spec = importlib.util.spec_from_file_location('warehouse_runtime', Path(__file__).with_name('runtime-control.py'))
rt = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(rt)
ROOT = rt.ROOT
EXECUTION = Path(__file__).resolve().parent
AIR_PID, OLD_API_PID, VITE_PID = 31138, 37081, 26877
USER_CONTAINER = '0ab33deabf37af110b2865227342e8b1c0884e3b624418ecd8e2adb2c18e0572'
USER_IMAGE = 'sha256:6def1cb8d5ffa3443c527419cc13f395ab328c27bf90fcb1e80831aae4103bc3'
DRAINED_RECEIPT = Path('/private/tmp/justixauto-warehouse-local-activation-bkf4lka1/activation.json')
CASE_NAMES = ('mixed', 'rollback', 'identify', 'correction', 'contention', 'race', 'ui-seed')
RECEIPT = {}
DIRECTORY = None


def checkpoint(stage, **values):
    RECEIPT.update(stage=stage, **values)
    if DIRECTORY:
        rt.private_write(DIRECTORY / 'activation.json', RECEIPT)
    print(f'local_activation_stage={stage}', flush=True)


def process(pid):
    out = subprocess.run(['ps', '-p', str(pid), '-o', 'pid=,ppid=,stat=,lstart=,comm='],
                         capture_output=True, text=True)
    if out.returncode != 0 or not out.stdout.strip():
        return None
    parts = out.stdout.strip().split(None, 8)
    rt.require(len(parts) == 9 and int(parts[0]) == pid, 'Unexpected process metadata')
    return {'pid': pid, 'parent': int(parts[1]), 'status': parts[2],
            'started': ' '.join(parts[3:8]), 'executable': parts[8]}


def same_process(expected):
    current = process(expected['pid'])
    rt.require(current is not None and current['started'] == expected['started'] and
               current['executable'] == expected['executable'], 'Process identity changed; refusing to signal')
    return current


def cwd_is(pid, path):
    return 'n' + str(path) in rt.command(['lsof', '-a', '-p', str(pid), '-d', 'cwd', '-Fn']).splitlines()


def listeners(port):
    out = subprocess.run(['lsof', '-nP', f'-iTCP:{port}', '-sTCP:LISTEN', '-FpPn'], capture_output=True, text=True)
    rt.require(out.returncode in (0, 1), 'Listener metadata unavailable')
    result = {}
    pid = None
    for line in out.stdout.splitlines():
        if line.startswith('p'):
            pid = int(line[1:])
            result.setdefault(pid, [])
        elif line.startswith('n') and pid:
            result[pid].append(line[1:])
    return result


def user_container():
    template = '{"id":{{json .Id}},"name":{{json .Name}},"image":{{json .Image}},"running":{{json .State.Running}},"ports":{{json .NetworkSettings.Ports}}}'
    metadata = json.loads(rt.command(['docker', 'inspect', USER_CONTAINER, '--format', template]))
    rt.require(metadata['id'] == USER_CONTAINER and metadata['name'] == '/justixauto-postgres-1' and
               metadata['image'] == USER_IMAGE and metadata['running'], 'User-local PostgreSQL identity changed')
    rt.require(metadata['ports'].get('5432/tcp') == [{'HostIp': '127.0.0.1', 'HostPort': '55433'}], 'User-local PostgreSQL port mapping changed')
    return metadata


def db_metadata(statement):
    # Call sites below contain schema/connection metadata only, never application rows.
    return rt.command(['docker', 'exec', '-i', USER_CONTAINER, 'psql', '-X', '-qAt',
                       '-v', 'ON_ERROR_STOP=1', '-U', 'justixauto', '-d', 'justixauto'],
                       stdin=statement, timeout=10).strip()


def schema():
    return db_metadata("SELECT version::text || ' ' || dirty::text FROM public.schema_migrations;")


def verify_preserved(vite):
    same_process(vite)
    rt.require(cwd_is(VITE_PID, ROOT / 'web/apps/portal'), 'Vite cwd changed')
    rt.require(listeners(5191) == {VITE_PID: ['127.0.0.1:5191']}, 'User Vite listener changed')
    user_container()


def drain_api(expected):
    current = same_process(expected)
    rt.require(current['parent'] == AIR_PID and current['executable'] == str(ROOT / 'var/bin/dev-api'), 'API ownership changed')
    os.kill(expected['pid'], signal.SIGTERM)
    deadline = time.monotonic() + 30
    while time.monotonic() < deadline:
        current = process(expected['pid'])
        if current is not None:
            # Validate this observation; a second ps could race graceful exit.
            rt.require(current['started'] == expected['started'] and current['parent'] == expected['parent'] and
                       (current['status'].startswith('Z') or current['executable'] == expected['executable']),
                       'API process identity changed during drain; refusing further signals')
        # Paused Air cannot reap its child; a zombie has exited and released DB connections.
        exited = current is None or current['status'].startswith('Z')
        if exited and not listeners(8090):
            return
        time.sleep(0.2)
    raise RuntimeError('Owned API did not drain within 30 seconds; no force-kill or migration attempted')


def backup():
    path = DIRECTORY / 'before-schema46.dump'
    with os.fdopen(os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600), 'wb') as archive:
        result = subprocess.run(['docker', 'exec', USER_CONTAINER, 'pg_dump', '-U', 'justixauto', '-d', 'justixauto', '-Fc'],
                                stdout=archive, stderr=subprocess.PIPE, timeout=60)
    rt.private_write(DIRECTORY / 'pg-dump.stderr', result.stderr.decode(errors='replace'))
    rt.require(result.returncode == 0 and path.stat().st_size > 0, 'Fresh pg_dump failed; old API stays drained and Air paused')
    with path.open('rb') as archive:
        listing = subprocess.run(['docker', 'exec', '-i', USER_CONTAINER, 'pg_restore', '--list'],
                                 stdin=archive, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=30)
    rt.private_write(DIRECTORY / 'pg-restore-list.stderr', listing.stderr.decode(errors='replace'))
    rt.require(listing.returncode == 0 and b'schema_migrations' in listing.stdout, 'Archive listing validation failed')
    digest = hashlib.sha256()
    with path.open('rb') as archive:
        for chunk in iter(lambda: archive.read(1024 * 1024), b''):
            digest.update(chunk)
    return {'path': str(path), 'bytes': path.stat().st_size, 'sha256': digest.hexdigest(), 'archiveListValidated': True}


def ready():
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))
    try:
        with opener.open('http://127.0.0.1:8090/readyz', timeout=2) as response:
            return response.status == 204
    except (OSError, urllib.error.URLError):
        return False


def contain_after_resume(air):
    """Contain automatic retries; never roll back, force migration or kill a migrator."""
    try:
        current = same_process(air)
        if not current['status'].startswith('T') and cwd_is(AIR_PID, ROOT):
            os.kill(AIR_PID, signal.SIGSTOP)
        active = listeners(8090)
        if len(active) == 1:
            pid = next(iter(active))
            api = process(pid)
            if api and api['parent'] == AIR_PID and api['executable'] == str(ROOT / 'var/bin/dev-api') and cwd_is(pid, ROOT):
                drain_api(api)
        return 'verified Air paused; owned API drained when identifiable; no migrator/backup restore touched'
    except Exception as exc:
        return 'containment requires primary attention: ' + str(exc)


def main():
    global DIRECTORY
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--go-ahead', required=True)
    parser.add_argument('--state', required=True, help='Private isolated-runtime state with final passing HTTP evidence')
    parser.add_argument('--expected-sha', required=True)
    parser.add_argument('--cua-accepted', action='store_true')
    parser.add_argument('--acceptance-note', required=True, help='Primary-owned evidence/acceptance artifact in this execution directory')
    parser.add_argument('--continue-drained-receipt', help='Exact accepted failed receipt; continue after verified API exit without another SIGTERM')
    args = parser.parse_args()
    rt.require(args.go_ahead == 'warehouse-local-activation' and args.cua_accepted, 'Explicit final source/HTTP/CUA activation authorization required')
    os.umask(0o077)
    state = rt.load(args.state)
    rt.check_source(state)
    rt.require(rt.command(['git', 'rev-parse', '--show-toplevel']).strip() == str(ROOT), 'Wrong repository root')
    rt.require(rt.command(['git', 'rev-parse', 'HEAD']).strip() == args.expected_sha, 'Accepted checkout HEAD changed')
    rt.require(all(state.get('httpCases', {}).get(name) == 'passed' for name in CASE_NAMES), 'Focused HTTP cases are not all passed')
    rt.require(state.get('migrationEvidence') == {'from': 45, 'to': 46, 'originalReceiptRowsUnchanged': True,
               'allOriginalBalancesMatch': True, 'oldWriterDrained': True}, 'Migration rehearsal evidence absent')
    note = Path(args.acceptance_note).resolve()
    rt.require(note.parent == EXECUTION and note.suffix == '.md' and note.is_file(), 'Acceptance note must be a primary-owned task evidence artifact')
    rt.require(hashlib.sha256((ROOT / 'migrations/000046_inventory_receipt_locations.up.sql').read_bytes()).hexdigest() == rt.MIGRATION46_HASH,
               'Accepted migration46 SQL changed')
    candidate = Path(state['directory']) / 'api'
    rt.require(candidate.is_file(), 'Verified runtime API binary absent')
    candidate_hash = hashlib.sha256(candidate.read_bytes()).hexdigest()
    continuation = None
    if args.continue_drained_receipt:
        original_path = Path(args.continue_drained_receipt).resolve()
        rt.require(original_path == DRAINED_RECEIPT and original_path.is_file() and
                   original_path.stat().st_mode & 0o077 == 0, 'Unexpected drained receipt path or permissions')
        original_bytes = original_path.read_bytes()
        previous = json.loads(original_bytes)
        rt.require(previous.get('stage') == 'blocked' and previous.get('airResumeCount') == 0 and
                   previous.get('initialSchema') == '45 false' and not previous.get('backup') and
                   previous.get('error') == 'Process identity changed; refusing to signal',
                   'Receipt is not the accepted pre-backup drain interruption')
        rt.require(previous.get('task') == rt.TASK and previous.get('acceptedSourceSHA') == args.expected_sha and
                   previous.get('acceptedSourceHash') == state['sourceHash'] and
                   previous.get('candidateBinaryHash') == candidate_hash and
                   previous.get('migrationUpHash') == rt.MIGRATION46_HASH and
                   previous.get('acceptanceNote') == str(note) and
                   previous.get('acceptanceNoteHash') == hashlib.sha256(note.read_bytes()).hexdigest(),
                   'Continuation source or acceptance binding changed')
        air, old_api, vite = previous['airBefore'], previous['oldAPIBefore'], previous['viteBefore']
        rt.require(air['pid'] == AIR_PID and vite['pid'] == VITE_PID and old_api['pid'] == OLD_API_PID and
                   old_api['parent'] == AIR_PID and old_api['executable'] == str(ROOT / 'var/bin/dev-api'),
                   'Continuation process identities differ from the accepted resources')
        observed_air, observed_vite = same_process(air), same_process(vite)
        rt.require(observed_air['parent'] == air['parent'] and observed_vite['parent'] == vite['parent'],
                   'Original supervisor or Vite parent changed')
        air = observed_air
        rt.require(process(OLD_API_PID) is None and not listeners(8090), 'Old PID or API listener exists; no repeated shutdown attempted')
        rt.require(user_container() == previous['postgres'], 'Original PostgreSQL metadata changed')
        amendment = EXECUTION / 'activation-amendment.md'
        rt.require(amendment.is_file(), 'Primary continuation amendment is absent')
        continuation = {'path': str(original_path), 'sha256': hashlib.sha256(original_bytes).hexdigest(),
                        'oldAPIAbsent': True, 'shutdownRepeated': False,
                        'amendment': str(amendment), 'amendmentHash': hashlib.sha256(amendment.read_bytes()).hexdigest()}
    else:
        air, old_api, vite = process(AIR_PID), process(OLD_API_PID), process(VITE_PID)
    rt.require(air and air['status'].startswith('T') and air['executable'].endswith('/air') and cwd_is(AIR_PID, ROOT), 'Expected project Air is not paused')
    if not continuation:
        rt.require(old_api and old_api['parent'] == AIR_PID and old_api['executable'] == str(ROOT / 'var/bin/dev-api') and cwd_is(OLD_API_PID, ROOT), 'Expected old API identity changed')
        rt.require(listeners(8090) == {OLD_API_PID: ['127.0.0.1:8090']}, 'Expected single old API listener absent')
    rt.require(vite is not None, 'Expected user Vite process absent')
    verify_preserved(vite)
    rt.require(schema() == '45 false', 'User-local schema is not clean45; obtain a revised activation packet')
    if continuation:
        rt.require(db_metadata("SELECT count(*) FROM pg_stat_activity WHERE datname='justixauto' AND pid<>pg_backend_pid() AND backend_type='client backend';") == '0',
                   'Other user-local DB clients remain; continuation requires no writers')
    DIRECTORY = Path(tempfile.mkdtemp(prefix='justixauto-warehouse-local-activation-', dir='/private/tmp'))
    os.chmod(DIRECTORY, 0o700)
    rt.PRIVATE_DIR = DIRECTORY
    RECEIPT.update(task=rt.TASK, acceptedSourceSHA=args.expected_sha, acceptedSourceHash=state['sourceHash'],
                   acceptanceNote=str(note), acceptanceNoteHash=hashlib.sha256(note.read_bytes()).hexdigest(),
                   candidateBinaryHash=candidate_hash, airBefore=air, oldAPIBefore=old_api,
                   viteBefore=vite, postgres=user_container(), migrationUpHash=rt.MIGRATION46_HASH,
                   initialSchema='45 false', airResumeCount=0)
    if continuation:
        RECEIPT['continuedFrom'] = continuation
    print('private_activation_receipt=' + str(DIRECTORY / 'activation.json'), flush=True)
    checkpoint('verified')
    resumed = False
    try:
        if not continuation:
            drain_api(old_api)
        checkpoint('old_api_drained')
        rt.require(same_process(air)['status'].startswith('T'), 'Air unexpectedly resumed before backup')
        clients = db_metadata("SELECT count(*) FROM pg_stat_activity WHERE datname='justixauto' AND pid<>pg_backend_pid() AND backend_type='client backend';")
        rt.require(clients == '0', 'Other user-local DB clients remain; no migration or connection termination attempted')
        checkpoint('backup_validated', backup=backup())
        rt.require(schema() == '45 false', 'Schema changed during backup')
        verify_preserved(vite)
        rt.check_source(state)
        rt.require(same_process(air)['status'].startswith('T') and not listeners(8090), 'Old API or watcher resumed unexpectedly')
        rt.require(db_metadata("SELECT count(*) FROM pg_stat_activity WHERE datname='justixauto' AND pid<>pg_backend_pid() AND backend_type='client backend';") == '0',
                   'New DB clients appeared before cutover; no migration attempted')
        os.kill(AIR_PID, signal.SIGCONT)
        resumed = True
        checkpoint('air_resumed_once', airResumeCount=1)
        deadline = time.monotonic() + 90
        progress_at = time.monotonic()
        while time.monotonic() < deadline:
            verify_preserved(vite)
            current_schema = schema()
            active = listeners(8090)
            if current_schema == '46 false' and len(active) == 1:
                api = process(next(iter(active)))
                if api and api['parent'] == AIR_PID and api['executable'] == str(ROOT / 'var/bin/dev-api') and cwd_is(api['pid'], ROOT):
                    binary_hash = hashlib.sha256((ROOT / 'var/bin/dev-api').read_bytes()).hexdigest()
                    rt.require(binary_hash == candidate_hash, 'Air API binary differs from the HTTP-verified candidate; stop for diagnosis')
                    if ready():
                        rt.check_source(state)
                        checkpoint('complete', finalSchema=current_schema, newAPI=api, newAPIBinaryHash=binary_hash,
                                   readinessStatus=204, vitePreserved=True, postgresPreserved=True)
                        print('local_api=http://127.0.0.1:8090 portal=http://127.0.0.1:5191 schema=46 readiness=204; backup retained privately')
                        return
            if time.monotonic() - progress_at >= 10:
                checkpoint('waiting_for_air_cutover', observedSchema=current_schema)
                progress_at = time.monotonic()
            time.sleep(1)
        raise RuntimeError('Air did not complete one cutover within 90 seconds; no extra rebuild trigger or retry sent')
    except Exception as exc:
        containment = contain_after_resume(air) if resumed else 'Air remains paused; old API may be drained; primary recovery required'
        checkpoint('blocked', errorType=type(exc).__name__, error=str(exc), containment=containment)
        raise


if __name__ == '__main__':
    try:
        main()
    except Exception as exc:
        print(f'LOCAL_ACTIVATION_BLOCKED: {type(exc).__name__}: {exc}', file=sys.stderr)
        sys.exit(1)
