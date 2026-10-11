# P06 — Percent-first installment down payment

Requirements: R8. Dependencies: P05. Read plan.md common execution rules and referenced source/contract slices. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 18k tokens (stop/split if exceeded).

## Exact owned paths

- `web/apps/realization/src/installment-terms-form.tsx`
- `web/apps/realization/src/installment-percent.ts`
- `web/apps/realization/src/installment-percent.test.ts`
- `web/apps/realization/src/installment-calculator.test.ts`
- `web/apps/realization/src/terms-dialog.test.tsx`
- `web/apps/realization/src/sale-form.test.tsx`

## Behavior, interfaces and data flow

Keep saved API downPayment Money unchanged. Percent is primary editable decimal-string input with synchronized money field; use BigInt rational parsing, no Number/parseFloat money. Select explicit nearest-minor-unit, half-up rule for positive percent calculation; document it in preview. Money-origin edits keep exact minor amount canonical; displayed derived percent may be rounded for display but must not re-round unchanged amount on blur/render. Percent-origin edits compute amount once, reject zero/full-price after rounding. Preserve fixedDown invoice lock for BOTH fields and original schedule generation, currency and saved schedules. Use bounded decimal precision, state the display precision without changing monetary precision.

## Acceptance and affected units

10% and fractional percent; half-unit rounding; huge 38-digit money; 0/100/out-of-range/invalid; small price rounding to 0 or full rejected; money→percent→render preserves cents; fixedDown blocks edits; existing schedule unchanged.

Run each affected command once after the relevant change:

- From `/Users/bakhromachilov/startups/justixauto/web`: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/installment-percent.test.ts src/installment-calculator.test.ts src/terms-dialog.test.tsx src/sale-form.test.tsx` (Realization only).

Keep these selections separate; repairing a failure reruns only affected suites, not a previously green selection. Require the output to name the intended suites; an omitted suite is not evidence. No other development verification or runtime actions.

Stop: Stop on any new product rule, ownership expansion or context ceiling; report exact missing decision/path. No unrelated refactoring.

