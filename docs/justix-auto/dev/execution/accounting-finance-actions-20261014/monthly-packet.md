# Explicit accountant issuance of monthly installment invoices

Approved by the user's 2026-10-14 answer: all invoices are issued by the accountant.
Read ../../accounting-finance-actions-20261014.md, internal/AGENTS.md, web/AGENTS.md,
and the updated installment sections in contracts/http-domain.md. Existing role
preset and separate Accounting cabinet from f958c783 remain the role model; the
user reconfirmed this. No new financial permission, role catalog or schema needed.

Current main checkout/branch: /Users/bakhromachilov/startups/justixauto,
fix/accounting-finance-actions-20261014, base f958c783. Frontend financial-scope
packet is complete and uncommitted; preserve its changes. Sole source writer.

Owned backend: internal/modules/retail/service/installment.go, installment_terms.go,
deal.go, contract.go; handler/handler.go and dto.go; existing adjacent in-package
unit fixtures. Owned frontend: realization/installment-servicing.tsx and
installment-terms.tsx if needed, pages/retail.tsx, pages/billing.tsx, data.ts and
existing installment-servicing/accountant test fixtures. Use existing finance
workspace context. No unrelated edits, generated OpenAPI, dependencies, schema,
runtime or live-data changes. Root owns canonical docs; no Git writes.

Implement POST /retail/deals/:id/installment-invoices, empty body, deal If-Match,
retail.invoices.manage in BOTH handler and service. Return HTTP200 existing full
deal envelope with fresh revision. Command validates company/branch scope, active
own-installment sale, supported valid saved draft, signed contract, unique issued
first invoice matching saved down payment, no existing immutable plan. Incomplete,
duplicate, stale, foreign scope and denied callers fail without mutation. Reuse
the existing exact materializeInstallmentPlan transaction to issue all saved rows,
update deal revision once and keep audit actor. No silent-success no-op. Require
the same finance permission for legacy manual SaveInstallmentPlan, since it also
issues invoices. No new permission key; actual constant is PermInvoicesManage.

Remove automatic monthly issuance from IssueInvoice(first installment),
Contract.signDraft and SaveInstallmentTerms. Preserve their existing relevant
validation/signature rules and first-invoice amount matching. Existing plans and
invoices are immutable and unchanged. Keep the signed-contract/terms locks that
were previously provided by immediate finalization; report any ambiguity before
changing contract policy. New generated schedules remain visible as drafts until
accountant action. Do not leave an unprotected legacy invoice-generation path.

Project issue-installment-invoices only for an eligible permitted actor, using
service/read-view facts. Existing generic invoice issuance remains its own action.
Accounting UI adds 'Выставить счета по графику' with saved total/month count,
explicit confirmation and the normal busy/stale/reload behavior. Render and submit
only in FinanceWorkspace + invoice grant + server action. Realization views draft
and pending-accountant message. Discover eligible deals in Billing including
delivered historical deals when only monthly issuance is pending. Do not pretend
unissued rows are invoice/payment records. Legacy manual plan UI, if present,
must require Accounting scope/finance authority too. Preserve commercial and
proof submission flows and optional insurance signing.

Adapt existing meaningful unit fixtures for no automatic generation and explicit
financial issuance, exact amounts/dates/revisions, denied sale-only caller,
incomplete/duplicate/stale and transaction rollback; frontend accountant action
and seller absence. No new test files or unrelated rewrites. Do NOT run checks or
tests: root runs final affected units once after all work, repairing only failures.
Run bash tools/check-git.sh before source edits, use apply_patch and normal gofmt
only on owned Go files. No lint/typecheck/build/vet/race/browser/E2E/review chains.

Protect internal/modules/identity/service/company_contract_test.go unchanged;
test_creds.txt remains unread/unmodified. Write monthly-result.md here with exact
files/hash list, final unit selectors, outstanding issues, then release source
lease. No nested agents.
