# P18a2 — Accounting HTTP and existing order-credit consumption

Original P18a, canonical `product-workflow-improvements-20261011`, R3/R10/R13; application hub `/root`. Base `4fa683ba4708bb71031534260ec5cb8742ce9977`; milestone `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`; branch `feature/company-commerce-workflows-20261011`. Actual predecessor is completed P18a1, supplied at dispatch with source hashes. Context ceiling 18k. Requires explicit hub acceptance of the two proposed additions; this document does not grant it.

## Read, locks and exact ownership

Read root AGENTS.md, agent-workflow.md, internal/AGENTS.md, plan.md common execution/R3/R10/R13/S2/S4, p18a-split-proposal.md, actual P18a1 result, result-04.md and actual P17 HTTP/composition handoff. Targeted source: ledger methods, Invoice.view/Issue/Void/SubmitEvidence/DecideEvidence/ForOrder/cancellationChecks, current reservation handler/module construction; http-domain.md §1/§3/§5 only. Do not load entire module/app or changing predecessor source before writer handoff.

Original owned paths, with serialized P18a1 overlap:

- `internal/modules/commerce/handler/reservation_payment.go`
- `internal/modules/commerce/handler/handler.go`
- `internal/modules/commerce/module.go`
- `internal/modules/commerce/service/reservation_payment.go`
- `internal/modules/commerce/service/reservation_payment_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/repository/reservation_payment.go`
- `migrations/000043_reservation_payments.up.sql`
- `migrations/000043_reservation_payments.down.sql`

Proposed additions; only writable after hub accepts these exact paths:

- `internal/modules/commerce/service/invoice.go`
- `internal/modules/commerce/handler/reservation_payment_test.go`

Schema overlap is only a concrete correction to the new, still-unapplied ledger seam. Do not modify invoice model/repository or historical migration 000010. Exclusive source-writer; no browser/PostgreSQL/runtime resource. Required `bash tools/check-git.sh` before source edits; no Git writes.

## Coherent exposure and consumer behavior

Wire P18a1 through existing commerce module/handler composition. Thin routes parse IDs/Money/revisions, read auth.Get, call one service method and map explicit DTOs via httpx.Data/List. Require company context, existing read/submission authority and commerce.payments.accept for accountant decisions, with service-level party/tenant checks retained. Use actual reservation/evidence If-Match conventions; report them precisely rather than guessing proposal endpoints. Expose evidence states, pending/accepted/refunded totals, full-refund obligation/confirmation, credit/source identity and permitted actions needed by P19. Never expose storage keys/public file URLs or serialize domain models directly. No public generic credit creation, expiry, paid-status setter or bank-transfer endpoint.

Connect the same accepted credit to actual invoice accounting in service/invoice.go. Existing migration 000010 permits one issued invoice per order; keep that invariant. Invoice.Issue keeps full terms total, schedule and due-date behavior. In its transaction, lock order credit with the documented shared order/reservation lock sequence, attach existing unassigned credit to the created issued invoice once, and roll back invoice/order revision/credit/event together on failure. Validate exact source/order/party/currency; do not derive linkage from amount. Reconcile an already-issued legitimate invoice within the credit-link transaction if predecessor flow can produce one; ambiguous multiple issued invoices fail closed, never spread credit.

Invoice.view includes only credit assigned to its own invoice: Paid = accepted ordinary evidence + assigned source credit; Outstanding = unchanged TotalMinor − Paid. Submitted/rejected reservation evidence and requested/unconfirmed refunds contribute zero credit. Ordinary orders return zero source credit and retain behavior. Historical void invoice views must not pick up the same order credit. Reads are side-effect free; repeated Get/ForOrder cannot consume or duplicate credit. Invalid negative/over-capacity/currency state fails rather than silently clipping or changing the agreed invoice amount.

SubmitEvidence continues deriving open capacity from this authoritative Outstanding minus Pending. Under the shared accounting lock, reject excess claims and recheck money before accepting evidence; retain existing invoice revision/party/decision semantics. The credited amount must never be represented by synthetic accepted invoice evidence, re-reviewed by the accountant, or counted twice when a later real payment is accepted. Mark/export in the handoff that Paid already includes credit; downstream P21 must not add it again.

Void uses credit-inclusive Paid so the existing no-accepted-money blocker remains effective. Update the existing order cancellation check to see accepted source-order credit even before an invoice is issued; this is preservation of its existing accepted-money blocker. No new policy to refund/transfer/reallocate converted-order credit, no automatic bank action and no changed cancellation eligibility for unrelated orders. P18b2 owns calling credit linkage during atomic conversion; P18b1 owns reservation release. This packet does neither stock release nor conversion.

## Unit acceptance

Add `TestReservationCredit_*` in the existing owned service test file using real Invoice methods and ledger/transaction fakes:

1. Accepted advance 20 against unchanged invoice total 100 yields Paid 20, Outstanding 80; pending/rejected evidence adds no credit. Ordinary submitted evidence 30 yields remaining submission capacity 50; accept that evidence yields Paid 50, Outstanding 50 with no copied advance evidence.
2. Repeated Issue/link/read cannot assign twice. ForOrder with historical void invoices and one issued invoice exposes credit only once. Unrelated order, wrong source/party/currency and unsupported duplicate-issued state cannot borrow/repeat it; ordinary order accounting unchanged.
3. Credit before invoice blocks ordinary order cancellation; credited issued invoice blocks Void. Empty credit preserves ordinary void/cancel guards. Do not invent a converted-order refund workaround.
4. Issue/credit attach/event/order-CAS failure rolls back every participating fake record; submission/review rechecks locked credit/capacity; stale/interleaved invocation never overcounts. Show actual Invoice service use, not only ledger helper tests.

Proposed handler test file adds `TestReservationPaymentHTTP_*` pure units: company/auth/role guards, malformed Money/ID/revision, service error/envelope propagation, pending versus accepted DTO/actions, private document references, and no public release/credit mutation. Use existing in-memory handler/service fakes and Echo without DB/application bootstrap. If pure route wiring cannot be exercised within owned paths, report exact missing seam rather than substitute a build or database test.

Run each actually affected family once after relevant change:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestReservationCredit($|_)'`

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/handler -run '^TestReservationPaymentHTTP($|_)'`

If changing P18a1 payment/refund behavior, run only the newly affected TestReservationPayment/TestReservationRefund families once too; don't rerun green families merely because this is the second packet. No automatic QA/review, whole-suite, integration, DB/runtime, migration application, browser, lint/typecheck/build/vet/race. These units do not prove runtime composition or release readiness.

## Handoff, final candidate and stops

Report source hashes, actual before/after SHAs supplied by hub/version-control, exact routes/DTOs/envelopes/permission and revision sources, order-credit/invoice mapping and inclusion semantics, commands/case results, remaining integration limits. P19 consumes actual HTTP contract; P18b2 consumes source-order credit; P21 consumes credit-inclusive accounting once. Hub maps the final SHA to S2/S4 evidence, requests only units affected by later source changes, and leaves CI/human integration separate. No automatic final-SHA test repetition.

Stop on unaccepted path addition, absent predecessor accounting/lock/transaction seam, applied migration, invented fees/thresholds/schedule or transfer/reallocation rule, need to alter one-issued-invoice semantics, unavailable source lock or 18k context overflow. Report exact gap and bounded reslice; no canonical backlog inflation or completion claim while invoice credit remains disconnected.
