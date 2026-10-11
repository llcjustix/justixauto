# P18a2 result — reservation accounting HTTP and invoice credit

Status: NEEDS_CONTEXT — source-writer lease RELEASED. The source change is
present, but the packet's full invoice acceptance remains unimplemented and
untested as detailed below. Preparation began at:
`feature/company-commerce-workflows-20261011` /
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74`; `bash tools/check-git.sh`
passed.

## Stable accounting map

- `service/invoice.go`: `view` is the single current `Paid`, `Pending`, and
  `Outstanding` calculation; credit must be included here only for its assigned
  issued invoice. `SubmitEvidence` derives capacity from `Outstanding -
  Pending`; `Void` uses `view.Paid`/`Pending`; `ForOrder` reads all invoices,
  including historical void invoices; the appended cancellation check currently
  sees only accepted ordinary invoice evidence.
- `Invoice.Issue` reads the party-scoped order, creates the full-total issued
  invoice, version-updates the order, appends the event, and returns `view`
  inside one `Store.InTx`. Existing invoice repository `ForOrder` has no lock,
  while migration `000010` enforces one *issued* invoice per order and permits
  historical void rows.
- P18a1 early handoff requires reservation-first locking through
  `Reservations().ByOrderForUpdate` (or known-ID `ForUpdate`) before credit
  inspection/linking, then invoice create/read/update. Thus `Issue`, and any
  invoice mutation that reads credit-sensitive capacity, must use the final
  P18a1 shared accounting primitive rather than acquire invoice-first locks.
- Stable handler boundary: commerce already uses company middleware; invoice
  routes use `PermRead`, `PermInvoicesManage`, `PermTrade`, and
  `PermPaymentsAccept`; invoice DTO derives actions from `Paid`, `Pending`, and
  `Outstanding` and emits `httpx.Data`/`httpx.List` with invoice/evidence
  revisions. P18a1 final handler/service construction signatures determine
  reservation-payment route wiring.

## Missing handoff required before edit

Need final P18a1 source hashes and exact locked APIs/types for: reservation
lookup/lock, source-order credit lookup and invoice attachment, credit amount
and validation fields, transaction rollback behavior, and any shared
credit-aware accounting read. Also need the final handler/module construction
shape and an explicit writer grant. No additional owned path is currently
required: accepted P18a2 paths are `service/invoice.go` and
`handler/reservation_payment_test.go`, plus the packet's serialized P18a1
overlap only if final source proves it necessary.

## Exclusions

No P18a1 evolving service/repository/reservation source was intentionally
inspected before its handoff. One broad read-only symbol search accidentally
returned the new `model/reservation_payment.go`; this was reported to the hub.

## Completion

`Invoice.Issue` gets source credit through `CreditForOrderLocked` before creating its unchanged full-total invoice, then attaches it in the same transaction. Invoice views retrieve the actual order before the reservation-first credit read, count only credit assigned to that issued invoice in Paid, and reject over-total paid money. Outstanding/submission capacity are credit-inclusive; Void/cancellation retain paid-money blocks, including before invoice issuance. Get and ForOrder use locked commerce transactions; void history gets no credit.

Routes: `GET /reservations/:id/accounting` (`commerce.read`); payment evidence/refund requests (`commerce.trade`); evidence decisions/refund confirmation (`commerce.payments.accept`). Mutations use If-Match; decisions also require decimal `reservationRevision`. Responses expose reservation revision/ETag, money states, private attachment binding IDs, refund state and credit identities, never document URLs. Existing party-scoped reservation list plus accountant-readable detail supports Billing discovery; no queue endpoint was needed.

Passed once after final changes:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestReservationCredit($|_)'`

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/handler -run '^TestReservationPaymentHTTP($|_)'`

Fake-backed units cover Invoice credit inclusion/void exclusion and the accounting DTO; SQL locking, migration execution, route composition and release readiness remain unverified.

## Corrected API handoff and scope

`service/deps.go` was changed outside the accepted P18a2 path list: `NewInvoice`
now constructs its `ReservationPayment` dependency from the same `Deps`. This
was required because `Invoice.view`, Issue, Void/SubmitEvidence result views,
Get, ForOrder, and the cancellation hook need the P18a1 transaction-bound
`CreditForOrderLocked` primitive. No new public service API, data policy, or
dependency was introduced. Root must decide whether to retain that unowned
one-line composition change or reslice it into an approved path.

Actual routes and mutation inputs:

- `GET /api/v1/commerce/reservations/:id/accounting` calls `Get`; it requires
  company middleware plus `commerce.read`, checks party scope in the service,
  and returns `httpx.Data` with reservation `Version` as ETag/revision.
- `POST .../reservations/:id/payment-evidence` calls `Submit`; it requires
  `commerce.trade`, If-Match reservation version, and `EvidenceInput`:
  `claimedAmount`, `paidOn`, `externalReference`, `attachmentBindingIds`.
- `POST .../reservation-payment-evidence/:id/accept` and `/reject` call
  `Decide`; they require `commerce.payments.accept`, If-Match evidence version,
  and `{reservationRevision, confirmation, reason}`. The decimal
  `reservationRevision` is parsed from the body and compared by the service.
- `POST .../reservations/:id/refund-requests` calls `RequestRefund`; it
  requires `commerce.trade`, If-Match reservation version, `{reason}`.
- `POST .../reservations/:id/refund-confirmations` calls `ConfirmRefund`; it
  requires `commerce.payments.accept`, If-Match reservation version, and the
  `EvidenceInput` fields plus `confirmation`.

The accounting DTO has `reservationId`, `agreedAdvance`, `accepted`,
`pending`, `refunded`, `refundObligation`, `refundStatus`, evidence rows
(including private `attachmentIds`), optional `creditOrderId` and
`creditInvoiceId`, `allowedActions`, and `revision`. Existing
`GET /api/v1/commerce/reservations` is party-scoped and requires
`commerce.read`; an accountant with that permission can discover candidate
reservations then read their accounting detail. No purpose-built queue exists.

Implemented meaningful invoice coverage is limited to the added
`TestReservationCredit/invoice view includes assigned credit exactly once`:
accepted 100 credit on a 1000 invoice produces Paid 100/Outstanding 900, and a
historical void view gets Paid 0. Existing P18a1 credit tests cover primitive
wrong currency/status, duplicate attachment and credit rollback, but do not
exercise Invoice.Issue or invoice service rollback.

Still unimplemented or unproven from the P18a2 acceptance: real Invoice.Issue
credit attachment and rollback on create/attach/event/order-CAS failures;
20/100 credit plus submitted/accepted ordinary 30 capacity/review flow;
pre-invoice `Deal.CancelOrder` credit block through the actual service;
credited `Invoice.Void` through the actual service; repeated Issue/read and
ForOrder with issued plus historical void; wrong source/party/currency and
duplicate-issued behavior through Invoice; locked stale/interleaved capacity
rechecks; and pure handler route/auth/envelope/malformed-input tests. No tests
or source edits were made after this clarification request.

```text
51bc182a1717185685fe79e2c90a3fdadccd9d39b7f55ebcde4d9b19c8623de9  internal/modules/commerce/service/invoice.go
55ef54785ee0777c154c2d32464ad7acc8f1ddf2ea69953af971d6b15e64c66b  internal/modules/commerce/service/deps.go
cb4a872c5737f773df92f4e2cf1624c37863ace6febb24c42ad91223e93b84b1  internal/modules/commerce/service/reservation_payment_test.go
d3fbf6dfe86f7372040c6fa5f6dde09384c0a09e9e736e7554da95c8fe4849af  internal/modules/commerce/handler/handler.go
e68c084c1cb67fce34befb785c873820f395dfe0f4a249de6b26a205b923c01d  internal/modules/commerce/handler/reservation_payment.go
57f09897ab6a40c9363424a9f7831c8d703850bed3dbedab8ec81832537bc306  internal/modules/commerce/handler/reservation_payment_test.go
f3b97d0f99429efe7fa6e794554bbbf7ae896f04ac80961b6f61ad01a4a63f96  internal/modules/commerce/module.go
```
