# P18a2 bounded completion and explicit risk routing

The initial worker's corrected result-18a2.md records only an invoice view
unit and a DTO unit, with the original Issue/capacity/review/cancellation/
rollback and HTTP behavior acceptance unfinished. Running named selectors
does not complete those requirements. Complete the original packet; this is
not an additional review/QA loop or a new product task.

The application hub explicitly selects a fresh gpt-6-sol/high worker under
agent-workflow's risk-escalation rule. The accepted-money consumer needs
complete lock/capacity/rollback semantics after a partial first implementation.
Context ceiling 19k; stop with precise unfinished scope before overflow.

Retain original packet-18a2.md plus p18a-split-approval.md ownership. Add:

- internal/modules/commerce/service/deps.go
- internal/modules/commerce/service/reservation_invoice_test.go

The first is the concrete NewInvoice composition seam omitted by the packet:
the initial worker changed it without prior scope authorization. The hub
adopts that necessary same-Deps ReservationPayment construction and permits
bounded corrections there. The second keeps actual Invoice/Deal consumer
transaction tests separate from the already completed payment ledger fixture.
No unrelated source or test paths are authorized.

Complete real service behavior for unchanged invoice total, assigned credit
once, accepted/pending capacity and locked rereads, ordinary-order behavior,
historical void exclusion, preinvoice cancellation and credited void blocks,
unique attachment and rollback. Credit linkage must also handle an already
issued legitimate invoice when that case is possible; no silent read mutation,
synthetic payment or new invoice policy. Use actual P18a1 bound primitives.

Finish thin HTTP wiring/auth/DTO/error/revision semantics and meaningful pure
route units. Final report supplies exact routes/input/output/actions and the
accountant-accessible list/detail path for Billing. No public credit/expiry/
paid-state setter or external bank action.

Only affected unit selections once per actual change. Prefer new Invoice
consumer case names under TestReservationCredit_Invoice and retain unrelated
green ledger evidence; run Payment/Refund families only if their behavior
changes. Handler family remains TestReservationPaymentHTTP. No whole-package,
DB/migration/runtime, build/typecheck/lint/vet/race/browser/integration or
independent review/QA. Report result-18a2-completion.md, release the writer
lease at completion and make no post-release source edits.
