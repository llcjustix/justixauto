# Development state

> **2026-10-13 — warehouse receipt runtime fix.** User reported useRefresh not
> defined at ManualReceipt on /warehouses. [Approved import correction](warehouse-receipt-refresh-20261013.md)
> restores the missing kit hook import at dependency 239dcfd. SOURCE COMPLETE
> on fix/warehouse-receipt-refresh-20261013. [Two existing final receipt cases](execution/warehouse-receipt-refresh-20261013/final-validation.md)
> passed once (1.40s), including rendering and successful submission/refresh in
> both VIN modes. Source/check leases released. No live data changes; unrelated
> dirty files preserved. Local commit only.

> **2026-10-13 — settings and reservation UI clarity.** User requested clearer /settings UI
> with a screenshot. [Approved correction](settings-clarity-20261013.md) at
> dependency b38fc75 separates company/branches/staff/profile, groups company
> details, and simplifies verification status. SOURCE COMPLETE on
> fix/settings-clarity-20261013. Reservation/supply forms now use grouped styled
> controls and contextual empty guidance. Existing actions/permissions preserved.
> [Final browser validation](execution/settings-clarity-20261013/final-validation.md)
> passed on isolated fixtures, with affected reruns after mobile shell/card fixes.
> No units or live API/database mutations. Source/browser leases released;
> local app and unrelated dirty files preserved. Canonical snapshots in
> execution/settings-clarity-20261013/entry/; local commit, no remote push.

> **2026-10-13 — company verification UI.** User reports missing submission and
> review actions. [Bounded correction](company-verification-ui-20261013.md) is
> SOURCE COMPLETE on fix/company-verification-ui-20261013 at dependency 6df3005.
> Visible state/action panels and Admin filter/review details use existing backend
> commands. [Final browser pass](execution/company-verification-ui-20261013/final-validation.md)
> verified the full return/resubmit/confirm flow, permission guidance, retry and
> tax/bank preservation on isolated fixtures. No units or real company mutations.
> Source/browser leases released; local app and unrelated dirty files preserved.
> Canonical snapshots: execution/company-verification-ui-20261013/entry/.

> **2026-10-13 — login section redirect.** User reported no access at the previous
> Admin URL after signing in as another user. [Correction](login-section-redirect-20261013.md)
> SOURCE COMPLETE on fix/login-section-redirect-20261013 at dependency d078b4b.
> Common entry handles denied sections, preserving deep links and confirmed
> context. [Final browser validation](execution/login-section-redirect-20261013/final-validation.md)
> passed five scenarios with intercepted API fixtures. Source/browser leases
> released; no units or live backend mutations. Unrelated dirty files preserved.

> **2026-10-13 — final validation policy.** User removed mandatory per-step
> unit-test work. [Applied decision](final-validation-policy-20261013.md): finish
> the full request, then perform the smallest useful final check; focused E2E
> is allowed when needed. Existing unit suites remain available, not mandatory.
> Active agent/scoped instructions updated; no tests for this docs-only change.

> **2026-10-13 — company form layout.** User requested legal form before company
> name on one row and seller business type below as radios, default dealer.
> [Bounded task](company-form-layout-20261013.md) SOURCE COMPLETE at dependency
> 3aeb7b8 on fix/company-form-layout-20261013. Source writer released; fixed-type
> permissions remain. [Completion](execution/company-form-layout-20261013/completion.md)
> records 25 distinct affected unit passes. No browser/layout validation or
> runtime/remote action. Unrelated dirty files preserved.
> Pre-edit snapshots: execution/company-form-layout-20261013/entry/.

> **2026-10-13 — organization legal form across company UI.** User requested
> ООО / ИП / Другое in all existing company forms and profiles.
> [Bounded task](company-legal-form-20261013.md) is SOURCE COMPLETE on
> fix/company-legal-form-20261013 at dependency 5ae7a93. Source writer released.
> [Completion](execution/company-legal-form-20261013/completion.md) records one
> frontend packet reusing legalForm storage/API and 17 focused unit passes.
> Unrelated company_contract_test.go changes and test_creds.txt are preserved.
> No runtime/remote actions. Snapshots: execution/company-legal-form-20261013/entry/.

