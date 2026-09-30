# LOCAL-STOP-1 — stop local app processes safely

One implementation packet; no canonical backlog addition. Authority: application
hub `/root`, DevOps `/root/local_db_cleanup`. Approved contract:
`docs/justix-auto/dev/local-stop-command-20260930.md` (Approved scope and Execution
and verification). No product-rule or open-decision IDs apply.

Branch `infra/local-stop-command`; source base
`3cc124c5a5858a5eaa2991edafc2771ee7232966`, carried forward from ancestor dev
`0e74dac4258402747e3a5cdc748a671112cf2fd0`. Branch preparation is complete, as
reported by the owning hub. Slicer performed no Git operations.

## Ownership and context

Worker owns exactly `Makefile`, `README.md`, new `tools/stop-apps.py`, new
`tools/test_stop_apps.py`, and its result artifact
`docs/justix-auto/dev/execution/local-stop-command-20260930/result.md`.
Do not edit this packet, canonical records, start-command behavior, app code,
configuration, dependency files, or unrelated dirty files.

Read root `AGENTS.md`, `tools/AGENTS.md`,
`docs/justix-auto/dev/agent-workflow.md` and
`docs/justix-auto/dev/AGENTS.md`, then the approved contract and this packet.
Relevant source slices: Makefile setup/run/help; README Quick start; `.air.toml`;
`tools/go.sh`; `internal/config/config.go` HTTPAddr and shutdown fields;
`cmd/api/main.go` shutdown handling; four `web/apps/*/package.json` dev scripts
and `vite.config.ts` server-port declarations. Do not read full product specs,
board, mocks or secrets. Context budget: 10,000 input tokens; implementation
should fit one short worker turn. If it requires a process-manager framework or
broader launch changes, stop and return `RESLICE_REQUIRED` to the hub.

Dependencies: approved contract and source-identical branch handoff above;
existing Python 3 and local process-discovery utilities, no installation.
Locks: worker holds sole implementation-writer lock for these owned paths;
existing live development session `59982`, app ports and PostgreSQL are reserved
to their current operators and must remain running. No browser/DB fixture lock
is needed. No children or Git operations. Use `apply_patch`; preserve user edits.
Run the mandated repository-root guard `bash tools/check-git.sh` before source
edits. This is a safety prerequisite, not a development verification suite.

## Behavior and acceptance coverage

| ID | Acceptance | Evidence in this packet |
|---|---|---|
| STOP-1 | `make stop` invokes the Python helper; target is phony and appears in help. README Quick start explains stop, database preservation, and unrelated-port refusal. | Required bounded Makefile/README edits, described in worker result; no text-mirroring tests. |
| STOP-2 | Find current repo-owned API and four Vite apps, including existing `make dev` Air/concurrently hierarchy and standalone `make api`/`make web`, without relying on newly recorded PIDs. | Mocked legacy full-tree and standalone discovery/stop unit cases. |
| STOP-3 | Ownership requires current UID, exact real repo/app cwd, known entrypoint/argv and validated ancestry as applicable; a matching name, cwd prefix, port or ancestor alone is insufficient. Never signal user/interactive shells, DB/Docker, other repos or unrelated processes, or whole process groups. | Mocked foreign UID/cwd/same-name/port occupants, shell and DB exclusions, cwd-prefix trap. |
| STOP-4 | Signal recognized restart supervisors first, then remaining verified API/Vite/launch wrappers; graceful SIGTERM, bounded wait, SIGKILL only for still-identical verified owned survivors. Revalidate identity before each signal and tolerate processes exiting concurrently. | Mocked supervisor ordering, surviving descendants, timeout/escalation, stale/reused PID and already-exited cases. |
| STOP-5 | Expected API port follows nonempty inherited `HTTP_ADDR`, then safely parsed root `.env`, then `127.0.0.1:8080`; also cover fixed 5191–5194 and observed owned listening ports. Never execute `.env` or expose its other values. | Pure config/discovery unit cases for precedence, quoted/plain values, missing file, invalid address, IPv6 address, ignored shell syntax, observed alternate owned port. |
| STOP-6 | Idempotent success when no owned apps and required ports free. Report unrelated occupants without signaling them; fail clearly if expected ports or owned processes remain, ownership discovery fails, or a required signal is denied. | Mocked empty/repeated stop, unrelated occupancy, missing/failed discovery, permission denial and surviving-port cases. |

STOP-1 through STOP-6 cover all approved scope bullets. No cross-packet shared
interface exists. There are zero independent QA/review/integration packets:
the latest explicit user policy permits only affected unit tests, once per
relevant change, and supersedes historical verification gates. Do not run live
stop, build, lint, typecheck, integration, browser, or broad test commands.

## Minimal implementation recommendation

Use Python standard-library functions with a small main guard and ordinary
mockable discovery, signal and sleep calls. No dependency, PID registry, daemon,
new supervisor or generic process-manager abstraction. Use explicit argv arrays
for process discovery, never shell interpolation. Fail closed on insufficient
identity evidence. Keep discovery output internal; user-facing diagnostics need
only useful app/PID/port information, never entire commands or environment data.

