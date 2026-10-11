package app

import (
	"context"
	"strconv"

	"justixauto/internal/modules/commerce"
	"justixauto/internal/modules/inventory"
)

type inventoryReservationStock interface {
	ReservableStock(context.Context, inventory.ReservableStockFilter) ([]inventory.ReservableStockVehicle, error)
}

type commerceReservationStockDiscovery struct{ inventory inventoryReservationStock }

var _ commerce.ReservationStockDiscovery = commerceReservationStockDiscovery{}

func (a commerceReservationStockDiscovery) ReservableStock(ctx context.Context, f commerce.ReservationStockFilter) ([]commerce.ReservationStockVehicle, error) {
	rows, err := a.inventory.ReservableStock(ctx, inventory.ReservableStockFilter{
		SupplierCompanyID: f.SupplierCompanyID, ModelID: f.ModelID, Limit: f.Limit, Offset: f.Offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]commerce.ReservationStockVehicle, 0, len(rows))
	for _, row := range rows {
		out = append(out, commerce.ReservationStockVehicle{
			ID: row.ID, VIN: row.VIN, ModelID: row.ModelID,
			ModelSpecificationVersion: strconv.Itoa(row.ModelSpecificationVersion),
			ExteriorColor: row.ExteriorColor, InteriorColor: row.InteriorColor,
		})
	}
	return out, nil
}
