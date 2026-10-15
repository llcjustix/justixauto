# Test-server administrator bootstrap

## Assignment and approval gate

- Execution-local task ID: `ADMIN-BOOTSTRAP-20261015`; this is not a new canonical backlog entry.
- Owning hub: application/delivery (`/root`). Infrastructure authority: `devops_orchestrator`; both must accept the runtime/deployment contract below.
- Status: joint runtime/deployment contract accepted by owning hub and infrastructure authority in dispatch follow-up. Owning hub will record `docs/justix-auto/adr-16-dev-admin-bootstrap.md`; that ADR is outside slicer/worker ownership. No live deployment is executed by this packet.
- Owned slicer artifact: `docs/justix-auto/dev/execution/admin-bootstrap-20261015/packet.md` only.
- Supplied base HEAD: `f91daa8e624da44f242a53c6677f127dcf0c87d8`, branch `fix/accounting-queue-ui-20261014`. Supplied dev/origin-dev ancestor: `4fa683ba4708bb71031534260ec5cb8742ce9977`. These are hub-provided facts, not independently checked with Git by the slicer. Preserve the current branch dependency; branch selection belongs to the hub/version_control.
- User request relayed by owning hub: seed admin credentials on the test server during Docker CI/CD if no admin user exists.
- Contract paths: this packet; `docs/justix-auto/adr-16-dev-admin-bootstrap.md` (root-owned, to be recorded); `AGENTS.md`; `docs/justix-auto/dev/agent-workflow.md`; `internal/AGENTS.md`; `deploy/AGENTS.md`; `tools/AGENTS.md`. No existing approved contract is edited by this packet.
- Canonical product/open-decision IDs were not supplied. The local requirement IDs below identify only this request. Both hubs accepted the existing active `platform_admin` predicate. Do not infer policy from mock/demo data.
- Approved GitHub secret names: `JUSTIX_AUTO_BOOTSTRAP_ADMIN_LOGIN`, `JUSTIX_AUTO_BOOTSTRAP_ADMIN_EMAIL`, `JUSTIX_AUTO_BOOTSTRAP_ADMIN_NAME`, `JUSTIX_AUTO_BOOTSTRAP_ADMIN_PASSWORD`. Name is optional at the workflow boundary, defaulting to `Platform Administrator`; login/email/password are required only when creation is needed. The workflow passes the resolved name explicitly to the CLI. The bootstrap container receives only `DATABASE_URL` from the existing deployment dotenv, not its full shared environment.

## Targeted findings

`cmd/bootstrap-admin/main.go` currently reads stdin before loading configuration or querying the database. `User.Bootstrap` in `internal/modules/identity/service/user.go:480` validates and hashes credentials, then locks the platform-admin role, counts active platform admins, checks login/email collisions, creates an active user and role assignment, and audits `user.bootstrapped` within one transaction. Existing admins produce `already_bootstrapped`; collisions produce `user_exists`. `internal/modules/identity/repository/role.go:164` defines the existing guard and active-admin predicate. This is the authoritative creation path.

`deploy/Dockerfile` describes a bootstrap command but currently builds only API and migration binaries. Dev Compose has API and migration services sharing the SHA-tagged image, external `justix-db` network and `.env`. The workflow executes migrate before API recreation. Bootstrap belongs between those steps. No fixture or credentials file was opened; preserve unread untracked `test_creds.txt`.

## Proposed acceptance contract

