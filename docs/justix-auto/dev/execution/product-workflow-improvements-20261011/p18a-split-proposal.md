# P18a bounded split proposal

Canonical task `product-workflow-improvements-20261011`, original P18a, requirements R3/R10/R13; application hub `/root`. Supplied feature base `4fa683ba4708bb71031534260ec5cb8742ce9977`, local milestone `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, branch `feature/company-commerce-workflows-20261011`. Actual predecessor SHA and source hashes must be supplied at each dispatch. Milestone is provenance, not a claim that the dirty shared checkout equals that commit.

Docs-only decomposition; original packet, plan, approved contracts and canonical board remain unchanged. No source/test/Git/runtime edits or verification. P17a was the active source writer; no changing reservation source was consumed. Stable invoice slices were read with explicit hub authorization.

## Intermediate findings and bounded contexts

Original P18a owns the advance/refund/credit ledger but omits its existing invoice consumer. The stable `service/invoice.go` has these relevant seams:

- `Invoice.view` sums accepted invoice evidence into Paid and subtracts it from TotalMinor. It cannot currently see reservation advance credit.
- `SubmitEvidence` derives open capacity from Outstanding minus Pending; adding display-only credit would still permit excess evidence unless this shared calculation changes.
- `Void` blocks Paid/Pending; `cancellationChecks` only checks accepted invoice evidence, leaving a credit-only order cancellable if not connected.
- `Issue` copies the full agreed terms total and payment schedule. `invoices_active_order_key` in migration 000010 permits one issued invoice per order, while `ForOrder` includes historical void invoices. Preserve those semantics; no advance subtraction from invoice face amount or schedule generation.

| Packet | Behavior | Ceiling | Dependencies |
|---|---|---|---|
| P18a1 | Row-serialized reservation evidence, full-refund accounting and unique source-order credit primitives | 18k | Actual P04, P17a/P17b results; P16 claim handoff |
| P18a2 | Expose accounting actions and consume existing credit in ordinary order invoice balances | 18k | P18a1 result and exact-path extension acceptance below |

Each uses a fresh worker context, targeted source slices and exclusive `source-writer`; no DB/browser/live fixture resource. Finish P18a1 before P18a2. The worker runs the required repository-root check before source edits; Git writes stay with version-control. No automatic review/QA chain. P18a stays incomplete until the consumer and HTTP boundary are connected; a ledger-only implementation is insufficient.

## Exact ownership proposal

P18a1 owns nine original paths: commerce `model/reservation_payment.go`, `service/reservation_payment.go`, `service/reservation_payment_test.go`, `service/ports.go`, `service/reservation.go`, `repository/reservation_payment.go`, `repository/store.go`, and migration 000043 up/down. P18a2 owns original `handler/reservation_payment.go`, `handler/handler.go`, `module.go`, with serialized overlap on `service/reservation_payment.go`, `service/reservation_payment_test.go`, `service/ports.go`, `repository/reservation_payment.go`, and migration 000043 up/down only if still unapplied and a concrete ledger seam requires adjustment. Together these cover exactly the original 12-path union before proposed extensions.

Two narrow additions are PROPOSED, not authorized by this document:

1. `internal/modules/commerce/service/invoice.go`: connect reservation credit to existing invoice balance/capacity, void protection and order cancellation check. This is required to complete original R3 accepted-advance accounting.
2. `internal/modules/commerce/handler/reservation_payment_test.go`: small pure HTTP boundary units for new routes/DTOs/auth; no module-root DB-backed tests.

No change to invoice model/repository, app adapter, permission catalog, existing migration 000010 or canonical contract is proposed. The new ledger stores nullable invoice association; existing invoice repository operations suffice. If dispatch disproves this, return exact missing path/interface to hub. This is an ownership correction within original canonical scope, not a new backlog task or RESLICE_REQUIRED product expansion. Neither packet may edit an unaccepted addition.

## Credit consumption contract

P18a1 creates immutable accepted-payment provenance and an exactly-once source reservation→order credit link within the caller's Store.InTx. P18b2 invokes that primitive in the same transaction as order creation, reservation conversion, claim/holder transfer and events. Currency/parties/source state must match; pending evidence cannot disappear, and conversion cannot bypass unresolved evidence or refund settlement. No independent commit and no new accepted invoice-evidence row.

The ledger retains source accepted evidence IDs, full accepted amount/currency, order ID and nullable invoice ID with uniqueness preventing a second source credit or second assignment. The order has this credit even before an invoice exists. P18a2 binds it atomically when the order's issued invoice is created, under the same accounting lock; a legitimate already-issued invoice must be reconciled in that same transaction if encountered. Do not attach to void invoices or infer order/source from equal amounts. A duplicate issued invoice violates the existing invariant and fails closed, rather than spreading one advance across invoices.

Invoice Paid = accepted ordinary invoice evidence + the credit assigned to this exact invoice. Outstanding = unchanged invoice TotalMinor − Paid; submission capacity = Outstanding − submitted ordinary evidence. Credit lookup validates currency and matching parties/order. Historical void invoice views get no credit, repeated reads do not consume it, and summing invoice credits for one order never exceeds the linked advance. Accepted credit protects an order from ordinary cancellation even before assignment to an invoice. Assigned credit also triggers the existing invoice paid-money void blocker. No new refund/transfer/reallocation policy for converted orders is authorized.

Do not allocate credit among schedule rows: the existing B2B invoice is full-order, not one invoice per installment. Preserve agreed face amount, schedule, due dates, purchase/consignment agreement and current issue semantics. P21 consumes these same order money facts; it must not add the linked credit again to already credit-inclusive invoice Paid. Export the source/credit identity and accounting inclusion contract in the result for P18b/P19/P21.

## Complete acceptance coverage

| Original acceptance / shared interface | Packet and meaningful unit family |
|---|---|
| Exact Money, currency and agreed amount bounds; immutable reference/private attachments | a1 TestReservationPayment |
| Submitted/accepted/rejected; pending and rejected never reduce debt; accountant authority/admin compatibility; actor/time/reason audit | a1 TestReservationPayment; a2 HTTP boundary |
| Money view under row lock includes pending/accepted/refunded and evidence-ever-submitted; rejected history not “never submitted” | a1 TestReservationPayment; P18b1 lifecycle consumer |
| Pending protects release; accepted cancellation awaits full refund; pending must be decided, never discarded | a1 TestReservationRefund; P18b1 release consumer |
| Full refund obligation, actual external evidence and accountant confirmation; wrong amount/currency/party denied; original accepted total immutable | a1 TestReservationRefund |
| No bank action, fee, retention, partial-refund policy or silent evidence mutation | a1; preserved a2 |
| Unique source-order credit, credit/refund exclusion, retried and stale actions cannot duplicate | a1 TestReservationCredit; P18b2 conversion consumer |
| Order/invoice balances and payment capacity include credit once; historical invoices do not repeat credit; ordinary orders unchanged | a2 TestReservationCredit |
| Credit-only order cancellation and credited-invoice void remain money-protected | a2 TestReservationCredit |
| Row-lock/state/CAS recheck and rollback on repository/document/event/credit failure | a1 all three families; a2 TestReservationCredit and HTTP boundary |
| No public documents; tenant isolation; thin HTTP, proper envelopes/revisions/actions | a1 service tests; a2 TestReservationPaymentHTTP |
| Exact full refund makes release eligible once; real hold release belongs to P18b1, never payment handler | a1 guard units plus P18b1 lifecycle units |

## Verification, final candidate and risks

Zero automatic QA/review packets. Only actually affected pure unit families once per relevant change, as specified in each packet; no full service suite, DB/repository integration, runtime, migration application, browser, build, lint, typecheck, vet/race or ceremonial repeated checks. No tests executed by slicer.

At final candidate the hub maps actual predecessor/final SHAs and existing evidence to S2/S4: accountant→reservation evidence; refund→release; conversion→unique order credit; credit→invoice/P21 balance. A later change invalidating those units triggers only affected units once. This is an acceptance mapping, not authority for integration execution or an independent QA claim. Database lock races, SQL uniqueness/migration execution, deployed document rollback and full runtime composition remain unverified.

Dispatch gaps: hub acceptance of two exact paths; actual P17a/P17b API/schema/lock-order handoff; final P18a1 method/DTO/source-hash handoff. No new product rule is required for the described seam. Undecided fees, thresholds, schedules, transfer/reallocation policy, or changing the one-issued-invoice rule must stop the affected work. Stop and reslice if either context cannot fit 18k; report RESLICE_REQUIRED only if canonical scope changes.

## SHA-256 manifest

Packet SHA-256:

```text
73a3b39d22100fd504a71b092e7ce5f313bf528882febc8aabd890fcca515312  packet-18a1.md
1a6376f8dd79e4b68217f595ed354bfb8fc3d2e3f8b03de15a91a93ffb4d34ac  packet-18a2.md
```

Inputs below are the actual planning fingerprints. Execution-directory paths are relative to this directory; other paths are repository-relative. Changing reservation source hashes are deliberately deferred to actual predecessor dispatch.

```text
a222419ecca4935c906454884eb62d6cacace6a5b80395c90a1cee870f8b8524  AGENTS.md
4f264fb5217a9613b00df455b0661cb9b955e90793a29b17c682cbcd0010830b  docs/justix-auto/dev/agent-workflow.md
b22213a76ca11a4ce4aeb04323260d1d577f28ef1230d48c15367d074e67650b  internal/AGENTS.md
5df5a66f4ad3526b36fc5f35c011eff7311eccb0e0b6b9a0e185636122542356  docs/justix-auto/dev/product-workflow-improvements-20261011.md
a2cd3b101a5d720e26aa7a96524b0252aa305b22db75895a52513b6b8777599e  docs/justix-auto/dev/product-workflow-interview-20261011.md
5bd0439c6aca0b5a824c340c4c15692734c70296b0125d82c63a739a21ba65ba  docs/justix-auto/contracts/http-domain.md
f8a0d756be25ba54b71b2b22294676f0e18adc3747f2bbc841d825e87ab88c69  plan.md
d7b1d7befb5055397de2a9d7b77e019535f8e8512eb3e5253bf289e7663b082b  packet-18a.md
4396cbbe49bdd9bda73c8e38668a30cd99084767e318f1f9f0f6baa2c2ffe3bb  result-04.md
fb6879df1fe88a3d41d1e426dee092d7e5ce0d06bcc72cb202de0dc9a34c9446  result-16-foundation.md
0d362be3f84dd9ca421ffc4334f0c28349cd30194204ce17ba2d1da3556f5472  packet-17a.md
ab748fe94d69d4ab5ec70540565bdaf50572da44a41bfdc19a6304c93c303b19  packet-17b.md
023703090b1d3bc741c9ad64d0f85bd9342cacabc98f240c40d7c5c9d55ad93c  p18b-split-proposal.md
93cfa1c5751ee7d568e587649e3b8d8ebe90ae3413a20553f7a8b292162ee7be  reservation-order-allocation-seam.md
6cf415a135e9326c9019a43d3e1d462f095398b4d6510c9d0033238d1e19b588  reservation-order-allocation-approval.md
28e76388a106f971b54771ed6ea8f913be5fd2c31482bbd9206fce6b59044364  internal/modules/commerce/service/invoice.go
8bcff56a60a4583f9d0009d49e359130e826a43be430bda87d61a9b053a60544  internal/modules/commerce/model/invoice.go
3c71e7b527ea7895da788ecded8ed0a7214a53badb80fc0477fb18e829e21e4e  internal/modules/commerce/repository/invoice.go
b4f0c34aba287c88c72d9b01f6036c9e40f087ee57402550932b187d6cd548e1  migrations/000010_commerce_invoices.up.sql
```
