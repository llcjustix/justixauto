# BE1 compatibility scope

/root approves the additional path
`internal/modules/inventory/service/receipt_colors_test.go` solely to keep its
existing `colorWarehouseRepo` compatible with the location repository methods
called by Receive and Identify. Add the minimal in-memory location behavior;
retain current assertions and cases. No new unit tests or intermediate execution.
The BE1 handoff includes the path and resulting hash.
