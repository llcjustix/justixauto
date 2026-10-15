package service

import (
	"context"
	"math"
	"slices"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/jsonx"
	"justixauto/internal/pkg/validate"
)

// Supply manages future capacity declared by verified distributors.
type Supply struct{ Deps }

type SupplyInput struct {
	ModelID                   string         `json:"modelId"`
	ModelSpecificationVersion string         `json:"modelSpecificationVersion"`
	ExteriorColor             string         `json:"exteriorColor"`
	InteriorColor             string         `json:"interiorColor"`
	TotalQuantity             jsonx.Quantity `json:"totalQuantity"`
	ExpectedArrival           *time.Time     `json:"expectedArrival"`
}

type SupplyView struct {
	Lot      model.SupplyLot
	Capacity model.SupplyCapacity
}

func (s *Supply) distributor(ctx context.Context, p *auth.Principal) error {
	if err := p.AllowNewCommercialOperation(); err != nil {
		return err
	}
	c, err := s.tradingCompany(ctx, p.CompanyID, "companyId")
	if err != nil {
		return err
	}
	if c.UserType == nil || *c.UserType != "distributor" {
		return apperr.New(apperr.ErrForbidden, "distributor_required", "only a verified distributor can manage future supply")
	}
	return nil
}

func (s *Supply) lotInput(ctx context.Context, in SupplyInput) (SupplyInput, error) {
	var v apperr.Validation
	in.ModelID = validate.Text(&v, "modelId", in.ModelID, 1, 100)
	in.ModelSpecificationVersion = validate.Text(&v, "modelSpecificationVersion", in.ModelSpecificationVersion, 1, 100)
	in.ExteriorColor = validate.Text(&v, "exteriorColor", in.ExteriorColor, 1, 100)
	in.InteriorColor = validate.Text(&v, "interiorColor", in.InteriorColor, 1, 100)
	if in.TotalQuantity <= 0 || in.TotalQuantity > math.MaxInt32 {
		v.Add("totalQuantity", "must be a positive whole quantity within the supported range")
	}
	if err := v.Err(); err != nil {
		return in, err
	}
	m, err := s.catalog.Model(ctx, in.ModelID)
	if err != nil {
		return in, apperr.FieldError("modelId", "model not found")
	}
	for _, spec := range m.Specifications {
		if spec.Version == in.ModelSpecificationVersion {
			if !slices.Contains(spec.ExteriorColors, in.ExteriorColor) {
				v.Add("exteriorColor", "is not available for this model specification")
			}
			if !slices.Contains(spec.InteriorColors, in.InteriorColor) {
				v.Add("interiorColor", "is not available for this model specification")
			}
			break
		}
	}
	if !slices.ContainsFunc(m.Specifications, func(spec ModelSpecification) bool { return spec.Version == in.ModelSpecificationVersion }) {
		v.Add("modelSpecificationVersion", "is not available for this model")
	}
	return in, v.Err()
}

func (s *Supply) view(ctx context.Context, lot *model.SupplyLot) (*SupplyView, error) {
	capacity, err := s.store.Supply().Capacity(ctx, lot.ID)
	if err != nil {
		return nil, err
	}
	return &SupplyView{Lot: *lot, Capacity: capacity}, nil
}

