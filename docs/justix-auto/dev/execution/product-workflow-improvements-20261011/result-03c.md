# P03c result — Commercial UI verification guard

Status: DONE

Implemented the P02 company verification capability at Realization's new
commercial-operation entry points. An unverified, submitted, returned, or
untyped seller sees the verification reason and a link to company settings;
new sale, order, stock receipt, retail/B2B offer, insurance application, and
financing application submits do not fire. Direct sale and order modal entry
is also blocked. The current verification value is read again on render, so a
company-context refresh updates action availability and prevents a form left
open under a stale session from submitting.

Historical servicing is preserved: warehouse/profile maintenance and existing
order payment/fulfilment actions remain enabled. The existing
`company_verification_required` API message remains the server-side fallback
when a stale browser session disagrees with the backend.

Changed paths:

- `web/packages/kit/src/session.tsx`
- `web/apps/realization/src/sale-form.tsx`
- `web/apps/realization/src/order-create.tsx`
- `web/apps/realization/src/pages/stock.tsx`
- `web/apps/realization/src/pages/inventory.tsx`
- `web/apps/realization/src/pages/offers.tsx`
- `web/apps/realization/src/pages/partners-finance.tsx`
- `web/apps/realization/src/commercial-access.test.tsx`

Verification (from `web/`):

`npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/commercial-access.test.tsx src/sale-offer-form.test.tsx`

Passed: 2 files, 17 tests. The first run failed only because the new test used
unavailable jest-dom matchers and assumed an exact repeated-notice count; the
test was repaired, then the same required selection passed. After adding the
stale-open receipt/form guard, the required selection was run again and passed.

Exclusions: no backend/handler/route/migration changes, no change to existing
order payment or fulfilment controls, no runtime/browser/QA action, and no Git
write. `order-actions.tsx` and `messages.ts` were inspected but deliberately
left unchanged: the former must retain servicing actions, and the latter
already maps the backend verification denial.