| ID | Required behavior | Evidence |
| --- | --- | --- |
| AB-01 | Existing CLI default stays strict. Add opt-in `-if-missing`; when an active platform admin exists, exit successfully before reading stdin or requiring creation fields. Do not modify any existing account, password, role, status, session or audit history. | V1 existing-admin and strict-mode scenarios |
| AB-02 | With no active platform admin, explicit login/email, resolved display name and stdin password are required and existing service validation applies. Workflow name defaults to `Platform Administrator`; CLI receives it explicitly. Successful creation makes one active platform admin with a password hash and one bootstrap audit event. No default login/email/password, demo credentials, implicit promotion or password reset. | V1 first creation and missing/invalid input scenarios |
| AB-03 | Existing transactional `Bootstrap` remains the only creation path and race authority. Opt-in mode treats only typed `already_bootstrapped` as successful concurrent skip. Database/configuration/validation errors and login/email collisions fail. Strict mode continues to fail on an existing admin. | V1 concurrent creation, collision and failure scenarios |
| AB-04 | The application image includes `/app/bootstrap-admin`. Dev Compose exposes a profile-isolated, explicitly invoked one-shot bootstrap service using the same SHA image, database configuration and `justix-db` network as migration/API. It is not started by ordinary API startup and does not restart. | V2 image/Compose inspection |
| AB-05 | Dev workflow sequence is exact-image preparation → migrate → bootstrap → API deployment. Bootstrap failure prevents API deployment; no success override or error suppression. It runs on subsequent dev deployments too. | V2 workflow inspection and V1 rerun |
| AB-06 | Only the bootstrap workflow step receives credential secret references. Password travels from a step-scoped environment variable through quoted `printf` to container stdin, never a CLI password flag, container environment variable, generated `.env`, image layer, log or API environment. Creation fields are passed explicitly without shell injection. Empty/missing secrets may reach the CLI so an existing-admin skip works without them; absence fails when creation is needed. | V2 secret-flow inspection; V1 stdin/empty-secret skip |
| AB-07 | Dev-only operational documentation explains secret identifiers, predicate, first run, no-op reruns, failure handling and lack of credential reset. ADR/decision acceptance records infrastructure guarantee and test-server scope. | V2 documentation inspection plus hub decision |

## I1 — One-shot conditional admin bootstrap during dev deployment

One coherent behavior: make the existing privileged bootstrap command safely rerunnable as a required, isolated dev-deployment step. This is one implementation packet, not separate CLI and infrastructure tasks.

**Dependencies:** joint contract accepted as recorded above; root records ADR-16; owning hub resolves branch/base dependency and reserves writer; source hashes below remain current or hub explicitly reconciles changes. Secret values are provisioned outside the packet. No permission to inspect `test_creds.txt` is implied.

**Exact implementation-owned paths:**

- `cmd/bootstrap-admin/main.go`
- `internal/modules/identity/service/user.go` — narrow read-only `HasActivePlatformAdmin(ctx)` using the existing service repository interface; preserve strict `Bootstrap` behavior.
- `deploy/Dockerfile`
- `deploy/dev/compose.yml`
- `.github/workflows/deploy-dev.yml`
- `deploy/dev/README.md` (new)

No repository, schema, service-port, public route, unrelated workflow cleanup or dependency changes are authorized. An ADR is not worker-owned until its exact path is assigned separately by the owning hub. Do not edit application code outside this list, canonical board or approved contracts. Do not write permanent tests automatically.

**Implementation outline:**

1. Run the required repository-root guard before application changes. Parse the new flag. In opt-in mode initialize the existing identity module and query its existing active-admin predicate through a small service method; return a neutral skip message before consuming stdin when present. Keep the CLI outside repository internals. Only a read-only preflight is added; it is not an authorization or concurrency guard.
2. If creation is needed, read stdin and call the unchanged `Users.Bootstrap`. Distinguish typed application error code `already_bootstrapped`; do not treat all conflicts as success or string-match a message. Preserve strict mode and validation behavior. Do not hold the admin guard while waiting for stdin. A valid concurrent winner produces a successful opt-in skip; malformed creation input is still an error when preflight found no admin.
3. Compile bootstrap alongside the existing image binaries; retain the image's nonroot runtime and API entrypoint. Add the isolated Compose service with bootstrap entrypoint, no restart loop and working stdin under `docker compose run --rm -T`. The service uses the same SHA image and existing network; its only environment entry is `DATABASE_URL` interpolated from the current generated dev dotenv. Do not attach shared `env_file` to bootstrap. Keep bootstrap credentials out of shared `.env` and API service.
4. Add a sequential workflow step after migration and before API recreation using the approved step-scoped secret names. Pass login/email/name as quoted arguments and password only via stdin. Avoid direct secret-expression interpolation into shell source, tracing and output of resolved Compose configuration. Preserve normal failure propagation, including pipeline status; ensure existing-admin skip succeeds with empty secrets and does not fail due to pipe handling. Do not require secrets in the shell before the CLI decides whether creation is needed. Keep pull/image selection consistent for the new service.
5. Write the dev README against the accepted contract, with secret names only and no sample usable credentials. Document that a suspended/pending admin is outside the existing active predicate; an existing colliding account is not promoted or reset. Any desired different predicate is a product decision, not an implementation choice.

