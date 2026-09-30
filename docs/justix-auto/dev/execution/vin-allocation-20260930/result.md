# VIN-EMPTY-01 result

Status: DONE

Changed owned paths:
- `web/apps/realization/src/pages/trade.tsx`: allocation dialog now explains how to obtain free VIN vehicles and throws the existing `ApiError` with a local `vehicleIds` field error before an empty/non-array selection can issue a POST.
- `web/apps/realization/src/order-allocation.test.tsx`: three mocked UI units cover empty inventory guidance, an unselected eligible vehicle, and the unchanged selected allocation request/revision/refresh path.

Verification run once after the relevant change:

```text
PATH=/Users/bakhromachilov/startups/justixauto/docs/justix-auto/dev/local/toolchains/node-24.21.0/bin:$PATH npm --prefix web run test:unit -- --project @justixauto/realization apps/realization/src/order-allocation.test.tsx
Test Files  1 passed (1)
Tests  3 passed (3)
```

Excluded by packet and latest unit-only policy: lint, typecheck, build, React Doctor, browser/integration checks, live HTTP/DB use, review/QA loops, Git writes, and all paths outside this packet. Existing unrelated `.claude` and manual-flow changes were not touched.
