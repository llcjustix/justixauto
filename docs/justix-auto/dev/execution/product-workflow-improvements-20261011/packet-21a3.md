# P21a3 — Purchase quantity receipt provenance

Proposed under P21a and accepted allocation seam, R3; application hub `/root`.
Requires accepted P21a2 and final P18b2 claim/progress handoff. Read common rules,
planning base/branch, hashes and stops in p21a-split-proposal.md; accepted
reservation-order-allocation-seam.md Release, receipt and physical handover.
Fresh context ceiling 14k; exclusive source-writer; no DB/browser/runtime.
Coverage C10/C11/C12; preserve C4/C5/C7. No automatic QA/review.

## Exact owned paths

- internal/modules/commerce/service/fulfilment.go
- internal/modules/commerce/service/reservation_allocation.go
- internal/modules/commerce/service/reservation_allocation_test.go
- internal/modules/commerce/service/ports.go
- internal/modules/commerce/repository/supply.go

Only original P21a plus already accepted seam additions. No extra schema,
inventory/app/handler or fulfilment-repository ownership. Internal AGENTS.md
applies: service-owned rules, typed commerce repositories, same bound transaction,
durable idempotency and immutable initial provenance. Read final supply claim
model/migrations and ShipmentLine fields as scoped dependencies only. If those
cannot represent the concrete provenance below, stop with the exact missing
column/index/path; do not silently expand ownership.

## Behavior and invariant

In the actual purchase quantity Ship receipt path, resolve the unique converted
reservation and exact accepted source-line→order-line mapping server-side.
Ordinary unrelated orders remain unchanged. An explicitly reservation-origin
order with missing/corrupt linkage fails closed. Validate source party, lot,
model/spec/colors and remaining future obligation under P18b's lock order.

Each received future unit materializes one existing order-owned unmaterialized
claim, in the same Store.InTx as Deliver's receipt batch, ShipmentLine rows,
order revision and event. Persist durable shipment-line/receipt-batch provenance
plus per-unit identity using final predecessor schema. Existing MaterializationRef,
MaterializationUnitOrdinal and ShipmentLine.ReceiptBatchID are candidate existing
fields, not authority to overwrite final schema or invent unconstrained strings.
Record the chosen exact relationship and uniqueness constraints in the result.
Generate stable row identities before using them as provenance. No free-text
only provenance, client-picked claim/lot, duplicate claim, reacquired capacity or
extra credit. Keep pending-money/consignment gating added by a2 before movement.

Retry, duplicate delivery and stale revision cannot consume a second unit.
Capacity conservation: total10/reserve4, deliver2 gives available6/future2/
materialized2; later delivery/allowed VIN allocation of remaining2 gives 6/0/4.
Mixed previously allocated and quantity-delivered units count separately once.
The existing shipment-line quantity guard must reject later Allocate of already
fulfilled quantity. More subtly, materialized receipt claims have no ordinary
VIN allocation association: do not let the existing reassociation helper reuse
them while some other future quantity remains. Reuse only legitimately retired
physical VIN claims under the final P18b contract; retain original provenance.

Later VIN identification of the same receipt batch changes inventory identity,
not commerce future consumption. Preserve batch linkage so identification and
commerce reads/ordinary Allocate cannot produce another materialization or
another delivery allocation for that quantity. Existing inventory Identify is
read-only to this packet; no new callback/public claim selector. If precise
identification evidence requires another module seam beyond final predecessors,
report the limitation and concrete hub ownership request; do not claim a fake
summary update proves inventory's real Identify execution. Consignment continues
to require genuine allocated VINs; no extension to receipt ownership is proposed.
Materialized physical release/rejection/return never reopens future availability.

## Affected units and handoff

Extend reservation_allocation_test.go with `TestReservationQuantityReceipt*`
cases invoking real Fulfilment.Ship and later real Fulfilment.Allocate through
transaction fakes. Cover all-future/mixed receipt, partial/full batches, exact
line/spec/colors/source, ordinary unrelated purchase, missing linkage, stale/
duplicate/retry, batch provenance preserved through simulated identification,
exhausted fulfilled quantity and attempt to reuse a receipt claim while another
held claim remains. Keep consignment quantity denial and accepted-money gating.
Inject Deliver, claim write, AddShipmentLines, order CAS and event failures;
deep-copy claims, receipt batches/stock, shipment lines, order and events so every
failure proves rollback. Test retired VIN reassociation remains valid without
new capacity consumption, while quantity receipt provenance is never recycled.

Run once after relevant changes:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^(TestReservationQuantityReceipt|TestReservationOrderAllocation)'`

If actual shared changes affect a2 money/mode dispatch, select its affected
`TestConsignmentPurchaseHandover` or `TestConsignmentHandover` family once as
well; no ceremonial repeat of app/inventory or all service units. No schema/DB
execution or integration claim. Report actual selectors and tests discovered.

Handoff exact provenance fields/keys, receipt→claim relationship, quantity versus
retired-VIN reassociation predicate, source lock order, final progress accounting,
actual changed paths/hashes and evidence limits. Hub maps these plus a1/a2
evidence to final candidate; P21a is complete only with all coverage C1–C12.
Stop on missing schema or cross-module authority, new product rule, unowned
fixture, changed final contract or context ceiling; no fixture-derived policy.
