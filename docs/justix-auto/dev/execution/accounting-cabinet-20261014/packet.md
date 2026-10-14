# Frontend: accountant cabinet, company onboarding, sale stepper

Approved task: ../../accounting-cabinet-20261014.md. Root, web/AGENTS.md apply.
Repo /Users/bakhromachilov/startups/justixauto, feature/accounting-cabinet-20261014
at dependency c92c21627efacd0604cb7f7591f9d5dfc8f6e37a. One source writer.
Latest user instructions override earlier four-section descriptions. R10/R13
accountant permission boundaries, R11 sale insurance, R12 optional scoring and
OD-01/OD-08 handover/provider boundaries remain authoritative.

Own only frontend paths needed for this coherent change:
- web/apps/portal/src/sections.tsx; routes.test.tsx
- web/apps/portal/src/features/accounting/ (new small navigation/profile files)
- web/apps/portal/src/features/realization/navigation.tsx; navigation.test.tsx
- realization/pages/billing.tsx, pages/settings.tsx, pages/retail.tsx
- realization/delivery-readiness.tsx and delivery-readiness.css (new if useful)
- realization/accountant.test.tsx, cash-sale.test.tsx, installment-servicing.test.tsx
- web/packages/kit/src/section-selection.ts, portal.tsx and existing matching tests
- kit/src/company-employees.tsx, company-roles.tsx, matching existing tests;
  a small accounting-access.ts helper and index.ts export if useful.
Avoid shell.tsx changes unless required; request expanded scope if necessary.

1. Extend PortalSection/SectionId to accounting at /accounting. Existing single
   session/company/query/router lifecycle remains. Provide accountant home with
   existing BillingPage and profile/security, using insurer-style separate shell
   or the shared workspace shell with its own brand/navigation. Financial access
   means (retail.read plus retail.invoices.manage or retail.payments.accept) OR
   equivalent commerce grants, in a seller company. Unguarded profile must not
   accidentally admit unauthorized users. Use a small general section-access
   predicate or equivalent permission-based admission, not role-name checks.
   Company administrator/mixed functional roles see the section switcher and
   valid last-section restoration. A pure accountant (existing minimal preset,
   no additional seller operational grants) should land in accounting, including
   when their previously saved/default seller route is no longer appropriate.
   Prevent switcher links that re-select the already active company merely to
   enter a denied section. Preserve deferred company-context selection.
2. Reuse BillingPage for both current /billing compatibility and accountant home;
   preserve invoice/payment server permissions, read-module separation, queue,
   whole-check acceptance, stale/error/refresh and dialog behavior. Accountant
   profile uses existing session and ChangePassword; staff creation stays in
   Realization settings. No backend/API/money changes. Do not create a fourth/fifth
   app, separate login, auth storage or request client.
3. Remove the seller Insurance sidebar item/import. Insurer cabinet remains;
   sale tab «Страхование» remains. Keep /billing compatible for permitted users.
4. Add an explicit accountant creation shortcut to CompanyEmployees for seller
   admins. Use the existing prepared role by actual permission set, excluding
   admin/extra-power roles; do not depend on role name alone. Standard preset is
   nine keys in internal/modules/identity/service/company_roles.go, already
   backfilled in migration36. The submitted employee payload must contain its
   role ID, plus normal name/login/temporary password fields. Disable/guide on
   missing/failed roles. No implicit role creation on render or role replacement
   on other employees. Normal employee/role actions remain. If preset is absent,
   existing CompanyRoles can expose an explicit accountant role template using
   normal role-create API after the user's click, or clear role-setup guidance.
5. Replace DeliveryReadiness kit-timeline in retail.tsx with a responsive stepper.
   Scoped CSS; use existing colors/tokens and icons. On desktop lay out compact
   cards/connected steps; on narrow screens vertical without overflow. Clearly
   show required completed count, each task status, and current outstanding
   action. Mark optional registration separate; do not label missing/undefined
   facts done or infer completion from amounts in other invoices. 'На проверке'
   only when matching invoice has pending amount/submitted evidence; otherwise
   appropriate outstanding text. Keep delivered/cancelled/absent checklist
   behavior, partner-finance unresolved restriction, no fake sequential locks.
   Buttons navigate inside current sale dialog to contracts/payments/insurance
   or existing registration action (with permission), never mutate automatically.
   Preserve readiness targetRef/focus/navigation used by payment flow, existing
   accessible wrapper label, and optional cash registration rules. Missing
   allowedActions must never cause new delivery or financial actions.
6. Update only meaningful existing tests/fixtures affected by changed routing,
   permission boundaries and readiness markup. Add scenarios within existing
   suites only where needed for new access/onboarding behavior; avoid cosmetic
   snapshot tests or new test files. Do not run tests during implementation.
   Primary performs one final affected-unit pass after all source is ready.

Run bash tools/check-git.sh before edits. Use apply_patch. No Git, canonical
docs, runtime, browser, build/lint/typecheck/React Doctor or unit execution.
No test_creds.txt reads and preserve unrelated identity test bytes. Write a
compact result + source SHA256 list here; release source lease when done.
