# Runtime and HTTP execution receipt

Owner `/root/warehouse_bulk_move_runtime`, 2026-10-13UTC / 2026-10-14 local.
**HTTP/CUA ACCEPTED; USER-LOCAL ACTIVATION COMPLETE; ISOLATED RUNTIME CLOSED.**
All seven prepared HTTP cases passed; primary acceptance is in
`final-validation.md`. User-local schema is clean46, Air31138 is resumed,
API86995 serves127.0.0.1:8090 with readiness204, and Vite26877/5191 plus the
original PostgreSQL55433 container were preserved. No user-local test data was
seeded or reset. A fresh pre-migration backup is retained privately.

Private state/evidence: `/private/tmp/justixauto-warehouse-e2e-l6dz41m1/state.json`
and adjacent `http-evidence.json` (mode600, directory mode700). Synthetic
credentials remain there and are omitted from this receipt. Former isolated services:
PostgreSQL55443, API8091, Vite bound only127.0.0.1:5192 and accessed through
**http://localhost:5192/**. API readiness204 and Vite document response200
were observed. Vite's first readiness probe omitted `Accept: text/html`, causing
the real document-policy middleware to return404 while the server was working.
A correctly headed request returned200 text/html; the owned helper was corrected.
The running server was retained for CUA, without a restart or fixture replay.
After accepted local activation, only these isolated resources were closed;
private state now records `closed`, no remaining owned processes or container.

## Source and retained failure provenance

Primary first released source manifest
`50a657849302ea6c9612173ede6229950241a6ea9c2a23d83780353497d40e18` and authorized
the isolated final pass. First prepare stopped at port preflight before any
private state/build/resource creation: macOS cannot bind 127.0.0.2 (errno49).
Primary accepted binding Vite only to **127.0.0.1:5192**, accessed by CUA as
**http://localhost:5192** with that explicit allowed origin. Host-only cookies
remain separate from the user's 127.0.0.1 session. No network alias/broad bind.
Only that failed preflight was rerun; no successful test/fixture was repeated.

Schema45 legacy seeding and the46 upgrade rehearsal passed. The isolated old API
was drained before backfill; every original receipt row remained byte-equal and
every original location balance matched. Initial mixed movement at18:53:34UTC
succeeded: two VINs, three manual pending units and two order pending units.
Its exact replay returned409 `placement_changed` instead of412 `stale_revision`.
Execution stopped immediately and the failure remains in the private evidence.

Primary approved the source-revision precedence repair and refroze only the
affected service change. Final manifest SHA256:
`c687fd6d2a4ae9b110c8aa0b7c6b5d4135703b033226046b710fd3e096df668a`.
Checkout HEAD: `4b92239779c5f3be1f7ea563123d8a829aee2fbe`.
Controller source fingerprint changed from
`367a47248fc64bd9c50095a35432a0fb2f134a0cd2d17701cf40d07dc8e329d7` to
`702b9507129575c8d6f5cf35ae9f87e08fe5738ed725272959573e2c40f41801`.
Only the isolated API was drained, rebuilt and restarted. Database, schema,
fixture and initial successful transfer were preserved. The authorized
`http-resume-mixed.py` continuation verified recorded200/409 provenance, then
observed412 on the repaired replay with unchanged inventory/commerce fingerprints.
It executed the remaining onward quantity and legacy VIN movement for the first
time. The other six cases then ran once; final HTTP completion18:56:40UTC.

Assigned files delivered: `runtime-control.py`, `http-check.py`,
`http-resume-mixed.py`, `local-activation.py`, `runtime-private.example.json`,
this receipt and earlier `runtime-plan.md`. No application source/Git/canonical
changes were made by this runtime owner.
The accepted design SHA is
`d6499298491722f2fb1ce507848b8a09e74f88476ac6f85ed1f5830393addafe`.
Controller pins the accepted migration46 up-file SHA
`af845b1de029f60e84b05a92afd9c5027d923e4867befbe9070a271c47e0e188`;
changed SQL requires renewed application acceptance before runtime execution.

## Controller commands and lifecycle

The following sequence documents the executed flow; do not rerun it against
the existing fixture. The replay-specific continuation described above replaced
only the failed replay and still-unstarted remainder of `mixed`.

Run from the project root. Each step stops on failure; do not blindly chain or
rerun successful mutation scenarios. `--go-ahead` is an execution interlock,
not evidence of user/hub authorization. The primary's final go-ahead and exact
finished-source checkpoint must exist first. Commands require local Docker/
process/socket/compiler sandbox access; no network downloads or browser driver.

```bash
python3 docs/justix-auto/dev/execution/warehouse-bulk-move-20261013/runtime-control.py prepare \
  --go-ahead warehouse-bulk-move-final --expected-sha <accepted-HEAD>
```

