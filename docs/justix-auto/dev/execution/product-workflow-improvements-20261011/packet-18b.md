# P18b — Expiry, extensions and agreed order continuation

Requirements: R3. Dependencies: P18a. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/commerce/service/reservation_lifecycle.go`
- `internal/modules/commerce/service/reservation_lifecycle_test.go`
- `internal/modules/commerce/service/reservation.go`
- `internal/modules/commerce/service/deal.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/model/order.go`
- `internal/modules/commerce/model/offer.go`
- `internal/modules/commerce/repository/reservation.go`
- `internal/modules/commerce/repository/deal.go`
- `internal/modules/commerce/handler/reservation.go`
- `internal/modules/commerce/handler/dto.go`

## Behavior, interfaces and data flow

Implement free reservation expiry (clock injected) and distributor-approved extension with versioned event. This packet owns the idempotent expiry service; P18c wires its automatic runtime invocation using existing RunOnce. Paid/pending reservations never auto-release. Latest user decision: at the agreed required-advance deadline automatically release only if dealer never submitted evidence; pending or accepted payment protects the hold. Continue to purchase or consignment by mutual exact terms acceptance, unique source reservation→order link and existing P17b hold transfer. Snapshot separately agreed payment/handover terms, route and continuation mode; explicitly agreed allowPartialPurchaseHandover default false, minimum/conditions as exact order agreement without invented percentages. Advance credit from P18a reconciles once. Existing direct order contracts retain historical behavior; new reservation-origin order uses approved constraints. Consignment does not imply ownership transfer; P21 handles movement. No automatically selected standard settlement schedule.

## Acceptance and affected units

Expiry frees only free/no-money reservation once; extension only supplier, stale rejected; money prevents cancellation/expiry. Conversion once, changed terms require other-party acceptance and terms digest. Partial-purchase handover flag absent false, explicit agreed true preserved. Price change only mutual.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/commerce/service`. No other development verification/runtime actions.

Deadline acceptance detail: automatically release unpaid reservation after its agreed mandatory-advance deadline if the dealer never submitted payment. Pending or accepted payment protects the hold. Test deadline before/at/after, no evidence versus pending/accepted, and idempotent release. A rejected earlier submission is not 'never submitted'; do not silently treat it as absent. No default deadline/duration is authorized.

Stop: no unrecorded partial-handover condition, refund fee or default expiry duration. No known reservation decision remains pending.

Read approved runtime seam `expiry-runtime-decision.md`. RunOnce callback does not inherit its lock transaction and lock connection loss can overlap passes; expiry correctness must come from reservation-row serialization shared with payment submission/acceptance, extension and cancellation, transactional money recheck, and idempotent quantity/VIN release. Bounded selection must progress past protected/failed candidates instead of starving later due rows. A completed candidate transaction remains committed if a later candidate fails. Unit fakes cover stale selection followed by payment/extension, repeated pass, partial pass failure, and protected-row progress. No exactly-once/hard-deadline guarantee.

