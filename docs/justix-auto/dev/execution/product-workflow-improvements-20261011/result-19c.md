# P19c result — reservation lifecycle, VIN history and continuation UI

Status: SOURCE COMPLETE. Sole source-writer lease RELEASED after this result. Work used the main checkout on `feature/company-commerce-workflows-20261011` at supplied HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`. `bash tools/check-git.sh` passed before edits. Existing dirty predecessor files were preserved; this worker made no Git write.

## Behavior and interfaces

- Reservation detail now shows the server's next actions, effective reservation and mandatory-advance deadlines, extension actor/time, cancellation state and durable release reason/time. Supplier extension sends `{expiresAt,advanceDueAt?,reason}`; buyer or supplier cancellation sends `{reason}`, each with the current reservation `If-Match`. An expiry-only extension never supplies a new advance deadline. Pending evidence and an accepted advance awaiting full accountant-confirmed refund are shown as distinct reasons cancellation is unavailable. The existing P19b accounting panel supplies refund actions and evidence history; no browser timer releases a reservation.
- `GET /commerce/reservations/:id/allocations` populates separate current and historical VIN associations with immutable source line and read-only claim provenance. The supplier selects an accepted future source line and a VIN from company-owned `GET /inventory/vehicle-units?placement=warehouse&eligible=true&modelId=…`; `POST /commerce/reservations/:id/allocations` sends only `{items:[{lineId,vehicleId}]}` with the reservation revision. Supply lot available/held/materialized values come from the refreshed server read. Historical association rows are never added to capacity.
- A distinct continuation panel keeps the accepted booking price/digest snapshot read-only. Either party can propose purchase or consignment with every accepted source line mapped through `offerLineId`, exact price/currency, route, explicit payment schedule (including a deliberately empty schedule), delivery/warranty/service and free-text handover terms. Partial purchase handover starts unchecked for every draft. Optional minimum is sent only when explicitly entered; no minimum or payment schedule is inferred. The counterparty accepts the proposed digest and proposed revision; conversion sends the accepted digest and accepted revision, each with the current reservation `If-Match`. Changed/stale requests refetch the server state and never replay themselves.
- Conversion consumes the returned `orderId`, closes the reservation workspace and opens the existing Purchases `OrderDialog`. A saved reservation `orderId` reopens that dialog without another conversion. The P19b panel displays linked credit as already included in invoice Paid; this UI does not add it again. Direct and offer order creation remain accessible; `order-create.tsx`, `order-workspace.tsx` and `trade.tsx` were untouched. Physical handover, custody and returns remain downstream P21/P23 behavior.
- Mutation sending guards block duplicate clicks. Successful and uncertain writes refresh company-scoped reservation detail/list, allocation history, supply counts, eligible stock, orders, invoice and Billing query families. If authoritative reread fails, the workspace keeps writes locked until explicit retry.

## Affected unit evidence

From `web/`, after final relevant edits:

```text
npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/reservation-workspace.test.tsx
Test Files 1 passed (1); Tests 19 passed (19)
```

The named `P19c lifecycle and continuation` cases render the real workspace and Purchases host with HTTP/session fixtures. They assert current versus historical VINs and conserved server counters, supplier source-line/VIN payload, server actions, extension, pending/accepted/refund/rejected-history states, release history, purchase/consignment terms, unchecked and explicitly accepted partial handover, optional exact minimum, zero/some/all VIN conversion, digest/revision and stale reconciliation, duplicate conversion blocking, and linked order opening. The existing P19a cases in that same full workspace suite remain green. The earlier first run had three asynchronous test-query failures; after waiting for the fetched UI, the suite passed. A later new assertion incorrectly expected a proposal checkbox when the server allowed only conversion; it was removed and the affected suite passed. `order-create.tsx` and accounting behavior were unchanged, so their separate unit suites were not rerun.

No lint, typecheck, build, React Doctor, browser, database, integration or automatic review/QA check ran. Unit HTTP fixtures do not establish live database or release readiness.

## Final source SHA-256

```text
a8904acffb3476fbb111f9a44f260c36d0211f6711bbd1824563fce68d68819c  web/apps/realization/src/reservation-workspace.tsx
449ba81d950bde35ad8d8f67e2be0f3e9e19def3a7e5aacc85b33ea953bd08ef  web/apps/realization/src/reservation-workspace.test.tsx
e3e0f9fd422f11c942707e4830f04eeb886740dc31a9544b5622cbbd29f686cf  web/apps/realization/src/data.ts
7af3ff64da42f928f298c408d0f3f5c9cf8004a22ddb8cfa8c28cd26b1779217  web/apps/realization/src/pages/purchases.tsx
```

Writer release: RELEASED. No post-release source edits are authorized by this result.