> **2026-10-13 — local UI and password corrections.** User reported a malformed
> Admin section switcher and requested an 8-character password minimum.
> [Bounded task](local-ui-password-fixes-20261013.md) is SOURCE COMPLETE on
> fix/portal-ui-password-20261013, dependency 31b018ce with dev ancestry 4fa683b.
> All source writers released. [Completion](execution/local-ui-password-fixes-20261013/completion.md)
> records 13 portal units, 20 password-form units and selected Go validator/hash
> units passed. Responsive selector correction has no browser/layout evidence.
> Running local app and reset admin database are preserved; no remote promotion.
> Canonical before-copies/hashes: execution/local-ui-password-fixes-20261013/entry/.

> **2026-10-13 — unified frontend.** User authorized one React app replacing
> four entries, opening sections by current access; confirmed last permitted
> section with an internal switcher. [Bounded task](unified-frontend-20261013.md)
> is SOURCE COMPLETE on feature/unified-frontend-20261013, base743bd6c from the
> completed company/commerce feature with dev ancestry4fa683b. Prior source is
> preserved. ADR-15 records the single-entry/output and context isolation contract.
> Source commit: 2b41eebb22b11c7e4ea680690d56385c94dd58d7. All writers released.
> [Completion and affected unit evidence](execution/unified-frontend-20261013/completion.md)
> records one portal entry/workspace, access/last-section switcher, context isolation,
> 94 feature moves and single-output serving/launch configuration. Existing private
> local WEB_DIR requires a manual change to web/apps/portal/dist.
> Affected units only; no runtime activation, deployment or automatic review/QA.
> Pre-edit copies/hashes: execution/unified-frontend-20261013/entry/.

> **2026-10-11 — company and commerce workflows.** User authorized implementation
> of [13 workflow changes](product-workflow-improvements-20261011.md), following
> [recorded interview](product-workflow-interview-20261011.md). Branch
> `feature/company-commerce-workflows-20261011` from dev `4fa683ba4708bb71031534260ec5cb8742ce9977`.
> All thirteen requirements are SOURCE COMPLETE; all source writers released.
> [Completion and unit evidence](execution/product-workflow-improvements-20261011/completion.md)
> and the delivery map identify the final reports. Legacy companies need
> manual type/verification, preserving history/servicing; full reservation-advance
> refunds require accountant confirmation; partial-purchase handover by explicit
> agreement. Retail sales are available to both company types. Affected unit
> evidence only; no automatic review/QA, migration execution or runtime activation.
> Local source Git provenance is in execution/product-workflow-improvements-20261011/source-git-result.md.
> No remote push, dev/main promotion or historical backlog completion is claimed.
> Before-copies/hashes: execution/product-workflow-improvements-20261011/entry/source-completion/manifest.md.

> **2026-10-02 — cash registration and exact-price invoice.** Latest clarification:
> one vehicle invoice equals sale price, multiple payments settle it. Cash
> registration optional; own installments mandatory. [Bounded task](sale-invoices-registration-20261002.md)
> source complete; all writers released. [Completion evidence](execution/sale-invoices-registration-20261002/completion.md)
> records173 selected terminal unit cases, targeted repairs and8source hashes.
> Preserve dirty work; no Git/runtime operations or new migration. Earlier multi-
> invoice interpretation withdrawn before edits. Running backend activation pending.
> Pre-completion snapshots: execution/sale-invoices-registration-20261002/entry/completion-manifest.md.

> **2026-10-02 — offers in sale creation.** User requested applying supplier and
> own offers, copying an editable offer price, only to existing owned warehouse
> VINs. [Bounded task](sale-offer-application-20261002.md) source is complete;
> all writers released. [Completion evidence](execution/sale-offer-application-20261002/completion.md)
> records 42 distinct affected unit passes, repairs/exclusions and nine source
> hashes. No backend policy/source persistence, runtime or Git operation. Earlier
> frontend edits remain preserved; prior migration034/API activation still pending.
> Pre-completion copies: execution/sale-offer-application-20261002/entry/completion-manifest.md.

