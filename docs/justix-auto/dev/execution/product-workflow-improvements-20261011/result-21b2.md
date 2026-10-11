# P21b2 — trusted consigned stock hold and customer delivery result

Status: SOURCE COMPLETE. The sole source-writer lease is RELEASED after this
report. Work used `feature/company-commerce-workflows-20261011` at unchanged
local Git HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, preserving
predecessor source. `bash tools/check-git.sh` passed before edits. The only
additional source path is the exact root type alias approved in
`p21b2-root-alias-scope.md`. No Git write, database/SQL/runtime action,
deployment, build, lint, typecheck, browser/integration test or automatic
review/QA loop ran.

## Root API and caller contract

`inventory.ConsignmentSaleInput` aliases the new service input in
`inventory/module.go`. Existing `inventory.StockService` exposes:

- `ReserveConsignmentSale(ctx, in inventory.ConsignmentSaleInput) error`
- `DeliverConsignmentSale(ctx, in inventory.ConsignmentSaleInput) (factID string, err error)`

The input contains `UnitID`, `AgreementID`, `OrderID`, `OrderLineID`,
`VehicleID`, `VIN`, `SupplierCompanyID`, `DealerCompanyID`, `HandoverFactID`,
`Revision int64`, `RetailDealID`, `ActorUserID`, and `At time.Time`. Both
operations require complete checked authority IDs, exact VIN, distinct
supplier/dealer and positive revision; actual delivery additionally requires
actor and time. The caller obtains these facts from P21b1 and **retains its
live commerce unit row lock in an outer bound retail transaction** throughout
stock and later commerce/retail writes. Inventory cannot read commerce tables
or independently establish the accepted agreement. Do not call these methods
from a standalone unbound retail flow.

The input has no warehouse ID: inventory row-locks the current placement and
its warehouse, verifying that the warehouse belongs to the dealer. Under the
vehicle → placement → active hold → dealer warehouse lock order, reserve
rechecks exact VIN, supplier ownership and dealer custody. It creates only the
actual dealer `retail-deal` hold; the same company/holder retry is a no-op and
a competing hold fails without release. Normal owner/custodian `Reserve` and
holder-scoped `Release` remain unchanged. A cancellation can use existing
`Release` for the exact retail holder, leaving supplier ownership, dealer
custody, placement and consignment authority intact.

Delivery repeats physical checks and requires the exact held dealer/retail
deal. It propagates `DeletePlacement` failure, clears ownership and custody,
touches the locked dealer warehouse, finalizes only that hold and appends
`vehicle.delivered_to_customer` with agreement/unit/order/handover/deal links.
The returned immutable fact ID is the `deliveryRef` for P21b1
`CompleteConsignmentSale`. A repeated or wrong-holder delivery fails. All
inventory writes join the caller's ambient transaction; any later commerce,
retail, listing or event failure must be propagated so that the outer
transaction rolls them back.

## Affected unit evidence

After the final relevant test change the authorized selector passed:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/inventory/service -run '^TestConsignmentStock'
ok  justixauto/internal/modules/inventory/service  0.448s
```

`TestConsignmentStock*` drives actual stock methods with deep-copy transaction
fakes. It covers dealer hold creation/retry/competing holder, matching-holder
release, false custody, owner/VIN/warehouse/placement/authority denial,
supplier bypass denial and ordinary dealer/distributor owner compatibility,
wrong-holder and repeated delivery, delivery fact/order links, and injected
lookup/hold creation/placement deletion/ownership/warehouse/hold finalization/
fact failures. Every participating fake method requires the outer bound
context marker. A preliminary selector ran before the final test addition;
the displayed pass is the final relevant run. These units do not prove actual
PostgreSQL row-lock behavior, installed migrations or deployed cross-module
rollback.

## Final source SHA-256

```text
0eb8ac30b06bfd434bc181405d0e9334333a7c6e800f7ff0b4b557efa2108e2e  internal/modules/inventory/service/stock.go
a5930c48cabc63ee3df95590ad0afe05ce23ba2071b7e62b058544921004ae45  internal/modules/inventory/service/consignment_sale_test.go
83fb1861eab95ebbe97b12cd3a16e2ec138abf34793f5affd8bafe1251e77b9f  internal/modules/inventory/module.go
```
