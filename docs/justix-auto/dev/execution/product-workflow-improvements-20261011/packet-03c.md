# P03c — Explain verification at commercial entry points

Requirements: R2. Dependencies: P02/P03a/P03b. Read plan.md common execution rules and referenced source/contract slices. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 18k tokens (stop/split if exceeded).

## Exact owned paths

- `web/apps/realization/src/sale-form.tsx`
- `web/apps/realization/src/order-create.tsx`
- `web/apps/realization/src/order-actions.tsx`
- `web/apps/realization/src/sale-offer-form.test.tsx`
- `web/apps/realization/src/commercial-access.test.tsx`
- `web/apps/realization/src/pages/stock.tsx`
- `web/apps/realization/src/pages/inventory.tsx`
- `web/apps/realization/src/pages/offers.tsx`
- `web/apps/realization/src/pages/partners-finance.tsx`
- `web/packages/kit/src/session.tsx`
- `web/packages/kit/src/messages.ts`

## Behavior, interfaces and data flow

Use P01/P02 current company capability to disable new stock, sales, offers, orders and new external application controls with visible verification explanation and link to settings. Cover direct modal/deep-link opening, not navigation only. Match P03 action matrix; do not disable existing-obligation payment/fulfilment controls. Keep server denial message when stale session disagrees.

## Acceptance and affected units

Unverified creation submit never fires; verification reason visible; setup/history remain. Changing company updates action availability. Existing obligation dialogs still usable.

Run each affected command once after the relevant change:

- From `/Users/bakhromachilov/startups/justixauto/web`: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/commercial-access.test.tsx src/sale-offer-form.test.tsx` (Realization only).

Keep these selections separate; repairing a failure reruns only affected suites, not a previously green selection. Require the output to name the intended suites; an omitted suite is not evidence. No other development verification or runtime actions.

Stop: Stop on any new product rule, ownership expansion or context ceiling; report exact missing decision/path. No unrelated refactoring.

