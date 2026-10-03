package service

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/jsonx"
	"justixauto/internal/pkg/validate"
)

type StockInput struct {
	Mode     string         `json:"mode"` // "unidentified" or "identified"
	Quantity jsonx.Quantity `json:"quantity"`
	VINs     []string       `json:"vins"`
}

type ReceiptInput struct {
	ModelID                   string         `json:"modelId"`
	ModelSpecificationVersion jsonx.Quantity `json:"modelSpecificationVersion"`
	ExteriorColor             string         `json:"exteriorColor"`
	InteriorColor             string         `json:"interiorColor"`
	Stock                     StockInput     `json:"stock"`
	ReceivedAt                time.Time      `json:"receivedAt"`
	EvidenceBindingIDs        []string       `json:"evidenceBindingIds"`
}

type ReceiptResult struct {
	Batch     model.ReceiptBatch
	Warehouse WarehouseView
	Vehicles  []model.VehicleUnit
}

// Receipt is the receiving and VIN-identification service.
type Receipt struct{ Deps }

// vins normalizes VINs and rejects malformed or repeated ones.
func vins(v *apperr.Validation, field string, raw []string) []string {
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		vin, ok := normalizeVIN(r)
		switch {
		case !ok:
			v.Add(field, "invalid VIN "+strings.TrimSpace(r)+": 17 characters, no I, O or Q")
		case slices.Contains(out, vin):
			v.Add(field, "VIN "+vin+" is listed twice")
		default:
			out = append(out, vin)
		}
	}
	return out
}

func (s *Receipt) checkOccurred(v *apperr.Validation, field string, at time.Time) {
	if at.IsZero() || at.After(s.clock().Add(5*time.Minute)) {
		v.Add(field, "required and not in the future")
	}
}

func copyColor(color *string) *string {
	if color == nil {
		return nil
	}
	return ptr(*color)
}

func newUnits(p *auth.Principal, vins []string, modelID string, specVersion int, exteriorColor, interiorColor *string, warehouseID, batchID string, at time.Time) ([]model.VehicleUnit, []model.Placement) {
	units := make([]model.VehicleUnit, len(vins))
	placements := make([]model.Placement, len(vins))
	for i, vin := range vins {
		units[i] = model.VehicleUnit{
			ID: uuid.NewString(), VIN: vin, ModelID: modelID, SpecVersion: specVersion,
			ExteriorColor: copyColor(exteriorColor), InteriorColor: copyColor(interiorColor),
			OwnerCompanyID: ptr(p.CompanyID), CustodianCompanyID: ptr(p.CompanyID), Version: 1, CreatedAt: at,
		}
		placements[i] = model.Placement{VehicleID: units[i].ID, WarehouseID: warehouseID, ReceiptBatchID: ptr(batchID), PlacedAt: at}
	}
	return units, placements
}

func specificationPalette(colors []string, scalar string) []string {
	if len(colors) > 0 {
		return colors
	}
	if scalar == "" {
		return nil
	}
	return []string{scalar}
}

func selectColor(v *apperr.Validation, field, value string, palette []string) string {
	value = strings.TrimSpace(value)
	if value == "" && len(palette) == 1 {
		return palette[0]
	}
	if value == "" {
		v.Add(field, "required unless the model specification has one color choice")
		return ""
	}
	for _, color := range palette {
		if strings.EqualFold(value, color) {
			return color
		}
	}
	v.Add(field, "must be a color choice of the model specification version")
	return ""
}

func resolveReceiptColors(spec *model.Specification, exteriorColor, interiorColor string) (*string, *string, error) {
	var v apperr.Validation
	exterior := selectColor(&v, "exteriorColor", exteriorColor, specificationPalette(spec.ExteriorColors, spec.ExteriorColor))
	interior := selectColor(&v, "interiorColor", interiorColor, specificationPalette(spec.InteriorColors, spec.InteriorColor))
	if err := v.Err(); err != nil {
		return nil, nil, err
	}
	return ptr(exterior), ptr(interior), nil
}

func suppliedColorMatches(field, supplied string, actual *string) error {
	if strings.TrimSpace(supplied) == "" {
		return nil
	}
	if actual == nil || !strings.EqualFold(strings.TrimSpace(supplied), *actual) {
		return apperr.FieldError(field, "must match the receipt batch color")
	}
	return nil
}