> **2026-10-02 — unified autocomplete controls.** User requested one searchable
> input instead of separate search/dropdown elements, consistently across forms.
> [Bounded task](autocomplete-controls-20261002.md) P1–P9 source is complete;
> all writers released. [Completion evidence](execution/autocomplete-controls-20261002/completion.md)
> records 112 distinct affected unit passes and approved exact stray-text cleanup.
> Three later frontend edits are preserved outside those receipt hashes; the
> entire current checkout is not certified by that evidence. No runtime/Git
> operations. Prior migration034/API activation remains unapproved.
> Pre-completion snapshots: execution/autocomplete-controls-20261002/entry/completion-manifest.md.

> **2026-10-02 — vehicle color choices.** User confirmed multiple body/interior
> options per catalog model and one selected combination per ordered car/VIN.
> [Bounded task](vehicle-colors-20261002.md) source is complete across P1–P11;
> all application writers released. [Completion evidence](execution/vehicle-colors-20261002/completion.md)
> records delivered behavior, affected units, deviations and final source pins.
> Preserve existing order/sale/payment work in the dirty checkout. Only
> affected units are authorized for development verification. Runtime activation
> remains pending explicit approval for local backup/migration034/build/restart;
> no migration034 or API replacement was performed. Canonical pre-completion
> copies/hashes: execution/vehicle-colors-20261002/entry/completion-manifest.md.

> **2026-10-01 — sale servicing and supplier-offer entry.** Requested source
> implementation is complete on the existing dirty `fix/empty-vin-allocation`
> checkout (HEAD `75252503356d0c98b067d6846e0c87ca04781d26`), preserving prior
> work. Manual multi-month/advance payments, early monthly payoff, parent finance
> decisions, sale action/status/close improvements and supplier-offer purchase
> entry are implemented. See [bounded task](sale-servicing-improvements-20261001.md)
> and [completion evidence](execution/sale-servicing-improvements-20261001/completion.md).
> Affected unit evidence only; all writers released. Local API activation remains
> pending a bounded build/restart authorization. The preexisting watcher applied
> migration033 before being held; its applied files are immutable. No automatic
> watcher resume or migration. No Git write, dev/main promotion or broader
> backlog completion is claimed. Older current-state entries below are historical.
> Prior canonical snapshots: execution/sale-servicing-improvements-20261001/entry/completion-snapshot-manifest.md.

> **2026-09-26 — current user-requested form correction.** Simplify Admin company
> onboarding into information, address, contacts and login sections, with one
> company name and one shared onboarding email. See [bounded record](admin-company-form-20260926.md).
> Branch: `fix/country-region-combobox`; tested source `f9c4f356`, dev base `eca22f60`.
> The bounded correction is **qa-green**: independent cumulative code review,
> final desktop/mobile browser integration and four-app builds passed. Selected
> units (28 tests), typecheck and lint passed on byte-equivalent non-CSS source.
> Existing geographic-input work is preserved and included in cumulative QA;
> the bounded record documents the separate session's collision report and
> adopted test-selector edit. Source/browser leases are released. Task-branch
> publication may proceed; dev/main integration remains human-controlled.
> This does not close T-426/B-11 or authorize deployment. Historical task/phase
> entries below do not describe the current tree.

> **2026-09-22 — backend reset ([ADR-14](../adr-14-classic-modular-monolith.md)).** The CQRS/ES microservice backend was removed (tag `archive/cqrs-es-backend`) and replaced by a modular monolith: `cmd/api`, `internal/modules/*` (handler → service → repository), Echo + GORM, SQL migrations. First module: `identity` companies. Backend tasks below that assume event sourcing, outbox/inbox, projections or per-owner databases are obsolete until re-planned. Entries below are historical.

