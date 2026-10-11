# P20 ordinary supply eligibility after custody handover

Targeted preparation identified a concrete dependency of the accepted holder
lifecycle: once P20 consumes the fulfilled supply hold, ordinary supplier stock
discovery and Reserve still check owner but not custodian. They would therefore
allow the supplier to reserve the same vehicle again while it is with a dealer
under the consignment agreement. This is unfinished physical-authority coverage
within R3/R5, not a new commercial policy or a review/QA gate.

Application hub accepts the narrow repair in P20: ordinary owner reserve and
eligible supply discovery require the acting owner also to be current custodian.
P21b's separately checked trusted consignment path remains the dealer's retail
reservation authority. Preserve owner-or-placement read visibility; ordinary
same-company Move already checks both warehouses belong to the actor, allowing
dealer warehouse moves while preventing a supplier from bypassing return
confirmation. Do not alter that Move behavior.

Original P20-owned stock.go, service/ports.go and repository/vehicle.go remain
available. Add exact ownership of these narrowly relevant paths if required:

- internal/modules/inventory/service/reservation_stock.go
- internal/modules/inventory/service/reservation_stock_test.go
- internal/modules/inventory/repository/vehicle_filter_test.go

The last repository file is the existing pure query-shape unit; no database
test is authorized. Do not add a public custody flag, loosen owner filters or
expose dealer stock through supplier impersonation. Preserve final stock read
DTO, pagination and partnership authorization. No new inventory API or table.

At dispatch choose actual affected custody/reservation/discovery unit families
once after source changes, including ordinary own-custody compatibility and
rejection of a supplier attempt after consignment handover. Narrow existing
fixture repairs require named ownership if outside this list. No broad passing
suite rerun, runtime, SQL migration or independent review is implied.

Preparation evidence: /root/custody_port_implementation reported exact guards
in service/stock.go:186, service/reservation_stock.go:48 and repository/vehicle.go:85;
the ordinary Move warehouse guard is in service/vehicle.go:105. These positions
are context references and may move after implementation.
