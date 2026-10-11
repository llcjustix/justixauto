# P09 — CRM passport editor and private upload UI

Requirements: R6. Dependencies: P08. Read plan.md common execution rules and referenced source/contract slices. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 18k tokens (stop/split if exceeded).

## Exact owned paths

- `web/apps/realization/src/pages/crm.tsx`
- `web/apps/realization/src/data.ts`
- `web/apps/realization/src/customer-passport.tsx`
- `web/apps/realization/src/customer-passport.test.tsx`

## Behavior, interfaces and data flow

Dedicated customer modal/section for optional P08 profile fields and passport-copy upload using existing FieldSpec file mechanism/purpose. Authenticated fileUrl only, sensitive-download authorization surfaced; no image public hosting/localStorage. Edit uses current customer revision; existing profile fields preserved. Keep passport details out of broad CRM listing/search columns.

## Acceptance and affected units

Create/edit optional fields and binding request match P08; failed upload/binding visible and does not claim saved; allowed file download uses authenticated route; no private details spilled in table.

Run each affected command once after the relevant change:

- From `/Users/bakhromachilov/startups/justixauto/web`: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/customer-passport.test.tsx` (Realization only).

Keep these selections separate; repairing a failure reruns only affected suites, not a previously green selection. Require the output to name the intended suites; an omitted suite is not evidence. No other development verification or runtime actions.

Stop: Stop on any new product rule, ownership expansion or context ceiling; report exact missing decision/path. No unrelated refactoring.

