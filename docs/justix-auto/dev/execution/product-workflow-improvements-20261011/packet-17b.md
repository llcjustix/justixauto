# P17b — Reservation VIN allocation and hold transfer

Requirements: R3. Dependencies: P17a. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/inventory/service/stock.go`
- `internal/modules/inventory/service/reservation_holder_test.go`
- `internal/modules/inventory/module.go`
- `internal/modules/inventory/repository/reservation.go`
- `internal/modules/inventory/service/ports.go`
- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/service/reservation_allocation.go`
- `internal/modules/commerce/service/reservation_allocation_test.go`
- `internal/modules/commerce/repository/reservation.go`
- `internal/modules/commerce/repository/supply.go`
- `internal/modules/commerce/handler/reservation.go`

## Behavior, interfaces and data flow

Reuse P17a exported inventory reservation operations with Holder{Type:commerce-reservation,ID} through commerce port/adapter; preserve commerce-order/retail-deal callers. For future reservation allocation, lock lot/reservation, validate VIN supplier ownership/model/spec/colors, reserve exact VIN via same bound transaction, decrement future quantity hold for allocated units and record VIN allocation once. Availability is remaining future held units plus distinct VIN holds, not two copies. Convert reservation→order by atomically reassigning existing VIN holder to commerce-order and future capacity obligation reference; no release/reacquire window, never creating duplicate inventory stock. Implement atomic holder-reassignment repository method in the explicitly owned inventory repository/reservation.go and service/ports.go. Version and stable business keys deny duplicate allocations. Refund/expiry release reuses holder-specific API.

## Acceptance and affected units

App seam unit: holder reassignment adapter preserves original reservation/order IDs and bound transaction context. Add/use a meaningful pure fake-port case named `TestWorkflowPort17` in the owned app test file. Use a small app-local interface around a module's exported service where required, never import its internal layer or start DB. Run this affected unit once: `bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort17($|_)'`. This compiles app wiring while running only the new pure case; existing OpenAPI test uses e2e.DB and must not run. If no meaningful pure seam can be tested without widening source, report that exact limitation instead of substituting a build/integration command or claiming no-tests as evidence.

Allocate partial/all VINs leaves exact total reserved units; wrong spec/color, duplicate VIN, outsider and concurrent stale revision fail. Order conversion holds same VIN exactly once; injected port failure rolls back lot decrement. Physical VIN uniqueness never weakened.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/inventory/service ./internal/modules/commerce/service`. No other development verification/runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

Clarification: P17a supplies Holder-based reserve/release. The explicit repository/reservation.go and service/ports.go ownership above authorizes the holder-reassignment seam here; no further generic approval is required.

Allocation conserves supply capacity: moving one future-held unit to a VIN hold also records one materialized/allocated supply claim. It must not increase the lot's available future quantity. Add repository/supply.go to exact ownership for this same-transaction update. Acceptance includes total=10, reserved=4, allocate 2 VINs: future unallocated held=2, VIN-held=2, available remains 6. Cancellation releases the appropriate unmaterialized claim or physical VIN, without inventing additional future supply.

