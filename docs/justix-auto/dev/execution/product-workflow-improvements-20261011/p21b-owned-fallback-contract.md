# P21b trusted absence versus denied consignment

Application hub consumes P21b3 read-only preparation. The current dealer-scoped
ConsignmentForVehicle discovery projection returns not-found to the supplier;
that response cannot establish that an owner VIN has no live consignment.
Inventory's ordinary Reserve now rejects owner without custody, but listing
eligibility still uses Owned/InWarehouse and also needs the accepted authority.

This is the original b1/b3 supplier-bypass acceptance, within existing paths.
B1 must expose a narrow trusted root sale check that distinguishes actual
absence of a live consignment VIN row (ordinary owned path may continue) from
a prepared/held row that does not authorize the acting seller (deny, no owned
fallback). LockUnitByVehicle already finds the live VIN row independent of
dealer; only the matched handed-over holding dealer can obtain consigned sale
authority. Preserve no-pending-return/current-retail rules and lock retention
through the outer mutation transaction. Propagate read failures as failures.

Do not loosen a public read, expose another company's financial data or add an
endpoint/schema/permission. The precise root API/error contract is b1's final
handoff to b3; absence must not be inferred from the existing dealer discovery
method. B3's original units include the supplier Owned bypass case. No new
check or source grant is made by this preparation clarification.
