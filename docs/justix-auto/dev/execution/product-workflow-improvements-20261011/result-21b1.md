# P21b1 — live consignment sale authority result

Status: SOURCE COMPLETE for the approved P21b1 packet; sole source-writer lease RELEASED. Work used the assigned `feature/company-commerce-workflows-20261011` checkout at hub-supplied local HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74` and preserved predecessor source. `bash tools/check-git.sh` passed before source edits. There was no Git write, database/SQL/runtime action, deployment, build, lint, typecheck, browser/integration test or automatic review/QA loop.

## Trusted root contract for P21b3

Commerce root exports `ConsignmentSaleAuthority` with `UnitID`, `AgreementID`, `OrderID`, `OrderLineID`, `VehicleID`, `VIN`, `SupplierCompanyID`, `DealerCompanyID`, `HandoverFactID` and `Revision`. `OrderID` is the existing accepted supplier settlement obligation; the implementation adds no payment, refund, invoice, commission, debt or new settlement copy. The immutable P20 handover fact is required before sale authorization. The root operations are:

- `CheckConsignmentSale(ctx, sellerID, vehicleID) (*ConsignmentSaleAuthority, error)`: listing/ordinary-fallback check. Only a truly absent live `prepared|held` VIN row returns `(nil, nil)`. A live row for the supplier, another dealer, a prepared/unhanded VIN or a pending return returns `ErrForbidden` with code `consignment_sale_denied`; an already retail-associated row returns `ErrConflict` with code `consignment_retail_reserved`. Read failures propagate. The old dealer-scoped `ConsignmentForVehicle` remains discovery only and must not decide ordinary fallback.
- `ReserveConsignmentSale(ctx, sellerID, vehicleID, dealID, actorID) (*ConsignmentSaleAuthority, error)`: locks the live VIN, requires unreserved authority, persists its exact `CurrentRetailDealID` by version CAS and appends an event. `(nil, nil)` means no live consignment row and permits the ordinary owned-stock branch; an error never permits fallback.
- `ConsignmentSaleForDeal(ctx, sellerID, vehicleID, dealID) (*ConsignmentSaleAuthority, error)`: relocks and requires the exact current retail deal before cancellation or physical delivery. Missing live authority is an error here, never ordinary fallback.
- `CancelConsignmentSale(ctx, sellerID, vehicleID, dealID, actorID) error`: clears only the matching current association by CAS, retains the unsold held unit, accepted order, original handover and event history. A stale/wrong deal cannot clear another association.
- `CompleteConsignmentSale(ctx, sellerID, vehicleID, dealID, deliveryRef, actorID, at) error`: after actual customer delivery in the same transaction, changes the exact held unit to sold and durably saves the retail deal, delivery reference/time/actor and existing order link. Repeated completion sees no live row and fails. It never writes payment or changes accepted financial terms.

Every operation acquires the P21a1 `commerce_consignment_units` live VIN row by `LockUnitByVehicle` (`FOR UPDATE`, independent of dealer) **before** agreement/order reads and inventory vehicle/hold locks. It validates dealer, supplier, VIN, accepted agreement/order parties, reservation provenance and order line; then checks pending return and the exact retail association for its action. The row ID is the serialization identity, and `UpdateUnit(expectedVersion)` supplies CAS. One VIN's pending return does not block another unit in the agreement. P22 must lock and recheck this same row before establishing a pending return. The partial unique live-vehicle index in migration 044 ensures at most one active authority row for the VIN.

P21b3 must call these operations with its retail `Store.Bind` context inside one outer retail transaction and retain that outer transaction through the inventory hold/release/customer-delivery operation and retail deal/event writes. Commerce `Store.InTx` joins the ambient transaction as a savepoint; an unbound standalone call would release its row lock before stock work and is not the mutation contract. Suggested order is `ReserveConsignmentSale` → trusted inventory retail hold → retail deal write; or `ConsignmentSaleForDeal` → matching hold release/customer delivery → `CancelConsignmentSale`/`CompleteConsignmentSale` → retail writes. Propagate every failure so the outer transaction rolls back all modules. Completion's `deliveryRef` should be the actual inventory customer-delivery fact/reference, not a generated request key.

## Affected unit evidence and limits

The approved selector passed after the final relevant source change:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestConsignmentSaleAuthority'
ok  justixauto/internal/modules/commerce/service  0.705s
```

Stateful transaction fakes cover ordinary absence versus live supplier/other-dealer denial, prepared/unhanded and terminal VINs, pending return, agreement/line/read failure, exact identity and obligation, wrong/repeated retail association, cancellation preserving history, both sale-first and return-first shared-unit schedules, stale CAS, event failure and outer rollback. They exercise actual service methods. They do not prove PostgreSQL lock races, installed migration, physical stock behavior or deployed cross-module rollback. Only the assigned unit family ran; no additional development verification was performed.

## Final source SHA-256

```text
54c9750633d254af3b588eb9fa48b8a00526bce6768658c6deefbd0c185cee11  internal/modules/commerce/service/consignment.go
481a4d03fd24bcdbd86909b652c3654ae6f6dee1fbff44f36decbe3a055e611b  internal/modules/commerce/service/consignment_sale_test.go
e15519a411b8ac5fc0554f50da584e11ee4249b7a58e5c1fa3d9e52285b0005d  internal/modules/commerce/module.go
```

`internal/modules/commerce/repository/consignment.go` and `internal/modules/commerce/service/ports.go` were already sufficient and were not changed. No other source path was edited.