// validateReceiptStock validates the stock mode of a Receive call and
// returns the quantity to reserve and, for identified stock, the VINs.
func validateReceiptStock(v *apperr.Validation, in ReceiptInput) (quantity int, list []string) {
	switch in.Stock.Mode {
	case "unidentified":
		if in.Stock.Quantity < 1 || in.Stock.Quantity > 10_000 || len(in.Stock.VINs) > 0 {
			v.Add("stock.quantity", "must be 1-10000, without VINs")
		}
		quantity = int(in.Stock.Quantity)
	case "identified":
		list = vins(v, "stock.vins", in.Stock.VINs)
		if len(in.Stock.VINs) == 0 || len(in.Stock.VINs) > 1000 {
			v.Add("stock.vins", "list 1-1000 VINs")
		}
		quantity = len(list)
	default:
		v.Add("stock.mode", "must be unidentified or identified")
	}
	return quantity, list
}

// anyWarehouseVersion skips the revision check for receipts made by the
// system (a delivery from another company), not from a form a user had open.
const anyWarehouseVersion int64 = -1

// lockReceivingWarehouse validates the model spec, locks the warehouse at
// the expected version and checks it has room and that no VIN is already
// taken. It returns the locked warehouse and its occupancy before this
// receipt.
func lockReceivingWarehouse(ctx context.Context, st Store, p *auth.Principal, warehouseID string, expected int64, in ReceiptInput, quantity int, list []string) (*model.Warehouse, map[string]int, *string, *string, error) {
	spec, err := st.Models().Spec(ctx, in.ModelID, int(in.ModelSpecificationVersion))
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, nil, nil, nil, apperr.FieldError("modelSpecificationVersion", "unknown model or specification version")
	} else if err != nil {
		return nil, nil, nil, nil, err
	}
	exteriorColor, interiorColor, err := resolveReceiptColors(spec, in.ExteriorColor, in.InteriorColor)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	w, err := st.Warehouses().Lock(ctx, p.CompanyID, warehouseID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if expected != anyWarehouseVersion && w.Version != expected {
		return nil, nil, nil, nil, apperr.ErrStale
	}
	occ, err := st.Warehouses().Occupied(ctx, []string{w.ID})
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if occ[w.ID]+quantity > w.Capacity {
		return nil, nil, nil, nil, apperr.New(apperr.ErrConflict, "capacity_exceeded", "not enough free space in the warehouse")
	}
	if taken, err := st.Vehicles().ExistingVINs(ctx, list); err != nil {
		return nil, nil, nil, nil, err
	} else if len(taken) > 0 {
		return nil, nil, nil, nil, vinUnavailable(taken)
	}
	return w, occ, exteriorColor, interiorColor, nil
}

