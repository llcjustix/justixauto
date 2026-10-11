# P05 result — accountant work surface and sale invoice waiting state

Status: DONE. Branch/head at dispatch: `feature/company-commerce-workflows-20261011` / `4fa683ba4708bb71031534260ec5cb8742ce9977`. No Git writes, runtime, browser, lint, typecheck, build, review or QA action occurred.

## Changed paths

- `web/apps/realization/src/navigation.tsx` — Billing is available through `retail.read`, which the accountant preset has, rather than `commerce.read` alone.
- `web/apps/realization/src/pages/billing.tsx` — retail invoice-waiting deals and deduplicated submitted parent payment groups form the accountant queue; review refreshes the deal/billing data. B2B invoice list now includes supplier-side rows so the existing projected void/review controls are reachable.
- `web/apps/realization/src/pages/retail.tsx` — retail issue controls additionally require `retail.invoices.manage`.
- `web/apps/realization/src/pages/trade.tsx` — B2B issue control additionally requires `commerce.invoices.manage`.
- `web/apps/realization/src/accountant.test.tsx` — covers the accountant preset’s exact UI-relevant grants (and no sale/trade grant), waiting state, and one grouped parent review.
- `web/packages/kit/src/permissions.test.ts` — includes the two new invoice permission labels in the catalog ordering unit.

## Unit evidence

- `npm --prefix web run test:unit -- packages/kit/src/permissions.test.ts` — PASS: 1 file, 2 tests.
- `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/accountant.test.tsx src/installment-payment-review.test.tsx` from `web/` — PASS: 2 files, 4 tests.

## Exclusions

No browser/runtime/DB checks, lint, typecheck, build, review, QA, Git staging/commit, or canonical-document changes. UI action visibility remains a convenience projection; API permission and party/state checks remain authoritative.