> **2026-09-22 — current state.** All backend modules are implemented (identity with MFA, inventory, commerce, retail, insurance, financing, documents with S3) and the four React apps were rebuilt on the shared `web/packages/kit`. `WEB_DIR=web/apps` makes `cmd/api` serve the built apps (`/`, `/finance/`, `/insurance/`, `/admin/`). Browser smoke test passed on a local stack for admin onboarding, a cash retail sale through delivery, a B2B offer → order → shipment → receipt → payment, RFQ → quotation → order, insurer approval, and bank program → application → terms → agreement → document exchange. Since then: provider snapshots carry VIN/model/client, retail contract scans are stored (review finding), receipt batch quantities have an audited correction, and warehouses can be a branch's main warehouse. The four apps now use the reference design system (mocks common.css/styles.css copied into web/packages/kit/src/design.css) and the reference navigation and page layouts; companies are registered by the platform admin only (user decision 2026-09-22). Still open: detailed parity of detail dialogs (deal stage bar, vehicle photos), pixel/interaction parity with the HTML mocks (independent browser QA), OD-01/OD-06/OD-12 decisions, the fixed-markup calculation policy approval, and company-level staff management (today only the platform admin assigns roles).

- Date: 2026-09-20
- Project: justix-auto / JustixAuto
- Target: `/Users/bakhromachilov/startups/justixauto`
- Current task branch: `task/T-950-eventsourcing-core` in the main project folder, based on human-approved `dev` at `727d3366704710d58b54969bc7da43621bc37f31`. No worktrees. T-939 is paused and its interrupted patch is preserved locally.
- Phase: development active; Go event mechanics and shared React components in progress
- Branch policy: `dev` is human-approved integration/development; `main` is human-only production promotion. Remote default remains main until the human changes it. Agent task branches may be pushed independently.
- Agent setup: [two-hub workflow](agent-workflow.md) is included in dev after the user's explicit push approval. [Initial handoff evidence](agent-setup-results.md) remains historical. Every subsequent dev merge/push still needs fresh human approval; main is unchanged.
- Remote: `git@github.com:achilovbakhrom/justixauto.git`
- Parent repository: `/Users/bakhromachilov/startups`; must NOT be used for tasks
- Git readiness: own root verified. T-933 was human-approved, fast-forwarded and pushed to `origin/dev` at `727d336`. T-950 starts from that exact local/tracking dev SHA. Runtime T-933 QA remains bound to `c75dfcd`; its publication candidate is `727d336`.
  Run `bash tools/check-git.sh` before assigning application tasks.
- Confirmed: Go microservices, CQRS + Event Sourcing, Gaze reference approach;
  React; one project folder; minified four-app HTML documentation references
- Reference: `/Users/bakhromachilov/golang/gaze-executor`, current clean HEAD
  `6a277d916b417754fe5d5e5b11cebe77ea3d8c42`; see the bounded
  [core refresh](../reference/gaze-core-refresh-2026-09-20.md). The historical
  full audit remains pinned to `913c018fb2182ec943ef308da63baa85b5628bec`.
- Additional reference: `/Users/bakhromachilov/gaze-executor-cc`, HEAD
  `f81b62ce346320ec229817dd73cf9ec42032e1d9`; audited 2026-09-14 in
  `../reference/gaze-executor-cc-reference.md`. User reconfirmed microservices.
  Existing seven-owner boundaries and approved reliability decisions remain.
- Frontend build tool/version: React/TypeScript/Vite/shared npm workspaces approved and locked; workspace config discovery is QA-gated
- Backend service ownership, contracts and reliability ADRs: see `../architecture.md`;
  approved on 2026-09-13. Go module/boundaries, seven PostgreSQL owners,
  restricted RabbitMQ topology, event envelopes, money values and internal
  caller authentication are integrated. Do not import trading dependencies or
  Gaze credentials.
- Package manager: npm 11.19.0 with Node 24.21.0; exact workspace lock included
- Validation: 106 mock domain tests pass against minified JS; four entries load;
  hashes, syntax and static local dependencies pass