The command prints only its new private state path and progress. Use that exact
path, never the example JSON or the user's `.env`/credential files:

```bash
python3 docs/justix-auto/dev/execution/warehouse-bulk-move-20261013/http-check.py seed-legacy \
  --go-ahead warehouse-bulk-move-final --state <private-state-path>
python3 docs/justix-auto/dev/execution/warehouse-bulk-move-20261013/runtime-control.py upgrade \
  --go-ahead warehouse-bulk-move-final --state <private-state-path>
python3 docs/justix-auto/dev/execution/warehouse-bulk-move-20261013/http-check.py check \
  --go-ahead warehouse-bulk-move-final --state <private-state-path>
python3 docs/justix-auto/dev/execution/warehouse-bulk-move-20261013/runtime-control.py vite \
  --go-ahead warehouse-bulk-move-final --state <private-state-path>
```

`prepare` verifies the paused project watcher and old API, snapshots the old
executable, pins source hashes, compiles runtime binaries privately and starts
only a cached-image disposable PostgreSQL55443 plus the old API8091 at schema45.
A private Go overlay provides a tiny entrypoint using existing
`database.NewMigrator.Migrate(45)/Up()/Version()`. Repository source is untouched;
no SQL migration ledger is fabricated. Compile logs and failed command
stdout/stderr are retained privately without printing process environments.

`seed-legacy` uses public HTTP onboarding/verification, company roles/users,
manual receipts and a normal direct-order quantity shipment. It requires no
contract/payment workaround. Newly generated fixture passwords remain private.
`upgrade` drains the isolated old API, applies46, compares all original receipt
rows unchanged and validates every original location balance, then starts the
finished API. Runtime cwd is private and contains no `.env`.

The primary opens a fresh **CUA** tab at `http://localhost:5192/`; the different
host separates cookies from the user's `127.0.0.1` session. Vite imports the
real portal config, uses a private cache and proxies only to API8091. The private
`uiFixture` gives source/destination/batch/order IDs and the `Buyer` account key.
No external Playwright/browser automation or browser installation is used.

After CUA evidence is captured and the primary closes its owned tab:

```bash
python3 docs/justix-auto/dev/execution/warehouse-bulk-move-20261013/runtime-control.py close \
  --go-ahead warehouse-bulk-move-final --state <private-state-path>
```

Completed cleanup verified exact isolated PIDs/start signatures and container ID/task label,
uses graceful termination, and preserves private evidence. It does **not** resume
Air or signal/stop the user's API/Vite/PostgreSQL. Its initial success message
contained obsolete wording that Air remained paused; final metadata confirmed
Air was resumed, and this owned helper's message was corrected without rerunning
cleanup.

## Passed coverage and limits

The driver has seven named cases: `mixed`, `rollback`, `identify`, `correction`,
`contention`, `race`, `ui-seed`. All are passed, with the precisely scoped source
repair and continuation provenance above retained rather than rewritten.
`--case <name>` runs one not-yet-started case after preceding dependencies; it is
not a blanket retry mechanism. Started/failed mutations require examination of
private evidence and an affected repair plan before another write. Passing
cases cannot be silently rerun on mutated fixtures.

- Mixed two-VIN/two-batch partial movement, exact occupancy/local/global counts,
  one revision touch, unchanged receipt/specification/color/ownership/custody
  fields, movement facts, unchanged commerce tables, replay rejection, quantity
  onward move and old single-VIN endpoint preserving receipt provenance.
- Twenty-one invalid/stale/capacity/local-quantity/tenant/permission/shape cases,
  each comparing complete inventory/commerce table fingerprints before/after.
  These SQL reads address only the exact disposable container; no identity
  password/session rows are queried. Fingerprints include facts and revisions.
  Cases include the added duplicate receipt UUID alias rejection422.
- Destination VIN identification, legacy original-location identification,
  drained-origin order progress, unchanged occupancy, original order/receipt
  identity and destination location discovery; no fabricated correction warning.
- Selected-location correction after a move and whole-state rollback when a
  global decrease would exceed that location's pending balance.
- Bounded target-warehouse-first, selected unit, placement and receipt-root
  row locks: expected `409 stock_busy` within two seconds with no state changes.
  Lock holders use a four-second lifetime and six-second statement timeout.
  One synchronized move/identify last-unit race checks exactly one success,
  conservation, capacity and no fact from a losing move. The race returned
  identify200 and move412 `stale_revision`.
- Fresh CUA stock with unsorted four-VIN input and two pending batches, plus the
  original order's moved pending receipt for order-origin identification UI.

