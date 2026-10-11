# P05 correction result — additive Billing access

Status: DONE. Branch/head at dispatch: `feature/company-commerce-workflows-20261011` / `4fa683ba4708bb71031534260ec5cb8742ce9977`. No Git writes, runtime, browser, lint, typecheck, build, review or QA action occurred.

## Changed paths

- `web/packages/kit/src/shell.tsx` — `NavItem.permission` accepts one key or an any-of key list; the shell filters navigation using that rule.
- `web/packages/kit/src/shell.test.ts` — new pure unit covers any-permission navigation access.
- `web/apps/realization/src/navigation.tsx` — Billing permits `retail.read` or `commerce.read`; existing Offers remains retail-only.
- `web/apps/realization/src/pages/billing.tsx` — recomputes the effective permitted tab every render, renders only permitted tabs, and disables inaccessible module queries.
- `web/apps/realization/src/accountant.test.tsx` — covers accountant, retail-only and commerce-only Billing access/query isolation.

## Test discovery correction

The accepted packet named `shell.test.tsx`, but the Kit Vitest project includes only `src/**/*.test.ts`; its prescribed selector therefore found no tests. The hub approved renaming that new pure test to `shell.test.ts` and the existing pure P02 `company-verification.test.tsx` to `company-verification.test.ts`. No Vitest configuration changed.

## Unit evidence

- `npm --prefix web run test:unit -- packages/kit/src/shell.test.ts packages/kit/src/company-verification.test.ts` — PASS: 2 files, 2 tests; output named both selectors.
- `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/accountant.test.tsx` from `web/` — PASS: 1 file, 3 tests.

## Exclusions

No unrelated P05 payment-review or permissions suites were repeated. No browser/runtime/DB checks, lint, typecheck, build, review, QA, Git staging/commit, or canonical-document changes occurred.
