# P20 — Inventory custody-only consignment port result

Status: DONE. The sole source-writer lease is released after this result. Work
used the assigned `feature/company-commerce-workflows-20261011` checkout at
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74` and preserved predecessor changes.
`bash tools/check-git.sh` passed before source edits. No Git write, migration,
runtime/database action, lint, build, vet/race, browser/integration test,
review, or QA loop ran.

## Trusted root port

Inventory now exposes `(*inventory.Module).Custody(ctx, inventory.CustodyInput)
(*inventory.CustodyResult, error)` only through its module root; no HTTP route
was added. `CustodyInput` carries purpose (`consignment` or `return`), owner,
source custodian, destination company and warehouse, expected holder, agreement
ID, caller operation reference, actor, vehicle IDs and movement time. Commerce
remains responsible for verifying the agreement because inventory has no
commerce-table access. On success `CustodyResult.FactIDs` returns the immutable
inventory fact IDs; each fact also stores `agreementId` and `operationRef` in
its details, giving P21a2/P22 an exact durable movement link without a table.

Consignment requires exactly the active `commerce-order` expected holder and
finalizes only those matching held rows in the same transaction. Return accepts
no active hold and finalizes none. Both require immutable owner plus exact
current source custodian; legal ownership never changes. The source warehouse
comes from each locked current placement, so a legitimate dealer same-company
Move after handover remains a valid return source. Repeats, changed custody or
placement, wrong holder, active competing return hold, or capacity failure fail
without appended facts. Existing purchase `Transfer` and ordinary `Move` were
not changed.

Vehicles are locked in sorted ID order, followed by their placements and active
holds; source/current and destination warehouses are then locked in sorted ID
order. Capacity is rechecked under those locks; vehicle custodian/placement,
matched finalization, warehouse revisions and facts share the ambient
transaction. Repository additions are `SetCustodian` and `LockHeldByVehicle`.

## Ordinary supply eligibility

Ordinary `Reserve` now requires the acting company to be both owner and current
custodian. The existing eligible query adds the same `custodian_company_id =
CompanyID` predicate beside owner, VIN, placement and active-hold predicates.
This prevents a supplier from rediscovering/reallocating a dealer-custodied
consignment VIN after its supply hold is consumed. Generic owner-or-placement
read visibility and `Vehicle.Move` are unchanged. The existing holder fixture
now supplies current custodian and asserts the supplier cannot reserve a
dealer-custodied vehicle.

## Unit evidence

Ran once after the final relevant edits with the pinned Go toolchain and
`GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011`:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/inventory/service -run '^(TestCustody|TestReservationHolderNamespace)$'
ok   justixauto/internal/modules/inventory/service  0.227s

env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/inventory/repository -run '^TestVehicleFilterEligibleQueryShape$'
ok   justixauto/internal/modules/inventory/repository  0.278s
```

`TestCustody` covers matching-hold consignment/history/output link, repeated
handover rejection, return from the dealer's changed current warehouse,
competing return hold rejection, and capacity failure before mutation.
`TestReservationHolderNamespace` covers normal owner/custodian reserve behavior
and rejects owner-only reserve after custody moves away. The query-shape test
asserts the added custodian predicate. The tests use in-memory repository fakes
for service transitions; they do not prove PostgreSQL locking or deployment
readiness.

## Final source SHA-256

```text
c2319f9a5e77564b652bf68f23d50433cd5e42f2436299c2adc82898f66d825c  internal/modules/inventory/module.go
0d4feb389d7680697c823df4fa34dbb82abcdaeab6a0c131324190ee9efaff57  internal/modules/inventory/repository/reservation.go
aeeff5b2d75ca636e429f6ee2f11fce312b2f662b835f48a2f21db59af68b0ac  internal/modules/inventory/repository/vehicle.go
7e2d97ca66685f171b9d9042e7a0ca00247f3e7b124558bd77e34ea83365bf08  internal/modules/inventory/repository/vehicle_filter_test.go
bd03fa4304bef4d6b7e1104ab8b42e7040f9d21abbe156086cf10a6a6a5e1912  internal/modules/inventory/service/custody.go
84631ba3cca84793c8d5685682d12945dcde11079fa291208c588176e69ba52a  internal/modules/inventory/service/custody_test.go
f1e982447672eb92cc88dccc7fb5943585a4b79926c6b52d1e1ef5004769cf48  internal/modules/inventory/service/ports.go
1152db351632ecf3880a16b305e99cc3ce7ac74594b3f996ef99f9efc79243e6  internal/modules/inventory/service/reservation_holder_test.go
f06918c38248aa9a02fc57dbb404c7fe66fde861149e777eaf11bacd0cca7834  internal/modules/inventory/service/stock.go
d64a95dcd85e6a3178817f8b3ea9a86d43e8e86f6a8736ac70d4da6a2664ea27  internal/modules/inventory/service/reservation_stock.go
c2b1a823d1777adccdfe852e429d86b599b3cd06241b88d8aed8869f53359e6e  internal/modules/inventory/service/reservation_stock_test.go
```

Source-writer lease: RELEASED. No further source edits are authorized by this result.
