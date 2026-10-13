# Warehouse bulk movement: local runtime plan

Authority: infrastructure hub `/root/warehouse_bulk_move_runtime`, assigned by
application hub `/root`. Date: 2026-10-13. Application owns migration content,
HTTP contract and fixtures; infrastructure owns execution/isolation. This is an
assigned execution artifact, not a canonical deployment record.

## Scope and current state

Initial assignment was read-only planning. The primary subsequently authorized
one reversible local operation: pause the positively identified Air supervisor
before source edits, leaving its existing API child, Vite and PostgreSQL running.
That operation is complete. Nothing else below has been executed.

Observed checkout: `fix/warehouse-bulk-move-20261013`, HEAD
`4b92239779c5f3be1f7ea563123d8a829aee2fbe`. Root, deploy, internal, tools and
development-doc AGENTS instructions and agent-workflow were read. The final-only
verification policy controls this task. No tests, builds, database creation,
migrations, restarts, application/Git/canonical edits or credential-file reads
were performed during this planning assignment.

| Resource | Observed identity | Disposition |
| --- | --- | --- |
| User portal | Vite PID 26877, cwd `web/apps/portal`, `127.0.0.1:5191` | Running, untouched |
| User API | `var/bin/dev-api`, PID 37081, parent 31138, `127.0.0.1:8090` | Running, untouched |
| API watcher | Air PID 31138, parent 31126, cwd project root | **Paused with SIGSTOP**, state `T` confirmed |
| User database | `justixauto-postgres-1`, container ID `0ab33deabf37af110b2865227342e8b1c0884e3b624418ecd8e2adb2c18e0572`, `127.0.0.1:55433` | Running, untouched |
| Proposed test ports | `127.0.0.1:55443`, `127.0.0.1:8091`, `127.0.0.2:5192` | No listeners at inspection; recheck before use |

The pause verified supervisor executable basename `air`, exact cwd, a sole
direct child at the project's `var/bin/dev-api`, and that child's API listener.
Only PID 31138 received SIGSTOP, never its process group. All three user-service
listeners remained present afterward. Historical PIDs in older startup notes
are stale and must not be signalled.

`.air.toml` watches Go and SQL under `cmd`, `internal`, `migrations`; its command
builds the API **and runs `cmd/migrate up` before restart**. Resuming the watcher
can therefore execute queued source changes and apply new migrations. Keep it
paused through source completion and isolated verification. Vite still serves
changing source through HMR; the old API remains available but cannot offer the
new endpoint until final activation. Primary should communicate that temporary
feature mismatch while the user is checking the app.

Last documented user-local migration state is version 45, clean, from
`../unified-frontend-20261013/result-local-runtime.md`; it was not queried again.
That note's version-34 backup predates later user activity and is not a current
activation backup.

## Isolated final verification environment

After the full backend and frontend change is complete and the primary releases
the writer, reserve one fixture and one browser owner. Pin the candidate SHA or
record exact dirty source hashes; freeze source throughout the final check.
Do not start `make dev`, `make api`, `make test-go`, the absent
`tools/test-go.sh`, or `internal/e2e.DB`. The first two target user-local setup;
the existing E2E helper truncates data and selects obsolete non-public schemas.
The broad Make test target also starts MinIO and runs race tests unnecessarily.

Use one separate, task-labelled PostgreSQL container with tmpfs storage, one
non-watching API process and a second instance of the actual portal Vite server.
No user DB clone, volume attachment or user identity credentials are needed.
The cached PostgreSQL image exactly matches the user-local database image:

```text
docker.io/library/postgres:18.6-alpine3.24@sha256:d3e1620b530c944afa6e887d22eb899824da68e19c52024bf98f5220c88a65b2
local image ID: sha256:6def1cb8d5ffa3443c527419cc13f395ab328c27bf90fcb1e80831aae4103bc3
```

Concrete setup commands for the execution owner (not executed here):

