# P07 — Islamic line terminology

Requirements: R7. Dependencies: P06. Read plan.md common execution rules and referenced source/contract slices. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 18k tokens (stop/split if exceeded).

## Exact owned paths

- `web/apps/realization/src/data.ts`
- `web/apps/realization/src/sale-form.test.tsx`
- `web/apps/realization/src/pages/partners-finance.tsx`
- `web/apps/realization/src/pages/billing.tsx`
- `web/apps/realization/src/pages/sales.tsx`
- `web/apps/realization/src/pages/retail.tsx`
- `web/apps/financing/src/main.tsx`
- `web/apps/admin/src/labels.ts`
- `web/apps/admin/src/pages.tsx`
- `web/apps/admin/src/main.tsx`
- `web/apps/admin/src/labels.test.tsx`
- `web/apps/admin/src/company-form.test.tsx`
- `web/packages/kit/src/org-settings.tsx`
- `web/packages/kit/src/applications.tsx`
- `web/packages/kit/src/messages.ts`
- `web/packages/kit/src/shell.tsx`

## Behavior, interfaces and data flow

Replace user-facing 'Банк / МФО' and corresponding grammatical combined finance-line labels with 'Исламская линия' across four apps/shared kit. Preserve bank/mfo API enums, provider kind distinction needed for Admin onboarding, registered organization names and error codes. Do not blanket replace bank details/requisites or fixture company names. Read insurance sources to confirm shared kit covers any labels; if a local insurance label exists report exact file for scope addition. Do not imply Sharia/legal/rate certification or change calculation.

## Acceptance and affected units

Affected label/component units expect new line label; selecting partner-finance still posts unchanged enum. Admin bank/mfo categories remain distinguishable where technical organization kind is necessary.

Run each affected command once after the relevant change:

- From `/Users/bakhromachilov/startups/justixauto`: `npm --prefix web run test:unit -- apps/admin/src/labels.test.tsx apps/admin/src/company-form.test.tsx` (Kit/Admin only).
- From `/Users/bakhromachilov/startups/justixauto/web`: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/sale-form.test.tsx` (Realization only).

Keep these selections separate; repairing a failure reruns only affected suites, not a previously green selection. Require the output to name the intended suites; an omitted suite is not evidence. No other development verification or runtime actions.

Stop: Stop on any new product rule, ownership expansion or context ceiling; report exact missing decision/path. No unrelated refactoring.

