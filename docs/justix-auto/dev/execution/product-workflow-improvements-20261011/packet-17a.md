# P17a — Reservation request and mutual fixed-term activation

Requirements: R3. Dependencies: P16. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/commerce/model/reservation.go`
- `internal/modules/commerce/service/reservation.go`
- `internal/modules/commerce/service/reservation_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/service/deps.go`
- `internal/modules/commerce/repository/reservation.go`
- `internal/modules/commerce/repository/supply.go`
- `internal/modules/commerce/repository/store.go`
- `internal/modules/commerce/handler/reservation.go`
- `internal/modules/commerce/handler/handler.go`
- `internal/modules/commerce/module.go`
- `internal/modules/inventory/service/stock.go`
- `internal/modules/inventory/service/reservation_holder_test.go`
- `internal/modules/inventory/module.go`
- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`
- `migrations/000042_commerce_reservations.up.sql`
- `migrations/000042_commerce_reservations.down.sql`

## Behavior, interfaces and data flow

Commerce reservation aggregate buyer/supplier, source stock VIN IDs OR future supply lot+quantity, desired/confirmed terms immutable revisions+digest, active hold state, expiry, optional advance/deadline, future continuation order reference. Dealer requests; request alone holds nothing. Distributor confirms quantities/price/currency/expiry/optional advance. Exact requested terms activate on supplier confirmation if stock available; changed terms require buyer acceptance of exact offered digest before activation. Active transition locks lot/reservation, increments held quantity atomically OR invokes existing Stock.Reserve with stable holder ID via bound context for VINs. Introduce reservation holder port distinct from commerce-order, never fabricate an order solely to get a holder. Use new ReserveReservation/ReleaseReservation commerce ports and the owned app/inventory root adapters to the existing Holder-aware inventory service. Signed accepted price never changed in place; later mutually accepted version only. Ensure reservation list/detail fields expose requested/offered/current versions and next actions. State/unique keys prevent double confirmation, stale acceptance, oversell.

## Acceptance and affected units

App seam unit: reservation holder adapter preserves commerce-reservation namespace and bound context. Add/use a meaningful pure fake-port case named `TestWorkflowPort17` in the owned app test file. Use a small app-local interface around a module's exported service where required, never import its internal layer or start DB. Run this affected unit once: `bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort17($|_)'`. This compiles app wiring while running only the new pure case; existing OpenAPI test uses e2e.DB and must not run. If no meaningful pure seam can be tested without widening source, report that exact limitation instead of substituting a build/integration command or claiming no-tests as evidence.

Request no hold; unchanged confirmation one hold; changed offer no hold until exact buyer acceptance; stale digest rejected; sold/held VIN and exhausted lot rejected; repeat activation never double holds; another tenant hidden. Free-by-default requires no payment.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/commerce/service ./internal/modules/inventory/service`. No other development verification/runtime actions.

Stock-holder port foundation is owned here: add commerce port methods ReserveReservation/ReleaseReservation, implemented by app adapters using inventory Holder{Type:commerce-reservation,ID}. Existing inventory Reserve/Release already accept Holder internally; export through inventory root. P17b subsequently owns reassignment for order conversion. Do not fabricate order rows or use the commerce-order holder namespace for reservation IDs. 

Stop: do not ship working future-only behavior as complete stock reservation support. Run affected inventory service units as well when changing the holder adapter.

