# P22 — Distributor return request and dealer confirmation

Requirements: R5. Dependencies: P21b. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/commerce/model/consignment_return.go`
- `internal/modules/commerce/service/consignment_return.go`
- `internal/modules/commerce/service/consignment_return_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/repository/consignment_return.go`
- `internal/modules/commerce/repository/store.go`
- `internal/modules/commerce/handler/consignment_return.go`
- `internal/modules/commerce/handler/handler.go`
- `internal/modules/commerce/module.go`
- `migrations/000045_consignment_returns.up.sql`
- `migrations/000045_consignment_returns.down.sql`

## Behavior, interfaces and data flow

Distributor requests return of its dealer-held unsold consignment vehicles, names owned destination warehouse and reason. Persist request/vehicle IDs/current agreement/version, requester/time. Only holding dealer confirms with actual movement time; confirmation rechecks owner/custody/sold/hold state and destination capacity, invokes P20 custody return within same transaction, records decision+movement reference once. Conflicting reservation blocks, never auto-release. Preserve financial obligations and payment records; return is not a refund or settlement. Wrong destination/source/party hidden/denied. No automatic return timer or obligation cancellation. If reject action offered record reason/history without physical change; no invented penalty.

## Acceptance and affected units

Distributor request cannot move stock; dealer confirmation moves custody only, records immutable decision; sold/reserved unit rejected, replay/stale no movement; another company cannot confirm; target capacity failure leaves pending request. Existing accepted money unchanged.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/commerce/service`. No other development verification/runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