```bash
cd /Users/bakhromachilov/startups/justixauto
umask 077
runtime_dir=$(mktemp -d /private/tmp/justixauto-warehouse-e2e-20261013-XXXXXX)
test_container="justixauto-warehouse-e2e-20261013-${runtime_dir##*-}"
test_image='docker.io/library/postgres:18.6-alpine3.24@sha256:d3e1620b530c944afa6e887d22eb899824da68e19c52024bf98f5220c88a65b2'

# Recheck all three ports and the unique container name before proceeding.
lsof -nP -iTCP:55443 -iTCP:8091 -iTCP:5192 -sTCP:LISTEN
docker container inspect "$test_container" --format '{{.Id}}'
# Continue only when the ports are free and that container does not exist.
docker run --detach --rm --pull=never --name "$test_container" \
  --label justixauto.task=warehouse-bulk-move-20261013 \
  --publish 127.0.0.1:55443:5432 \
  --tmpfs /var/lib/postgresql:rw \
  --env POSTGRES_DB=warehouse_bulk_move_e2e \
  --env POSTGRES_USER=postgres \
  --env POSTGRES_PASSWORD=warehouse-e2e-only \
  "$test_image"
docker exec "$test_container" pg_isready -U postgres -d warehouse_bulk_move_e2e
```

`warehouse-e2e-only` is a newly declared disposable fixture password, never a
user password. Readiness polling must be bounded, with no sleep longer than one
second per poll. Record the new container ID and task label before fixture use.
Docker commands need the scoped local sandbox escalation demonstrated by the
read-only inspection; no cluster or other container is in scope.

Application handoff: the new migration introduces per-receipt warehouse
balances while preserving receipt IDs and original received totals. Include
one populated pre-migration receipt in the isolated migration exercise. The
paused user's current `var/bin/dev-api` is the pre-change executable: copying
it to `$runtime_dir/api-before` preserves a way to seed schema-45 fixtures by
HTTP without ever connecting to the user's DB. Record that binary's SHA-256;
run it with the same isolated environment on 8091 only while seeding, stop that
owned process, then migrate and start the final API. If the old executable is
no longer available, obtain an application-owned synthetic SQL seed matching
schema 45 instead of booting new balance-dependent code against old tables.

Compile the required runtime executables once from the finished source. This is
necessary to run the requested final HTTP/browser scenario, not a separate build
gate. Existing pinned tools and dependencies are present; do not auto-install or
download anything on failure.

```bash
bash tools/go.sh build -o "$runtime_dir/api" ./cmd/api
bash tools/go.sh build -o "$runtime_dir/migrate" ./cmd/migrate
bash tools/go.sh build -o "$runtime_dir/bootstrap-admin" ./cmd/bootstrap-admin
```

Run those executables with cwd `$runtime_dir` (where there is **no `.env`**),
not the checkout. Use an allowlisted subprocess environment rather than copying
the entire parent environment. No production AWS/profile/S3/OTLP values should
enter this process. Required environment:

```text
DATABASE_URL=postgres://postgres:warehouse-e2e-only@127.0.0.1:55443/warehouse_bulk_move_e2e?sslmode=disable
HTTP_ADDR=127.0.0.1:8091
COOKIE_SECURE=false
ALLOWED_ORIGINS=http://127.0.0.2:5192,http://127.0.0.1:8091
FILE_STORAGE=local
DOCUMENTS_DIR=<runtime_dir>/documents
WEB_DIR=
API_DOCS=false
OTEL_EXPORTER_OTLP_ENDPOINT=
SHUTDOWN_DRAIN=0s
```

Use subprocess argument arrays: `[runtime_dir + '/migrate', 'up']`, then
`[runtime_dir + '/migrate', 'version']`; refuse dirty/unexpected version. Spawn
`[runtime_dir + '/api']`, save its exact PID and private log, wait boundedly for
`GET http://127.0.0.1:8091/readyz` to return 204. If the final migration backfills
existing unidentified quantities, the final fixture must additionally exercise
an old-shape populated batch through that migration: application supplies the
pre-migration fixture, and a bounded helper may call the existing
`database.NewMigrator(...).Migrate(45)` before seeding and `.Up()` afterward.
Do not manually fabricate `schema_migrations`, change applied SQL, or use a
user-data copy for this check. Final migration number/content remain pending.

