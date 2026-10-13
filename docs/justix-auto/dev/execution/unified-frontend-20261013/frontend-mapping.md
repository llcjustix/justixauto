# Frontend map — read-only handoff

Actor /root/unified_frontend_map, explorer gpt-6-sol/high; source base 743bd6c.
No edits, tests or runtime actions. Scoped codebase findings accepted by root.

- Four `web/apps/{realization,financing,insurance,admin}/src/main.tsx` call
  kit `mountApp` in shell.tsx, with basenames /, /finance, /insurance, /admin.
  Each boot creates its own QueryClient, SessionGate and BrowserRouter.
- kit/src/session.tsx SessionGate reads GET /identity/session, handles login and
  forced password. Context selection PUT /identity/session/context refreshes
  permissions. CompanyGate's sole-company auto-selection must move under coherent
  unified selection; Admin must not require a company.
- Backend effective permissions combine platform roles and the selected company
  roles (identity/service/auth.go). accessibleCompanies lists membership/kind,
  not inactive-company permissions. Do not infer entitlements before selection.
- Shared shell permitted() filters nav and mounted Route. Seller Dashboard and
  Settings/provider Settings lack a nav permission, so company-kind gate still
  matters. Accountant Billing uses existing OR permission; seller verification
  must stay at useCommercialAccess operation level.
- Navigation lives in seller/provider src/navigation.tsx; Admin nav is inline
  in src/main.tsx. Cabinet hrefs in shell.tsx are full reloads. CompanySwitch
  filters companies by cabinet kinds and clears query cache after switching.
  Many existing data query keys omit company; persistent single-client context
  changes must fence late results and clear state before new routes render.
- Finance pages.tsx uses absolute /applications and /programs navigation; Admin
  pages.tsx uses absolute paths and /audit. These assumed router basenames and
  must be correctly prefixed under one root router, as must nav/search links.
- Seller /dashboard,/sales etc and provider/admin current route families can
  stay stable. One entry must preserve permitted deep links and avoid mounting
  a denied section while choosing another company or route.
- Existing affected units include kit shell, seller navigation/accountant/
  commercial-access, plus new route/context behavior units. No whole-suite,
  browser or build verification is authorized by this mapping.
