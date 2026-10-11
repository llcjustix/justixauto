# Proposed reservation → order allocation seam

Application hub /root; existing R3, P17b/P18b/P21a scope. Base supplied by hub: 4fa683ba4708bb71031534260ec5cb8742ce9977; feature/company-commerce-workflows-20261011, shared dirty work preserved. Docs-only preparation while P14 writer owns source. Original packets/canonical plan are unchanged. Hub acceptance required before assigning the ownership clarification below; no review/QA loop or new product decision.

## Invariant and concrete gap

Reservation conversion is allowed with zero, some or all future units already assigned VINs. Conversion transfers both physical holds and remaining future obligations; it does not require all VINs. Consignment physical handover still requires actual VINs.

For a lot, available + unmaterialized active claims + permanently materialized/consumed claims = total. Materialization changes one existing claim's state; it never creates a second claim or returns that unit to future availability. Releasing/rejecting/returning a physical VIN does not unmaterialize its claim. Only an eligible cancellation of an unmaterialized claim restores future availability, under existing money/refund rules.

Example: total 10, reserve 4 → available 6, future 4, materialized 0. Assign 2 before conversion → 6/2/2. Convert → 6/2/2 and all four claims belong to the order obligation. Assign the last 2 after conversion → 6/0/4. An all-future conversion followed by four later assignments reaches the same result.

Current Fulfilment.Allocate (service/fulfilment.go) calls supplierOrder → planAllocations → Stock.Reserve(order holder) → AddAllocations → UpdateOrder/event. It never reconciles a supply claim. Merely transferring initial reservation holds therefore leaves later allocations unaccounted. Current CancelOrder releases order VINs but has no future-claim release. Existing receipt rejection releases physical holds; it must not restore lot future capacity.

## Proposed interface and transaction boundary

Use one durable reservation/lot claim identity across conversion. P16/P17a's still-planned supply/reservation schema must retain source lot/reservation, quantity/claim state, converted order reference and exact reservation-line→order-line mapping. Materialized claims additionally retain immutable materialization provenance plus the current allocation/fulfilment association. A grouped quantity ledger is acceptable if it provides equivalent exact per-transition identity and no double consumption. Do not encode provenance only in free text or infer it from equal model/color.

- Reservation repository: an order-scoped lookup/lock such as ByOrderForUpdate(ctx, orderID) resolves the unique converted source reservation and accepted line mapping. Ordinary orders return no source; an order explicitly marked reservation-origin with missing linkage fails closed.
- Supply repository: an atomic materialization operation keyed by existing claim/order/line and allocation ID transitions only the requested unmaterialized quantity into materialized quantity. A stable unique key rejects duplicate application. Existing materialized-claim associations can be reused for legitimate reallocation without a second future decrement; keep previous physical association history. This does not authorize new replacement pricing/refund policy.
- Commerce-local helper in service/reservation_allocation.go, for example materializeOrderClaims(ctx, st, order, plannedAllocations), runs within its caller's Store.InTx, with no nested independent commit. It loads the source, locks/rechecks claims, validates exact lot model/specification/colors and remaining obligation, then records materialization. No browser-supplied lot/reservation/claim owner is trusted.
- Fulfilment.Allocate keeps its existing route/input/If-Match and ordinary-order behavior. Inside the same transaction, after ordinary allocation validation and before commit, invoke that helper for reservation-origin orders, reserve each new VIN directly under commerce-order, add allocation rows, and bump the order revision/event. Claims, physical reservation, allocations and order revision rollback together on any failure. Do not call ReserveReservation and then transfer for new post-conversion VINs.
- Conversion moves existing materialized VIN holders commerce-reservation→commerce-order atomically and creates matching ordinary commerce allocation rows using existing FulfilmentRepository.AddAllocations. It transfers the SAME remaining future claims and line mapping to the unique order link. Do not release/reacquire, reacquire lot capacity or count already transferred VINs again on first ordinary Allocate.
- Once converted, original reservation allocate/expiry/cancel paths cannot mutate/release its claims. Order operations own them. Stale pre-conversion requests conflict and roll back; history stays readable. All mutating paths must use one documented lock order for lot/reservation/order rows and recheck state after locks, in addition to expected revision. No claim of exactly-once deployment behavior.

P17b owns the shared materialization/holder primitives; P18b wires conversion and the existing order Allocate/CancelOrder paths. This keeps future-only and partial conversions usable as soon as the continuation feature is connected.

## Release, receipt and physical handover

Under the existing allowed cancellation rules (do not weaken payment/allocation blockers), CancelOrder releases only still-unmaterialized order-owned claims once, and physical holds through the proper order holder. Materialized counters/provenance remain consumed. Neither old reservation expiry nor a repeated order cancellation releases transferred claims again.

Receipt rejection/physical return releases or moves the VIN as already authorized; it does not turn its materialized claim into future supply. A later valid reassignment reconciles the existing claim association rather than blindly decrementing remaining future quantity again. Order-line quantity checks and active VIN uniqueness remain intact.