Bootstrap a new synthetic platform administrator only in this disposable DB via
the compiled `bootstrap-admin` command. Supply a newly generated fixture password
on stdin, keep it in the test runner's memory/private artifact, and do not print
it. Create the company/users/roles/inventory via current public HTTP endpoints.
The application hub's finished contract supplies precise seed payloads and
permission cases; the old helper's HTTP shape is a reference, not a reset tool.
Use a fresh cookie jar, GET session to capture CSRF, and current If-Match values.
Assert the test base URL/DB name/port before any mutation. No writes to 8090 or
55433 are authorized for testing.

Start actual Vite from `web/apps/portal` with an allowlisted environment including
`JUSTIX_API=http://127.0.0.1:8091`; `loadEnv` permits that explicit override. Exact
argv, with project root expanded:

```text
<root>/docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node
<root>/web/node_modules/vite/bin/vite.js
--host 127.0.0.2 --port 5192 --strictPort
```

Give this Vite instance a task-private `cacheDir` (a tiny external config wrapper
can import the real portal config and override only `cacheDir`, or use the Vite
Node API). This prevents simultaneous optimizers from writing the user's shared
Vite cache. Preserve real portal entry, React plugin, routes and API proxy. Do
not change the running 5191 instance's proxy or `.env`. Record Vite PID/log.
One concrete private runner, created during final setup at
`$runtime_dir/vite.mjs`, is:

```js
import { createServer } from '/Users/bakhromachilov/startups/justixauto/web/node_modules/vite/dist/node/index.js';
const root = '/Users/bakhromachilov/startups/justixauto';
const server = await createServer({
  configFile: `${root}/web/apps/portal/vite.config.ts`,
  cacheDir: `${process.env.WAREHOUSE_RUNTIME_DIR}/vite-cache`,
  server: { host: '127.0.0.2', port: 5192, strictPort: true },
});
await server.listen();
```

Launch it with the pinned Node executable and
`WAREHOUSE_RUNTIME_DIR=$runtime_dir`, `JUSTIX_API=http://127.0.0.1:8091` in the
allowlisted environment. The imported real config still supplies its API proxy
and plugins; no repository Vite config mutation is needed.

All browser interaction uses **CUA**, per the primary's accepted constraint.
The primary owns a fresh CUA tab at `http://127.0.0.2:5192/`. This different
loopback hostname isolates test cookies from the user's `127.0.0.1:5191`
session; port separation alone would not. Do not open the user's signed-in tab,
copy its cookies, use external Playwright/browser drivers or download a browser.
Browser requests to `/api` stay on 127.0.0.2 and Vite proxies to the isolated
API8091. The API accepts `http://127.0.0.2:5192` as an explicit origin. HTTP
fixture scripts may use 8091 with their own independent cookie jar; the browser
signs in separately using only the synthetic test account. If CUA cannot reach
this loopback host, report the concrete block rather than mocking API results.

Final evidence should cover the application's minimal accepted matrix: one
atomic mixed VIN/unidentified movement with partial batch quantity; source and
destination counts plus subsequent destination identification and preserved
provenance; rollback under invalid/stale/capacity/tenant/permission cases; manual
and select-N VIN selection plus unidentified quantity entry; successful dialog
completion/refresh and retained draft on an error. Use actual Vite at
`127.0.0.2:5192` through CUA and
the actual API. Keep synthetic IDs, statuses and assertion counts in the receipt;
redact passwords, session cookies and CSRF values. No broad checks or repeated
successful scenarios. Repair and rerun only failed affected scenarios.

## User-local activation after the final pass

