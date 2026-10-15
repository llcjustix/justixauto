# Read-only switcher findings

Actor: /root/accounting_switch_scope, 2026-10-14. Recorded by root from the
agent's delivered findings; no source/checks were performed by this explorer.

The switcher is assembled in web/packages/kit/src/portal.tsx, PortalRoutes
(approximately lines 114–128 at base a99f72c). `destinations` is derived from
allowed sections. Shell only renders the injected switcher in its layouts.

Derive a separate display list, omitting Accounting while current section is
Realization, and Realization while current section is Accounting. Use that list
for section-switcher existence and link mapping. Do not mutate `available`,
`destinations`, `companyChoices`, sectionForPath, allowedSections, direct routes,
readLastSection or rememberSection. There is no API or role-policy dependency.

With just the pair and no other company choice the switcher disappears. Other
section/company choices remain; one-section behavior stays as before. Existing
web/packages/kit/src/portal.test.ts covers pure-accountant/root preference,
remembered mixed access, other-company Accounting and standard switchers.

Root accepted this bounded UI-only scope; no new cosmetic test. Existing portal
cases join the final affected-unit pass after all source edits finish.
