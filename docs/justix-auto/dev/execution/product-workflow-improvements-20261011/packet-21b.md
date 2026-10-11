# P21b — Retail eligibility for dealer-held consignment

Requirements: R3/R5. Dependencies: P21a. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/commerce/service/consignment.go`
- `internal/modules/commerce/service/consignment_sale_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/module.go`
- `internal/modules/commerce/repository/consignment.go`
- `internal/modules/retail/service/ports.go`
- `internal/modules/retail/service/deps.go`
- `internal/modules/retail/service/deal.go`
- `internal/modules/retail/service/listing.go`
- `internal/modules/retail/service/consignment_test.go`
- `internal/modules/retail/module.go`
- `internal/modules/inventory/service/stock.go`
- `internal/modules/inventory/service/consignment_sale_test.go`
- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`

## Behavior, interfaces and data flow

Current retail requires Vehicle.Owned, so consignment custody alone cannot sell. Introduce narrow commerce exported consignment authorization/completion port adapted in app, providing agreement/owner/custodian/current sale eligibility for vehicle. Retail permits verified dealer's active consigned VIN only if explicitly covered by accepted consignment agreement; do not reinterpret generic warehouse presence as sale permission. Inventory hold permits authorized custodian only via trusted service call carrying checked agreement, not browser owner flag. At retail delivery mark consignment sold and record linked sale/settlement obligation within bound transaction; distributor ownership ends only actual customer delivery. Existing owner sales unchanged. Return P22 checks this same live agreement/hold state. No auto debt forgiveness or invented commission; per-order settlement terms remain.

## Acceptance and affected units

App seam unit: consignment eligibility/completion adapters use actual dealer, agreement and VIN and forward the bound transaction. Add/use a meaningful pure fake-port case named `TestWorkflowPort21` in the owned app test file. Use a small app-local interface around a module's exported service where required, never import its internal layer or start DB. Run this affected unit once: `bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort21($|_)'`. This compiles app wiring while running only the new pure case; existing OpenAPI test uses e2e.DB and must not run. If no meaningful pure seam can be tested without widening source, report that exact limitation instead of substituting a build/integration command or claiming no-tests as evidence.

Dealer can list/reserve permitted consigned VIN; unrelated custodian/owner cannot. Active return/other hold prevents sale. Sale delivery marks sold once and retains settlement linkage; cancellation releases retail hold while consignment remains. Port failure rolls back inventory/retail/commerce changes.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/retail/service ./internal/modules/commerce/service ./internal/modules/inventory/service`. No other development verification/runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