Standing local-run authorization permits restoring the updated local app, but
the primary must accept the final migration SQL and this named scope before
execution: this project, existing `justixauto-postgres-1` / DB `justixauto`,
API `127.0.0.1:8090`, Vite `127.0.0.1:5191`; forward migration, existing API
restart and readiness only. No cluster, Git promotion, live fixture seed, reset,
down migration, schema force, user-account change or production operation.

1. Reverify container ID/image/port and Air PID/comm/cwd/child ownership. Recheck
   `public.schema_migrations` read-only inside the named container using
   `psql -X -U justixauto -d justixauto -At -v ON_ERROR_STOP=1`; compare version
   and dirty flag with the migration intended for activation. Do not query
   credential or other application row values.
2. Make a **fresh private** backup before any migration. Through subprocess
   stdout directed to a mode-0600 file, execute:
   `docker exec justixauto-postgres-1 pg_dump -U justixauto -d justixauto -Fc`.
   Check exit success, nonzero size and SHA-256, and validate archive listing
   with the same container's `pg_restore --list` through stdin (capture rather
   than print its output). Keep backup path/hash in the receipt. This avoids
   reading `.env`, password files or database credentials. Backup contents must
   never be printed or committed. The older version-34 backup is insufficient.
3. Record whether the final migration is transactional/additive and safe while
   the old API serves. If it changes tables in an incompatible way, the primary
   must accept a brief API interruption and drain the verified API child before
   applying it. The final SQL is not yet known, so uninterrupted activation is
   not promised. Do not undo user writes to make a rollback appear safe.
4. Prefer the repository's existing Air migration-before-restart path, after
   backup and final acceptance: reverify PID 31138 and cwd; then
   `kill -CONT 31138` **only if it is still this paused supervisor**. Queued
   filesystem events should trigger the final build, migration and restart.
   Wait boundedly, check the new API listener's owner, the clean expected schema
   version and `/readyz` 204. Vite already sees final source. No warehouse/user
   data mutation is required to verify activation.
5. If Air does not consume queued changes, do not repeatedly touch source or
   fire extra watchers. Use one explicitly scoped manual activation: run the
   finished migration binary with checkout cwd so the normal application loader
   consumes the existing `.env` without exposing it, then restart only the
   identified project API under a single owner. Do not run `make stop`, which
   also terminates Vite; do not kill all Go/Node processes. Final process-control
   argv depends on the reverified supervisor tree and must be recorded before
   execution. No new user permission is needed solely because a documented
   scoped local startup method differs, but primary acceptance remains required.

If migration fails/turns dirty, leave the backup and logs intact, stop automatic
retries, report the exact version/error with sensitive fields redacted, and
return to the application hub for migration-content diagnosis. Do not force the
version, run down, restore over current user data or apply an ad hoc SQL fix.

## Cleanup and concrete limits

- The paused watcher is an outstanding operational obligation. Do not end the
  overall task with Air silently paused. Before `kill -CONT 31138`, revalidate
  exact PID identity and ensure the final activation prerequisites are met;
  never resume merely as generic cleanup while unvalidated migrations exist.
- For isolated cleanup, verify owned API/Vite PIDs/cwd/ports before SIGTERM;
  wait for exit, have the primary close the owned CUA tab, and stop only the recorded
  disposable container ID after verifying its task label and name. `--rm`
  removes that container's tmpfs-backed state. Do not use compose down, prune,
  volume deletion, unqualified pkill or any user-local service cleanup.
- Preserve the private backup and final evidence; do not delete user files or
  credential artifacts. No secret file, including `test_creds.txt` or an admin
  credential artifact, was read or is needed by this plan.
- Remaining concrete inputs: finished migration number/SQL/backfill contract,
  exact final HTTP/fixture matrix, final source checkpoint and writer release.
  No external package or image installation is needed based on inspection.
  Docker/process control may require sandbox escalation;
  read-only Docker/process escalation succeeded during planning.
