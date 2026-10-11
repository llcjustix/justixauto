package service

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/database"
)

type ReservationStockFilter struct {
	SupplierCompanyID string
	ModelID           string
	Limit             int
	Offset            int
}

type ReservationStockVehicle struct {
	ID, VIN, ModelID                                        string
	ModelSpecificationVersion, ExteriorColor, InteriorColor string
}

// ReservationStockDiscovery is separate from the mutation Stock port.
type ReservationStockDiscovery interface {
	ReservableStock(context.Context, ReservationStockFilter) ([]ReservationStockVehicle, error)
}

type ReservationStockView struct {
	VehicleID, VIN, SupplierCompanyID, SupplierCompanyName string
	ModelID, ModelSpecificationVersion                     string
	ExteriorColor, InteriorColor                            string
}

type ReservationStockService struct {
	Deps
	stock ReservationStockDiscovery
}

func NewReservationStockService(d Deps, stock ReservationStockDiscovery) *ReservationStockService {
	return &ReservationStockService{Deps: d, stock: stock}
}

// List authorizes the real dealer and active distributor partnership before
// crossing the inventory boundary. Reads do not require trade verification.
func (s *ReservationStockService) List(ctx context.Context, p *auth.Principal, f ReservationStockFilter) ([]ReservationStockView, error) {
	if err := p.Allow(model.PermRead); err != nil {
		return nil, err
	}
	if p.CompanyID == "" {
		return nil, apperr.New(apperr.ErrConflict, "no_active_company", "select a company first")
	}
	caller, err := s.tradingCompany(ctx, p.CompanyID, "companyId")
	if err != nil {
		return nil, err
	}
	if caller.UserType == nil || *caller.UserType != "dealer" {
		return nil, apperr.New(apperr.ErrForbidden, "dealer_required", "only a dealer can discover partner stock")
	}
	var v apperr.Validation
	if uuid.Validate(f.SupplierCompanyID) != nil {
		v.Add("supplierCompanyId", "must be a valid ID")
	} else if f.SupplierCompanyID == p.CompanyID {
		v.Add("supplierCompanyId", "must be another company")
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
	supplier, err := s.directory.Company(ctx, f.SupplierCompanyID)
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, apperr.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if supplier == nil || supplier.Kind != "seller" || !supplier.Active || supplier.UserType == nil || *supplier.UserType != "distributor" {
		return nil, apperr.ErrNotFound
	}
	active, err := s.store.Partnerships().ActiveBetween(ctx, p.CompanyID, f.SupplierCompanyID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, apperr.ErrNotFound
	}
	f.Limit, f.Offset = database.Page(f.Limit, f.Offset)
	vehicles, err := s.stock.ReservableStock(ctx, f)
	if err != nil {
		return nil, err
	}
	out := make([]ReservationStockView, 0, len(vehicles))
	for _, vehicle := range vehicles {
		out = append(out, ReservationStockView{
			VehicleID: vehicle.ID, VIN: vehicle.VIN,
			SupplierCompanyID: f.SupplierCompanyID, SupplierCompanyName: supplier.Name,
			ModelID: vehicle.ModelID, ModelSpecificationVersion: vehicle.ModelSpecificationVersion,
			ExteriorColor: vehicle.ExteriorColor, InteriorColor: vehicle.InteriorColor,
		})
	}
	return out, nil
}