**Scoped rules:** root and workflow 2026-10-13 verification override controls older checklists: complete the whole request before choosing final verification; no automatic unit runs/writes, build, lint, typecheck, vet/race or review/QA chain. `internal/AGENTS.md`: Go via `bash tools/go.sh`, service owns rules, no GORM in service, imports through identity module root, existing transaction/repository boundaries. `deploy/AGENTS.md`: infrastructure authority, immutable image selection, no secret printing, named environment authorization for live mutations. `tools/AGENTS.md`: only version_control mutates Git, explicit paths, preserve unknown work. Slicer/worker do not mutate Git. Existing CI image build remains part of authorized CI, not an extra local validation gate.

**Resources:** exclusive main-checkout writer lock; no simultaneous source edits/branch switches while final verification uses the checkout. Disposable PostgreSQL fixture reserved separately for V1. No browser, live database, runner, Docker registry or deployed server lock is granted. Deployment execution/provisioning requires the hub's separately established authority.

**Context budget:** maximum 12,000 input tokens for I1, including this packet, targeted source, relevant scoped rules and accepted decision slice. Read only bootstrap section of `user.go`, relevant role predicate, CLI, Dockerfile, dev Compose and the workflow image/env/migration/deploy slices. If additional module internals must change, stop and request exact-path expansion; split the packet if it cannot fit. Report `RESLICE_REQUIRED` if that changes canonical scope; do not add backlog entries.

**Stop conditions:** undecided admin predicate; unaccepted infrastructure contract or unknown secret identifiers; stale source/base not reconciled by hub; need for account mutation, public bootstrap endpoint, schema/dependency/permission changes or unowned files; existing account collision; missing/invalid creation fields; database failure; credentials available only in an unauthorized file; unexpected dirty paths or missing resource lock. Runtime collision/config/database failures stop deployment rather than skipping.

## Final verification packets (specifications only; no automatic QA chain)

The following are independent, bounded evidence specifications for the smallest useful final pass after all implementation is complete. The hub chooses/assigns final verification under the user override; this plan does not dispatch agents, add mandatory review cycles, run checks or require persistent test files. Record one exact final candidate SHA and its base in evidence; do not reuse earlier packet-level green results. After repairs rerun only affected scenarios. Documentation-only planning has no tests.

### V1 — Disposable PostgreSQL bootstrap behavior

- Requirements: AB-01, AB-02, AB-03, AB-06.
- Dependencies: all I1 code complete; exact candidate SHA supplied; a disposable, migrated PostgreSQL database and only synthetic credentials. No production/test-server database access.
- Owned source paths: none. Evidence path, if assigned: `docs/justix-auto/dev/execution/admin-bootstrap-20261015/verification-bootstrap.md`. Temporary harness/executable may live only in an assigned temporary directory, not committed source.
- Locks: immutable checkout reader and exclusive named disposable PostgreSQL fixture; serialize against I1 edits and VCS branch changes.
- Exercise the actual CLI executable, flags and stdin, not only service-level units. Scenarios: (1) empty fixture + explicit valid inputs creates exactly one active platform admin, non-plaintext hash and bootstrap audit; verify the resulting credentials authenticate as admin; (2) opt-in rerun with no fields and stdin kept open returns promptly without input, leaves user/hash/role/session/audit snapshots identical; (3) default strict invocation with valid inputs fails `already_bootstrapped`; (4) no active admin + missing fields/password and invalid input fail without rows; (5) two simultaneous valid opt-in creations both exit successfully but create/audit once; (6) no active admin + existing login/email collision fails without changing that account; (7) invalid/unreachable database fails, never successful skip. A suspended/pending admin alone must not satisfy the approved active-admin predicate, and its collision still fails.
- Use project-pinned Go only for the minimal executable/harness needed for this behavioral check, not a broad build/test suite. Database inspection is confined to the disposable fixture. Capture exit status, sanitized outcomes and row-count/snapshot assertions, never passwords/hashes in reports.
- Context budget: 8,000 tokens. Stop on missing fixture authority, source changes, absent runtime tooling or any need to access live credentials. Report unexecuted scenarios explicitly.

### V2 — Final-SHA CLI/image/Compose/workflow integration

