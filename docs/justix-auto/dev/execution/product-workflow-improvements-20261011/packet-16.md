# P16 — Distributor future-supply availability

Requirements: R3. Dependencies: P03b/P15. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/commerce/model/supply.go`
- `internal/modules/commerce/service/supply.go`
- `internal/modules/commerce/service/supply_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/service/deps.go`
- `internal/modules/commerce/repository/supply.go`
- `internal/modules/commerce/repository/store.go`
- `internal/modules/commerce/handler/supply.go`
- `internal/modules/commerce/handler/handler.go`
- `internal/modules/commerce/module.go`
- `migrations/000041_commerce_future_supply.up.sql`
- `migrations/000041_commerce_future_supply.down.sql`

## Behavior, interfaces and data flow

Commerce owns future supply lots {supplierCompanyId,modelId,specificationVersion,exteriorColor,interiorColor,totalQuantity,expectedArrival,version}. Validate exact model version/colors via Catalog port; nullable future VIN not a fake physical unit. Verified distributor creates/updates supply, permitted dealer partner reads availability. Enforce tenant/party role via Directory.Company userType from P03b, plus existing permissions/partnership. Add repository row lock and available/reserved quantity projection used by P17; cannot reduce lot below committed amount. Numeric quantities positive exact integers. Do not spawn inventory vehicles on supply entry. HTTP list/detail/create/update supply-lots with envelopes/If-Match.

## Acceptance and affected units

Model/spec/color mismatch, invalid/overflow quantity and outsider rejected; update below committed blocked; empty/available view exact. Existing physical VIN stock unaffected.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/commerce/service`. No other development verification/runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

Explicit compatibility: existing offer quantity does not cap ordinary order quantity. This new supply-lot capacity constrains only reservations against that lot, never silently changes ordinary offer/order semantics.

Capacity ledger must distinguish unallocated held quantity, units materialized/allocated to VIN and unreserved future availability. Assigning a VIN never makes the same supply unit available for another future reservation. Provide consumed/allocated counters or durable unit claims in this supply-owned schema for P17b; do not represent all commitments as a single decrementable held counter.

