# P19b — Reservation advance and refund accounting

Canonical task `product-workflow-improvements-20261011`, parent P19, R3/U5/U9–U10; shared accounting R10/R13 is consumed, not expanded. Application hub `/root`; feature base `4fa683ba4708bb71031534260ec5cb8742ce9977`, branch `feature/company-commerce-workflows-20261011`. Hub accepts hashes/scope and supplies actual P19a predecessor SHA/source hashes. Dependencies: accepted P19a and final P18a2/P18b results with actual money, permission, queue, revision and credit contracts. Context ceiling 12k; fresh worker.

Read root AGENTS.md, agent-workflow.md, web/AGENTS.md, plan common rules/R3 and p19-split-proposal.md. Contract reference: http-domain.md common values, identity §2, commerce §3 and private attachments §5 as needed. Read only P19a detail/query seams, BillingPage/SupplierInvoices and existing evidence/modal helpers. P18a2 final handoff defines endpoints, review action names, money/credit fields and accountant visibility; do not derive these from proposals or fixtures.

## Exact owned paths

- `web/apps/realization/src/reservation-payment.tsx`
- `web/apps/realization/src/reservation-payment.test.tsx`
- `web/apps/realization/src/reservation-workspace.tsx`
- `web/apps/realization/src/data.ts`
- `web/apps/realization/src/pages/billing.tsx`

Exclusive source-writer; no browser/DB fixture/runtime locks. Run required `bash tools/check-git.sh` before edits; no Git writes. Serial overlap preserves P19a. Shared Money/modal/query/session/private-file conventions apply. Readonly history/old obligations remain accessible according to existing permission rules.

## One behavior and boundaries

Implement reservation money evidence through authoritative accountant decision and full refund confirmation. Workspace displays agreed advance/deadline, submitted/accepted/rejected evidence with immutable references/attachments and returned actor/time/reason; submit evidence through the actual API. Upload/submission is visibly pending and does not imply received funds. Expose allowed review/decline controls only to actual authorized accountant/admin-compatible roles using permissions plus server actions; do not add a guessed role matrix.

Integrate a discoverable advance/refund work queue into Billing using final API reads. Queue item opens the same reservation accounting component; preserve existing commerce and retail invoices. Show pending review and refund obligation/pending/confirmed distinctly. A refund is the full accepted advance with external evidence and accountant confirmation; a dealer request or uploaded proof alone does not release a reservation or confirm a refund. Backend drives cancellation eligibility and authoritative status.

After decisions refresh affected reservation detail/list, Billing queue and order/invoice views using final query identities. Render server accepted credit and credit-inclusive invoice totals once: no optimistic accepted amount, no browser double-add to Paid, no floating arithmetic, no invented invoice/schedule allocation. Preserve pending/rejected history; neither reduces outstanding debt. P19c connects these states to lifecycle and conversion.

## Acceptance and affected units

Use named `P19b reservation accounting` cases in reservation-payment.test.tsx. Render workspace payment entry and actual Billing host with HTTP/session fakes. Exercise submit→pending→accepted/rejected, exact minor-unit/currency/reference payload, attachments through existing private-file seam, permission-denied reviewer, review reason/error handling, full refund obligation→evidence→accountant confirmed, pending versus confirmed labels, query refresh and credit-inclusive invoice totals without a second subtraction. Include stale revision/retry and double-submit guard. Assert calls and visible state, not typeof exports or helper-only formatting.

From `/Users/bakhromachilov/startups/justixauto/web`, once after relevant changes:

`npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/reservation-payment.test.tsx -t 'P19b reservation accounting'`

If shared P19a interaction behavior was changed, additionally run its affected named selection once; unchanged green evidence is retained. Require named intended suite/nonzero cases. No other development verification or runtime/review/QA loop.

Report final queue/detail props, money/credit DTOs, affected query identities, tests and exclusions for P19c. Stop if final APIs cannot populate Billing, lack review/refund authority or credit inclusion is ambiguous; return exact missing seam to hub. Stop on new refund policy/unowned paths or context ceiling; never fabricate pending/accepted money or broaden canonical scope.