P21a retains physical consignment VIN enforcement and custody distinct from ownership. Conversion itself has no such VIN prerequisite. Preserve ordinary purchase quantity receipts: for a reservation-origin quantity shipment, consume the corresponding future obligation exactly once with durable shipment-line/receipt-batch provenance in the same transaction; subsequent identification of that receipt's VIN does not consume the lot again or allocate the same delivered quantity. Do not solve this by globally forbidding purchase receipt batches or requiring all VINs at conversion. Existing shipment-line quantity checks continue to prevent later ordinary allocation of already fulfilled units. Actual consignment receipt must use allocated VIN/custody flow as already specified in P21a.

## Exact ownership clarification

No new production path outside the existing feature packets is required.

| Packet | Existing owned seams to use | Narrow addition to that packet's ownership |
|---|---|---|
| P17b | commerce service/reservation_allocation.go, service/ports.go, repository/reservation.go, repository/supply.go; inventory holder reassignment and app adapter already owned | None |
| P18b | commerce service/reservation_lifecycle.go, service/reservation.go, service/deal.go, service/ports.go, model/order.go, repository/reservation.go, repository/deal.go | internal/modules/commerce/service/fulfilment.go; internal/modules/commerce/service/reservation_allocation.go; internal/modules/commerce/repository/supply.go; internal/modules/commerce/service/reservation_allocation_test.go |
| P21a | commerce service/fulfilment.go, service/consignment.go, service/ports.go and existing consignment_test.go | internal/modules/commerce/service/reservation_allocation.go; internal/modules/commerce/repository/supply.go; internal/modules/commerce/service/reservation_allocation_test.go |

Those additions are paths already owned elsewhere in P17b/P21a; serialize the source writer. No handler change is needed for existing POST /commerce/orders/:id/allocations. Existing repository/fulfilment.go AddAllocations is sufficient, so do not add that path without concrete need. Keep these service helpers in the existing module; no new app/inventory port for later ordinary Reserve is needed.

P16 owns supply model/migration000041; P17a owns reservation model/migration000042. Supply claim durability and nullable converted-order/line linkage are existing requirements to include when those unapplied migrations are first written. Do not add a duplicate schema or rewrite any applied migration. If predecessor schema omits a concrete necessary column/index, report that exact ownership/migration seam to hub before editing.

## Affected unit cases only

Extend meaningful service fakes in existing reservation_allocation_test.go and reservation_lifecycle_test.go, exercising actual Fulfilment.Allocate and conversion/cancel paths, not only a ledger helper:

1. Zero VIN conversion then partial/full ordinary Allocate; partial before+after conversion; all VIN conversion. All yield the same four exact claims/order allocations/holders and unchanged available 6 for total10/reserve4.
2. Ordinary unrelated order remains unchanged and does not consume a supply lot. Reservation-origin wrong model/spec/color, wrong line/source party, duplicate VIN/claim, missing linkage and stale revision fail without mutations.
3. Converted reservation allocation/expiry/cancellation cannot release order claims. Allowed cancellation releases only residual future quantity once; materialized physical release/rejection leaves materialized count and future availability unchanged.
4. Retried or interleaved conversion/allocation/cancel rechecks the owning state; reused materialized claim does not consume a second future unit. Fake transaction state deep-copies claims/allocations/holders/orders/events.
5. Inject stock reserve/reassignment, claim write, AddAllocations, order CAS and event failures: all participating state rolls back. Bound transaction context reaches inventory.
6. P21a: future consignment conversion succeeds, later VIN assignment then custody handover succeeds; no-VIN physical consignment handover remains blocked. Purchase quantity receipt consumes once; later receipt VIN identification/repeated delivery cannot double-consume or reopen future capacity.

Implementation worker runs affected selectors once after each relevant change: from repo root,
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^(TestReservationAllocation|TestReservationLifecycle|TestReservationOrderAllocation|TestConsignment)'`
Select only families actually affected by that packet; new ordinary-order compatibility case lives in TestReservationOrderAllocation. Existing app/inventory units run only when their actual adapter/holder code changes per original packet. No tests/source edits were performed for this proposal; no integration/build/lint/review/QA or database-concurrency guarantee.

## Targeted input SHA-256

```text
ab748fe94d69d4ab5ec70540565bdaf50572da44a41bfdc19a6304c93c303b19  packet-17b.md
398a8ba2bbf4e21613ce07cd30b0f004d23ae9146d37d3d1f337ceeee7ce1a8d  packet-18b.md
398f8ac001f63cb9510bfaeda035d2609809f9a2e4fb5a55a5f44f26e79d910b  packet-21a.md
db377700c932f5e806cb7a96ddb469ee114d692243ffac63be696a270466c6f0  internal/modules/commerce/service/fulfilment.go
4b8fd87b0b74f39236d6ffaa4fef7f2e5104bed968bdc87f139a65b4d4b6aa0c  internal/modules/commerce/service/deal.go
787364f89023c458361636143c84607ded429f0edfc38a65c844b1b1aef78951  internal/modules/commerce/service/ports.go
1e00f93f99405fc1d75cf1f0e5806bfefec3d5915e0444680fb48df1bbe71977  internal/modules/commerce/repository/fulfilment.go
543db825109bc860a84e9aac672bc5f1f39c5c7f25689828b189827dac0273cf  internal/modules/commerce/handler/deal.go
```

