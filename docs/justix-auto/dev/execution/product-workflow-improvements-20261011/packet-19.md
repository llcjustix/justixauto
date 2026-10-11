# P19 — Reservation and advance UI

Requirements: R3. Dependencies: P18b. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `web/apps/realization/src/reservation-workspace.tsx`
- `web/apps/realization/src/reservation-workspace.test.tsx`
- `web/apps/realization/src/reservation-payment.tsx`
- `web/apps/realization/src/reservation-payment.test.tsx`
- `web/apps/realization/src/order-create.tsx`
- `web/apps/realization/src/order-workspace.tsx`
- `web/apps/realization/src/data.ts`
- `web/apps/realization/src/pages/purchases.tsx`
- `web/apps/realization/src/pages/trade.tsx`
- `web/apps/realization/src/pages/billing.tsx`

## Behavior, interfaces and data flow

Distributor enters available future supply model/spec/colors/quantity/date; dealer browses stock/future lines and requests reservation. Modal shows request/proposed/accepted terms digest, price/currency/expiry/optional advance, counterparty accept/confirm, allocated VINs without double counting, extension and current next actions. UserType+permissions determine party controls. Integrate accountant advance/refund queue into Billing; submitted evidence labelled pending, accepted credit and refund confirmed distinguished. Continue reservation to purchase/consignment with explicitly agreed terms and opt-in partial-purchase handover field; default unchecked. Preserve direct order route. Display server expiry/release reason and agreed advance deadline; no browser-only expiry.

## Acceptance and affected units

Future/no-VIN and stock-VIN request payloads; changed proposal requires accept; confirmed price readonly until proposal; evidence pending vs accepted; refund pending vs confirmed; conversion includes digest and explicit flag; errors/stale refresh without double submit.

Run each affected command once after the relevant change:

- From `/Users/bakhromachilov/startups/justixauto/web`: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/reservation-workspace.test.tsx src/reservation-payment.test.tsx` (Realization only).

Keep these selections separate; repairing a failure reruns only affected suites, not a previously green selection. Require the output to name the intended suites; an omitted suite is not evidence. No other development verification or runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

Deadline rule: show automatic release after the agreed mandatory-advance deadline only when dealer never submitted evidence; pending/accepted advance protects reservation. Render released status and reason from server; do not emulate expiry with a browser-only timer.

