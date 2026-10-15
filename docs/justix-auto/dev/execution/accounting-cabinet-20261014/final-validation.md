# Final affected-unit validation

2026-10-14, primary `/root`; branch `feature/accounting-cabinet-20261014`, base
`c92c21627efacd0604cb7f7591f9d5dfc8f6e37a`. Source writer released before checks.

All 42 selected cases are green: 33 accounting/routing/onboarding cases and nine
sale-readiness cases. Application source was unchanged after the first pass;
only the two newly added test fixtures were corrected, with only their failures
rerun. All 13 source/test hashes match the final entries in [worker handoff](result.md).

The first accounting run had 31 passes and two failures: the accountant shortcut
test clicked during role loading; the same-kind company switch test used a
MemoryRouter that reset its initial URL on session retirement. It was changed
to the existing BrowserRouter fixture pattern. The two-case rerun passed the
company switch; the employee test used an unavailable jest-dom matcher. It now
uses the repository's native DOM disabled-attribute assertion and passed its
single-case rerun. No product authorization or financial rules were relaxed.

Commands below ran from `web/` using the pinned Node24.21.0 and installed Vitest.
These actual commands supersede the worker's suggested pnpm commands.

## accounting

```sh
../docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node node_modules/vitest/vitest.mjs run --config vitest.workspace.ts --configLoader runner packages/kit/src/section-selection.test.ts packages/kit/src/portal.test.ts packages/kit/src/company-employees.test.ts apps/portal/src/routes.test.tsx apps/portal/src/features/realization/navigation.test.tsx apps/portal/src/features/realization/accountant.test.tsx
```

31 passed, 2 failed (33); exit1. The two fixture failures are explained above.

## readiness

```sh
../docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node node_modules/vitest/vitest.mjs run --config vitest.workspace.ts --configLoader runner apps/portal/src/features/realization/cash-sale.test.tsx apps/portal/src/features/realization/installment-servicing.test.tsx -t 'uses explicit registration policy|defaults to deal readiness|selects readiness before focus|suppresses completed contract and registration|falls back to stable details focus|shows draft terms in contract action|shows supplied readiness facts|gates delivery on server permission|targets preparation readiness'
```

9 passed, 69 skipped; exit0. Covers cash optional registration, readiness rendering/focus, existing action suppression, missing checklist, contract without scoring, policy blocker and server delivery permission.

## recheck

```sh
../docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node node_modules/vitest/vitest.mjs run --config vitest.workspace.ts --configLoader runner packages/kit/src/company-employees.test.ts packages/kit/src/portal.test.ts -t 'adds an accountant only through the exact prepared permission set|selects another seller for accounting even when the active seller is also a membership candidate'
```

1 passed, 1 failed, 17 skipped; exit1. Company selection passes; employee assertion matcher corrected.

## employee-recheck

```sh
../docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node node_modules/vitest/vitest.mjs run --config vitest.workspace.ts --configLoader runner packages/kit/src/company-employees.test.ts -t 'adds an accountant only through the exact prepared permission set'
```

1 passed, 3 skipped; exit0. Prepared role assigned in the employee POST.

## Scope and limits

Routing checks cover accountant default/remembered mixed roles, refreshed company
context, seller-kind and read-plus-financial admission. Existing billing units
cover the minimal accountant role, grouped whole-check acceptance and retail/
commerce query separation. Creation tests cover both normal employees and exact
prepared-accountant permissions. Readiness checks preserve payment focus and
server action gates; missing facts remain incomplete, cash registration optional.

No live accounts, roles or payments were created. No backend/schema/API/dependency
changes, runtime restart, browser/E2E, lint, typecheck, build or automated QA/review
chain. Visual layout is source-inspected, not browser-verified. Units are not a
release-readiness claim. Local commit only; no remote push or production action.
Protected `internal/modules/identity/service/company_contract_test.go` remains
SHA-256 `a4d4b3f67107b5362810dcc96f68f1de8e59cae466dacafb7e100e29a59fe99f`;
`test_creds.txt` was not read or modified. Source/check leases released.
