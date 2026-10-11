# P18a1 result — reservation payment, full refund and order credit ledger

Status: SOURCE COMPLETE. Source-writer lease RELEASED after this report. Branch
`feature/company-commerce-workflows-20261011`, predecessor HEAD
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74` with P17b dirty source
handoff in `result-17b-completion.md` and dispatch source pins in
`entry/p18a1-dispatch/manifest.md`. No Git write or new commit was made; the
after commit SHA is therefore still the supplied HEAD, while working-tree
source hashes below identify this packet's actual bytes. `bash tools/check-git.sh`
passed before source edits. All predecessor changes were preserved.

## Implemented contract

- `NewReservationPayment(Deps)` supplies public `Get`, `Submit`, `Decide`,
  `RequestRefund`, `ConfirmRefund`. Every public mutation runs in
  `Store.InTx`, locks `Reservations().ForUpdate` before money reads, checks
  party, status and reservation/evidence revisions, and uses versioned
  reservation/evidence writes. `Decide` uses its initial evidence lookup only
  to find the source, then re-reads it after the reservation lock. Accountant
  decisions and refund confirmation require `commerce.payments.accept` alone;
  buyer submission uses the existing `commerce.trade` permission. Neither
  path requires newly verified commercial access for an old obligation.
- `LockMoney(ctx, st, reservationID)` and `MoneyLocked(ctx, st, lockedReservation)`
  expose the authoritative `ReservationMoneyView` within the caller's commerce
  transaction. It includes the agreement's exact advance/currency, accepted
  and pending totals, pending count, confirmed refunded total, dynamic full
  refund obligation/status, unique credit/order identity, evidence list and
  `EvidenceEverSubmitted` (including rejected records). Only accepted evidence
  reduces debt. Submitted evidence consumes open submission capacity but does
  not reduce debt. Rejected evidence remains immutable history; a later
  submission is a new record. Evidence binds the current accepted digest and
  revision, exact amount/currency, date, external reference, private file IDs,
  actor/time and decision reason.
- `RequestRefund` records cancellation/refund intent. It prevents new
  submissions but allows pending evidence to be explicitly decided. The
  obligation follows the accepted total until pending count reaches zero.
  `ConfirmRefund` requires the supplier accountant's affirmative confirmation,
  actual external reference/date/private evidence, zero pending claims and
  exactly the full accepted amount/currency. It preserves accepted evidence
  and records a separate confirmed refund; it performs no bank action or
  fee/partial arithmetic. `ReleaseEligibleLocked` returns true only with no
  pending claims, no converted/credited source and either zero accepted money
  or full confirmed refund. It never releases physical stock or claims.
- `LinkCreditLocked(ctx, st, lockedReservation, order)` is the P18b2 primitive.
  It validates reservation/order source, parties, currency, accepted amount,
  accepted evidence IDs, no pending/refund conflict and order face amount.
  Accepted zero money creates no credit row. A nonzero credit is unique by
  reservation and order, is immutable, and initially has nullable invoice ID.
  `CreditForOrderLocked(ctx, st, order)` obtains the source reservation lock,
  validates source/parties/order money and returns nil for ordinary orders.
  `AttachCreditInvoiceLocked(ctx, st, credit, issuedInvoice)` rechecks the
  locked source and assigns exactly one matching issued invoice; a void,
  mismatched or already assigned invoice fails. No ordinary invoice evidence
  is synthesized and invoice face amount/schedule are untouched.

Lock order for downstream consumers is reservation row first, then sorted
future lots/claims, then inventory VIN locks; the credit and invoice reads or
assignment follow the reservation row lock. P18b2 must pass its existing
`Store` and `st.Bind(ctx)` in the same conversion transaction when it calls
`LinkCreditLocked`; order/claims/holders/credit/events then share commit or
rollback. P18b1 must call `MoneyLocked` or `ReleaseEligibleLocked` under its
own reservation lock and settle pending evidence before final release.
P18a2 must acquire the converted source reservation lock before issuing or
attaching its invoice. After attachment, invoice Paid is ordinary accepted
evidence plus this exact linked credit once; outstanding and ordinary evidence
capacity subtract that Paid. Historical void invoices receive no credit.
Order cancellation must see the credit even before invoice association.

Migration 043 creates commerce-owned reservation evidence, one refund record
per reservation, and a one-reservation/one-order credit with unique nullable
`invoice_id`. The reservation row is the serialization lock; evidence and
ledger rows are always read and written under it. `Files.Share` receives
`st.Bind(ctx)` and owner company from the actual evidence submitter (buyer
for advance, supplier for refund). The documents repository resolves the
bound context through `database.Conn`, so its share row participates in the
commerce transaction. Foreign files fail. Events carry safe source/evidence
IDs; the unit fake rolls back share, evidence, refund, credit, reservation
revision and event state on injected failures.

## Evidence and exclusions

The final affected selector passed after the last relevant edits:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^(TestReservationPayment|TestReservationRefund|TestReservationCredit)($|_)'`

Result: `ok justixauto/internal/modules/commerce/service 0.646s`. Earlier
selected runs after earlier relevant edits also passed. These are fake-backed
unit results, not SQL lock-race or database migration execution evidence. No
other tests, database/runtime action, build, lint, typecheck, browser,
integration, review or QA loop was run. P18a2 HTTP and invoice consumer and
P18b1/b2 lifecycle wiring remain downstream work; P18a as a whole is not
complete from this ledger alone.

## Final source SHA-256

```text
dc6e902a823a93e2d49bac57f9aed2ab0a901bed61e49b60c8bd96026299b552  internal/modules/commerce/model/reservation_payment.go
6ba99e323348cdbd4910ff07b609360a769265250ea2c70145674aa649d297d5  internal/modules/commerce/service/reservation_payment.go
f1c2f0d8e6446b973f99e910e40859fa4968f7efdef6d0397388a38d3a49cc92  internal/modules/commerce/service/reservation_payment_test.go
0532c36ee13d894e90291f2ef743a414bd37f22505ebe8580144e848e1ca0f0d  internal/modules/commerce/service/ports.go
54851ea42c9ae6ad5e19cc7ca524c5f94188bbe6a8eb367889d2ba37390783bd  internal/modules/commerce/repository/reservation_payment.go
8ccf43ace2027875da05a58e98bfb00550b7f60d6adad22b627478f438a0b6b3  internal/modules/commerce/repository/store.go
3fb53981c00ad67ce5e1e8182695f2d9ab143e966a56aaabd63cbc34a7378df0  migrations/000043_reservation_payments.up.sql
3cec6790d3ea7cdb78e10ed7ec853779915736bcd09e90cc447456c39af8d6fe  migrations/000043_reservation_payments.down.sql
```
