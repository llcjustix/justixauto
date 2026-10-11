# P23 consigned retail selection seam proposal

Status: proposed; root acceptance and final predecessor API handoffs required.
Feature `product-workflow-improvements-20261011`, application hub `/root`, R3
authorized consigned retail with R1/R2/R5 compatibility. Planning base
`4fa683ba4708bb71031534260ec5cb8742ce9977`, branch
`feature/company-commerce-workflows-20261011`. Only this proposal is writable
by this slicer; source, canonical packets/contracts and Git remain untouched.

## Mapped findings

- readSaleVehicles reads only inventory `placement=warehouse&eligible=true`.
  Ordinary eligible inventory requires acting-company ownership; accepted P20
  additionally requires own custody. Dealer-held supplier-owned VINs are absent.
- SaleForm uses that helper for options, applied-offer preflight and success
  refresh. matchesSaleSource only checks actual vehicle facts after discovery;
  it cannot confer consignment authority.
- Actual existing permissions are company-scoped `retail.read` for retail
  customer/lead/listing/deal reads, `inventory.read` for vehicle list/detail, and
  `retail.deals.manage` for deal creation. Session.can checks held permissions.
  Commerce handlers currently gate individual B2B routes with commerce.read;
  the company group itself has RequireCompany only.
- Inventory detail is already tenant-visible by owner OR warehouse company,
  and its response is `data.vehicle` plus specification/history, not flat data.
  This permits hydration of authorized dealer IDs without widening eligible.
- Existing retail listing routes are not an exhaustive stock discovery seam:
  a consigned unit need not already have a listing. Supplier offers are not
  dealer sale authority. Final recommendation and bounded ownership follow.

## Recommendation and permission boundary

Add a separate minimal **retail candidate projection** using commerce's existing
P21a consignment authority, then merge those vehicle IDs with unchanged ordinary
eligible stock inside readSaleVehicles. Use the existing inventory detail route
for actual VIN/model/specification/colors/placement/reserved facts. No better
existing exhaustive authorized route was found in the targeted surfaces.
This is discovery only: P21b's bound mutation remains the final sale authority.

The projection uses authenticated RequireCompany (including context-revision
checks) and the existing company permission **retail.read**, checked also at
the service boundary. The active principal supplies company identity; reject or
ignore no client-selected alternative dealer identity. No new permission key,
role/preset grant or commerce.read dependency. Commerce must not import retail's
internal model to refer to its already registered permission key; follow the
existing string-key auth contract within owned files. Do not widen a normal
consignment-detail/B2B terms route to retail.read. Keep that route's existing
commerce permissions and create a distinct redacted projection operation.

Inventory hydration continues to require inventory.read under existing routes.
The current form already needs retail.read for customers/leads plus inventory
reads; this proposal grants neither. Creation still requires retail.deals.manage,
verification and current company/branch authorization. Existing inventory detail
reads are company-scoped, not branch-filtered; preserve that behavior and existing
retail branch restrictions rather than inventing a new warehouse/branch policy.
Permission/context failures must remain visible, not silently become empty or
authorized candidate sets. Both dealer and distributor retain ordinary owned
sales. A distributor's normal discovery must never reacquire supplier-owned stock
already held by a dealer under consignment; P20's own-custody filter remains.

## Required data contract, without a guessed committed API

Final route, DTO names, field names and pagination are supplied by the accepted
backend handoff before UI implementation. No endpoint name is prescribed here.

- Server returns only candidate vehicle IDs for the principal's **holding
  dealer**, backed by accepted agreement and actual handover. Exclude unhanded,
  future-only, sale-reserved, pending-return, sold and returned authority. Where
  the final authority has an ordinary external hold not reflected in its state,
  inventory reserved hydration excludes it; no false claim that the projection
  alone proves stock free. A supplier/other dealer cannot query another dealer's
  candidates. Distributor current-company calls may return an empty candidate
  list while retaining ordinary owner results; they must not receive its
  supplier-side consignments as sellable stock.
