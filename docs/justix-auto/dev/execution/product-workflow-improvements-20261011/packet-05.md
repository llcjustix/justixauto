# P05 — Accountant work surface and sale invoice waiting state

Requirements: R10/R13. Dependencies: P04. Read plan.md common execution rules and referenced source/contract slices. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 18k tokens (stop/split if exceeded).

## Exact owned paths

- `web/apps/realization/src/navigation.tsx`
- `web/apps/realization/src/retail-actions.tsx`
- `web/apps/realization/src/installment-payment-review.tsx`
- `web/apps/realization/src/accountant.test.tsx`
- `web/apps/realization/src/installment-payment-review.test.tsx`
- `web/apps/realization/src/pages/billing.tsx`
- `web/apps/realization/src/pages/retail.tsx`
- `web/apps/realization/src/pages/trade.tsx`
- `web/packages/kit/src/permissions.ts`
- `web/packages/kit/src/permissions.test.ts`

## Behavior, interfaces and data flow

Expose accountant surface through existing Billing page/navigation, gated by invoice/read/review rights rather than commerce.read alone. Show retail deals waiting for invoice and submitted grouped installment payments; reuse issued-invoice and payment-review components. Sale dialog shows existing issued invoice(s), and waiting-for-accountant state when invoice absent; only invoice managers see issue action. Normal seller registers evidence under existing right; accountant accepts/rejects with reason via grouped endpoints. Query refresh updates sale/debt/invoice/queue. No duplicate per-child review actions for grouped evidence.

## Acceptance and affected units

Accountant with no sale-manage permission reaches queue and invoice action; seller waits and sees issued invoice after refresh. Mixed role/admin works. Accepted/rejected/pending amount statuses retained. Group review operates once; server errors leave old balance.

Run each affected command once after the relevant change:

- From `/Users/bakhromachilov/startups/justixauto`: `npm --prefix web run test:unit -- packages/kit/src/permissions.test.ts` (Kit/Admin only).
- From `/Users/bakhromachilov/startups/justixauto/web`: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/accountant.test.tsx src/installment-payment-review.test.tsx` (Realization only).

Keep these selections separate; repairing a failure reruns only affected suites, not a previously green selection. Require the output to name the intended suites; an omitted suite is not evidence. No other development verification or runtime actions.

Stop: Stop on any new product rule, ownership expansion or context ceiling; report exact missing decision/path. No unrelated refactoring.

Also wire commerce.invoices.manage into B2B order invoice issue/void controls and accountant queue; commerce.trade alone must no longer display invoice-management controls. Existing trade evidence submission remains under its existing permission.

