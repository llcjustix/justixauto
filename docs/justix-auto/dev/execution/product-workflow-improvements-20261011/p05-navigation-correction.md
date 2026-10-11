# P05 additive Billing access correction

Status: accepted bounded correction, scheduled immediately after P06/P07.
Source writer remains P06/P07 until its explicit release. This corrects reported
P05 behavior, not an automatic review/QA gate.

Finding from implementation follow-up: changing Billing navigation solely to
retail.read hides it from commerce-only users; default Suppliers tab can request
commerce data for retail-only users. R10 is additive and existing invoice
servicing must remain available.

Exact source ownership for the corrective P05 worker:

- web/packages/kit/src/shell.tsx
- web/packages/kit/src/shell.test.tsx (new)
- web/apps/realization/src/navigation.tsx
- web/apps/realization/src/pages/billing.tsx
- web/apps/realization/src/accountant.test.tsx

Allow NavItem.permission string or array, with any-permission semantics. Billing
uses retail.read OR commerce.read. Render only permitted module tabs, select a
permitted initial tab, and disable queries for inaccessible modules. Retain
accountant queue, invoice manage/review permission gates and role isolation.

Meaningful affected units: Kit shell any-permission behavior; Realization
accountant with retail-only/commerce-only permissions and fetch isolation.
Run each changed suite once after repair, using the separate Kit/Admin and
Realization configurations from the plan. Do not repeat already-green unrelated
payment-review or permissions suites. No other verification/runtime/Git actions.
Output result-05-correction.md, release source writer to P08.
