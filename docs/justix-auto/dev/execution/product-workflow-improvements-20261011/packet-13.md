# P13 — Retail and order contract workspaces

Requirements: R9. Dependencies: P11/P12. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `web/apps/realization/src/contract-workspace.tsx`
- `web/apps/realization/src/contract-workspace.test.tsx`
- `web/apps/realization/src/data.ts`
- `web/apps/realization/src/pages/retail.tsx`
- `web/apps/realization/src/pages/trade.tsx`

## Behavior, interfaces and data flow

Embed reusable contract workspace in sale and order dialogs. P11/P12 endpoint/DTO-specific adapters supply template editing and filled preview, external upload, signed scan/reference/date, version history and amendment creation. Clearly label template as configurable company text and paper signature scan. Signed versions read-only; no replacement button. Private authenticated downloads only. Display snapshotted parties/vehicle/order/price and current-versus-version mismatch without rewriting old facts. Preserve modal keyboard/back flow and invoice/insurance tabs.

## Acceptance and affected units

Both owner adapters send correct IDs/revision; file failure not signed; signed history readonly; amendment targets parent; edits to live deal do not change displayed historical snapshot.

Run each affected command once after the relevant change:

- From `/Users/bakhromachilov/startups/justixauto/web`: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/contract-workspace.test.tsx` (Realization only).

Keep these selections separate; repairing a failure reruns only affected suites, not a previously green selection. Require the output to name the intended suites; an omitted suite is not evidence. No other development verification or runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

