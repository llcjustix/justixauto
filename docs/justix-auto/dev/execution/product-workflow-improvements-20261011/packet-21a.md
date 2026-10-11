# P21a — Consignment handover and agreed purchase handover

Requirements: R3. Dependencies: P18b/P20. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/commerce/model/consignment.go`
- `internal/modules/commerce/service/consignment.go`
- `internal/modules/commerce/service/consignment_test.go`
- `internal/modules/commerce/service/fulfilment.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/repository/consignment.go`
- `internal/modules/commerce/repository/store.go`
- `internal/modules/commerce/handler/consignment.go`
- `internal/modules/commerce/handler/handler.go`
- `internal/modules/commerce/handler/dto.go`
- `internal/modules/commerce/module.go`
- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`
- `migrations/000044_commerce_consignments.up.sql`
- `migrations/000044_commerce_consignments.down.sql`

## Behavior, interfaces and data flow

Persist consignment agreement/order/vehicle records and per-order accepted settlement/handover terms; supplier owner/dealer custodian are distinct. Wire Stock.Custody port via inventory root adapter. Existing allocated-VIN ship and legacy receipt paths dispatch based on accepted continuation mode. Purchase originating reservation may physically transfer before full payment only when exact mutually accepted partial-handover agreement permits; otherwise outstanding accepted debt blocks. Do not count pending payments as paid. For future quantity delivery, do not use current Deliver path that creates buyer-owned units for consignment; identify/allocate actual VINs before consignment handover or extend receipt ownership port explicitly with hub-approved scope. Original reservation supports future quantities independently; no retail sale without VIN. Keep order financial obligation after consignment custody change. Expose active consignment info port to retail in P21b.

## Acceptance and affected units

App seam unit: custody adapter preserves owner and distinct recipient custodian instead of invoking ownership Transfer. Add/use a meaningful pure fake-port case named `TestWorkflowPort21` in the owned app test file. Use a small app-local interface around a module's exported service where required, never import its internal layer or start DB. Run this affected unit once: `bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort21($|_)'`. This compiles app wiring while running only the new pure case; existing OpenAPI test uses e2e.DB and must not run. If no meaningful pure seam can be tested without widening source, report that exact limitation instead of substituting a build/integration command or claiming no-tests as evidence.

Purchase outstanding without agreement blocked; same accepted amount with explicit agreement allowed; pending evidence no paid credit. Consignment preserves supplier owner across ship/receive exactly once. Sold/returned status cannot repeat handover. Stock failure rolls back commerce transition.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/commerce/service`. No other development verification/runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