- Minimal read projection needs unique vehicle identity and the list envelope/
  pagination metadata only. Do not expose order prices, accepted terms/digests,
  debt, invoice/payment/settlement amounts, commission, supplier requisites or
  the normal B2B agreement DTO. Add a revision only if the actual handoff needs
  it for freshness; it is never a client-held sale authorization token.
- Use existing httpx.List semantics and a deterministic paginated order. If
  offset pagination is chosen, consume every page with the helper's existing
  non-progress detection and no partial publication. Adapt exactly if final
  cursor contract differs; do not infer pagination from fixtures.
- Hydrate each distinct ID through existing `/inventory/vehicle-units/{id}`,
  extracting **response.data.vehicle**. Preserve actual vehicle ID, VIN, model,
  specification version, colors, placement, reserved and revision. Do not cast
  outer data/specification/history into a Vehicle. Keep hydration concurrency
  bounded, deduplicate IDs/requests and retain owner stream filtering unchanged.
- Merge candidate detail and owned results by vehicle ID. Require nonblank VIN,
  actual warehouse placement and reserved === false using existing predicates.
  Authorization comes solely from the server projection for the additional
  stream, not from those predicates, offers, warehouse presence or session type.
  Stale/missing detail or identity mismatch never adds a fabricated available
  vehicle. Network/permission/page errors surface as a failed load with retry;
  no stale or incomplete result is reported as a complete successful snapshot.
- Read projection takes no physical hold and cannot promise atomicity across
  page/detail reads. A return/hold after load is rejected by P21b's same-live-row
  reservation. UI must show that response and reload selection; no client-owned
  company/consignment flags or accepted contract/financial inputs are sent.

## Bounded work definitions and exact paths

These are proposed execution refinements under existing R3/P21/P23, not new
canonical tasks. They are not dispatch grants. Root must accept exact ownership
first. Per root's scheduling direction, use **P21b4** for the projection after
final P21b1/b2/b3 authority and **P23a** for selector integration before the
original P23 consignment/return workspace. P21a1's accepted persistence
prerequisites remain unchanged; this does not add work to its 12k implementation.
Ready final reports, hashes and APIs for every predecessor are required before
dispatch. No final API is guessed now. Both refinements have 8k fresh contexts,
below the root's 10k maximum; keep their suites separate from original P23.

**P21b4 (S23-read below), ceiling 8k: one minimal redacted candidate read.**

Dependencies: all final P21a1/a2/a3 and P21b1/b2/b3 reports, plus final P20
own-custody eligibility/stock DTO handoff. Retail authority, state predicates
and permissions must be ready before defining the read's exact API.

Existing P21a paths required for this read-only behavior:

- `internal/modules/commerce/service/consignment.go`
- `internal/modules/commerce/service/consignment_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/repository/consignment.go`
- `internal/modules/commerce/handler/consignment.go`
- `internal/modules/commerce/handler/handler.go`
- `internal/modules/commerce/handler/dto.go`
- `internal/modules/commerce/module.go`

One exact proposed additional unit path:

- `internal/modules/commerce/handler/consignment_test.go`

Service uses final authority predicates and scoped repository read, with no
cross-module table queries. Handler performs auth/parse/map only; use dedicated
minimal DTO so B2B fields cannot leak. No model/schema/migration/store edit is
anticipated. If the final source already supplies the approved projection, this
work definition reduces to consuming its handoff; do not duplicate it. Missing
repo/type support requiring another path is an explicit stop. Do not edit
canonical a1/b1/23 packets or approved contracts as part of this proposal.

**P23a (S23-selector below), ceiling 8k: one complete consigned selection flow.**

Dependencies: accepted P21b4 final route/DTO/permission/pagination, final P20
own-custody behavior, all P21a and P21b1/b2/b3 handoffs. Proposed exact additional
P23a ownership is five existing paths:

- `web/apps/realization/src/sale-offer-source.ts`
- `web/apps/realization/src/sale-offer-source.test.ts`
- `web/apps/realization/src/sale-form.tsx`
- `web/apps/realization/src/sale-form.test.tsx`
- `web/apps/realization/src/sale-offer-form.test.tsx`

Keep discovery DTO typing local to the helper; no data.ts, shared session/kit,
picker, workspace or billing edit is required. P19b's changing files remain
unread/unowned. Update fixtures only in the three owned suites to include exact
new reads; do not change the session permission catalog to make tests pass.