- Execution authorization: user said "ok lets start" on 2026-09-14; proceed through bounded implementation and QA gates.
- Active task: T-950 pure aggregate lifecycle is independent exact-commit QA GREEN at `4a6be488`; human approval is required before its candidate may update `dev`. T-951 will bind it to the existing durable event store; T-952 will prove typed CQRS composition. T-939 returned to todo; no auth-generator implementation was merged. No new dev merge/push is approved. T-002/T-920 remain blocked; T-928's missing live authorization is not cleared. Do not use historical `.worktrees` paths.
- Go preflight correction: the standard-library source files exist. The shared package index reported them missing; a fresh isolated `GOCACHE` resolved the issue without reinstalling the compiler. Independent final QA with `/private/tmp/justixauto-go-cache.Ew93Ez` passed the scoped race suite (120 top-level passes; 58 expected live-opt-in skips), all nine required migration tests without skips, and both vet checks.
- Integrated count: 56/952 after human-approved T-933 integration and the three new core tasks. T-938's prior reconciliation remains backed by independent GREEN at `931ce802` and integration evidence at `a853ce12`. This does not certify the new core tasks.
- Active architecture handoff: owner migration, retained-label and [auth transport](auth-transport-assignment.md) handoffs approved; independent auth canonical promotion QA is GREEN. T-935–T-944 may be assigned when their task dependencies and worker slots permit.
- Additional architecture: [durable membership storage](membership-version-assignment.md) final r3 proposal and canonical promotion QA GREEN. Pure membership identities (T-947) are integrated; T-945, T-946 and T-948 await their own dependencies. T-920 remains blocked on its new state dependency. T-928 live verification authorization remains pending.
- Reviewed PO backlog: 52 items — 42 Must, 7 Should, 3 deferred; scope confirmed by user continuation
- Formal PM task files: 952 (916 reviewed baseline + thirty-six bounded additions); see task-board.md, task-index.json and backlog-coverage.md
- Coverage: 167 acceptance clauses across 49 active backlog items; one closing verification per item
- Execution limit: three delegates total across both hubs (DevOps counts as one), one writer in the main project checkout, one browser-heavy QA and one holder per live fixture; one canonical task/branch, sequential code tasks with frozen source during review/QA. No separate development worktrees. Existing worktrees are retained historical state, not current execution locations.

## Laptop preflight observed

Project-local Go `1.27.1` and Node `24.21.0`/npm `11.19.0` are installed under
the ignored `docs/justix-auto/dev/local/toolchains/` directory. The pinned
wrapper enforces Go 1.27.1; use `bash tools/go.sh`, not an unqualified Go
command. Docker daemon `29.0.1` and Compose `v2.40.3-desktop.1` are available.
Pinned PostgreSQL 18.6 and RabbitMQ 4.3.5 images were verified through
disposable loopback-only QA containers; no project infrastructure remains
running after the checks.

## Commands available now

```sh
npm ci --ignore-scripts
npm run typecheck:config
npm run mocks:verify
npm run mocks:test
npm run mocks:serve
bash tools/go.sh test -race -mod=readonly ./services/... ./pkg/... ./tests/...
bash tools/go.sh vet ./services/... ./pkg/... ./tests/...
```

The confirmed outbox relay, transaction-bound sequence/gap adapters, explicit
owner-history SQL template and deterministic Go/TypeScript contract generator
are integrated. Generator checks use the approved Node toolchain and installed
TypeScript; the checked-in config stays empty until concrete owner schemas are
approved and registered. Sealed owner persistence profiles/read-only checks and
the strict raw response JSON decoder are now independently integrated. Auth
Fetch metadata/429 composition is integrated as T-936; the generation-label correction remains
inside T-928's reviewed storage fix cycle. Identity's explicit profile-compatible
UOW is integrated as T-931 and checks each transaction before repository binding.
The migration driver's initial merge failed integration; the task board now lists
T-932's connection cleanup correction integrated. Preserve historical reports.
The runner, concrete baseline
manifests/default provisioning and custody profile support remain separate tasks
with their explicit prerequisites.

