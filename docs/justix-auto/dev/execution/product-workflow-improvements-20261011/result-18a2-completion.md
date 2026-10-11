# P18a2 completion — reservation accounting HTTP and invoice credit consumer

Status: SOURCE COMPLETE for the approved bounded packet. Source-writer lease RELEASED after this report. Branch `feature/company-commerce-workflows-20261011`; before/after Git HEAD remains `ab088d97054fbabac65caae1f3f8bfe7f71fbe74` because this worker made no Git write. The predecessor dirty worktree was preserved. `bash tools/check-git.sh` passed before source edits. The final source bytes are pinned below.

## Contract and consumer behavior

`Invoice.Issue` reads the order to locate its converted source, locks the reservation through `CreditForOrderLocked`, then re-reads the order/revision and issued invoice rows. For a new invoice it retains the full order total, schedule and due-date rule, creates one issued invoice, attaches the existing credit once, updates the order by version and appends the event in one commerce transaction. A matching issued invoice with unassigned credit is reconciled in that transaction; a repeat with that same association is read-only. A different association, mismatched source/party/currency/face amount or multiple issued invoices fails closed. Create, attach, order-CAS, event and over-capacity failures roll back the participating fake invoice, credit association, order revision and event.

`Invoice.view` locks the converted source before reading invoice evidence. `Get`, `ForOrder`, `Void`, `SubmitEvidence` and `DecideEvidence` run through this accounting boundary; invoice actions re-read their invoice/evidence after the source lock. Only credit assigned to that exact issued invoice contributes to `Paid`; historical void rows do not receive it. `Paid` is accepted ordinary invoice evidence plus that credit, and `Outstanding` is unchanged face total minus `Paid`. Submitted/rejected reservation evidence and refund intents create no invoice credit or synthetic ordinary evidence. Invalid money/currency/status or `Paid + Pending > Total` fails. Buyer claim capacity uses `Outstanding - Pending`; accountant review rechecks the locked state. Credited Void and preinvoice `Deal.CancelOrder` are blocked by accepted money. Ordinary invoice/payment and cancellation guards remain in place.

The real service unit flow uses an accepted advance of 20 against a face total of 100: Issue returns Paid 20/Outstanding 80; an ordinary submitted 30 remains pending with 50 further submission capacity; accepting it returns Paid 50/Outstanding 50 with one ordinary evidence row. A committed ordinary payment arriving between invoice locator and reservation lock makes an 80 claim fail against the post-lock 50 remainder. Repeated reads and Issue do not duplicate credit. The fake transaction tests also cover existing invoice reconciliation, overfilled reconciliation rollback, wrong source/party/currency, duplicate issued state, ordinary orders, preinvoice cancellation and credited Void.

The same `Deps` in `service/deps.go` constructs `Invoice` and its `ReservationPayment` collaborator. Downstream P18b2 must call `LinkCreditLocked` while holding its reservation row lock in the conversion transaction; it must pass the same store and transaction-bound context. Invoice Issue and views use `CreditForOrderLocked` in their own transaction before credit-sensitive invoice/evidence reads. Downstream P21 must consume Invoice `Paid` as already credit-inclusive and must not add the reservation credit again.

## HTTP handoff for Billing

Commerce mounts all routes below under `/api/v1/commerce`, with active-company middleware and service party checks. `GET /reservations` is the existing party-scoped `commerce.read` discovery list; `GET /reservations/:id/accounting` requires `commerce.read` and supplies the per-reservation detail for Billing. No separate queue policy was added.

| Route | Permission | Input/revision | Response |
| --- | --- | --- | --- |
| `GET /reservations/:id/accounting` | `commerce.read` | Reservation UUID | `httpx.Data` 200, reservation ETag/revision |
| `POST /reservations/:id/payment-evidence` | `commerce.trade` | If-Match reservation revision; `claimedAmount`, `paidOn`, `externalReference`, `attachmentBindingIds` | `httpx.Data` 201, new reservation revision |
| `POST /reservation-payment-evidence/:id/accept` and `/reject` | `commerce.payments.accept` | If-Match evidence revision; decimal `reservationRevision`, `confirmation`, `reason` | `httpx.Data` 200, new reservation revision |
| `POST /reservations/:id/refund-requests` | `commerce.trade` | If-Match reservation revision; `reason` | `httpx.Data` 201, new reservation revision |
| `POST /reservations/:id/refund-confirmations` | `commerce.payments.accept` | If-Match reservation revision; evidence fields plus `confirmation` | `httpx.Data` 200, new reservation revision |

The DTO exposes `reservationId`, agreed advance, accepted, pending, refunded, refund obligation/status, evidence rows and their revisions, optional credit order/invoice IDs, allowed actions and reservation revision. Attachments are private binding IDs, not URLs or storage keys. Allowed actions follow company party, permission, unsettled evidence/refund and open advance capacity. HTTP units cover anonymous/no-company/permission/foreign-company guards, bad JSON/Money/revisions, If-Match, response ETag/envelope, service error envelope and absence of public release/credit/expiry routes. The invoice HTTP contract remains the existing full-face invoice DTO; its `paid` already includes assigned credit.

## Unit evidence and limits

Final affected selectors passed after their last relevant edits:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestReservationCredit($|_)'
ok justixauto/internal/modules/commerce/service 0.436s

env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/handler -run '^TestReservationPaymentHTTP($|_)'
ok justixauto/internal/modules/commerce/handler 1.054s
```

An earlier handler run failed because the test-only service fixture omitted its clock; the fixture was corrected and the affected handler selector passed. No database, SQL lock-race, migration, app bootstrap, runtime, browser, integration, build, lint, typecheck, vet/race or independent review/QA was run. These units do not establish release readiness.

## Final source SHA-256

```text
7a6f0bb32ed56936510f38cacac03848abb778400173f650b7d32830d19e36f5  internal/modules/commerce/service/invoice.go
55ef54785ee0777c154c2d32464ad7acc8f1ddf2ea69953af971d6b15e64c66b  internal/modules/commerce/service/deps.go
af594a3d47c82cf31d19cf40538f4d87e684e48e48d07a9a5fd82a3ff533f60f  internal/modules/commerce/service/reservation_payment.go
78e0f6a49f4028517089ed6509341abc94c6a2b2d77a10f25ecd5d6dcdaa8b34  internal/modules/commerce/service/reservation_invoice_test.go
aa5da6b3f2eb63999c7db0e7d375b896146a6384577f51acb8f25b30c6d2f0b8  internal/modules/commerce/handler/reservation_payment.go
820f516fab08ee1ef3c52de6b150b5aac73263beedc6f75548cc7e187b07d99c  internal/modules/commerce/handler/reservation_payment_test.go
d3fbf6dfe86f7372040c6fa5f6dde09384c0a09e9e736e7554da95c8fe4849af  internal/modules/commerce/handler/handler.go
f3b97d0f99429efe7fa6e794554bbbf7ae896f04ac80961b6f61ad01a4a63f96  internal/modules/commerce/module.go
```