- Requirements: AB-04, AB-05, AB-06, AB-07, with AB-01/AB-03 cross-interface evidence from V1 at the same SHA.
- Dependencies: all I1 files and accepted infrastructure record complete; exact same final SHA as V1. This can be inspected independently; final acceptance joins the evidence without a repeated QA chain.
- Owned source paths: none. Evidence path, if assigned: `docs/justix-auto/dev/execution/admin-bootstrap-20261015/verification-wiring.md`.
- Locks: immutable checkout reader. No live deployment/registry/network access or image build is required by this packet.
- Quietly validate Compose with synthetic local values where tooling is available, without printing rendered configuration or overwriting any real `.env`. Inspect the workflow for migrate → bootstrap → API sequencing, nonzero propagation, quoted secret-variable handling and exact image selection. Confirm bootstrap profile isolation, no restart, stdin enabled/non-TTY invocation, same image/database/network, no credential keys in API/shared env, and image build/install path matches the Compose entrypoint. Check empty-secret rerun wiring does not introduce shell-level failure before the CLI no-op. Inspect docs and approved decision for matching secret names and environment.
- Final-SHA integration condition: the CLI contract exercised by V1 must be the command/flags and password channel used by V2; inspect all six owned paths together. A successful image build or live server deployment is not claimed by static wiring evidence. If later authorized CI supplies same-SHA image evidence, attach it without an extra local image build.
- Context budget: 6,000 tokens. Stop on real secrets in output, environment ambiguity, source drift or mismatch with the approved infrastructure contract. Report missing parser/Docker tooling as an evidence gap, not permission to run live infrastructure.

## Source hashes (SHA-256)

Observed from the shared checkout before implementation; base SHA is supplied separately above. Reconcile a changed hash rather than overwriting concurrent work.

| Source | SHA-256 |
| --- | --- |
| `AGENTS.md` | `9535ac1f07c3c61574633449375b85feba89baaaca33474c5ed6d875c4953469` |
| `docs/justix-auto/dev/agent-workflow.md` | `f2714233d1619a2cb2d8c1b10b2622b8cebf2c2b60108b226efaff0f15999fb6` |
| `internal/AGENTS.md` | `4c43c2d904382a3aefc313dffd7f28a04238e51496f6f0be87269b54b99f9dd6` |
| `deploy/AGENTS.md` | `28d03b9e6d5971860a9e4376d2b9c18d086c06a212697fbb7031244b8171f23a` |
| `tools/AGENTS.md` | `361f5e895fc93d6bb232941ae5f173d868b6ce4038f1b4810138bed8c4c631f6` |
| `cmd/bootstrap-admin/main.go` | `6d52b2cf6f2a3d2e8e01c00688a6c8379e9ae53d31e72eee590b9ecb33528fd9` |
| `internal/modules/identity/service/user.go` | `a21733b3df2543611dca41d3b239276c54bf6733d808bc8c7b9adbc2f392ae73` |
| `internal/modules/identity/repository/role.go` (read-only) | `4c03f7f0ddead31c8bc4c72cc2e77e2d0f18fc94898e5cd6ddeb6846a53c91a1` |
| `deploy/Dockerfile` | `580b19b4f5466834d9a3a546c1c09754853cfb2d8bab850c1773024f6fa49015` |
| `deploy/dev/compose.yml` | `5d431a2920c7bccb159e9e08f6b9bca3410b56411c686a11e6293fd0ff74c8b6` |
| `.github/workflows/deploy-dev.yml` | `8fd605344b901332852066dfd88c8b9ede7bafcaa52f8101beec2eb28574bde1` |

## Gaps and risks

All seven local acceptance requirements map to final evidence specifications; none has been executed. Both hubs accepted the contract, active-admin semantics and exact secret names; root must persist ADR-16. Canonical task linkage, if applicable, remains hub-owned. The user request itself does not supply credentials. Existing login/email collision intentionally stops bootstrap and requires a separate authorized resolution. Preflight does not eliminate races; the unchanged locked transaction is essential. Static wiring validation does not prove a runtime image or live deployment. Existing workflow cleanup/concurrency behavior is outside this bounded change. Preserve unread `test_creds.txt` and unrelated branch work. Root will perform the narrow final verification after the complete change; no independent review/QA loop is required by this packet.