Historical preflight below predates the authorized Git setup on 2026-09-14.
`doctor` checks the independent Git root and initial commit.
`tools/check-git.sh` reports the detected root and exact user commands. The old
spec-team checker returned this misleading inherited-parent success:

```text
GIT CHECK OK: /Users/bakhromachilov/startups/justixauto (branch: main, dirty tree, 4 commits)
```

Read-only `git rev-parse --show-toplevel` instead returned
`/Users/bakhromachilov/startups`. The new project guard explicitly rejects that.
No changes were staged/committed in either repository and no remote was added.

Final project guard result:

```text
GIT CHECK FAILED: project needs its own Git repository: /Users/bakhromachilov/startups/justixauto
Detected Git root: /Users/bakhromachilov/startups. Do not use the parent repository.
```

Docker socket access is denied in the restricted shell; the approved read-only
doctor check outside the sandbox confirms daemon 29.0.1. This is a permission
boundary, not a missing Docker installation.

## Commands not available yet

No owner service composition roots, shared local Compose runner, owner migration
runner or CI pipeline exist yet. All four Admin, Realization, Financing and Insurance React workspaces and shared API,
tokens, UI and test-kit packages are integrated. `npm run build:apps` builds
all four apps; their default session boundaries render no protected children until session
integration. Explicit synthetic QA fixtures display the empty shell. This is
not a working business application; feature routes and service composition
remain in their assigned tasks. Explicit SQL installation templates and
shared Go durability primitives are integrated. Identity, Inventory, Commerce,
Retail, Financing and Insurance now have typed owner
transaction ports, explicit migrations and fail-closed persistence readiness.
Their composition roots, business features and custody-mode adapters remain
separate tasks; these scaffolds do not start service APIs or workers.
Broad `go test ./...` from the main checkout also discovers ignored Go sources
inside the local compiler and `node_modules`; use the project-package commands
above until tooling relocates or isolates those development artifacts. Do not
describe the repository as a running Go/React application yet.

## Next coordinator action

Architecture confirmed by user "yes" on 2026-09-13; release scope confirmed by
"conmtinue" in `../state/backlog-approval.md`. Planning is complete; do not repeat
these checkpoints or regenerate the plan. `task-board.md` is the formal plan;
`task-preview.md` remains historical examples only. Coordinator review and actual
validation results are in `../state/planning-review.md`.

Event schema/append, shared API/tokens and runtime registration convention are
integrated. The Active task field and task-index.json carry current assignments;
continue their exact-commit QA and dependency transitions. Follow the live task
dependencies for replay/receipts/outbox/inbox and the app harness, and resolve scoped
choices only before affected tasks. No policy invention. The user initially
authorized Git initialization/push on 2026-09-14. The newer policy permits agent
pushes to task/feature/fix/infra branches, requires human approval before every
dev integration, and reserves main promotion to the human.

Git readiness blocks application code, NOT documentation architecture/planning.
Policy decisions block only affected operations. Release approval, PM task
generation and the initial dependency/foundation gates are complete.
Keep current mocks immutable. Use the approved contract, not raw slice alternatives.

### Pending user decisions — asked 2026-09-15

- OD-02: Cars registry default should include all scoped company vehicles,
  including warehouse stock, or only in-work vehicles outside warehouses.
  The all-vehicles option was recommended but no response has been received.
  T-042 remains unapproved; supported owner query filters are unaffected.
- T-002 / OD-12: implementation security profile and restricted recovery contract
  proposal at task commit `a262f075bd5338c03c7049141c6c9f7e674110e5` passed
  proposal-readiness QA (`qa/T-002-r2.md`). Explicit acceptance was requested;
  no answer has been received. The proposal stays in `.worktrees/T-002` and is
  not an approved production configuration. Delivery/target benchmarks and
  all-factor-loss/pending-user activation remain separately gated.

### Pending test-environment authorization — asked 2026-09-16

