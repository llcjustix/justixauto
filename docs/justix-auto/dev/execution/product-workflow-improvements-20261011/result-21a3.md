# P21a3 — purchase quantity receipt provenance result

Status: SOURCE COMPLETE for the approved P21a3 packet; sole source-writer lease RELEASED after this report. Work used `feature/company-commerce-workflows-20261011` at the hub-supplied local HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, preserving the a1/a2 and earlier source. `bash tools/check-git.sh` passed before edits. No Git write, database/SQL/runtime action, deployment, build, lint, typecheck, browser/integration check or automatic review/QA loop ran.

## Receipt and claim relationship

The purchase quantity `Fulfilment.Ship` path retains P21a2's `physicalOrder` source lock, accepted-mode dispatch and `purchaseHandoverAllowed` money gate before movement. Consignment quantity shipping still rejects before `Deliver`; unrelated ordinary orders retain their prior receipt path. After the exact shipment plan and inventory `Deliver`, `shipQuantity` persists the new `ShipmentLine` rows, then `materializeReceiptClaims` consumes one existing held order-owned claim for each received unit before the order CAS/event, all inside the same `Store.InTx` and bound inventory context. It creates no claim or capacity and takes no client-selected reservation, lot or claim ID.

Each claim has `MaterializationKind = quantity-receipt`, `MaterializationRef =` the preassigned and persisted `ShipmentLine.ID`, and a positive ordinal `1..ShipmentLine.Quantity`. The line's `ReceiptBatchID` links to the inventory receipt batch, while claim order/reservation/source-line and order-line IDs retain the accepted obligation. Migration 041 enforces uniqueness of `(materialization_kind, materialization_ref, materialization_unit_ordinal)` and immutable materialization fields; it has no foreign key from the generic reference to shipment lines. The service validates persisted row UUIDs, the unique converted reservation, source and order parties, accepted digests/revision/mode, exact source-line-to-order-line mapping and quantity/model/specification/colors, supplier lot identity and held claim ownership under the source reservation lock followed by sorted lot and claim locks. Missing or corrupt linkage and insufficient matching held units fail closed and roll back the receipt transaction.

For total 10/reserved 4, receiving 2 yields available 6/held 2/materialized 2. Receiving or assigning the final 2 yields 6/0/4; mixed previously allocated VINs and quantity receipts count once each. Existing shipment-line quantity guards prevent later ordinary `Allocate` from exceeding the order line. More specifically, `materializeOrderClaims` now reuses a materialized unassociated claim only when its immutable kind is `vin-allocation`; a quantity receipt claim never becomes a retired physical VIN claim, even when held future claims remain. Legitimate retired VIN reassociation still preserves its original reference and does not consume a second unit. Materialized claims remain consumed after physical release/rejection/return.

Inventory `Identify` remains outside this packet and creates VIN units while changing counts on the existing receipt batch. The commerce shipment-line/batch link and claim materialization stay unchanged; Identify has no second commerce consumption call. Service units simulate a later identified batch and verify the original linkage, but **real inventory Identify integration was not executed** and no direct VIN-to-claim-ordinal mapping is claimed.

## Affected unit evidence

After the last relevant source change, the approved selector passed:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^(TestReservationQuantityReceipt|TestReservationOrderAllocation)'
ok  justixauto/internal/modules/commerce/service  0.450s
```

The real `Fulfilment.Ship`/`Allocate` service methods run against transaction fakes that deep-copy claims, receipt batches, shipment lines, shipments, order state, allocations, holders and events on failure. Cases cover full and partial receipt batches, mixed allocated VINs, ordinary-order compatibility, identified-batch linkage, retired VIN reuse without receipt-claim reuse, exhausted fulfilled quantity, stale/repeated calls, missing or altered source/party/line and lot model/specification/colors, and injected `Deliver`, shipment, `AddShipmentLines`, claim write, order CAS and event failures. The existing `TestReservationOrderAllocation` family continues to exercise post-conversion allocation and retired-VIN conservation. Neither SQL constraints/row-lock races nor deployed cross-module rollback were executed under unit-only development verification. P21a3 did not change P21a2 money or mode dispatch; its consignment unit selector was not repeated.

## Changed source SHA-256

```text
e24236ff67240e4e863b131608d25278b1d5bc25f4eb5ec5a74262ccf24ed8d6  internal/modules/commerce/service/fulfilment.go
207161bca51ed74e1a7264b51bfdb1be48f509e7eb029295785b08ef96a3c363  internal/modules/commerce/service/reservation_allocation.go
ede05f16550a0ac718e000c6483b35d9d205a6f062addeb1732330add8520fec  internal/modules/commerce/service/reservation_allocation_test.go
```
