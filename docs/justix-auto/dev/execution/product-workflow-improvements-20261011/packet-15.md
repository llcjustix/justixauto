# P15 — Insurance and demo scoring inside sale dialog

Requirements: R11/R12. Dependencies: P13/P14. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `web/apps/realization/src/sale-insurance.tsx`
- `web/apps/realization/src/sale-insurance.test.tsx`
- `web/apps/realization/src/data.ts`
- `web/apps/realization/src/contract-workspace.tsx`
- `web/apps/realization/src/pages/retail.tsx`
- `web/packages/kit/src/applications.tsx`

## Behavior, interfaces and data flow

Within own-installment sale dialog show demo scoring run/status and run/rerun action before signing, prominent 'демонстрационный' label and no coverage claim. Embed insurance create/select insurer/submit/current status/information-request response using existing insurance endpoints and InsuranceCase component from shared kit; retain insurer application and its decisions. Reuse current deal revision on submit, reload current insurance application after action. No navigation to standalone insurance page needed. Caller role gates seller actions; insurer approve/decline controls remain only addressed insurer side. Insurance requests/status and demo score are distinct state.

## Acceptance and affected units

Can request and respond without leaving sale modal; draft/under-review/information-request/approved/declined rendering; demo state does not display coverage; role cannot approve own request; stale deal revision reload; signing UI requires latest demo run.

Run each affected command once after the relevant change:

- From `/Users/bakhromachilov/startups/justixauto/web`: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/sale-insurance.test.tsx` (Realization only).

Keep these selections separate; repairing a failure reruns only affected suites, not a previously green selection. Require the output to name the intended suites; an omitted suite is not evidence. No other development verification or runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

