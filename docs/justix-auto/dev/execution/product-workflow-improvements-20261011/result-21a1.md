# P21a1 — durable consignment authority result

Status: SOURCE COMPLETE for P21a1; the sole source-writer lease is RELEASED. Work used `feature/company-commerce-workflows-20261011` at unchanged Git HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, preserving predecessor dirty source. `bash tools/check-git.sh` passed before edits. The application hub accepted the two-path conversion extension in `p21a1-conversion-scope.md`. No Git write, database/migration execution, runtime action, deployment, broad test, lint, build, typecheck, browser/integration test, review or QA loop ran.

## Authority and persistence

`Reservation.Convert` conditionally calls `Consignment.CreateFromConversionLocked` after its accepted order, transferred claims/holders, allocations and credit are written, before reservation CAS/events, inside the same `Store.InTx`. A successful consignment conversion always inserts exactly one agreement, including when zero VINs are allocated. The helper checks the current mutual continuation digest, accepted revision, order parties/source, complete order terms and immutable reservation provenance. An agreement/unit write failure rolls back order, claim/holder, credit, reservation and event state. Purchase conversion retains its existing path.

Migration `000044_commerce_consignments` defines `commerce_consignment_agreements` (unique order and reservation) and `commerce_consignment_units` (one row per agreement/VIN, plus a partial unique live VIN index). Agreement fields pin `ID`, `OrderID`, `ReservationID`, supplier owner, dealer custodian, booking digest, continuation digest/revision, creation time and version. **The converted order and accepted reservation snapshot remain the durable commercial authority** for schedule, price, handover terms and credit; the agreement stores references, not an independently mutable payment copy. Every unit keeps its order ID as its existing settlement obligation link. No invoice, commission, payment, refund or debt state is created or altered here.

Unit rows hold ID, agreement/order/order-line/vehicle/VIN, distinct owner and custodian, `prepared|held|sold|returned`, version, actual handover fact/operation/time/actor, current retail deal, pending-return ID, actual retail delivery reference/time/actor, and return time/fact. A unit is `prepared` on conversion if already allocated, with no custody or retail authority. A future-only order has its agreement and no unit rows. `PrepareUnitLocked` registers a later allocated VIN only after matching the stored current order allocation, accepted order line, parties and provenance; duplicate or invented VIN registration fails. `MarkHandedOverLocked` transitions only a matching prepared VIN to `held` using the actual P20 fact ID, operation reference, actor and time in the caller's transaction. P21a2 owns the physical P20 call and performs that bound update only after its success; P21a1 alone does not establish physical custody.

## Transaction and downstream contract

Repository operations are `CreateAgreement`, `AgreementByOrder`, `Agreement`, `CreateUnit`, `Unit`, `LockUnitByVehicle`, `UpdateUnit(expectedVersion)` and `Units`. They use `database.Conn(ctx, db)`; `LockUnitByVehicle` selects the active `prepared|held` VIN row `FOR UPDATE`, and `UpdateUnit` uses version CAS. These methods are available through `Store.Consignments()`/`storeAdapter`. `Consignment.LockUnitForDecision(ctx, st, dealerID, vehicleID)` requires the actual handed-over dealer row and leaves its lock in the caller's outer transaction. P21b/P22 must additionally recheck their retail/pending-return/physical-holder rules there, then use the same row and `UpdateUnit`; the agreement row is not the serialization point for all VINs. Current retail and pending-return associations can be cleared by later cancellation/resolution while the retail/return records and commerce events retain history. Terminal sold/returned rows cease being active and keep the order obligation link.

Bound service signatures for P21a2 are `CreateFromConversionLocked(ctx, st, reservation, order, allocations)`, `PrepareUnitLocked(ctx, st, agreement, allocation)` and `MarkHandedOverLocked(ctx, st, agreementID, vehicleID, expectedRevision, factID, operationRef, actorID, at)`. P21a2 must prepare any VIN allocated **after** conversion before or with handover in the same transaction. P20 returns fact IDs in an array while sorting input vehicle IDs internally; call P20 `Custody` for **one VIN per call** within the outer transaction and pass that call's single fact ID to `MarkHandedOverLocked`. This gives exact VIN→fact linkage without positional assumptions. The P20 operation reference and agreement ID should be the same values used in the unit row. P20 consumes only the matching commerce-order holder at consignment handover; prepared authority alone does not release a hold.

