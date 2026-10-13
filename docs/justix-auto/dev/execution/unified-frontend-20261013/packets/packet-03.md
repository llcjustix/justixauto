# P3 — Actual portal workspace, feature relocation and stable browser URLs

Task/base/hub: plan.md. U1 actual single app; U2/U4 preservation; U5 Vite; U6.
Preserves all predecessor R1–R13 via full feature-source manifest.
Dependency: accepted P2 result-02.md exact exports and P1 lifecycle. Worker with
18k input ceiling, exclusive checkout/source/test lease, no children/Git/runtime.
Read root/workflow/web rules, ADR-15, move-manifest.md and relevant config slices.
Do NOT load every moved source file. Use apply_patch Move to with narrow context;
read semantic files incrementally and keep a result-03.md checkpoint before
continuing. Context ceiling includes new source reads, not just prompt size.

## Exact ownership

Move the 94 move rows in move-manifest.md; delete the four replace-entry rows.
No destination main.tsx inside features. Every original source/test/CSS is
accounted for, with entry composition transferred into sections.tsx.
Delete these old app configs after transferring the shared relevant configuration:
- web/apps/realization/index.html
- web/apps/realization/package.json
- web/apps/realization/vite.config.ts
- web/apps/realization/vitest.config.ts
- web/apps/realization/tsconfig.json
- web/apps/financing/index.html
- web/apps/financing/package.json
- web/apps/financing/vite.config.ts
- web/apps/financing/vitest.config.ts
- web/apps/financing/tsconfig.json
- web/apps/insurance/index.html
- web/apps/insurance/package.json
- web/apps/insurance/vite.config.ts
- web/apps/insurance/vitest.config.ts
- web/apps/insurance/tsconfig.json
- web/apps/admin/index.html
- web/apps/admin/package.json
- web/apps/admin/vite.config.ts
- web/apps/admin/vitest.config.ts
- web/apps/admin/tsconfig.json

Add:
- web/apps/portal/package.json
- web/apps/portal/index.html
- web/apps/portal/vite.config.ts
- web/apps/portal/vitest.config.ts
- web/apps/portal/tsconfig.json
- web/apps/portal/src/main.tsx
- web/apps/portal/src/sections.tsx
- web/apps/portal/src/routes.test.tsx
- web/apps/portal/src/document-policy.ts
- web/apps/portal/src/document-policy.test.ts

Modify web/package.json and web/package-lock.json.
Modify web/packages/kit/src/shell.tsx ONLY to remove the unused legacy mountApp
bootstrap and its now-unused imports after all old entries are gone; preserve
ShellOptions used in PortalSection. Do not otherwise alter P1/P2 implementation.

Permitted semantic changes within moved destinations:
- features/{realization,financing,insurance}/navigation.tsx: final global URLs
  (seller paths unchanged), explanatory stale multi-app comment/test adaptation.
- features/realization/navigation.test.tsx: wording/expectations only as needed.
- features/financing/pages.tsx: exact three navigate calls in OverviewPage.
- features/admin/pages.tsx: OverviewPage card browser paths and /audit navigate.
- features/{realization,financing,admin}/design.css: only concrete selector
  collision scoping required by combined imports; preserve existing appearance.
Other moved files must remain byte-identical unless a relative import demonstrably
breaks because of relocation; report exact old/new import and reason. Additional
behavior changes require root-approved ownership amendment, not opportunistic fixes.
Result: execution/unified-frontend-20261013/result-03.md.

## Composition and workspace behavior

One portal root, one main.tsx invoking mountPortal, sections.tsx exporting ordered
sections with IDs realization, financing, insurance, admin. Transfer every old nav
entry, permission array, group/icon/bottom item, search/banners/variant/class.
Seller root stays /dashboard; seller brand Реализация, finance Исламская линия,
insurance Страховая компания, Admin Администрирование. Finance/insurance nav
to values gain /finance and /insurance; inline Admin nav moves from old main
into sections.tsx with /admin prefixes. Existing shell classes/variant wrappers
must still match feature CSS. Check targeted global selectors before importing
three styles together; no broad stylesheet rewrite or new visual design.

Finance navigate('/applications') -> '/finance/applications', '/programs' ->
'/finance/programs'. Admin Overview card values '/integrations/{mfo,bank,insurance}'
and '/companies' gain /admin; '/audit' becomes '/admin/audit'. Do not change API
paths such as /financing/* or /identity/admin/*. Search q/hash and SPA NavLink
behavior stay intact. Insurance/realization feature URLs remain as mapped.
Do not add a /realization prefix or four hidden bootstraps.

Portal manifest merges the existing identical locked dependency identities,
including existing react-router-dom; no new libraries/upgrades. Retain shared
packages/* workspaces and make apps/portal the sole frontend application workspace.
Set root build:apps to build @justixauto/portal; preserve existing CI command intent.
Portal Vitest project name @justixauto/portal, include src/**/*.test.{ts,tsx}, jsdom;
this discovers every relocated suite including pure policy units. Existing
vitest.workspace.ts, eslint and coverage globs need no change.
Reconcile package-lock canonical workspace keys, @justixauto app link records and
old extraneous web/apps/* records to one portal identity without install/network
or changing dependency versions/integrities. Do not run npm install/ci to do this.

## Document fallback policy

Port5191 strict, base /, appType custom, dev/preview API proxy unchanged.
Keep Vite internal module serving and dev-only html-proxy functionality.
Remove sibling section exclusion so /finance, /insurance, /admin documents work.
Pure helper src/document-policy.ts must test only request properties; config may
continue existing env loading at actual runtime, but units never import config.
Entry fallback requires GET/HEAD plus positive text/html Accept, compatible
Fetch Metadata if supplied. Explicit /index.html requires document intent.
API including bare /api, /healthz, missing /assets entries (including extensionless),
source/module paths, malformed encodings, raw/decoded traversal and file-shaped
paths never fall back to index. Real static/module requests continue to their
static middleware; do not reject real Vite modules solely for non-HTML intent.
Keep no-store/non-stale entry response and HEAD body absence. Cross-reference P4
case matrix without claiming Vite server verification.

## Affected units only

npm --prefix web run test:unit -- --project @justixauto/portal apps/portal/src/routes.test.tsx apps/portal/src/document-policy.test.ts apps/portal/src/features/realization/navigation.test.tsx

routes.test.tsx uses actual section definitions and targeted rendered Finance/Admin
overview actions with MemoryRouter and mocked network/data; assert clicked browser
destination, seller search, insurer prefix and denied page non-mount. Include
company-scoped settings/setup and accountant billing nav from real sections.
Do not test strings/manifest copies as a substitute for navigation behavior.
document-policy.test.ts is pure: method/Accept q=0,wildcard,Fetch Metadata,
prefix/index/HEAD, API/health/source/missing asset/traversal cases; no listener/env.
No tests for byte-identical moves alone. If an actual moved test's import changes,
run that named suite once and report why it became affected. Do not rerun all 50
feature suites because their directory moved.

Stop for missing move row, out-of-scope import/config dependency, unresolved CSS
collision requiring redesign, or inability to preserve the lock without dependency
changes. Missing/stale local workspace symlinks may prevent unit tooling; report
rather than installing. Handoff exact final PortalSection/sections, pure policy
export signature, moved/adapted exceptions, retained dependency identities and
unit evidence. Unit success does not establish an installable build.