func (s *Supply) Create(ctx context.Context, p *auth.Principal, in SupplyInput) (*SupplyView, error) {
	if err := s.distributor(ctx, p); err != nil {
		return nil, err
	}
	in, err := s.lotInput(ctx, in)
	if err != nil {
		return nil, err
	}
	now := s.clock()
	lot := &model.SupplyLot{ID: uuid.NewString(), SupplierCompanyID: p.CompanyID, ModelID: in.ModelID, ModelSpecificationVersion: in.ModelSpecificationVersion, ExteriorColor: in.ExteriorColor, InteriorColor: in.InteriorColor, TotalQuantity: int(in.TotalQuantity), ExpectedArrival: in.ExpectedArrival, Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := s.store.InTx(ctx, func(st Store) error {
		if err := st.Supply().Create(ctx, lot); err != nil {
			return err
		}
		return s.event(ctx, st, p, "supply.created", "supply-lot", lot.ID, "", nil)
	}); err != nil {
		return nil, err
	}
	return &SupplyView{Lot: *lot, Capacity: model.SupplyCapacity{Available: lot.TotalQuantity}}, nil
}

func (s *Supply) Get(ctx context.Context, p *auth.Principal, id string) (*SupplyView, error) {
	if err := validate.IDs(id); err != nil {
		return nil, err
	}
	lot, err := s.store.Supply().Lot(ctx, id)
	if err != nil {
		return nil, err
	}
	if lot.SupplierCompanyID != p.CompanyID {
		active, err := s.store.Partnerships().ActiveBetween(ctx, p.CompanyID, lot.SupplierCompanyID)
		if err != nil {
			return nil, err
		}
		if !active {
			return nil, apperr.ErrNotFound
		}
		c, err := s.directory.Company(ctx, p.CompanyID)
		if err != nil {
			return nil, err
		}
		if c.UserType == nil || *c.UserType != "dealer" {
			return nil, apperr.ErrNotFound
		}
	}
	return s.view(ctx, lot)
}

func (s *Supply) List(ctx context.Context, p *auth.Principal, limit, offset int) ([]SupplyView, error) {
	c, err := s.tradingCompany(ctx, p.CompanyID, "companyId")
	if err != nil {
		return nil, err
	}
	var lots []model.SupplyLot
	switch {
	case c.UserType != nil && *c.UserType == "distributor":
		lots, err = s.store.Supply().ListSupplier(ctx, p.CompanyID, limit, offset)
	case c.UserType != nil && *c.UserType == "dealer":
		lots, err = s.store.Supply().ListVisible(ctx, p.CompanyID, limit, offset)
	default:
		return nil, apperr.New(apperr.ErrForbidden, "company_type_required", "only a dealer or distributor can view future supply")
	}
	if err != nil {
		return nil, err
	}
	out := make([]SupplyView, 0, len(lots))
	for i := range lots {
		v, err := s.view(ctx, &lots[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, nil
}

func (s *Supply) Update(ctx context.Context, p *auth.Principal, id string, expected int64, in SupplyInput) (*SupplyView, error) {
	if err := validate.IDs(id); err != nil {
		return nil, err
	}
	if err := s.distributor(ctx, p); err != nil {
		return nil, err
	}
	in, err := s.lotInput(ctx, in)
	if err != nil {
		return nil, err
	}
	var result *SupplyView
	err = s.store.InTx(ctx, func(st Store) error {
		lot, err := st.Supply().LockLot(ctx, id)
		if err != nil {
			return err
		}
		if lot.SupplierCompanyID != p.CompanyID {
			return apperr.ErrNotFound
		}
		if lot.Version != expected {
			return apperr.ErrStale
		}
		capacity, err := st.Supply().Capacity(ctx, lot.ID)
		if err != nil {
			return err
		}
		committed := capacity.Held + capacity.Materialized
		if int(in.TotalQuantity) < committed {
			return apperr.New(apperr.ErrConflict, "supply_committed", "total quantity cannot be reduced below committed future supply")
		}
		if committed > 0 && (lot.ModelID != in.ModelID || lot.ModelSpecificationVersion != in.ModelSpecificationVersion || lot.ExteriorColor != in.ExteriorColor || lot.InteriorColor != in.InteriorColor) {
			return apperr.New(apperr.ErrConflict, "supply_committed_identity", "model, specification and colors cannot change after future supply is committed")
		}
		lot.ModelID, lot.ModelSpecificationVersion, lot.ExteriorColor, lot.InteriorColor = in.ModelID, in.ModelSpecificationVersion, in.ExteriorColor, in.InteriorColor
		lot.TotalQuantity, lot.ExpectedArrival, lot.UpdatedAt = int(in.TotalQuantity), in.ExpectedArrival, s.clock()
		if err := st.Supply().Update(ctx, lot, expected); err != nil {
			return err
		}
		result = &SupplyView{Lot: *lot, Capacity: model.SupplyCapacity{Available: lot.TotalQuantity - committed, Held: capacity.Held, Materialized: capacity.Materialized}}
		return s.event(ctx, st, p, "supply.updated", "supply-lot", lot.ID, "", nil)
	})
	return result, err
}
