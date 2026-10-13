# P2 section router result

Actor: `/root/unified_portal_router`. Branch: `feature/unified-frontend-20261013`; base `743bd6c4accadcd50afb59534bb66dabe2f3baf8`. Accepted dependency: `result-01.md`. Before editing source, `bash tools/check-git.sh` returned `GIT CHECK OK: /Users/bakhromachilov/startups/justixauto`. Only P2's seven owned Kit paths were modified or added. No Git, runtime, network/install, or unowned source actions. **P2 source/test writer lease released by this report.**

## Exact P3 handoff

`@justixauto/kit` now exports:

```ts
type SectionId = 'realization' | 'financing' | 'insurance' | 'admin';
type PortalSection = Omit<ShellOptions, 'rootId' | 'basename'> & {
  id: SectionId;
  prefix: '/' | '/finance' | '/insurance' | '/admin';
};
function PortalRoutes({ sections }: { sections: PortalSection[] }): ReactNode;
function mountPortal({ rootId, sections }: { rootId: string; sections: PortalSection[] }): void;
```

`mountPortal` creates one root, one QueryClient provider, one `SessionGate` with no kind gate, and one root `BrowserRouter`. `PortalRoutes` consumes that router and admitted session. Existing `mountApp` remains available for the temporary old entries. `Shell` is exported internally for portal composition; its existing options and variants remain compatible, with an optional `sectionSwitcher` node.

P3 must supply final **absolute browser paths** for every `NavItem.to` and `search.path` result: seller paths remain `/dashboard`, `/settings`, etc.; finance uses `/finance/overview`, `/finance/applications`, `/finance/programs`, `/finance/partners`, `/finance/settings`; insurance uses `/insurance/overview`, `/insurance/applications`, `/insurance/settings`; Admin uses `/admin/overview`, `/admin/companies`, `/admin/catalog`, `/admin/users`, `/admin/roles`, `/admin/permissions`, `/admin/integrations/*`, `/admin/audit`. Keep the existing nav permission declarations, including seller billing's OR array. Provider and Admin page-internal absolute navigation identified in `frontend-mapping.md` still belongs to P3.

Route selection uses the effective session permission array and active company kind. Seller dashboard and provider settings remain usable for a matching active company without a nav permission; Admin requires an actually permitted Admin nav item and works with no company. A requested route checks its nav item before its element can mount. The root chooses a validated last allowed section for the actual user/company-or-platform key, then first usable section in supplied order. Permitted direct URL search/hash survive. Wrong-kind deep links show matching company choices (one attempted once, multiple explicit, zero explicit no access); selected company context is admitted by P1 before rendering the page. Inactive memberships appear only as labelled company-selection controls in the internal switcher. The shell/page subtree is keyed by admitted `scopeKey` and section. All three shell variants contain the internal switcher; old full-reload cabinet anchors are gone. P1's now redundant post-admission `qc.clear()` and `qc.invalidateQueries()` calls were removed from CompanySwitch and BranchSwitch. No role names, permission names, verification-wide gate, or new endpoint were added.

## Focused unit evidence

Command, from repository root:

```text
npm --prefix web run test:unit -- --project @justixauto/kit packages/kit/src/section-selection.test.ts packages/kit/src/portal.test.ts packages/kit/src/shell.test.ts
```

First run after implementation: `Test Files 3 passed (3); Tests 16 passed (16); exit 0`. After a relevant P2 UI/test addition, one further run: `Test Files 3 passed (3); Tests 19 passed (19); exit 0` (Vitest 4.1.11, 966 ms). No unit failures. The tests cover company-kind/permission decisions, seller accountant OR access, platform Admin with no company, advisory history, direct query/hash, denied element render count zero, pending/failing/multiple company selection, internal section navigation across three variants, inactive-company switch, and final absolute search path.

Only these affected Kit unit files were run. No lint, typecheck, build, React Doctor, browser/E2E/integration, server, review/QA loop or release check was run under U6. Unit success does not establish installation or release readiness.

## Final SHA-256

| P2 path | SHA-256 |
|---|---|
| `web/packages/kit/src/shell.tsx` | `594b3b49628d88cc9c2ed07178446843f1cdcc4657059979684559d5f113fd96` |
| `web/packages/kit/src/index.ts` | `bb03c9965aa3d24752949e75e15fc11c1eca306e0d4d25c06a61fafabbd41877` |
| `web/packages/kit/src/portal.tsx` | `3ea2f60e6d75c93b8c140500924c47a2c1b464ff6f01281364d6f1e75902a7e6` |
| `web/packages/kit/src/section-selection.ts` | `1c9427c383751fd80b475a7934f84a210877744e2650d9696ba25ccab1d0e801` |
| `web/packages/kit/src/section-selection.test.ts` | `f58fb8fba144e5456db263fbafcf8611c8bdcf1613d7b9abd6fa0714e219db70` |
| `web/packages/kit/src/portal.test.ts` | `5480819f0ffb4079a6a6554697c6248c91d41284f2a470f38c1683b40ecc255f` |
| `web/packages/kit/src/shell.test.ts` (unchanged) | `3e76c39e86bb8f65402833e83d92dd989583e8314f98e42648cd36849487e0f8` |

No reported dependency defect or unowned path need. P3 can consume these exact exports and absolute-path contract after the root accepts this handoff.