Automatic approval review rejected PostgreSQL role/parameter configuration
changes in a disposable membership QA fixture before process creation. Explicit
authorization was requested for those changes only inside newly created,
disposable JustixAuto containers; no answer has been received. T-928 exact-commit
live verification remains blocked. Other checks omit the restricted mutations
and must report those exclusions; ordinary isolated fixtures and pure checks
continue. Proposal feasibility QA used a separately approved safe probe and
does not establish these unexecuted runtime configuration checks.

### Historical Git diagnostic — 2026-09-13

```text
GIT CHECK FAILED: project needs its own Git repository: /Users/bakhromachilov/startups/justixauto
Detected Git root: /Users/bakhromachilov/startups. Do not use the parent repository.
User action: cd '/Users/bakhromachilov/startups/justixauto' && git init -b main && git add . && git commit -m 'Prepare development workspace'
```

This is tool output, not a command executed by the coordinator. User should review
what will be staged before creating the initial commit; parent repo is untouched.

### Approved messaging correction — 2026-09-15

The reviewed [amendment](../state/drafts/contracts/messaging-delivery.md) is
approved by [coordinator decision](../state/approvals/messaging-delivery.md).
T-917–T-924 implement recipient plans, durable intake and logical dispatch;
T-922 is the definite transaction-composable inbox prerequisite identified by QA.
No affected composition is ready until its added graph dependencies integrate.

### Source route compatibility correction pending — 2026-09-15

T-918 live work exposed a mismatch between T-011's established
`source.target.fullOwnerQualifiedEventType` route and T-917's schema allowlist
reconstruction. Correctly admitted populated legacy routes can be rejected.
An isolated architect proposal is assigned in
`.worktrees/messaging-route-correction` at base
`f0d0454e0794dbdc4bddd3cb3b3bb4e331e9a51a`. Preserve routes and existing
migrations; a separately reviewed forward correction must precede dependent
writer/intake/topology activation. T-918 admission logic may proceed with its
explicitly documented synthetic fixture limit. See messaging-readiness.md.

### Route correction approved for bounded implementation — 2026-09-15

Independent proposal QA is GREEN; [coordinator approval](../state/approvals/messaging-route-correction.md)
adopts the exact reviewed [forward correction](../state/drafts/contracts/messaging-route-correction.md).
T-925 and explicit dependent edges now gate canonical route and correction-marker
readiness. This approves implementation, not an applied migration or working
delivery runtime. The earlier diagnosis and historical QA evidence stay intact.

### App fallback corrective handoff — 2026-09-15

T-038 independent review reproduced non-document fetch/script requests receiving
entry HTML. Financing fix cycle1 also owns the identical Admin and Realization
Vite predicates, with explicit prior-task dependency edges and renewed dev/preview
checks for all three apps. Their existing visuals and reviewed artifacts remain
unchanged. This is the existing prefix/document-only requirement, not a new
authentication or deployment guarantee.

### Projection storage handoff approved — 2026-09-15

The [independently reviewed storage handoff](../state/approvals/projection-storage-handoff.md)
adds T-926–T-928 and explicit checkpoint/quarantine/generation dependencies.
This approves bounded implementation, not installed storage or running recovery.
See projection-storage-readiness.md; existing policy gates stay scoped.

### Owner migration compatibility handoff adopted — 2026-09-15

The [technical adoption](../state/approvals/owner-migration-compatibility.md)
adds T-929–T-934 and exact T-059/060/022/580 prerequisites. Existing owner defaults
remain legacy-only. Later owner roots retain OWNER-COMPATIBILITY until their
explicit adapter/profile follow-ups are assigned and independently integrated.
This is implementation architecture, not an executed migration or running API.

### Auth schema task resliced before implementation — 2026-09-16

[Coordinator reslice](../state/approvals/auth-schema-reslice.md) retains T-059
wire/schema fixtures (3h) and assigns private SQL/manifest plus serial migration
tests to T-949 (4h). T-060 and B-02 closure require T-949; T-641 can consume
wire schemas independently. No release security setting or contract is changed.