// Receive records vehicles arriving at a warehouse: either known VINs or a
// quantity whose VINs come later. Everything commits together or not at all,
// and the warehouse capacity is checked under a row lock.
func (s *Receipt) Receive(ctx context.Context, p *auth.Principal, warehouseID string, expected int64, in ReceiptInput) (*ReceiptResult, error) {
	if err := validate.IDs(warehouseID); err != nil {
		return nil, err
	}
	var v apperr.Validation
	if uuid.Validate(in.ModelID) != nil {
		v.Add("modelId", "must be a valid ID")
	}
	quantity, list := validateReceiptStock(&v, in)
	s.checkOccurred(&v, "receivedAt", in.ReceivedAt)
	if len(in.EvidenceBindingIDs) > 0 {
		v.Add("evidenceBindingIds", "evidence attachments are not supported yet")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	return s.receive(ctx, p, warehouseID, expected, in, quantity, list)
}

// receive stores a validated receipt: quantity vehicles, of which list are
// already known by VIN.
func (s *Receipt) receive(ctx context.Context, p *auth.Principal, warehouseID string, expected int64, in ReceiptInput, quantity int, list []string) (*ReceiptResult, error) {
	var result *ReceiptResult
	err := s.store.InTx(ctx, func(st Store) error {
		w, occ, exteriorColor, interiorColor, err := lockReceivingWarehouse(ctx, st, p, warehouseID, expected, in, quantity, list)
		if err != nil {
			return err
		}
		now := s.clock()
		received := in.ReceivedAt.UTC()
		b := &model.ReceiptBatch{
			ID: uuid.NewString(), CompanyID: p.CompanyID, WarehouseID: w.ID, ModelID: in.ModelID,
			SpecVersion: int(in.ModelSpecificationVersion), ConfirmedQuantity: quantity, IdentifiedCount: len(list),
			ExteriorColor: exteriorColor, InteriorColor: interiorColor,
			UnidentifiedCount: quantity - len(list), ReceivedAt: received, CreatedBy: p.UserID, Version: 1, CreatedAt: now,
		}
		if err := st.Warehouses().CreateBatch(ctx, b); err != nil {
			return err
		}
		units, placements := newUnits(p, list, b.ModelID, b.SpecVersion, b.ExteriorColor, b.InteriorColor, w.ID, b.ID, received)
		if err := st.Vehicles().Create(ctx, units, placements); err != nil {
			if errors.Is(err, apperr.ErrConflict) {
				return vinUnavailable(list) // a concurrent receipt registered one of them
			}
			return err
		}
		if err := st.Warehouses().Touch(ctx, w, now); err != nil {
			return err
		}
		facts := []model.Fact{s.fact(p, "receipt.recorded", nil, &w.ID, &b.ID, received, "",
			map[string]any{"mode": in.Stock.Mode, "quantity": quantity, "modelId": b.ModelID, "specVersion": b.SpecVersion})}
		for i := range units {
			facts = append(facts, s.fact(p, "vehicle.received", &units[i].ID, &w.ID, &b.ID, received, "", map[string]any{"vin": units[i].VIN}))
		}
		if err := st.Facts().Append(ctx, facts...); err != nil {
			return err
		}
		result = &ReceiptResult{Batch: *b, Warehouse: WarehouseView{Warehouse: *w, Occupied: occ[w.ID] + quantity}, Vehicles: units}
		return nil
	})
	return result, err
}

type IdentifyInput struct {
	ExteriorColor string `json:"exteriorColor"`
	InteriorColor string `json:"interiorColor"`
	Items         []struct {
		VIN     string `json:"vin"`
		ModelID string `json:"modelId"`
	} `json:"items"`
	Atomic bool `json:"atomic"`
}

// validateIdentifyItems validates an Identify call and returns the VINs to
// assign.
func validateIdentifyItems(v *apperr.Validation, in IdentifyInput) []string {
	if !in.Atomic {
		v.Add("atomic", "only atomic identification is supported")
	}
	raw := make([]string, len(in.Items))
	for i, item := range in.Items {
		raw[i] = item.VIN
	}
	list := vins(v, "items", raw)
	if len(in.Items) == 0 || len(in.Items) > 1000 {
		v.Add("items", "list 1-1000 vehicles")
	}
	return list
}

// checkItemsMatchBatch rejects an Identify call that mixes in a model
// different from the batch being identified.
func checkItemsMatchBatch(in IdentifyInput, modelID string) error {
	for _, item := range in.Items {
		if item.ModelID != modelID {
			return apperr.FieldError("items", "every vehicle must match the batch model")
		}
	}
	return nil
}

// lockIdentifyWarehouse locks the batch's warehouse and checks that none of
// the VINs being assigned is already taken.
func lockIdentifyWarehouse(ctx context.Context, st Store, p *auth.Principal, b *model.ReceiptBatch, list []string) (*model.Warehouse, error) {
	w, err := st.Warehouses().Lock(ctx, p.CompanyID, b.WarehouseID)
	if err != nil {
		return nil, err
	}
	if taken, err := st.Vehicles().ExistingVINs(ctx, list); err != nil {
		return nil, err
	} else if len(taken) > 0 {
		return nil, vinUnavailable(taken)
	}
	return w, nil
}

// Identify turns unidentified stock of a batch into vehicles with VINs, one
// unit each. Occupancy does not change. All items succeed or none do.
func (s *Receipt) Identify(ctx context.Context, p *auth.Principal, batchID string, in IdentifyInput) (*ReceiptResult, error) {
	if err := validate.IDs(batchID); err != nil {
		return nil, err
	}
	var v apperr.Validation
	list := validateIdentifyItems(&v, in)
	if err := v.Err(); err != nil {
		return nil, err
	}
	var result *ReceiptResult
	err := s.store.InTx(ctx, func(st Store) error {
		b, err := st.Warehouses().LockBatch(ctx, p.CompanyID, batchID)
		if err != nil {
			return err
		}
		if err := checkItemsMatchBatch(in, b.ModelID); err != nil {
			return err
		}
		if len(list) > b.UnidentifiedCount {
			return apperr.New(apperr.ErrConflict, "exceeds_unidentified", "the batch has fewer vehicles waiting for a VIN")
		}
		if b.ExteriorColor != nil {
			if err := suppliedColorMatches("exteriorColor", in.ExteriorColor, b.ExteriorColor); err != nil {
				return err
			}
		}
		if b.InteriorColor != nil {
			if err := suppliedColorMatches("interiorColor", in.InteriorColor, b.InteriorColor); err != nil {
				return err
			}
		}
		exteriorColor, interiorColor := b.ExteriorColor, b.InteriorColor
		if exteriorColor == nil || interiorColor == nil {
			spec, err := st.Models().Spec(ctx, b.ModelID, b.SpecVersion)
			if errors.Is(err, apperr.ErrNotFound) || (err == nil && spec == nil) {
				return apperr.FieldError("modelSpecificationVersion", "unknown model or specification version")
			} else if err != nil {
				return err
			}
			// Resolve only missing facts. Existing selections and existing VINs
			// remain historical facts, including their original spelling.
			var colors apperr.Validation
			if exteriorColor == nil {
				value := selectColor(&colors, "exteriorColor", in.ExteriorColor, specificationPalette(spec.ExteriorColors, spec.ExteriorColor))
				exteriorColor = &value
			}
			if interiorColor == nil {
				value := selectColor(&colors, "interiorColor", in.InteriorColor, specificationPalette(spec.InteriorColors, spec.InteriorColor))
				interiorColor = &value
			}
			if err := colors.Err(); err != nil {
				return err
			}
		}
		w, err := lockIdentifyWarehouse(ctx, st, p, b, list)
		if err != nil {
			return err
		}
		now := s.clock()
		units, placements := newUnits(p, list, b.ModelID, b.SpecVersion, exteriorColor, interiorColor, w.ID, b.ID, now)
		if err := st.Vehicles().Create(ctx, units, placements); err != nil {
			if errors.Is(err, apperr.ErrConflict) {
				return vinUnavailable(list)
			}
			return err
		}
		b.ExteriorColor, b.InteriorColor = exteriorColor, interiorColor
		b.IdentifiedCount += len(list)
		b.UnidentifiedCount -= len(list)
		if err := st.Warehouses().UpdateBatchCounts(ctx, b); err != nil {
			return err
		}
		if err := st.Warehouses().Touch(ctx, w, now); err != nil {
			return err
		}
		facts := make([]model.Fact, len(units))
		for i := range units {
			facts[i] = s.fact(p, "vehicle.identified", &units[i].ID, &w.ID, &b.ID, now, "", map[string]any{"vin": units[i].VIN})
		}
		if err := st.Facts().Append(ctx, facts...); err != nil {
			return err
		}
		occ, err := st.Warehouses().Occupied(ctx, []string{w.ID})
		if err != nil {
			return err
		}
		result = &ReceiptResult{Batch: *b, Warehouse: WarehouseView{Warehouse: *w, Occupied: occ[w.ID]}, Vehicles: units}
		return nil
	})
	return result, err
}

// CorrectQuantity fixes the confirmed quantity of a batch after a recount. It
// is a separate audited action (business-logic §5.7): a reason is required,
// the quantity cannot drop below the vehicles already identified, and an
// increase must fit the warehouse.
func (s *Receipt) CorrectQuantity(ctx context.Context, p *auth.Principal, batchID string, expected int64, quantity int, reason string) (*ReceiptResult, error) {
	if err := validate.IDs(batchID); err != nil {
		return nil, err
	}
	var v apperr.Validation
	if quantity < 1 || quantity > 10000 {
		v.Add("quantity", "must be 1-10000")
	}
	why := validate.Reason(&v, reason)
	if err := v.Err(); err != nil {
		return nil, err
	}
	var result *ReceiptResult
	err := s.store.InTx(ctx, func(st Store) error {
		b, err := st.Warehouses().LockBatch(ctx, p.CompanyID, batchID)
		if err != nil {
			return err
		}
		if b.Version != expected {
			return apperr.ErrStale
		}
		if quantity == b.ConfirmedQuantity {
			return apperr.FieldError("quantity", "equals the current quantity")
		}
		if quantity < b.IdentifiedCount {
			return apperr.FieldError("quantity", "cannot be below the vehicles already identified")
		}
		w, err := st.Warehouses().Lock(ctx, p.CompanyID, b.WarehouseID)
		if err != nil {
			return err
		}
		occ, err := st.Warehouses().Occupied(ctx, []string{w.ID})
		if err != nil {
			return err
		}
		delta := quantity - b.ConfirmedQuantity
		if delta > 0 && occ[w.ID]+delta > w.Capacity {
			return apperr.New(apperr.ErrConflict, "capacity_exceeded", "not enough free space in the warehouse")
		}
		previous := b.ConfirmedQuantity
		b.ConfirmedQuantity, b.UnidentifiedCount = quantity, quantity-b.IdentifiedCount
		if err := st.Warehouses().UpdateBatchCounts(ctx, b); err != nil {
			return err
		}
		now := s.clock()
		if err := st.Warehouses().Touch(ctx, w, now); err != nil {
			return err
		}
		if err := st.Facts().Append(ctx, s.fact(p, "receipt.quantity_corrected", nil, &w.ID, &b.ID, now, why,
			map[string]any{"from": previous, "to": quantity})); err != nil {
			return err
		}
		result = &ReceiptResult{Batch: *b, Warehouse: WarehouseView{Warehouse: *w, Occupied: occ[w.ID] + delta}}
		return nil
	})
	return result, err
}