All initial/preflight/success-refresh calls must use the same augmented helper.
Keep company/context-scoped query keys and remount semantics; discard late
responses after context changes. Existing applied-source matching by model/
specification/colors or exact listing vehicle remains unchanged. A supplier
offer alone never makes a VIN available. For a selected consigned VIN, refresh
discovery/detail before submit even without an applied offer, then reject a
vanished/reserved/pending-return selection; this can use local provenance
metadata or refresh the common helper, without changing the create-deal body.
Do not introduce a new sale wizard or consignment agreement ceremony.
Surface conflict/permission/load errors, clear unavailable selections on refresh,
retain entered customer/price details, and prevent retry after successful create
from posting twice. Existing refresh retry remains refresh-only.

## Scoped execution rules and affected units

Root AGENTS.md, dev/agent-workflow.md and plan.md Execution rules apply;
internal/AGENTS.md for S23-read, web/AGENTS.md for S23-selector. Latest unit-only
policy overrides old scoped lint/build/React Doctor/browser/QA prescriptions.
One exclusive source-writer lease in the main checkout; no browser/PostgreSQL/
runtime resource. Check tools/check-git.sh before implementation; no worker Git
writes. Root supplies actual dispatch SHA, dirty-source hashes and report path.
Map exact final symbols incrementally; stop before context ceiling/unowned edit.

S23-read meaningful pure units:

- `TestConsignmentRetailCandidates*` exercises service/repository-port fakes:
  current holding dealer only; exact handed unit; each ineligible lifecycle;
  independent VINs under one agreement; empty supplier/other tenant results;
  pagination; retail.read without commerce.read allowed; missing retail.read
  denied; no financial-state mutation or physical reservation.
- `TestConsignmentRetailProjection*` exercises actual route guards and DTO
  mapping with pure handler setup: missing auth/company/stale context/permission
  denied, retail.read-only route authorization, known positive minimal shape
  and absence of all B2B fields. No e2e.DB. If concrete handler wiring prevents
  a meaningful pure positive seam within owned paths, return that precise gap.

