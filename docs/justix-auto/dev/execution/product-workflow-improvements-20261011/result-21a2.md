# P21a2 — accepted-mode physical handover result

Status: SOURCE COMPLETE for P21a2. Sole source-writer lease RELEASED after this report. Work used `feature/company-commerce-workflows-20261011` at unchanged local Git HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, preserving predecessor dirty source. `bash tools/check-git.sh` passed before source edits. No Git write, database, SQL/migration execution, deployment, browser/integration check, lint, build, typecheck, vet/race, review or QA loop ran.

## Physical dispatch and agreement

`Fulfilment.Ship` has all three accepted-mode outcomes: quantity shipment keeps ordinary and reservation-purchase receipt batches after its money gate, while consignment quantity shipment fails before `Deliver` or shipment creation even if VIN strings are supplied; allocated shipment without a receiving warehouse remains in transit; allocated shipment with a receiving warehouse dispatches purchase to `Transfer` and consignment to custody. `DecideReceipt` applies the same purchase money gate or consignment custody on acceptance of only the selected shipped VINs. Its rejection path still retires P18b order claims before stock release, without granting custody. Existing ordinary orders continue through their prior paths.

Before a reservation physical decision, `physicalOrder` validates the converted source under `CreditForOrderLocked`, rereads the order after that source lock, and compares its immutable handover provenance against the exact mutually accepted continuation digest/revision, partial flag, optional minimum and handover text. Purchase debt is calculated from the accepted invoice view when an issued invoice exists: its `Paid` already includes the assigned reservation advance once, and submitted/rejected evidence contributes no paid amount. Before invoice issue, the unique accepted source credit is the only paid amount. Positive outstanding debt blocks absent the explicit partial-purchase flag. With that flag, the optional exact `MinimumPaid` is enforced; no default minimum is imposed. Commerce retains the order's face obligation and accepted settlement terms after custody.

Consignment allocation now registers later assigned VINs as `prepared` in the same order allocation transaction. Actual handover requires a matching current allocation, stored agreement and prepared VIN row. Each selected VIN gets one bound P20 `Custody` call with supplier owner/source custodian, dealer destination and warehouse, exact commerce-order holder, agreement ID, operation reference, actor and time. That call's single immutable fact ID is saved by `MarkHandedOverLocked` on that VIN in the same commerce transaction. Supplier ownership stays intact; the P20 hold and capacity checks apply. Repeated, sold, returned, mismatched or stale units cannot hand over. The app adapter uses the inventory module-root custody API, and the focused fake verifies its complete input, bound context, fact result and failure propagation without invoking `Transfer`.

## P21a3 receipt seam

Purchase quantity work enters `shipQuantity` after `physicalOrder` and `purchaseHandoverAllowed`, then validates the exact shipment lines and colors. It creates `sh`, calls `stock.Deliver` per planned line and persists `ShipmentLine{ShipmentID, OrderID, LineID, Quantity, ReceiptBatchID}` before order CAS/event. P21a3 must consume held order-owned future claims once with shipment-line/receipt-batch provenance in this transaction, without changing the consignment quantity rejection or re-consuming on later VIN identification. Allocated VINs continue through `Ship`/`DecideReceipt`; no quantity claim consumption is claimed by P21a2.

## Unit evidence and limits

Final affected selectors passed after their last relevant edits:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestConsignment(Handover|PurchaseHandover)'
ok justixauto/internal/modules/commerce/service 0.507s

env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort21($|_)'
ok justixauto/internal/app 0.757s
```

The first service run exposed a missing catalogue in the new quantity test fixture; that owned fixture was corrected. The first app run exposed an identical duplicate `httpx` import left in P21a1's `handler/dto.go`. The application hub approved the exact one-line predecessor correction in `p21a2-handler-import-scope.md`; the app selector then passed. Later service-only changes were followed by the affected service selector, not a repeat of the app selector. Stateful service fakes cover immediate and later receipt custody, future VIN allocation, quantity consignment rejection, purchase partial/full/pending money, ordinary transfer, and rollback on custody/capacity/agreement/unit/shipment/allocation/order/event failures. The unit fakes do not prove SQL locking, installed migrations or deployed cross-module transaction behavior.

## Final source SHA-256

```text
6d8fc21c9b70773113c4f90599121b7bf355a9beaba570f83977bd7f4105ff44  internal/modules/commerce/service/consignment.go
247fdbca7c6f103f76f535b17a96aeb538b2e95d8e5971be2781ba4d015caf63  internal/modules/commerce/service/consignment_test.go
76fac1493a85e31930fb0208190eaf3f0e4a24fb8c08925ef04d639d8d4fc28a  internal/modules/commerce/service/fulfilment.go
711d93b787b19a3d177436c73192b64217caceb89d06899c2be033e4a84a45d9  internal/modules/commerce/service/ports.go
0e6646f4508fc3377c5eabfbc965715df4618c713fb8ddab79e97d7ca3cf9571  internal/modules/commerce/module.go
34a8f663a711ea8180ecf23c3675018c759a75477f2c9dc3cf4557c99462c266  internal/app/app.go
db0bc3a5bc890279ca68aa3d727c4a0b1d6ffa65aee3d0a9b9dd4e4a8af56b5b  internal/app/workflow_ports_test.go
bc2b67c80c81128f417b662a6909ff6740df299ba5da437ba6796f62c217a753  internal/modules/commerce/handler/dto.go
```

No other source path was edited. P21a remains incomplete until P21a3's quantity claim provenance and the dependent retail/return work are done.
