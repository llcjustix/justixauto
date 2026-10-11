# P19a result — PARTIAL / RESLICE_REQUIRED

Source-writer lease RELEASED. The four owned frontend paths contain an
incomplete draft; no source is to be changed by this worker after this report.
`bash tools/check-git.sh` passed before edits. No Git write, runtime, browser,
build, lint, typecheck, review or QA was run.

## Implemented draft seams

- `PurchasesPage` has a new modal entry while retaining existing direct-order
  and offer-order entries.
- `data.ts` adds supply lot, reservation and reservation-stock DTOs plus
  `['supply-lots']` and `['reservations']` list queries.
- The workspace uses `GET /commerce/reservation-stock?supplierCompanyId=…&modelId=…&limit=50&offset=0`, selects its returned `vehicleId`, and posts
  `POST /commerce/reservations` with `{supplierCompanyId, lines:[{vehicleId,
  quantity:'1'}], desiredTerms:{}}`; the empty desired terms deliberately omit
  unknown economics. Future selection filters the existing dealer-visible
  `/commerce/supply-lots` response by its `supplierCompanyId` and posts a
  `supplyLotId` line. Supply creation posts the final P16 field names.
- Mutation buttons apply commercial-access plus `commerce.trade`; reads remain
  available through the component queries. The draft includes supplier
  confirmation/offer acceptance display only; confirmation UI is unfinished.

## Failed affected unit evidence

The required command was run twice while repairing the same VIN-discovery test:

```text
npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/reservation-workspace.test.tsx -t 'P19a supply and agreement'
```

Both runs selected one case and failed at `Unable to find role="option" and
name "VIN-1"`. The mocked discovery response was `{items:[{vehicleId:'v',
vin:'VIN-1',…}],nextCursor:null}`; after the first failure the implementation
was changed from `get<T>` to the normal list-envelope reader, but no third run
is authorized. The response object was observed only through this failing
interaction; the underlying mapping cause is not established.

## Unfinished acceptance

U1–U4/U9–U10 are not complete: stock discovery test is red; no future request,
supply edit/capacity refresh, confirmation exact terms/If-Match, changed-offer
acceptance, accepted readonly terms, stale/error/duplicate, permission and
verification, modal keyboard/close, or Purchases-host interaction cases are
complete. A fresh bounded continuation must diagnose the discovery render
failure before expanding coverage.

## Source SHA-256

```text
156d3a353dfa39b810d9226381d59a809f1e8d759c6876d5f990a435231ba49f  web/apps/realization/src/data.ts
f3e077cb6c778173ed0b5165edd904734a1b0769f1d3551e9b685310f18c2e0d  web/apps/realization/src/reservation-workspace.tsx
ea80b00be125d07507d602160eb7810b98343d21263c29187b3d78bdf2f18149  web/apps/realization/src/reservation-workspace.test.tsx
28414c9ef849bd6f9cf70991e258d3f97639c6703da74b1cda94c20b4bbcba76  web/apps/realization/src/pages/purchases.tsx
```
