package inventory

import "justixauto/internal/modules/inventory/model"

const (
	PermRead          = model.PermRead
	PermCatalogManage = model.PermCatalogManage
	// Deprecated: retired 2026-09-27 (only the platform admin maintains the
	// car catalog, with PermCatalogManage); no route checks it any more.
	PermModelsEdit       = "inventory.models.edit"
	PermWarehousesManage = model.PermWarehousesManage
	PermReceiptsCreate   = model.PermReceiptsCreate
	PermVehiclesMove     = model.PermVehiclesMove
)

// Permissions are registered in the identity catalog at startup.
var Permissions = model.Permissions