Commerce root exports `ConsignmentUnit` and `Module.ConsignmentForVehicle(ctx, dealerID, vehicleID)`, a read-only active discovery projection. It excludes prepared, sold, returned and wrong-dealer VINs; mutation callers must lock and recheck inside their own transaction. HTTP `GET /api/v1/commerce/consignments/orders/:id` requires company scope and `commerce.read`, verifies actual order party and returns `httpx.Data` with agreement ETag/revision. DTO fields are `id`, `orderId`, `reservationId`, `supplierCompanyId`, `dealerCompanyId`, `bookingDigest`, `continuationDigest`, decimal-string `continuationRevision`/`revision`, and `units` containing `id`, `orderLineId`, `vehicleId`, `vin`, `status`, decimal-string `revision`, optional `handedOverAt`, `currentRetailDealId`, `pendingReturnId`, `soldAt`, `returnedAt`. No new POST acceptance/authority route or `allowedActions` was introduced. Unit revisions are independent of the immutable agreement revision; consumers of changing unit state must use each unit revision, not the agreement ETag.

## Affected unit evidence and limits

After the final source change, ran the approved affected selector once:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^(TestConsignmentAgreement|TestReservationConversion)'
ok  justixauto/internal/modules/commerce/service  0.604s
```

Nine top-level cases were selected: four new `TestConsignmentAgreement*` and five existing `TestReservationConversion*` (with subcases). Stateful transaction fakes cover future-only agreement, allocated/prepared VIN, party isolation, changed/repeated continuation, duplicate/unallocated VIN rejection, one-VIN handover, sale/return terminal exclusion, independent per-VIN pending-return state, retained order obligation, and agreement/unit/source-CAS/event rollback. Sold/return mutations in these units exercise the generic persisted row contract; P21b/P22 own their product-policy services. Unit fakes do not prove PostgreSQL constraints/row-lock races, installed migration, P20 physical composition, HTTP route execution or release readiness.

## Final source SHA-256

```text
adee714f419fc42f2247174ae521fcddf4af49d60ded238a846e72098458a475  internal/modules/commerce/model/consignment.go
cfca49e5e238830ef1ebf1fe86f1095ec8aab0bdafbe6ba631f7cf2bb29bc003  internal/modules/commerce/service/consignment.go
68c9f3979c9a7daa99250450067dfadb5780bca8af72c68b9a4ee3e019254671  internal/modules/commerce/service/consignment_test.go
687660d686d3ce39e77af3d651eb025df6e68e17cea9f62a5986aca2d4664017  internal/modules/commerce/service/ports.go
cc0b577944b56b258a5e82ae1749e8d8731ac79228c89d98b467c87575f09ac4  internal/modules/commerce/service/reservation.go
2224d696a4c02870e08c4901711c780d5622b1c38d722fdfdf076c77b0c78533  internal/modules/commerce/service/reservation_allocation_test.go
e7dec3a8969460245fd22415faeb53065bac9adceb2d27c6ed415319ebf50f91  internal/modules/commerce/repository/consignment.go
c484020c03d87d6a259d7244aadca898e7ab3f17f408a63efddc74dc9722a5f1  internal/modules/commerce/repository/store.go
6ebb8c9a1afd1779b21770673d5a67e934ff0a309e29426f588430a2bb4e5214  internal/modules/commerce/handler/consignment.go
92c9c6a0c700191f03ab5d57bf5b1984cdc52ed0d18d85e8c4801466538a1d86  internal/modules/commerce/handler/handler.go
3e1b9aab6e41c3803b94ee64b3984b9bd99bd7a8c95d2676246e03d376ac1f16  internal/modules/commerce/handler/dto.go
56453f927e869b205f3363cac8e6c74c6951e63265f14bc29086d26e9bc186a4  internal/modules/commerce/module.go
58ac2bba747fcb17c2e0e4f66c8148bc0427160fdfd7cb567b3191caa5df8a8c  migrations/000044_commerce_consignments.up.sql
e9fa02ca26647baac0e6f72d564cd44417fc6eef9f710c5fb8fea89aea9a5c59  migrations/000044_commerce_consignments.down.sql
```

The migration pair was created at the hub-confirmed available source number 044; no database state or application of this migration is claimed.
