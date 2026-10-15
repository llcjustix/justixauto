# Accessible section after changing users — 2026-10-13

Status: SOURCE COMPLETE; final browser validation passed. Application authority: /root.
User reports logging out of Admin and signing in as another user leaves
/admin/companies showing no access.

Cause: PortalRoutes finds the previous Admin URL, but the new session has no
active company or Admin grant. Its denied-section branch only considers sections
available in the current context, missing accessible company choices at root.
Similarly, a stale provider URL without matching memberships can block an
otherwise available section.

Approved behavior: unavailable section URLs return to the common entry, which
selects an allowed remembered/default section or offers/selects a company from
the new user's memberships. Select one matching company automatically, ask when
several match, and confirm its context/session before rendering company pages.
Preserve permitted deep URLs, query/hash on those URLs, explicit company choice,
user/context-scoped section memory, grants and genuine no-access behavior.
No unconditional login redirect that discards an authorized deep link.

Scope: shared portal routing in web/packages/kit/src/portal.tsx; no backend,
permission or session-contract change expected. Existing tests remain in place.
Implement first; primary performs one focused browser regression at the end with
isolated mocked API sessions (Admin logout -> seller login at the stale URL,
company selection, provider fallback and permitted deep URL). No automatic unit
test additions/runs, builds, lint, typecheck or review/QA loop. Final browser
evidence covers frontend behavior, not live backend authorization.

Dependency d078b4b0e200942735d581ea6c4be761d84e9add; branch
fix/login-section-redirect-20261013 at identical current source. Main checkout
only, one writer; version_control owns Git. Preserve unrelated dirty
internal/modules/identity/service/company_contract_test.go and unread
test_creds.txt. No credentials, database reset, restart or remote actions.
Root/web AGENTS and final-validation-policy-20261013.md apply. Reference ADR-15
and business-logic.md Applications and access. No unresolved domain policy.

Completion: denied section URLs now replace-navigate to the common entry;
requested company selection remains available only when that section has actual
matching memberships. One source file changed. The final Chromium pass verified
five frontend scenarios on the actual local app with intercepted API/session
fixtures: account switch, explicit multiple-company choice, stale provider URL,
permitted deep URL and genuine no access. Context confirmation precedes content.
No unit tests, builds or live backend mutations. Source/browser leases released.
See execution/login-section-redirect-20261013/final-validation.md for source
identity, command, exact results and limits.
