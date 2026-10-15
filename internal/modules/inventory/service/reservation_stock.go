package service

import (
	"context"

	"github.com/google/uuid"

	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/database"
)

// ReservableStockFilter is a trusted, bounded cross-module inventory read.
type ReservableStockFilter struct {
	SupplierCompanyID string
	ModelID           string
	Limit             int
	Offset            int
}

// ReservableStockVehicle exposes only the identity needed for a reservation request.
type ReservableStockVehicle struct {
	ID, VIN, ModelID             string
	ModelSpecificationVersion    int
	ExteriorColor, InteriorColor string
}

// ReservableStock reads currently eligible supplier-owned warehouse VINs.
// Availability is advisory: a later reservation action rechecks under its lock.
func (s *Stock) ReservableStock(ctx context.Context, f ReservableStockFilter) ([]ReservableStockVehicle, error) {
	var v apperr.Validation
	if uuid.Validate(f.SupplierCompanyID) != nil {
		v.Add("supplierCompanyId", "must be a valid ID")
	}
	if f.ModelID != "" && uuid.Validate(f.ModelID) != nil {
		v.Add("modelId", "must be a valid ID")
	}
	if f.Limit < 0 || f.Limit > 100 {
		v.Add("limit", "must be 1-100")
	}
	if f.Offset < 0 {
		v.Add("offset", "must be zero or greater")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	limit, offset := database.Page(f.Limit, f.Offset)
	rows, err := s.store.Vehicles().List(ctx, model.VehicleFilter{
		CompanyID: f.SupplierCompanyID, ModelID: f.ModelID,
		Placement: "warehouse", Eligible: true, Limit: limit, Offset: offset,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ReservableStockVehicle, 0, len(rows))
	for _, row := range rows {
		item := ReservableStockVehicle{
			ID: row.ID, VIN: row.VIN, ModelID: row.ModelID,
			ModelSpecificationVersion: row.SpecVersion,
		}
		if row.ExteriorColor != nil {
			item.ExteriorColor = *row.ExteriorColor
		}
		if row.InteriorColor != nil {
			item.InteriorColor = *row.InteriorColor
		}
		out = append(out, item)
	}
	return out, nil
}
