# P18b2 — Agreed conversion and post-conversion order allocation

Proposed split of original P18b, canonical R3. Depends on accepted P18b1 plus actual P17b/P18a interfaces and accepted reservation-order-allocation seam. Common rules in p18b-split-proposal.md. One source writer, fresh context ceiling 19k. Domain+HTTP behavior stays one order-continuation transition; P21a retains physical handover/quantity-receipt work.

## Exact owned paths

- internal/modules/commerce/service/reservation_lifecycle.go
- internal/modules/commerce/service/reservation_lifecycle_test.go
- internal/modules/commerce/service/reservation.go
- internal/modules/commerce/service/reservation_allocation.go
- internal/modules/commerce/service/reservation_allocation_test.go
- internal/modules/commerce/service/fulfilment.go
- internal/modules/commerce/service/deal.go
- internal/modules/commerce/service/ports.go
- internal/modules/commerce/model/order.go
- internal/modules/commerce/model/offer.go
- internal/modules/commerce/repository/reservation.go
- internal/modules/commerce/repository/supply.go
- internal/modules/commerce/repository/deal.go
- internal/modules/commerce/handler/reservation.go
- internal/modules/commerce/handler/dto.go

This is original P18b union accepted allocation-seam additions. Do not alter b1 expiry algorithm merely while sharing its file. No new schema/app/inventory/payment-ledger files, no quantity-receipt dispatch.

## Mutual agreement and conversion

1. Reuse predecessor reservation version/digest workflow for an exact proposal and other-party acceptance of continuation terms. Persist purchase versus consignment mode, accepted price/currency/quantity/model/spec/colors, explicit route, payment/settlement schedule and handover agreement. Neither party may silently alter accepted terms. A changed proposal invalidates earlier acceptance. Do not invent a default schedule, deadline, percentage, fee or extra role policy.
2. allowPartialPurchaseHandover is absent/false unless explicitly mutually agreed. Preserve exact agreed minimum/conditions, represented without browser float arithmetic. This flag records the agreement; physical delivery/payment enforcement is P21a. Consignment mode preserves supplier ownership and does not itself create custody movement or retail availability.
3. Permit conversion with no VINs, some VINs or all VINs. In one Store.InTx, lock/recheck source reservation, current accepted continuation digest/version, party authorization and P18a money/refund/credit state. Use shared lock ordering from b1/P17b. No new all-VINs business prerequisite. Expired/cancelled or invalid/stale agreement fails; converted source cannot create a second order.
4. Create the unique reservation→order link and exact source-line→order-line mapping. Preserve the SAME future/materialized claim identities. Transfer existing VIN holders commerce-reservation→commerce-order through P17b atomic reassignment; create corresponding ordinary allocation rows using existing FulfilmentRepository.AddAllocations. Transfer residual future obligations without reacquiring lot capacity. The order is the sole owner of subsequent claim mutations.
5. Reconcile accepted advance via P18a credit-once operation bound to this order. Pending/rejected evidence never reduces order debt; refunded amounts cannot also be credited. Preserve unresolved money obligations under predecessor rules rather than silently discarding/reclassifying them. If P18a disallows a particular unsettled state, propagate it; do not invent an additional settlement rule. Refund/conversion race serializes on same source row and ledger business key. Credit links existing accepted money, not a second payment. Port/ledger/claim/order/event failure rolls all conversion state back.
6. Order accepted terms snapshot retains reservation/continuation provenance and all fields for contracts/settlement consumers. Existing ordinary direct-order economics/routes/revision behavior remain unchanged. Do not bypass economic-addendum acceptance or mutate a signed documentary snapshot.

## Existing order Allocate and CancelOrder

Keep POST /commerce/orders/:id/allocations input and order If-Match unchanged. Fulfilment.Allocate resolves the actual converted reservation through a locked repository lookup, never client-selected lot IDs. No source for an ordinary order means unchanged behavior; marked reservation-origin with missing linkage fails closed.

After existing order-line/quantity/VIN checks, reconcile requested allocations against the order-owned source claims using the shared reservation_allocation helper, within the same transaction as Stock.Reserve(order holder), AddAllocations, order CAS and event. Enforce exact lot specification/colors as well as ordinary line constraints. Materialize only existing unmaterialized obligations once. Reassociation of a previously materialized obligation uses its durable claim/history without a second future decrement. New post-conversion VINs reserve directly under commerce-order, without temporary reservation holder or release/reacquire.

Example total10/reserve4: 0, 2 or 4 pre-conversion VINs all converge after later allocations to available6/future0/materialized4. Duplicate request/VIN/claim or stale revision cannot change those totals. Prior transferred VINs must already appear in ordinary allocations, so the next order call neither loses nor recounts them.

Deal.CancelOrder retains current state/payment/allocation blockers. On an otherwise allowed reservation-origin cancellation, release only residual unmaterialized order claims once alongside the existing physical-holder release/state/event. Materialized claim provenance/counters stay consumed. b1 original-reservation cancellation/expiry cannot release these order-owned claims. Receipt rejection/physical return never reopens materialized future supply. Do not add new order cancellation eligibility or weaken full-refund/accounting rules.

P21a later consumes reservation-origin purchase quantity receipts with durable shipment/receipt provenance and enforces actual VIN custody for consignment. Keep that implementation outside this packet; hand off source claim helper and unresolved physical-progress count explicitly. Conversion and later VIN Allocate must already work here.

## HTTP/data boundaries and handoff

Use actual predecessor proposal/acceptance route and DTO vocabulary, with reservation If-Match and exact terms digest/version in the action that accepts/converts. Final conversion response exposes created order ID/current order revision plus source reservation ID/current revision and accepted continuation data, without ambiguous revision reuse. Record actual route/status/envelope field names for P19/P21/P13 consumers at completion. Do not change ordinary order response shape merely to expose this new action.

No auto-conversion, physical shipment or advance bank action on a read. Both parties can inspect accepted terms/source history under existing permissions. Publish precise source IDs and decimal-string revision semantics; no speculative DTO assumption survives the result handoff.

## Meaningful affected units

Use actual conversion, Fulfilment.Allocate and Deal.CancelOrder with deep-copy fake transaction state including claims, order/allocations, inventory holders, money credit and events:
- exact mutual agreement, changed/stale digest and cross-party authorization; explicit/absent handover flag and unchanged accepted pricing;
- conversion no/some/all VINs, single order link/credit/holder transfer, retry and conflicting refund/conversion;
- later partial/full Allocate against same claims, ordinary-order compatibility, wrong source/spec/color/line/duplicate/stale failures;
- source reservation expiry/cancel/allocate after conversion cannot release/reassign order claims;
- allowed order cancellation releases only residual future claims; existing blockers retained; materialized physical release does not restore future availability;
- failures at holder transfer/new reserve, credit, claim materialization, AddAllocations, order CAS and event produce no partial results; transaction context preserved.

Run affected unit families once after this change, repo root:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^(TestReservationConversion|TestReservationOrderAllocation|TestReservationAllocation|TestReservationLifecycle)'`
Name new cases accordingly. TestReservationLifecycle is affected only for shared transition/converted-state behavior; do not rerun b1 scan/extension families if unchanged. New ordinary Allocate/CancelOrder compatibility cases live in this owned fixture. No app/inventory/payment suite rerun when those implementations remain untouched.

Report result-18b2.md with actual agreement/HTTP schema, source linkage/claims, credit API, P21 quantity-receipt seam, changed paths, tests and remaining exclusions. If predecessor schema/API cannot support the accepted seam, report exact needed path to hub; no unowned edit or invented restriction. Split again before context ceiling, without dropping any acceptance.

