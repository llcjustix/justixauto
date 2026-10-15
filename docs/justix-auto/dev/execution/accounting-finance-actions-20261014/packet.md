# Frontend packet: accounting-only invoice and payment decisions

Read ../../accounting-finance-actions-20261014.md and web/AGENTS.md. Root rules
apply; R10/R13 scope narrowed by latest 2026-10-14 user instructions. OD-01/08/09
provider financing, insurance and installment policy boundaries unchanged.
Repo /Users/bakhromachilov/startups/justixauto; branch
fix/accounting-finance-actions-20261014 at f958c783568b0f08c1854a9c7b6b4dc453b898f0.

Owned paths: web/apps/portal/src/features/accounting/; realization/pages/billing.tsx,
pages/retail.tsx, pages/trade.tsx; installment-payment-review.tsx,
installment-servicing.tsx, reservation-payment.tsx; existing adjacent tests plus
accountant.test.tsx, cash-sale.test.tsx, installment-servicing.test.tsx,
reservation-payment.test.tsx, order-workspace.test.tsx, order-fulfilment.test.tsx,
order-create.test.tsx if affected. Small shared accounting helper allowed inside
features/accounting; no kit/session/server changes without primary scope approval.

Implement a default-deny accounting workspace context (or equivalent explicit
scope) enabled only by the /accounting navigation entry around BillingPage. Shared
components outside that scope, including legacy /billing and Realization sale/
order/reservation/payment dialogs, must never issue/void invoices or accept/reject
payment evidence/refunds. Keep explicit financial permission and server allowed
actions; scope alone must not grant authority. Existing section/context lifecycle
removes old dialogs on switching. Guard mutation handlers as well as visibility.
No insecure backend header or role-name checks.

Cover ALL existing entry points:
- DealDialogActions: first-installment, cash full-price, registration/manual invoice.
- OrderActions: supplier invoice; InvoicePanel: void and supplier evidence accept/reject.
- RetailInvoicePanel: legacy/non-group accept/reject.
- InstallmentPaymentReview plus open-review affordances: full grouped decisions only
  within Accounting; history read-only outside. No accidental child decision path.
- ReservationPayment: advance accept/reject and full refund confirmation only within
  Accounting; submit/request-refund remain in their normal permitted workspaces.
- Billing queue: review buttons/labels must not imply actions outside Accounting.
Leave commercial order/addendum, consignment physical return, insurer/contract
approvals and payment-proof submission alone.

Ensure discoverability in Accounting: B2B data already loads orders of both parties,
but has no invoice-issuance action/empty-order list. Preserve order records alongside
invoice rows and expose eligible supplier orders for accountant issuing, reusing
current order form/dialog, statuses, due-date/terms and revision safeguards. Current
retail waiting queue only lists deals with zero invoices: make eligible deals
reachable for additional invoices (e.g. registration after first invoice) too.
Existing amounts/immutable draft and automatic installment generation unchanged.
Use appropriate B2B tab/counterparty copy instead of referring users to Realization
for wholesale confirmation. Keep read-module separation, no unnecessary queries
for a module without read permission, loading/error/refresh/duplicate-submit guards.
No fabricated additional invoice state or backend command.

Clear product guidance outside Accounting ('Счета выставляет и оплаты подтверждает
бухгалтер') plus a link only when actual accounting grants admit the section. Keep
current routes and viewing/submission capabilities. Do not add a new app/session.

Update only meaningful existing tests/fixtures that expect financial actions to
be available outside the cabinet. In positive financial test harnesses provide
the explicit Accounting scope and proper session grants; retain/add small negative
scenarios proving admin-granted Realization still has no issuing or acceptance,
including shared retail, commerce, grouped and reserve paths. Cover accounting
issuing without existing invoice and extra retail invoice discoverability. Avoid
new test files, cosmetic tests and implementation-mirroring assertions. Native
Vitest DOM assertions only (no uninstalled jest-dom); wait for loaded data before
clicking disabled actions. Use BrowserRouter for session-retiring company switches.

Run bash tools/check-git.sh before application edits. Use apply_patch. One source
writer; no Git, canonical doc edits, tests, runtime, live data, browser or checks.
Primary runs one final affected-unit pass once all code is ready, rerunning only
actual failures. Preserve internal/modules/identity/service/company_contract_test.go
and do not read test_creds.txt. Write compact result.md and exact source SHA256
list here, list focused final tests, and release source lease.
