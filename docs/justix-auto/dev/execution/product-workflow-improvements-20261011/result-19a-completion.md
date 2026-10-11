# P19a completion — supply and exact reservation agreement

Status: SOURCE COMPLETE. Sole source-writer lease RELEASED after this report.
The work remained in the main checkout on
`feature/company-commerce-workflows-20261011` at supplied HEAD
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74`. `bash tools/check-git.sh`
passed before application edits. Existing dirty predecessor files were preserved;
`pages/purchases.tsx` already contained the P19a modal entry and was not changed
in this continuation. No Git write, build, typecheck, lint, browser, database,
integration, review or QA check ran.

## Behavior and contracts

- Distributor supply create/edit now selects a real catalog model, exact
  historical specification version and its exterior/interior palettes. It
  sends `POST /commerce/supply-lots` or `PUT /commerce/supply-lots/:id` with
  `modelId`, `modelSpecificationVersion`, `exteriorColor`,
  `interiorColor`, `totalQuantity` (decimal string), and `expectedArrival`
  (timestamp). Edit uses the exact `If-Match` revision. Identity controls
  lock after held/materialized claims, while quantity and arrival remain
  editable. The list shows available, held and materialized capacity and
  refreshes after writes.
- Dealer request chooses an active partnership, real warehouse VINs from
  `GET /commerce/reservation-stock?supplierCompanyId=…&modelId=…&limit=50&offset=…`
  (the actual `{items,nextCursor}` list envelope), or dealer-visible active
  partner supply lots from `GET /commerce/supply-lots` filtered by the
  returned `supplierCompanyId`. Multiple distinct VINs can be selected across
  stock pages. `POST /commerce/reservations` sends their actual `vehicleId`
  or a real `supplyLotId` with decimal quantities. Unknown desired economics
  are omitted as `desiredTerms:{}`; optional desired price/expiry/advance
  fields use exact integer minor units. Request sends no hold/claim action.
- Reservation detail is keyed by the selected ID and reads
  `GET /commerce/reservations/:id`. It displays buyer/supplier names and
  immutable requested, offered and current line/term snapshots with their
  server digests. It shows price, currency, expiry, optional advance and
  deadline as read-only facts once accepted. Supplier confirmation posts
  `{terms,lines:[{lineId,quantity}]}` to
  `/reservations/:id/supplier-confirmations` with the exact current
  `If-Match` revision. Buyer acceptance posts the exact offered digest to
  `/reservations/:id/acceptances` with the same revision contract. The
  server's `allowedActions`, selected company type/party, `commerce.trade`
  and commercial verification jointly gate each operation. Historic reads
  remain available when mutation access is blocked.
- Query keys for reservation lists/detail, supply lots, stock and reservation
  partner selection include the current company ID. Mutations refresh
  authoritative lists/capacity and detail. A failed or stale detail mutation
  preserves the selected reservation and refetches it; a pending mutation
  blocks duplicate submission. Loading, empty, error and retry states appear
  in the workspace. The existing Purchases sent/received tabs and direct/offer
  creation remain present alongside its reservation entry.

The first worker's failed VIN test was an asynchronous selection fixture
problem: it changed the supplier select before its active partner option was
loaded. The completion test waits for the actual partnership option, then
observes the stock list envelope and selected VIN. A separate passing case
exercises stock failure and retry. A real local-time round-trip bug in the
confirmation form was repaired so an existing UTC expiry keeps the same
instant when sent back.

## Affected unit evidence

Only the packet's named selection was run after relevant changes. The first
completion run selected 8 cases and failed 3 (UTC date display conversion and
two premature test queries). After repair, 8/8 passed. Added changed-offer and
stock-failure cases passed 10/10. The final multiple-VIN change passed 11/11:

```text
npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/reservation-workspace.test.tsx -t 'P19a supply and agreement'

Test Files  1 passed (1)
     Tests  11 passed (11)
```

Cases exercise real rendered components and the Purchases host with mocked
HTTP/session. They assert method/path/body, exact `If-Match` and digest,
visible transitions, supplier/dealer/permission/verification gates,
catalog-backed supply create/edit, two VINs, future lot request without VIN
or price, no request hold endpoint, offered versus accepted snapshots,
stock retry, stale refetch, duplicate blocking and Escape close. Unit mocks
do not establish live database, deployment or release readiness.

## Final source SHA-256

```text
ab414a014765c174abb59e4f47a0f31ced473afc5d5077e2b1dcd4f0e83544df  web/apps/realization/src/reservation-workspace.tsx
2e92940e1b3b697457d77fba3fdbce26bb501a96c607141df28ce0d52c4ef6f4  web/apps/realization/src/reservation-workspace.test.tsx
a6ce6647cde21d169be8e349399c6e00aa0b48d09efa625174155528e5cd9cca  web/apps/realization/src/data.ts
28414c9ef849bd6f9cf70991e258d3f97639c6703da74b1cda94c20b4bbcba76  web/apps/realization/src/pages/purchases.tsx
```
