package model

import "justixauto/internal/pkg/auth"

const (
	PermRead = "inventory.read"
	// PermCatalogManage is the platform permission that maintains the shared
	// car catalog (user decision 2026-09-27: only the platform admin; the
	// former company permission inventory.models.edit is retired).
	PermCatalogManage    = "platform.catalog.manage"
	PermWarehousesManage = "inventory.warehouses.manage"
	PermReceiptsCreate   = "inventory.receipts.create"
	PermVehiclesMove     = "inventory.vehicles.move"
)

// Permissions are registered in the identity catalog at startup.
var Permissions = []auth.PermissionInfo{
	{Key: PermRead, Scope: "company", Assignable: true},
	{Key: PermWarehousesManage, Scope: "company", Assignable: true},
	{Key: PermReceiptsCreate, Scope: "company", Assignable: true},
	{Key: PermVehiclesMove, Scope: "company", Assignable: true},
}