Run once per relevant change, from repo root:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service ./internal/modules/commerce/handler -run '^(TestConsignmentRetailCandidates|TestConsignmentRetailProjection)'`

S23-selector units in the three existing suites cover both streams, all pages,
deduplication/detail envelope, exact facts/source matching, reserved/blank/missing
placement exclusion, failed later page/detail/permission with retry, actual
manual and offer-seeded consigned selection, no commerce.read prerequisite,
company/context change while hydration pending, preflight candidate removal,
server conflict after preflight, successful create/refresh once and both-type
ordinary-owner compatibility. Preserve the own-custody inventory URL assertion.
Use fetch stubs matching the final backend handoff, never invent an API to make
a fixture pass. Run the affected suites once from `web`:

`npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/sale-offer-source.test.ts src/sale-form.test.tsx src/sale-offer-form.test.tsx`

No tests/review were run in this docs-only preparation. No automatic review/QA,
lint/build/typecheck, integration/browser or deployment is authorized.

## Acceptance coverage and final candidate

| Label | Acceptance | Owner/evidence |
|---|---|---|
| D1 | Actual handed active dealer-consigned VIN appears in real SaleForm | S23-read + S23-selector |
| D2 | Current company/retail.read boundary; no financial DTO leakage or commerce.read grant | S23-read service/handler units |
| D3 | Ordinary owner/custody filter preserved; both company types work; no supplier bypass | P20/P21b evidence + helper/form compatibility |
| D4 | Exact VIN/spec/colors/placement/hold facts; no offer/custody-only authorization | Projection + detail hydration + existing matcher |
| D5 | Pagination, partial failures, tenant/context races, refresh/retry and stale candidate behavior | Three selector suites |
| D6 | Real reserve uses existing server body and P21b authority; pending return/hold conflict shown | Form units + P21b mutation evidence |
| D7 | R2 history servicing, debt/settlement and no extra sale ceremony unchanged | Read-only nature; untouched APIs; P21b preserved evidence |

At final candidate SHA tie projection permission/DTO/hash to helper fixture and
form result; tie final P21b reservation and P20 eligible contracts to those same
hashes. Invalidate only evidence affected by later changes; do not rerun passing
units ceremonially. If separately requested, independent small QA definitions
are Q23-read (4k, D1/D2/D4) and Q23-selector (5k, D1/D3–D6), immutable read lease,
no source edits, relevant units only, stop on SHA drift/missing authorization.
Future live final-SHA candidate→detail→reservation/return-race integration needs
explicit user authority and named fixture/environment lease. None is executed
or implied here; unit evidence does not prove release readiness.

## Gaps, risk and stop conditions

Original P23 lacks all five selector paths; P23a supplies that bounded extension
before the original workspace implementation. The backend dedicated handler unit
is another concrete proposed ownership addition. Root acceptance is required
before source edits. This closes existing R3 acceptance, so no canonical product
scope change/RESLICE_REQUIRED is proposed. Missing final projection/state API,
unowned compatibility edit or any newly undecided authorization rule stops work.
Do not loosen inventory eligible, expose normal B2B DTOs, guess an endpoint or
silently swallow 403 as an empty valid discovery list. Detail hydration can be
many requests; bound concurrency and preserve complete-load/error semantics.

## Source SHA-256

Planning working-file snapshots only; refresh after predecessors. Root/scoped
baseline hashes remain in p21b-split-proposal.md. No changing data.ts/workspace/
billing source was opened. Execution basenames below are in this directory.

```text
b8dee411a0a1043a038d72ccde611f5d12e70a6065a3d60eb64fde72600bdd02  web/AGENTS.md
64c140c57156cbfdf997e634f9038bf7357c3e5ea6cb96d2551c8f3276ddc51a  web/apps/realization/src/sale-offer-source.ts
ea8566ea804410f55dba0ac52588f832002be07cca23f29997b667b59525220b  web/apps/realization/src/sale-offer-source.test.ts
b84d411bd4b2728770931cd3ff4349f330f1f8f25e4fad65943fac2b2d53db1c  web/apps/realization/src/sale-form.tsx
838ad100036102c79e927149dd08dd7dcb1802cd92ed72bffe11e940857e0d40  web/apps/realization/src/sale-form.test.tsx
792f2b5da2487ece2bb0e895cc108bc61f0b2b324612d06531a6d56a11589a97  web/apps/realization/src/sale-offer-form.test.tsx
9bc3107f613b0f2c28eff34555b474681bbef2ee0db9058ae504765270489c20  web/packages/kit/src/session.tsx
dee346c24a3cd6f3fe586dcb5db7baafb1d883af99e4c0a7b4ba3eaf40d7cd82  internal/modules/retail/model/permissions.go
d465e5a31b300ffdca94af1e0588d8652cd517047f99d439c6507238be6894c4  internal/modules/retail/handler/handler.go
c999445fa393db0f8b1bcc864e3d68532a7e1cc6c79277436afce090c7e9cb19  internal/modules/commerce/handler/handler.go
e46c03030e244815085d1e4588a23b007aef1c377913149f67b6ac73846ef71a  internal/modules/inventory/handler/handler.go
fb1555b67a38280c4733902fc51ce3d0807d71f1d567c7912826eb0fe230c62d  internal/modules/inventory/handler/dto.go
657e4718bea9c8302b60864976a948a6fec80c6c73dab6ee986af156c298bece  internal/modules/inventory/service/vehicle.go
5d91771af595f3f2add6218a964c757f275ebe7a89653315fce504a2ba589c19  internal/modules/inventory/repository/vehicle.go
099c48b63042c401c34207e8130fe72f9e027e93973de73c045e2f4c48ac99b6  internal/pkg/auth/principal.go
d9b81053f2530e6b31d01e3fe894245ebf344cd4b791c18215192f39606f897a  p20-owner-custody-eligibility.md
fd780f08372b2ffca228aa5eb9cfa70db6d7bf8ba4153ad2a4f59b45852d31c1  packet-23.md
```

Digest reported separately. No source/Git/test/runtime/canonical changes.
