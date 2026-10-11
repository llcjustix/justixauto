# P23a — consigned retail sale selector result

Status: DONE. The P23a sole source-writer lease is released. Work used
`feature/company-commerce-workflows-20261011` at local HEAD
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74`; `bash tools/check-git.sh`
confirmed this project root before source edits. No Git write, SQL/runtime,
browser/integration, build, lint, typecheck, React Doctor, review or QA ran.

The SaleForm vehicle helper now combines the unchanged owned warehouse/eligible
query with the final current-company `retail.read` projection at
`GET /commerce/consignments/retail-candidates?limit=100&offset=...`. It consumes
the List `nextCursor` through every page, deduplicates IDs, rejects nonprogress,
and fetches distinct vehicle details through
`GET /inventory/vehicle-units/{id}` with at most six concurrent requests. It
extracts `data.vehicle`, requires the requested ID and actual VIN, warehouse
placement and `reserved === false`, then merges by ID. A page, permission or
detail failure rejects the whole load. The additional stream takes no client
company or commerce permission input; offer and inventory facts alone do not
authorize a candidate. `ownerCompanyId` and `custodianCompanyId` are not used as
client authority.

Manual and offer-seeded sales now refresh this same combined stream before
submit. A vanished or reserved VIN clears selection and leaves entered customer
and price fields intact. A final server 409 is shown and reloads the selection;
the POST body is unchanged and server P21b reserve remains final authority.
Success refresh still invalidates vehicle data once, while refresh retry cannot
POST again. All form query keys use company plus context revision; a late old
context hydration cannot populate the new form.

## Affected unit evidence

From `web`, the three-suite command was run twice after the implementation.
The first run had 15/15 helper tests passing, 11/12 form tests passing (an old
immediate-POST assertion failed), and the offer suite did not collect because
of a new test syntax error. After those two corrections, the second combined
run had 15/15 helper and 12/12 form tests passing, plus 20/21 offer tests
passing; the remaining offer test assumed a canceled picker had closed. The
unchanged helper suite was repeated in that second combined run by mistake,
an extra run against the unit-once development instruction. After correcting
the picker test, only the offer suite ran. A later distributor-owned-stock test
and mounted-state error guard changed only the form suite, which then ran once:

```text
npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/sale-offer-source.test.ts src/sale-form.test.tsx src/sale-offer-form.test.tsx
  first run: helper 15 passed, form 11 passed/1 failed, offer did not collect
  second run: helper 15 passed, form 12 passed, offer 20 passed/1 failed
npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/sale-offer-form.test.tsx
  21 passed
npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/sale-form.test.tsx
  13 passed
```

The tests cover candidate pages, deduplication, the detail envelope and exact
facts, later page/detail/permission failures, manual and offer-seeded consigned
selection, vanished manual preflight, 409 reload, same-company revision race,
owned-stock dealer/distributor compatibility, unchanged sale body and refresh
retry. Unit tests do not prove live route wiring, database races or release
readiness.

## Final source SHA-256

```text
8ed082138fb2fd785e780d1da829611a3268fd577590f2502e10ae53ee32b808  web/apps/realization/src/sale-offer-source.ts
cd0dfd2fdf92cb401192491963eb88ff0973a33fc5c7a3859c3c0d6a167c43d9  web/apps/realization/src/sale-offer-source.test.ts
bddff200a6cd3e5841889976499c26193715b3380ce81d0501a1316eaa988657  web/apps/realization/src/sale-form.tsx
67640f5c434268fb339fc9ed9071c927a56f8017d33ea269271e637a71386757  web/apps/realization/src/sale-form.test.tsx
2b0160d125c4f5c16f6fb888d8d84d5e9c155b74cdcdabf7ef8fe9491bd13b4b  web/apps/realization/src/sale-offer-form.test.tsx
```
