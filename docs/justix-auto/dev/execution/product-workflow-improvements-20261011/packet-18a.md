# P18a — Reservation advance and full refund accounting

Requirements: R3/R10/R13. Dependencies: P04/P17b. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/commerce/model/reservation_payment.go`
- `internal/modules/commerce/service/reservation_payment.go`
- `internal/modules/commerce/service/reservation_payment_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/service/reservation.go`
- `internal/modules/commerce/repository/reservation_payment.go`
- `internal/modules/commerce/repository/store.go`
- `internal/modules/commerce/handler/reservation_payment.go`
- `internal/modules/commerce/handler/handler.go`
- `internal/modules/commerce/module.go`
- `migrations/000043_reservation_payments.up.sql`
- `migrations/000043_reservation_payments.down.sql`

## Behavior, interfaces and data flow

Attach advance payment evidence to reservation with Money currency matching terms, immutable external reference and file IDs through existing Files.Share. Distinct submitted/accepted/rejected states, accountant existing commerce payment permission for decide, actor/time/reason audit. Pending or accepted money blocks automatic expiry/cancellation. Cancellation with accepted advance creates full-amount refund obligation; accountant confirms actual refund evidence, no automatic bank transfer/fee/retention. Requesting refund does not count as refund; confirmed amount must equal full accepted advance and accepted total immutable. Pending evidence must be decided before settlement can finalize; no silent discard. On continuation advance credited exactly once to purchase/consignment settlement (link ledger credit to order; don't duplicate payment). Add port/service view for current pending/accepted/refunded totals under row lock.

## Acceptance and affected units

Pending/rejected do not reduce debt; accepted once, currency/overpayment invalid; pending blocks release; accepted cancellation remains pending refund; wrong amount/unauthorized accountant denied; confirmed full refund permits release once; retry cannot refund or credit twice. No public document access.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/commerce/service`. No other development verification/runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

