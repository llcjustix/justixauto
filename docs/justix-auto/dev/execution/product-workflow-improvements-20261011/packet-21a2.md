# P21a2 — Accepted-mode physical handover

Proposed under P21a, R3; application hub `/root`. Requires accepted P21a1 result,
final P18b2 accepted terms/money/claim result and final P20 custody root API.
Planning base/branch, source hashes, scoped rules and stops are in
p21a-split-proposal.md. Fresh context ceiling 14k; exclusive source-writer,
no DB/browser/runtime. Coverage C3–C9/C12; preserve a1 C1/C2. No automatic QA.

## Exact owned paths

- internal/modules/commerce/service/consignment.go
- internal/modules/commerce/service/consignment_test.go
- internal/modules/commerce/service/fulfilment.go
- internal/modules/commerce/service/ports.go
- internal/modules/commerce/module.go
- internal/app/app.go
- internal/app/workflow_ports_test.go

No new paths. Read only final P18b2 fulfilment/accepted-money slices and P20
exported interface. Inventory implementation, payment ledger, order model,
schema, supply helpers and handlers are read-only. Internal AGENTS.md applies;
services own rules, app adapts module-root exports, bound transaction contexts
and no module-internal imports. Do not modify ordinary Allocate/cancel logic.

## Behavior and physical entry paths

Dispatch from the saved accepted continuation mode, not a client-supplied mode
or fixture. Preserve unrelated ordinary direct-purchase semantics. At each
actual reservation-purchase physical transfer boundary, read authoritative
accepted debt and exact mutually accepted partial-handover conditions using
P18a/P18b final contracts. Outstanding debt blocks absent explicit agreement;
agreement permits handover only when its exact conditions hold. Pending or
rejected evidence gives no paid credit; never count an advance twice. Preserve
current accepted route/economics and transaction/lock/revision behavior.

Cover all existing entry paths explicitly:
1. Allocated Ship with receiving warehouse: physical handover occurs now.
   Purchase uses ownership Transfer after money checks; consignment uses P20
   custody with actual allocated VINs and a1 authority, retaining supplier owner.
2. Allocated Ship without receiving warehouse: retain existing in-transit path;
   legacy DecideReceipt acceptance later dispatches by the saved accepted mode
   and applies the same money/authority rules at physical movement. Partial
   decisions must not move a VIN twice. Rejection retains P18b release/claim
   retirement semantics and never restores materialized future capacity.
3. Quantity Ship/Deliver: ordinary and reservation purchase batches remain
   supported subject to the applicable money agreement; consignment must fail
   before creating buyer-owned stock even if the quantity input includes VIN
   strings. Those are not proof of existing supplier-owned allocated vehicles.
   Actual consignment uses allocated VIN/custody flow. P21a3 adds quantity claim
   consumption; preserve and hand off the exact entry point.

All-future consignment conversion is valid upstream; handover waits for genuine
VIN allocation. No new all-VIN prerequisite for conversion and no requirement
that every order unit be allocated before a valid partial handover. Sold,
returned, repeated/stale or mismatched agreement/vehicle cannot hand over again.
Keep order settlement/debt despite custody change; custody is no payment.

Wire commerce Stock custody port via app into P20's actual module-root method.
Forward supplier owner, expected source custodian, destination company/warehouse,
order holder, agreement reference, selected vehicles, purpose, actor/time and
bound context exactly. Use P20 validations for capacity/placements/holds; never
silently release a hold or substitute Transfer/Deliver. Record consignment,
allocation, shipment, order revision and event changes in the same transaction.
Failure at any stock or commerce write leaves all participating fake state intact.

## Affected units

service/consignment_test.go: name `TestConsignmentHandover*` and
`TestConsignmentPurchaseHandover*`; invoke real Ship and DecideReceipt services
with deep-copy stateful fakes. Cover each path above, same accepted amount
without/with qualifying agreement, unmet agreement, full-paid, pending/rejected
money, wrong party/source/mode, no-VIN consignment, later allocated future VIN,
ordinary purchase compatibility, partial/repeat/sold/returned, owner unchanged,
custody moved once and financial obligation unchanged. Inject custody/Transfer/
Deliver, capacity, status/agreement persistence, shipment/order CAS and event
failures; assert rollback and bound transaction. Keep P18b rejection ledger logic.

App seam: add/use a meaningful `TestWorkflowPort21` or `TestWorkflowPort21_*`
pure fake-port case in workflow_ports_test.go. Introduce a small app-local
interface around P20's exported root service if needed, within app.go. Assert
all custody inputs/context and propagated failure; prove Transfer is not called.
Do not start DB or import inventory internal layers. Report a concrete coverage
gap if the final adapter cannot be meaningfully faked within these paths.

Run each affected family once after relevant changes:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestConsignment(Handover|PurchaseHandover)'`
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort21($|_)'`

Do not run app OpenAPI/e2e.DB or substitute build/integration for the seam unit.
If shared a1 behavior changes, run only its actually affected unit family once.
Handoff dispatch matrix, money source/conditions, custody signature, active
lookup, preserved rejection behavior and quantity seam to a3/P21b. Report tests,
actual changed paths/hashes and exclusions. Stop on missing predecessor API,
ownership expansion, undefined money rule or context ceiling.
