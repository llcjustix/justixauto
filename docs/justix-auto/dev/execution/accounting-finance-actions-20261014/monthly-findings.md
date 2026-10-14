# Read-only findings used for the monthly issuance packet

Actor: /root/installment_invoice_scope (explorer), 2026-10-14. No source edits or
checks. Primary accepted the existing retail.invoices.manage permission and
HTTP200 full-deal response. No new permission or role model.

- Automatic paths were IssueInvoice(first-installment), Contract.signDraft and
  SaveInstallmentTerms, all calling finalizeInstallmentDraft, which creates the
  immutable plan plus every monthly invoice. Legacy SaveInstallmentPlan is a
  separate generation path and also needs invoice-management authority.
- Removing immediate materialization leaves a signed saved draft editable while
  no final plan exists. Lock existing draft terms once the contract is signed;
  preserve initial no-draft legacy setup and all existing sign-time stale checks.
- Load the deal with normal company/branch access and check revision explicitly;
  the command helper that assumes reserved-only is unsuitable for legacy
  delivered deals, which can still need monthly invoice issuance.
- Project the new financial action from service eligibility; include eligible
  delivered deals in the Accounting queue. Existing issued plans stay immutable.

Latest user reconfirmed an explicit Accountant role and separate cabinet like
Admin. Existing seeded company role (migration36/new-company preset) and the
separate /accounting section from f958c783 satisfy this structure. Company
administrators assign that role through the company employees UI.
