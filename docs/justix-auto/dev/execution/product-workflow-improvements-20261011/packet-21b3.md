# P21b3 — Retail lifecycle for authorized consigned VINs

Proposed under original P21b, R3/R5 with R1/R2 compatibility; hub `/root`.
Planning base `4fa683ba4708bb71031534260ec5cb8742ce9977`, branch
`feature/company-commerce-workflows-20261011`. Read p21b-split-proposal.md,
root AGENTS.md, dev/agent-workflow.md, internal/AGENTS.md, plan.md Execution
rules and final b1/b2 result/API slices. All final P21a1/a2/a3 handoffs remain
prerequisites. Fresh context ceiling **14k**; map symbols before reading source.
Exclusive source-writer, no browser/DB/runtime lease. Coverage B1–B10, focused
on retail behavior and app seam; predecessor evidence is not rerun ceremonially.

## Exact owned paths

- `internal/modules/retail/service/ports.go`
- `internal/modules/retail/service/deps.go`
- `internal/modules/retail/service/deal.go`
- `internal/modules/retail/service/listing.go`
- `internal/modules/retail/service/consignment_test.go`
- `internal/modules/retail/module.go`
- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`

Read deal.go only at checkDealVehicle/Create, Deliver/checklist, Cancel and
their directly called state/version helpers. Read listing eligibility/create/
publication, constructors and the app retailStock/composition slices. Existing
cash/color/verification unit helpers and public signatures are read-only.
Preserve constructor and ordinary Stock-port compatibility where needed so
unowned callers/fakes do not require edits; never silently expand ownership.
Add narrowly scoped ports/injection through these owned files. Missing authority
must fail closed for consignment without blocking historical ordinary owners.

## One behavior

Wire b1's live commerce authority and b2's trusted physical operations into the
retail listing→reservation→cancellation/delivery lifecycle. Company/VIN/agreement
facts come from trusted module-root services, never client owner flags. Preserve
immutable vehicle snapshots and exact money/accepted payment behavior. Only
internal application adapters import other module roots; retail declares ports
and never imports commerce or inventory, including their internal layers.

Listing creation/publication allows verified holding dealer's actually handed
over active consigned VIN via commerce authority. Creation currently checks
eligibility before InTx: move the relevant consignment recheck into the bound
mutation transaction so authority is current when creating/publishing. Existing
list/read/update/withdraw history behavior remains. Do not equate publication
with a physical hold or make a stale listing permission to reserve.

Deal.Create rechecks authority under b1's live row lock and creates b2's exact
retail holder within the same retail Store.Bind transaction, retaining the lock
until outer commit. Reject pending return and other hold; do not release them.
Use the per-unit durable retail association exposed by b1; do not add retail
model fields or let one VIN's pending return block the entire agreement.
Live active consignment must not be bypassed merely because the supplier also
passes the old Owned check: owner retail remains the normal path only where
that does not evade the existing consignment authority. Do not invent a new
seller-type restriction: both dealer and distributor retain normal owner retail.

Deliver preserves current permissions, tenant/revision/state/checklist/payment/
contract/insurance gates and OD-01 partner-finance exclusion. For an actual
consigned deal revalidate b1 authority and call b2 customer delivery plus b1
sold/retail-deal/settlement linkage inside the same bound transaction. Only that
successful actual delivery ends supplier ownership. Subsequent retail update,
listing withdrawal, lead/event or commerce completion failure rolls all changes
back. Retain historical snapshots; avoid forcing ordinary delivered-history
reads to require now-cleared inventory ownership or an active consignment.

Cancel preserves existing accepted-payment and pending-installment blockers,
revision/state/tenant checks. On eligible cancellation release only the matching
retail hold and keep unsold consignment, dealer custody, supplier ownership and
all B2B debt/settlement/payment history. Coordinate authority state/lock in the
same transaction where needed; no sale completion, return or refund is implied.

R2's new-operation verification guards remain on new listing/publication/deal
creation; do not add blanket verification guards to history or preexisting
obligation servicing. Preserve current company/branch/role scopes.

## Affected units and app seam

Use meaningful `TestRetailConsignment*` cases in the owned file with stateful
transaction fakes for listing/create/publish, permitted dealer reservation,
wrong dealer/VIN/agreement, owner bypass attempt on live dealer consignment,
pending return and competing hold, stale/repeated actions, cancellation retaining
authority and debt, actual delivery exactly once with settlement link, and
failure rollback across retail/commerce/stock. Include both ordinary owner
company types, R2 new-operation denial/history servicing, preserved cash and
own-installment delivery prerequisites and unresolved partner finance. Fakes
must drive public service behavior, not just selector helpers.

In app workflow_ports_test.go add distinct `TestWorkflowPort21_Retail*` cases,
preserving P21a's `TestWorkflowPort21` cases. Use small app-local interfaces
around module-root exported methods to fake actual adapters. Assert dealer,
supplier, agreement, VIN, retail holder, actor/time, settlement reference and
bound-context forwarding; errors remain errors and do not downgrade to owned
flow. Verify constructor wiring reaches these adapters. No e2e.DB or imports
of another module's service/model layer to force a test.

Run each relevant selection once after changes:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/service -run '^(TestRetailConsignment|TestSaleColors|TestCashSalePartialPaymentLifecycleAndDelivery$|TestCashSaleDeliveryGuardsAndRollback$|TestCashSaleOtherSchemesRemainStrict$)'`

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort21_Retail'`

The listed existing retail families cover affected reservation snapshots and
delivery behavior. Do not run unrelated invoice units, whole packages or
TestOpenAPICoversRoutes. Do not rerun b1/b2 green families absent new changes.
Nonempty named results required; report inability to make an app seam unit
meaningful as a gap, never substitute build/integration checks.

Before implementation run tools/check-git.sh; no worker Git writes. Return
actual ports/constructor changes, hashes, test evidence and P22 shared-authority
handoff. No new HTTP route/DTO, model/schema, legal/fee policy, browser or SQL
verification. Stop on missing predecessor state/API, unowned compatibility
change, source drift or 14k limit. Hub reslices before additional reads/edits;
do not solve a fixture-derived or unresolved product rule.
