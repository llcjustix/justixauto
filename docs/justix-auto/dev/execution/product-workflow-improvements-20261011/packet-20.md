# P20 — Inventory custody-only consignment port

Requirements: R3/R5. Dependencies: P17b/P19. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/inventory/service/stock.go`
- `internal/modules/inventory/service/custody.go`
- `internal/modules/inventory/service/custody_test.go`
- `internal/modules/inventory/service/ports.go`
- `internal/modules/inventory/repository/vehicle.go`
- `internal/modules/inventory/repository/reservation.go`
- `internal/modules/inventory/module.go`

## Behavior, interfaces and data flow

VehicleUnit already has distinct OwnerCompanyID/CustodianCompanyID. Add explicit exported custody handover method with owner/source-custodian/destination company+warehouse, expected holder/agreement reference, vehicles, actor/time and purpose consignment/return. Lock vehicles/placements/warehouses in stable order; same ambient transaction; capacity and live placement rechecked. Consignment keeps distributor owner, moves custody/placement; return keeps same owner, requires dealer custody and matching active consignment authorization, and rejects sold/retail-reserved/conflicting held units. Never silently release hold. No browser direct access to cross-company custody primitive: only commerce service through typed port. Append immutable stock facts visible under existing tenant policies. Existing purchase Transfer semantics unchanged.

## Acceptance and affected units

Consignment owner unchanged/custodian changes; return reverts custody; sold vehicle/conflicting reservation rejected; capacity failure atomic; repeated or source-changed move fails without extra facts; existing purchase transfer still sets both ownership/custody.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/inventory/service`. No other development verification/runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

