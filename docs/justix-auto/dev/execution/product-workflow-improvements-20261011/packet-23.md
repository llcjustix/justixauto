# P23 — Consignment and return UI

Requirements: R3/R5. Dependencies: P22. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `web/apps/realization/src/consignment-workspace.tsx`
- `web/apps/realization/src/consignment-workspace.test.tsx`
- `web/apps/realization/src/data.ts`
- `web/apps/realization/src/order-fulfilment.tsx`
- `web/apps/realization/src/order-actions.tsx`
- `web/apps/realization/src/pages/trade.tsx`
- `web/apps/realization/src/pages/inventory.tsx`
- `web/apps/realization/src/pages/stock.tsx`

## Behavior, interfaces and data flow

Show ownership and custody separately in vehicle/order details. Accepted mode/settlement and handover agreement visible; consignment handover uses P21 mode-aware endpoints. Distributor requests return choosing its destination warehouse; dealer sees request and confirms actual movement (reason/error/state visible). No controls for sold stock; reservation conflict clearly explains why blocked. Preserve financial obligation summary on return. Refresh stock/warehouse/order/consignment/return queries after success. Use exact P21/P22 DTOs; no frontend owner mutation.

## Acceptance and affected units

Correct party controls; return requested state does not change stock; confirmation success updates custody while owner persists; sold/hold/capacity errors visible; accepted partial purchase agreement displayed before handover; no approval inferred from UI checkbox without server accepted terms.

Run each affected command once after the relevant change:

- From `/Users/bakhromachilov/startups/justixauto/web`: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/consignment-workspace.test.tsx` (Realization only).

Keep these selections separate; repairing a failure reruns only affected suites, not a previously green selection. Require the output to name the intended suites; an omitted suite is not evidence. No other development verification or runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