Current launch identities come from the source above: concurrently invokes Air
through the Go tool wrapper and four npm workspaces; Air runs
`var/bin/dev-api`; standalone API uses `go run ./cmd/api`; Vite workspaces run
`vite --host 127.0.0.1`. The wrapper execs the pinned Go compiler. Recognize
validated non-shell launch wrappers where necessary and let caller shells/make
parents return normally. Do not indiscriminately kill every descendant.
Rediscover surviving recognized children so supervisor exit does not strand
listeners or permit restart loops; retained identity evidence must remain valid.

Keep the deadline predictable: a fixed documented grace ceiling (for example
20 seconds) is acceptable; no full Go-duration parser is required. API shutdown
has a 10-second shutdown timeout plus optional `SHUTDOWN_DRAIN`; existing Air
has its own 5-second kill delay. Do not promise an unlimited graceful drain or
stronger guarantees than existing supervisors provide. Identity checks reduce
PID-reuse risk but cannot make separate OS inspection and signal calls atomic.

Safely parse only the needed `HTTP_ADDR` assignment from `.env`, without sourcing,
evaluating substitutions, or printing file content. Ports assist discovery and
post-stop reporting; they never authorize termination by themselves. If an
unsupported config syntax cannot be interpreted safely, return a clear error
instead of choosing an arbitrary kill target.

## Unit execution and stop conditions

Run only once for the relevant completed change:

```sh
python3 -m unittest discover -s tools -p 'test_stop_apps.py'
```

Tests import the helper without running its CLI. Mock process discovery,
signals, time and environment/file reads as needed; do not discover or signal
real processes, wait real grace periods, bind ports, start children/apps or touch
PostgreSQL. Use compact representative fixtures, including legacy supervisor
and standalone trees. These are behavioral tests, not code/README text mirrors.

Stop and report to the hub on source/contract mismatch, unexpected dirty owned
paths, missing process-discovery primitives, ambiguous ownership requiring broad
signals, necessity for live shutdown verification, or new product/runtime policy.
Do not expand ownership or resolve policy from mock fixtures. A failure may be
fixed within this packet with only newly affected unit tests; no repeated passing
checks or automatic review/QA chain.

Write `result.md` with changed paths, requirement coverage, exact unit command and
result, limitations, and explicit confirmation that no live stop was executed.
Unit evidence does not claim live process-release or release readiness.

## Source SHA-256 manifest

Hashes were read after branch preparation; new helper/test/result files do not
yet exist. Hashes pin inputs, not approval of later worker edits.

```text
a222419ecca4935c906454884eb62d6cacace6a5b80395c90a1cee870f8b8524  AGENTS.md
361f5e895fc93d6bb232941ae5f173d868b6ce4038f1b4810138bed8c4c631f6  tools/AGENTS.md
b2d5153012dfec17e20a5e55a65af858d693cae27adb66ef6a5c440599149b27  docs/justix-auto/dev/AGENTS.md
4f264fb5217a9613b00df455b0661cb9b955e90793a29b17c682cbcd0010830b  docs/justix-auto/dev/agent-workflow.md
04ee26799bd5a0c4f0ca03f43f7ab6b7971cc6e4b8ba092838acd5ab65eb89d8  docs/justix-auto/dev/local-stop-command-20260930.md
1cdc3b359e216cc78b002ecc1724e8f6ce438319cdf60cc339b563288aff73ae  Makefile
04ca6b21776bb4882883d823f7ed168c049d09af301f2cc960a8e24f888a66af  README.md
87e14c4b41327336e57e184c85d8c89393c87e7f759b6e41bf132020444f9480  .air.toml
a2ca8e48f3cd00fa4eb4bc560b394726e52ca481853e738ce1c7f671726294af  tools/go.sh
121a9a470024884e70deaa9a67e399c2afb92c2b89f6e0c7699151816389f6ef  internal/config/config.go
c4430395e810edaaffdb24b43e485b38efb57369701a62d8113ebd25d87e0735  cmd/api/main.go
081a98f5f8e7730c6a11a51d313641f80923ff840ba59638b64110979d0bc51d  web/package.json
a5180f90c87bb001727681028ae4254e8bd41cb20b04b0e3cbadf5813cd8ce2c  web/apps/realization/package.json
a5c701bbfa568ef1b49823e8dc2f277bff1e8596013c7b2c588d35a86c4d5418  web/apps/financing/package.json
5a8bdbef7d49617121b095ea21395ce23c2b34c8f428a29d3535875efa81bb9c  web/apps/insurance/package.json
44d5556f1b42e88bb9aa075b478564a26be1b92fff34197dbceacac74861b7fe  web/apps/admin/package.json
850a4750e263aa3fc6e3649ffaa7cdc1dd3a98b5400ace6f8039e44d72705a9e  web/apps/realization/vite.config.ts
1af498ac503eba81df15728dae1d8e28f3aef41b4df814f66fdf8683b0ed1500  web/apps/financing/vite.config.ts
98795820e80f097a0f7d691e6fbf96b01055517541e3efe78472b19e3d04e984  web/apps/insurance/vite.config.ts
5d8ab92e113717f5aa915e361b5073fa56f11f0d7499e143ca39e6bc0e4912c8  web/apps/admin/vite.config.ts
```
