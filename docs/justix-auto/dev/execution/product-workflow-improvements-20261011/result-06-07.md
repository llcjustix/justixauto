# P06/P07 result

Status: DONE. Writer dispatch began on `feature/company-commerce-workflows-20261011` at `4fa683ba4708bb71031534260ec5cb8742ce9977`; the pre-existing dirty checkout was preserved. No Git writes, runtime, browser, lint, typecheck, build, review or QA action occurred.

## R8 — percent-first installment terms

- Added `web/apps/realization/src/installment-percent.ts` with bounded six-decimal percent parsing and `BigInt` rational conversion. Positive half minor units round up; the preview states that rule and display precision.
- `installment-terms-form.tsx` now presents percent before money. Percent-origin values calculate a minor-unit amount once; money-origin values remain canonical and only derive a displayed percent, so an unchanged amount is never re-rounded.
- Both controls are read-only and event-guarded when `fixedDown` comes from an issued invoice. The saved API `downPayment`, currency, validation and schedule generation remain unchanged.
- Added exact percent units (`installment-percent.test.ts`) and sale-form coverage for fractional percent, money-cent preservation, invalid percent and both-field invoice lock. The existing sale fixture now supplies the verified seller context required by the predecessor commercial-access gate.

## R7 — Islamic line terminology

- Replaced user-facing combined Bank/MFO finance-line text in Realization, Financing and shared Kit with `Исламская линия`, including selection, application, status/error and cabinet labels.
- Preserved API `bank`/`mfo` enums, Admin routes/onboarding categories, organization-kind labels and bank requisites. Targeted Insurance search found no local combined label; shared Kit supplies its relevant wording.

## Unit evidence

- `npm --prefix web run test:unit -- apps/admin/src/labels.test.tsx apps/admin/src/company-form.test.tsx` — PASS, 2 files / 7 tests.
- `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/installment-percent.test.ts src/installment-calculator.test.ts src/terms-dialog.test.tsx src/sale-form.test.tsx` from `web/` — initial run: `installment-percent`, `installment-calculator` and `terms-dialog` passed; `sale-form` was blocked by the predecessor verification fixture.
- Repair selector: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/sale-form.test.tsx` from `web/` — PASS, 1 file / 12 tests, after the fixture and fixed-down event guard repair.

## Exclusions

No server API or enum change, legal/Sharia/rate claim, money float arithmetic, browser/runtime/DB verification, lint, typecheck, build, Git staging/commit or canonical-document edit.