The executed driver used the actual public HTTP DTOs and SQL implementation,
including public onboarding and a normal direct-order quantity shipment; no
business-data SQL bypass or `internal/e2e` helper was used. Original failure,
repair continuation and passing cases remain in private `http-evidence.json`;
service/compile logs remain private. Passwords/cookies/CSRF are excluded from
HTTP evidence. This is a task-specific harness, not a test-framework change.
No unit/lint/typecheck/race/vet suites were run. Runtime compilation served the
approved final HTTP pass; only the affected API was rebuilt after repair.

Explicit exclusions: populated reservation/consignment variants are not seeded
(the ordinary VIN flags are compared); UI selection, keyboard/mobile/error/draft
behavior is primary-owned CUA; transport-drop simulation and request maximum
1000/10000 boundary fixtures are not automated. Target-first contention uses a
controlled matching warehouse row lock, not an unrelated commercial transfer
workflow. No release, production or user-local business-data mutation is covered.

## User-local activation and bounded recovery

The accepted migration46 adds/backfills per-receipt warehouse balances. **Drain
the positively identified old API before backfill**, even though the migration
is additive: old writers do not maintain balances. This supersedes the earlier
runtime-plan suggestion that old API service might continue during migration.

The accepted procedure required the primary/runtime owner to reverify named
user-local process/container identities, gracefully drain old API37081 while
Air31138 is still paused, and take/validate a fresh private pg_dump plus schema
version/dirty metadata. Preserve the backup privately. Then resume the verified
Air supervisor once to run its existing build → migrate → restart path. Confirm
expected clean schema46, a single current API8090 listener and `/readyz`204;
preserve Vite5191 and all original database data. No reset/down/force/version
bypass, live fixture seeding or automatic backup restore. Exact PID reuse or
failed migration blocks activation for diagnosis. The controller deliberately
has no user-local activation subcommand. The separate prepared
`local-activation.py` required the primary's explicit activation authorization,
CUA acceptance note and exact accepted HEAD. It verifies preserved process and
container identities, captures a fresh private backup, resumes Air once and
requires the rebuilt API to match the HTTP-verified binary. Execution occurred
before any Git commit, preserving the source and Go VCS build metadata binding.

The first activation stopped at the process-identity guard while the old API
exited. Original receipt is immutable at
`/private/tmp/justixauto-warehouse-local-activation-bkf4lka1/activation.json`,
SHA256 `97cdb9a95e950a53027f5235e6297c64307c3b17bb1dd53f1342c25c730774c1`.
It recorded clean45, resume count0 and no backup. Read-only inspection then
confirmed old PID37081 absent, no8090 listener, zero other database clients,
Air still paused and original Vite/PostgreSQL identities unchanged. A graceful
exit between the drain loop's two process observations is the likely cause;
the original script did not record a signal trace, so this cannot be proven
retrospectively. No migration or Air resume occurred in that failed attempt.

Primary authorized `activation-amendment.md`, SHA256
`7a3ad542eeaf1b7ef67d99be0321d294d4c544936c7ed2c13855b3ffa6a0a28c`.
The revised `local-activation.py`, SHA256
`b56a8f0267880896986585b726fb008a8a8dee31f684b77f906e512d948a0880`,
validates each observed process without a second racing lookup. Its explicit
`--continue-drained-receipt` binds the original receipt/hash, acceptance/source/
HEAD, original Air/Vite/PostgreSQL identities and absent old PID/listener,
clean45 and zero database clients. It did not repeat SIGTERM. A new receipt
links the original failure and amendment while preserving both unchanged.

Successful continuation receipt:
`/private/tmp/justixauto-warehouse-local-activation-lo_tvf7q/activation.json`.
Fresh custom-format backup:
`/private/tmp/justixauto-warehouse-local-activation-lo_tvf7q/before-schema46.dump`,
224540 bytes, mode600 within a mode700 private directory; archive listing
validated. Backup SHA256:
`f940ef2ca27eca680f9567b0457e7df314bc2c8ab0c59a158b81ec3b539fdfae`.
Air resumed exactly once; migration changed clean45→clean46. New API PID86995
is a child of Air31138 and is the sole127.0.0.1:8090 listener; `/readyz` returned204.
Its binary equals the HTTP-verified candidate, SHA256
`eccf6c5d387e05d5be99543853b653ce98b9db9c08ed4f63a0c15bb48ba68862`.
Vite PID26877 remains the127.0.0.1:5191 listener with its original start time;
the named original PostgreSQL container and port mapping remain unchanged.
Final post-cleanup metadata observed Air31138 state `S`, not paused.

No forced/down migration, database reset, restore, live fixture write, unrelated
process signal, production action or Git change occurred. Private backups and
all original/final evidence remain available; isolated test services are closed.
