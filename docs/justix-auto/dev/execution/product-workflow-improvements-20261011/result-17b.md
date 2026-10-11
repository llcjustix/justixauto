# P17b result

Status: RESLICE_REQUIRED.

The bounded source slice implemented the required inventory-side atomic holder
transfer seam: `Stock.Reassign` and the repository `Reassign` change only the
active holder columns in one ambient transaction. They preserve the original
inventory reservation row ID, company and creation time, reject a stale source
holder, and never release/reacquire the VIN. Commerce's typed
`ReservationStock` port, root adapter and `TestWorkflowPort17` now carry the
reservation-to-order transfer with the same bound context.

Affected units passed once with pinned Go and `-vet=off`:

- `./internal/modules/inventory/service -run '^TestReservationHolder'`
- `./internal/app -run '^TestWorkflowPort17($|_)'`

No database/runtime/Git/review/QA/build check ran. The source writer is
released. This is intentionally not a P17b completion: the actual future VIN
allocation route/service, materialization claim update, durable
current/replacement allocation history bridge, supplier model/spec/color
validation, and transaction rollback units remain unimplemented. Those pieces
cannot be claimed from the reassignment seam alone and require a corrective
packet or continued P17b source lease.
