# P18b2 result — mutual reservation continuation and order claim conservation

Status: SOURCE COMPLETE for approved P18b2. Sole source-writer lease RELEASED after this report. Work used `feature/company-commerce-workflows-20261011` at milestone HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, preserving predecessor dirty source. `bash tools/check-git.sh` passed before edits. No Git write, database/runtime action, deployment, build, lint, typecheck, browser/integration check or review/QA loop ran.

## Saved agreement and HTTP contract

The accepted booking `CurrentTerms`, digest, revision, effective deadlines and release fields remain separate. Migration 042 and the reservation model now persist a proposed and accepted continuation JSON snapshot, digest, revision and proposing/accepting company. A new proposal clears previous acceptance; the other party must accept its exact digest and proposal revision. Digests are checked after decoding/normalizing JSONB, so database key ordering does not invalidate the agreement. Both parties need `commerce.trade`; a converted, released, expired or effectively advance-due source without any submitted evidence cannot convert.

`ReservationContinuation` contains `mode` (`purchase` or `consignment`), `terms` with exact source-line IDs in each line's `offerLineId`, `allowPartialPurchaseHandover`, optional `minimumPaid` exact Money, `handoverConditions` and explicit `handoverTerms`. `terms` has exact accepted source quantities/model/spec/colors, negotiated positive unit prices/currency, chosen route and an explicitly supplied settlement schedule (an explicit empty array is valid). No fee, default schedule, deposit threshold or all-VIN prerequisite was introduced. The partial flag defaults false and is valid only for purchase. Physical enforcement belongs to P21a.

Routes under `/api/v1/commerce`, active-company middleware and `commerce.trade`:

| Route | Input | Response |
| --- | --- | --- |
| `POST /reservations/:id/continuation-proposals` | Reservation If-Match; `ReservationContinuation` body | `httpx.Data` 200, reservation ETag/revision |
| `POST /reservations/:id/continuation-acceptances` | Reservation If-Match; `{digest, proposedRevision}` with decimal-string revision | `httpx.Data` 200, reservation ETag/revision |
| `POST /reservations/:id/conversions` | Reservation If-Match; `{digest, acceptedRevision}` with decimal-string revision | `httpx.Data` 201, reservation ETag/revision |

Reservation GET/list and mutation DTOs expose `continuationProposed`, `continuationAccepted`, their digests/revisions and party IDs. The conversion response data has `reservation`, `reservationId`, `reservationRevision`, `orderId`, `orderRevision`, `continuationAccepted`; the top-level ETag refers to the reservation revision. Order revision is separately named. The source reservation becomes `converted`, retains readable history and points to its unique order. Its proposed/accepted continuation and original accepted booking remain intact.

## Atomic order and claim seam

Conversion locks the reservation, rechecks agreement/state and P18a money, creates one accepted `Source: reservation` order, transfers P17b's exact existing held/materialized claims and physical VIN holders to it, inserts ordinary allocations for existing VINs, links the accepted advance once via `LinkCreditLocked`, updates the reservation by version and appends both events in one `Store.InTx`. Failure rolls back all participating fake state. The order's terms JSON carries `reservation` provenance: source reservation ID, booking/continuation digests, accepted continuation revision/mode, partial-handover flag, optional exact minimum and agreed handover text. Each order line's `offerLineId` is its immutable reservation source line mapping. Economic addenda preserve this provenance, rather than accepting client replacement.

Existing `POST /orders/:id/allocations` keeps its input and order If-Match. `Fulfilment.Allocate` now calls the P17b `materializeOrderClaims` helper after ordinary line/quantity/VIN validation and before bound order-holder Stock.Reserve, AddAllocations, order CAS and event. The helper requires the saved source/digest/line identity, exact VIN lot specification/colors and same order-owned claim, reusing retired materialized claims without another future decrement. Ordinary orders with no reservation source keep their previous path; a marked order with missing source fails closed. Rejected receipt retires its current order/VIN claim association without returning materialized capacity. An otherwise eligible `Deal.CancelOrder` locks the converted source before credit-sensitive checks, retains existing state/payment blockers, releases only residual held order-owned claims once and retires current physical associations alongside ordinary order-holder release. Reservation expiry/cancellation cannot release transferred claims.

P21a consumes `order.DecodeTerms().Reservation` as the authoritative purchase/consignment and partial-handover agreement. It must check credit-inclusive invoice Paid once, enforce actual allocated VIN custody for consignment, and at purchase quantity receipt consume held order-owned claims exactly once with shipment-line/receipt-batch provenance. `SupplyClaim.MaterializationKind/Ref/UnitOrdinal` is the durable seam; later VIN identification must reuse that claim and never reopen future capacity. This packet did not implement physical handover, ownership/custody, retail authorization or quantity receipt materialization.

## Affected unit evidence and limits

Final relevant service selector passed after the last service change:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^(TestReservationConversion|TestReservationOrderAllocation|TestReservationAllocation|TestReservationLifecycle)'`

Result: `ok justixauto/internal/modules/commerce/service 0.413s`. A prior combined affected service/handler selector passed after the handler route/DTO edit; later edits changed service and service tests only, so handler units were not repeated. New fake-backed cases exercise no/some/all VIN conversion to the same 6/0/4 capacity, changed/stale agreement and cross-party acceptance, future-only consignment, accepted-credit rollback and cancellation blocker, holder/claim/allocation/order/event failures, later ordinary Allocate, wrong specification/missing source, ordinary-order compatibility and cancellation's materialized-versus-residual conservation. The fake transaction deep-copies reservation/order/claim/association/holder/credit/allocation/event state. SQL schema, live row-lock races, deployed cross-module rollback and the P21 receipt/custody paths remain outside unit-only development evidence.

## Final source SHA-256

```text
02f2a03c8ce09ceed7bbbfcd71e0e4a95ee1414ba5461c300bb9dea2291cfe33  internal/modules/commerce/model/reservation.go
b24712aec2fb0b4dd1682049ba4e6bf5bfbc731bf263967bedffeb2ebb23439b  internal/modules/commerce/model/order.go
6bed60aa879914c9420da76ee8b52ab16589b56d3c14c4d157fd829142f11c9b  internal/modules/commerce/model/offer.go
a54bf75d5525f421e6c8802f96867327697375f3418ebef8f7369054b1b53ba8  internal/modules/commerce/service/reservation.go
26964ddf07eb5e8c1159ebecef76d20477cdcfe84fa716678b12d5f8c0b69c07  internal/modules/commerce/service/reservation_allocation.go
182251216a08b694b4d96f05abc33461321e57ec7e1463a5af85ebe11961ab9e  internal/modules/commerce/service/reservation_allocation_test.go
d168f006b0c362e41e32be0d9f46e156f8ec041b1fedf28b836357744621bed6  internal/modules/commerce/service/fulfilment.go
0b2ba251aa9af75bc8009626663578a8bd71589cb34c362fbb0635184c58abe2  internal/modules/commerce/service/deal.go
67b0a1a4899407d4afb3f2b1637147462445e85a6eed1aa46abd6481c44bcde9  internal/modules/commerce/repository/reservation.go
dfbd033002fb4dbdfc6ab42b82f1f386daefc8cb9841d213fbcfed034af63a40  internal/modules/commerce/handler/reservation.go
4fd3d4bba8383bf723a9f493f87d230f338cf1a117d443b29392a24bb7fb4b93  internal/modules/commerce/handler/handler.go
ff3549c29d82baac53da49f5c727a38cb6b36de77f97db8c5732b68773e6b8fd  migrations/000042_commerce_reservations.up.sql
```
