package service

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/jsonx"
	"justixauto/internal/pkg/validate"
)

// Fulfilment manages vehicle allocation, shipment and receipt.
type Fulfilment struct{ Deps }

// AllocationItem assigns one concrete vehicle to an order line.
type AllocationItem struct {
	OrderLineID string `json:"orderLineId"`
	VehicleID   string `json:"vehicleId"`
}

// Allocate reserves concrete vehicles of the supplier for order lines. Each
// vehicle must match the line's model; a line never gets more vehicles than
// its quantity; a vehicle can be held by only one deal at a time.
func (s *Fulfilment) Allocate(ctx context.Context, p *auth.Principal, orderID string, expected int64, items []AllocationItem) (*model.Order, error) {
	if len(items) == 0 || len(items) > 1000 {
		return nil, apperr.FieldError("items", "list 1-1000 vehicles")
	}
	var o *model.Order
	err := s.store.InTx(ctx, func(st Store) error {
		var err error
		if o, err = s.supplierOrder(ctx, st, p, orderID, expected); err != nil {
			return err
		}
		if o.Status != model.OrderAccepted && o.Status != model.OrderFulfilling {
			return apperr.New(apperr.ErrConflict, "invalid_transition", "vehicles are allocated to accepted orders")
		}
		existing, err := st.Fulfilment().Allocations(ctx, o.ID)
		if err != nil {
			return err
		}
		shipped, err := st.Fulfilment().ShipmentLines(ctx, o.ID)
		if err != nil {
			return err
		}
		now := s.clock()
		add, vehicleIDs, err := s.planAllocations(ctx, st, p, o, existing, shippedByLine(shipped), items, now)
		if err != nil {
			return err
		}
		if err := s.stock.Reserve(st.Bind(ctx), p.CompanyID, o.ID, vehicleIDs); err != nil {
			return err
		}
		if err := st.Fulfilment().AddAllocations(ctx, add); err != nil {
			return err
		}
		o.UpdatedAt = now
		if err := st.Deals().UpdateOrder(ctx, o, expected); err != nil {
			return err
		}
		return s.event(ctx, st, p, "order.vehicles_allocated", "order", o.ID, "", map[string]any{"vehicleIds": vehicleIDs})
	})
	return o, err
}

// planAllocations validates each requested item against the order's lines,
// already-used quantities and the caller's stock, returning the allocations
// to insert and the vehicle IDs to reserve. It preserves the original
// per-item validation order and error precedence.
func (s *Fulfilment) planAllocations(ctx context.Context, st Store, p *auth.Principal, o *model.Order, existing []model.Allocation, used map[string]int, items []AllocationItem, now time.Time) ([]model.Allocation, []string, error) {
	for _, a := range existing {
		if a.Counts() {
			used[a.LineID]++
		}
	}
	lines := map[string]model.Line{}
	for _, l := range o.DecodeTerms().Lines {
		lines[l.LineID] = l
	}
	var v apperr.Validation
	var add []model.Allocation
	var vehicleIDs []string
	for i, it := range items {
		field := "items." + strconv.Itoa(i)
		line, ok := lines[it.OrderLineID]
		if !ok {
			v.Add(field+".orderLineId", "not a line of this order")
			continue
		}
		if slices.Contains(vehicleIDs, it.VehicleID) || slices.ContainsFunc(existing, func(a model.Allocation) bool { return a.VehicleID == it.VehicleID && a.Counts() }) {
			v.Add(field+".vehicleId", "already allocated")
			continue
		}
		vehicle, err := s.stock.Vehicle(st.Bind(ctx), p.CompanyID, it.VehicleID)
		if err != nil {
			v.Add(field+".vehicleId", "not one of your vehicles")
			continue
		}
		if vehicle.ModelID != line.ModelID {
			v.Add(field+".vehicleId", "VIN "+vehicle.VIN+" is a different model than the order line")
			continue
		}
		if (line.ExteriorColor != "" && !strings.EqualFold(line.ExteriorColor, vehicle.ExteriorColor)) ||
			(line.InteriorColor != "" && !strings.EqualFold(line.InteriorColor, vehicle.InteriorColor)) {
			v.Add(field+".vehicleId", "VIN "+vehicle.VIN+" does not match the ordered colors")
			continue
		}
		used[line.LineID]++
		if used[line.LineID] > int(line.Quantity) {
			v.Add(field+".orderLineId", "more vehicles than the ordered quantity")
		}
		vehicleIDs = append(vehicleIDs, it.VehicleID)
		add = append(add, model.Allocation{ID: uuid.NewString(), OrderID: o.ID, LineID: line.LineID, VehicleID: it.VehicleID, VIN: vehicle.VIN, Status: "allocated", CreatedAt: now})
	}
	if err := v.Err(); err != nil {
		return nil, nil, err
	}
	return add, vehicleIDs, nil
}

func (s *Fulfilment) supplierOrder(ctx context.Context, st Store, p *auth.Principal, id string, expected int64) (*model.Order, error) {
	if err := validate.IDs(id); err != nil {
		return nil, err
	}
	o, err := st.Deals().Order(ctx, p.CompanyID, id)
	if err != nil {
		return nil, err
	}
	if o.Version != expected {
		return nil, apperr.ErrStale
	}
	if o.Party(p.CompanyID) != "supplier" {
		return nil, apperr.New(apperr.ErrForbidden, "wrong_party", "only the supplier can do this")
	}
	return o, nil
}

// ShipmentLineInput ships a quantity of one order line straight into the
// buyer's receiving warehouse; VINs are optional and may cover only a part.
type ShipmentLineInput struct {
	OrderLineID               string         `json:"orderLineId"`
	Quantity                  jsonx.Quantity `json:"quantity"`
	VINs                      []string       `json:"vins"`
	ModelSpecificationVersion string         `json:"modelSpecificationVersion,omitempty"`
	ExteriorColor             string         `json:"exteriorColor,omitempty"`
	InteriorColor             string         `json:"interiorColor,omitempty"`
}

// ShipmentInput ships either allocated vehicles (VehicleIDs) or quantities
// of order lines (Lines), never both.
type ShipmentInput struct {
	VehicleIDs []string            `json:"vehicleIds"`
	Lines      []ShipmentLineInput `json:"lines"`
	Route      string              `json:"route"`
}

// shippedByLine sums what was shipped by quantity per order line.
func shippedByLine(ls []model.ShipmentLine) map[string]int {
	out := map[string]int{}
	for _, l := range ls {
		out[l.LineID] += l.Quantity
	}
	return out
}

// buyerWarehouseErr turns the receiving warehouse's capacity error into the
// supplier-facing reason a shipment cannot go out.
func buyerWarehouseErr(err error) error {
	var ae *apperr.Error
	if errors.As(err, &ae) && ae.Code == "capacity_exceeded" {
		return errBuyerWarehouseFull
	}
	return err
}

type plannedLine struct {
	line     model.Line
	quantity int
	vins     []string
}

// planShipmentLines validates quantities to ship against the order's lines.
// taken holds what each line already has shipped or allocated; it is updated
// with the planned quantities.
func planShipmentLines(o *model.Order, taken map[string]int, in []ShipmentLineInput) ([]plannedLine, error) {
	lines := map[string]model.Line{}
	for _, l := range o.DecodeTerms().Lines {
		lines[l.LineID] = l
	}
	var v apperr.Validation
	var plan []plannedLine
	seen := map[string]bool{}
	for i, it := range in {
		field := "lines." + strconv.Itoa(i)
		line, ok := lines[it.OrderLineID]
		switch {
		case !ok:
			v.Add(field+".orderLineId", "not a line of this order")
		case seen[it.OrderLineID]:
			v.Add(field+".orderLineId", "listed twice")
		case it.Quantity < 1:
			v.Add(field+".quantity", "must be at least 1")
		case taken[line.LineID]+int(it.Quantity) > int(line.Quantity):
			v.Add(field+".quantity", "more than remains to ship: "+strconv.Itoa(int(line.Quantity)-taken[line.LineID]))
		case len(it.VINs) > int(it.Quantity):
			v.Add(field+".vins", "more VINs than vehicles")
		default:
			// Only the plan is completed; the stored historical order stays unchanged.
			line.ModelSpecificationVersion = shipmentFact(&v, field+".modelSpecificationVersion", line.ModelSpecificationVersion, it.ModelSpecificationVersion, false)
			line.ExteriorColor = shipmentFact(&v, field+".exteriorColor", line.ExteriorColor, it.ExteriorColor, true)
			line.InteriorColor = shipmentFact(&v, field+".interiorColor", line.InteriorColor, it.InteriorColor, true)
			taken[line.LineID] += int(it.Quantity)
			plan = append(plan, plannedLine{line: line, quantity: int(it.Quantity), vins: it.VINs})
		}
		seen[it.OrderLineID] = true
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	return plan, nil
}

func shipmentFact(v *apperr.Validation, field, saved, supplied string, color bool) string {
	supplied = strings.TrimSpace(supplied)
	if saved == "" {
		return supplied
	}
	if supplied != "" && supplied != saved && !(color && strings.EqualFold(saved, supplied)) {
		v.Add(field, "must match the ordered selection")
	}
	return saved
}

// Incoming stock must have an explicit pin, including for historical orders.
// Color omissions can resolve only from that version's singleton palettes.
func (s *Fulfilment) resolveShipmentColors(ctx context.Context, plan []plannedLine) error {
	var v apperr.Validation
	models := map[string]*Model{}
	for i := range plan {
		field := "lines." + strconv.Itoa(i)
		if plan[i].line.ModelSpecificationVersion == "" {
			v.Add(field+".modelSpecificationVersion", "choose the incoming specification version for this legacy order line")
			continue
		}
		plan[i].line = s.Deps.selectColors(ctx, &v, field, plan[i].line, true, models)
	}
	return v.Err()
}

// shipQuantity ships quantities of order lines without allocated vehicles
// (user decision 2026-10-01): the supplier needs nothing in its own stock,
// the vehicles enter the buyer's receiving warehouse as receipt batches and
// the buyer enters missing VINs there. A full warehouse blocks the shipment.
func (s *Fulfilment) shipQuantity(ctx context.Context, p *auth.Principal, orderID string, expected int64, in ShipmentInput) (*model.Shipment, error) {
	var v apperr.Validation
	if len(in.VehicleIDs) > 0 {
		v.Add("vehicleIds", "ship either allocated vehicles or quantities")
	}
	if !slices.Contains(model.Routes, in.Route) {
		v.Add("route", "must be factory, foreign-direct, in-transit or local")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	var sh *model.Shipment
	err := s.store.InTx(ctx, func(st Store) error {
		o, err := s.supplierOrder(ctx, st, p, orderID, expected)
		if err != nil {
			return err
		}
		if o.Status != model.OrderAccepted && o.Status != model.OrderFulfilling {
			return apperr.New(apperr.ErrConflict, "invalid_transition", "only accepted orders can be shipped")
		}
		if o.ReceivingWarehouseID == nil {
			return apperr.New(apperr.ErrConflict, "no_receiving_warehouse", "the order has no receiving warehouse: allocate vehicles and ship them")
		}
		allocs, err := st.Fulfilment().Allocations(ctx, o.ID)
		if err != nil {
			return err
		}
		shipped, err := st.Fulfilment().ShipmentLines(ctx, o.ID)
		if err != nil {
			return err
		}
		taken, delivered := shippedByLine(shipped), shippedByLine(shipped)
		for _, a := range allocs {
			if a.Counts() {
				taken[a.LineID]++
			}
			if a.Status == "delivered" {
				delivered[a.LineID]++
			}
		}
		plan, err := planShipmentLines(o, taken, in.Lines)
		if err != nil {
			return err
		}
		if err := s.resolveShipmentColors(st.Bind(ctx), plan); err != nil {
			return err
		}
		now := s.clock()
		sh = &model.Shipment{
			ID: uuid.NewString(), OrderID: o.ID, Route: in.Route, Status: "received", Version: 1,
			CreatedBy: p.UserID, CreatedAt: now, UpdatedAt: now,
		}
		if err := st.Fulfilment().CreateShipment(ctx, sh); err != nil {
			return err
		}
		rows := make([]model.ShipmentLine, 0, len(plan))
		details := make([]map[string]any, 0, len(plan))
		for _, l := range plan {
			batchID, err := s.stock.Deliver(st.Bind(ctx), Delivery{
				ToCompanyID: o.BuyerCompanyID, ToWarehouseID: *o.ReceivingWarehouseID, ModelID: l.line.ModelID,
				ModelSpecificationVersion: l.line.ModelSpecificationVersion, ExteriorColor: l.line.ExteriorColor, InteriorColor: l.line.InteriorColor,
				Quantity: l.quantity, VINs: l.vins, ActorUserID: p.UserID, At: now,
			})
			if err != nil {
				return buyerWarehouseErr(err)
			}
			delivered[l.line.LineID] += l.quantity
			rows = append(rows, model.ShipmentLine{
				ID: uuid.NewString(), ShipmentID: sh.ID, OrderID: o.ID, LineID: l.line.LineID,
				Quantity: l.quantity, ReceiptBatchID: batchID, CreatedAt: now,
			})
			details = append(details, map[string]any{"orderLineId": l.line.LineID, "quantity": l.quantity, "vins": len(l.vins)})
		}
		if err := st.Fulfilment().AddShipmentLines(ctx, rows); err != nil {
			return err
		}
		o.Status, o.UpdatedAt = model.OrderFulfilling, now
		if orderComplete(o, delivered) {
			o.Status = model.OrderCompleted
		}
		if err := st.Deals().UpdateOrder(ctx, o, expected); err != nil {
			return err
		}
		return s.event(ctx, st, p, "order.shipped_delivered", "order", o.ID, "", map[string]any{"shipmentId": sh.ID, "lines": details})
	})
	return sh, err
}

// errBuyerWarehouseFull tells the supplier a shipment cannot go out yet.
var errBuyerWarehouseFull = apperr.New(apperr.ErrConflict, "buyer_warehouse_full", "the buyer's receiving warehouse has not enough free space")

// deliverOnShipment hands shipped vehicles straight to the buyer's receiving
// warehouse (user decision 2026-10-01): they become the buyer's at once and
// the shipment needs no receipt. A full warehouse blocks the shipment.
func (s *Fulfilment) deliverOnShipment(ctx context.Context, st Store, p *auth.Principal, o *model.Order, sh *model.Shipment, allocs []model.Allocation, shipped map[string]int, ids []string, now time.Time) error {
	err := s.stock.Transfer(st.Bind(ctx), o.ID, ids, o.BuyerCompanyID, *o.ReceivingWarehouseID, p.UserID, now)
	if err != nil {
		return buyerWarehouseErr(err)
	}
	sh.Status = "received"
	isShipped := func(a model.Allocation) bool { return a.Status == "allocated" && slices.Contains(ids, a.VehicleID) }
	_, delivered := receiptProgress(allocs, ids, isShipped, "delivered")
	for line, n := range shipped {
		delivered[line] += n
	}
	if orderComplete(o, delivered) {
		o.Status = model.OrderCompleted
	}
	return nil
}

// Ship sends allocated vehicles; only concrete, allocated VINs can be shipped.
// With a receiving warehouse on the order the vehicles are delivered at once.
func (s *Fulfilment) Ship(ctx context.Context, p *auth.Principal, orderID string, expected int64, in ShipmentInput) (*model.Shipment, error) {
	if len(in.Lines) > 0 {
		return s.shipQuantity(ctx, p, orderID, expected, in)
	}
	var v apperr.Validation
	ids := validate.UniqueIDs(&v, "vehicleIds", in.VehicleIDs)
	if len(ids) == 0 {
		v.Add("vehicleIds", "list the vehicles to ship")
	}
	if !slices.Contains(model.Routes, in.Route) {
		v.Add("route", "must be factory, foreign-direct, in-transit or local")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	var sh *model.Shipment
	err := s.store.InTx(ctx, func(st Store) error {
		o, err := s.supplierOrder(ctx, st, p, orderID, expected)
		if err != nil {
			return err
		}
		if o.Status != model.OrderAccepted && o.Status != model.OrderFulfilling {
			return apperr.New(apperr.ErrConflict, "invalid_transition", "only accepted orders can be shipped")
		}
		allocs, err := st.Fulfilment().Allocations(ctx, o.ID)
		if err != nil {
			return err
		}
		for _, id := range ids {
			if !slices.ContainsFunc(allocs, func(a model.Allocation) bool { return a.VehicleID == id && a.Status == "allocated" }) {
				return apperr.FieldError("vehicleIds", "only vehicles allocated to this order and not yet shipped")
			}
		}
		now := s.clock()
		sh = &model.Shipment{
			ID: uuid.NewString(), OrderID: o.ID, Route: in.Route, Status: "in-transit", Version: 1,
			CreatedBy: p.UserID, CreatedAt: now, UpdatedAt: now,
		}
		o.Status, o.UpdatedAt = model.OrderFulfilling, now
		allocStatus, event := "shipped", "order.shipped"
		if o.ReceivingWarehouseID != nil {
			shipped, err := st.Fulfilment().ShipmentLines(ctx, o.ID)
			if err != nil {
				return err
			}
			if err := s.deliverOnShipment(ctx, st, p, o, sh, allocs, shippedByLine(shipped), ids, now); err != nil {
				return err
			}
			allocStatus, event = "delivered", "order.shipped_delivered"
		}
		if err := st.Fulfilment().CreateShipment(ctx, sh); err != nil {
			return err
		}
		if err := st.Fulfilment().SetAllocationStatus(ctx, o.ID, ids, allocStatus, &sh.ID); err != nil {
			return err
		}
		if err := st.Deals().UpdateOrder(ctx, o, expected); err != nil {
			return err
		}
		return s.event(ctx, st, p, event, "order", o.ID, "", map[string]any{"shipmentId": sh.ID, "vehicleIds": ids})
	})
	return sh, err
}

// shipment returns a shipment of an order the company takes part in.
func (s *Fulfilment) shipment(ctx context.Context, st Store, p *auth.Principal, id string) (*model.Shipment, *model.Order, error) {
	if err := validate.IDs(id); err != nil {
		return nil, nil, err
	}
	sh, err := st.Fulfilment().Shipment(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	o, err := st.Deals().Order(ctx, p.CompanyID, sh.OrderID)
	if err != nil {
		return nil, nil, err
	}
	return sh, o, nil
}

type MilestoneInput struct {
	MilestoneType string    `json:"milestoneType"`
	OccurredAt    time.Time `json:"occurredAt"`
	Location      string    `json:"location"`
	Note          string    `json:"note"`
}

// AddMilestone records a route fact (who, when, where). It changes no state:
// receipt is a separate decision by the buyer.
func (s *Fulfilment) AddMilestone(ctx context.Context, p *auth.Principal, shipmentID string, in MilestoneInput) (*ShipmentView, error) {
	var v apperr.Validation
	if !slices.Contains(model.MilestoneTypes, in.MilestoneType) {
		v.Add("milestoneType", "unknown milestone")
	}
	if in.OccurredAt.IsZero() || in.OccurredAt.After(s.clock().Add(5*time.Minute)) {
		v.Add("occurredAt", "required and not in the future")
	}
	location := validate.Text(&v, "location", in.Location, 1, 300)
	note := validate.Text(&v, "note", in.Note, 0, 2000)
	if in.MilestoneType == "damage-reported" && note == "" {
		v.Add("note", "describe the damage")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	err := s.store.InTx(ctx, func(st Store) error {
		sh, _, err := s.shipment(ctx, st, p, shipmentID)
		if err != nil {
			return err
		}
		if sh.Status != "in-transit" && in.MilestoneType != "damage-reported" {
			return apperr.New(apperr.ErrConflict, "shipment_received", "the shipment was already received")
		}
		m := &model.Milestone{
			ID: uuid.NewString(), ShipmentID: sh.ID, MilestoneType: in.MilestoneType, OccurredAt: in.OccurredAt.UTC(),
			Location: location, Note: note, RecordedBy: p.UserID, CompanyID: p.CompanyID, RecordedAt: s.clock(),
		}
		if err := st.Fulfilment().AddMilestone(ctx, m); err != nil {
			return err
		}
		return s.event(ctx, st, p, "shipment."+in.MilestoneType, "order", sh.OrderID, note, map[string]any{"shipmentId": sh.ID, "location": location})
	})
	if err != nil {
		return nil, err
	}
	return s.GetShipment(ctx, p, shipmentID)
}

type ReceiptDecision struct {
	Decision    string   `json:"decision"` // accept | reject
	VehicleIDs  []string `json:"vehicleIds"`
	WarehouseID string   `json:"warehouseId"`
	Reason      string   `json:"reason"`
}

// DecideReceipt is the buyer's receipt of shipped vehicles. Accepted vehicles
// become the buyer's, placed in its warehouse (capacity checked). Rejected
// vehicles need a reason and go back to the supplier's free stock. When every
// line is delivered, the order is completed.
func (s *Fulfilment) DecideReceipt(ctx context.Context, p *auth.Principal, shipmentID string, expected int64, in ReceiptDecision) (*ShipmentView, error) {
	var v apperr.Validation
	ids := validate.UniqueIDs(&v, "vehicleIds", in.VehicleIDs)
	if len(ids) == 0 {
		v.Add("vehicleIds", "list the vehicles")
	}
	switch in.Decision {
	case "accept":
		if uuid.Validate(in.WarehouseID) != nil {
			v.Add("warehouseId", "choose the receiving warehouse")
		}
	case "reject":
		in.Reason = validate.Reason(&v, in.Reason)
	default:
		v.Add("decision", "must be accept or reject")
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	err := s.store.InTx(ctx, func(st Store) error {
		sh, o, err := s.shipment(ctx, st, p, shipmentID)
		if err != nil {
			return err
		}
		if o.Party(p.CompanyID) != "buyer" {
			return apperr.New(apperr.ErrForbidden, "wrong_party", "only the buyer receives a shipment")
		}
		if sh.Version != expected {
			return apperr.ErrStale
		}
		allocs, err := st.Fulfilment().Allocations(ctx, o.ID)
		if err != nil {
			return err
		}
		inShipment := func(a model.Allocation) bool {
			return a.ShipmentID != nil && *a.ShipmentID == sh.ID && a.Status == "shipped"
		}
		if err := requireShipmentVehicles(allocs, ids, inShipment); err != nil {
			return err
		}
		now := s.clock()
		status, err := s.applyReceiptDecision(st.Bind(ctx), o.ID, ids, p, in, now)
		if err != nil {
			return err
		}
		if err := st.Fulfilment().SetAllocationStatus(ctx, o.ID, ids, status, nil); err != nil {
			return err
		}
		pending, delivered := receiptProgress(allocs, ids, inShipment, status)
		if pending == 0 {
			sh.Status, sh.UpdatedAt = "received", now
		}
		sh.UpdatedAt = now
		if err := st.Fulfilment().UpdateShipment(ctx, sh, expected); err != nil {
			return err
		}
		if orderComplete(o, delivered) {
			o.Status = model.OrderCompleted
		}
		o.UpdatedAt = now
		if err := st.Deals().UpdateOrder(ctx, o, o.Version); err != nil {
			return err
		}
		return s.event(ctx, st, p, "shipment.receipt_"+in.Decision+"ed", "order", o.ID, in.Reason,
			map[string]any{"shipmentId": sh.ID, "vehicleIds": ids, "warehouseId": in.WarehouseID})
	})
	if err != nil {
		return nil, err
	}
	return s.GetShipment(ctx, p, shipmentID)
}

// requireShipmentVehicles checks that every id in ids is an allocation of
// this shipment awaiting receipt.
func requireShipmentVehicles(allocs []model.Allocation, ids []string, inShipment func(model.Allocation) bool) error {
	for _, id := range ids {
		if !slices.ContainsFunc(allocs, func(a model.Allocation) bool { return a.VehicleID == id && inShipment(a) }) {
			return apperr.FieldError("vehicleIds", "only vehicles of this shipment awaiting receipt")
		}
	}
	return nil
}

// applyReceiptDecision moves the vehicles into the buyer's stock (accept) or
// back to the supplier's free stock (reject), returning the resulting
// allocation status.
func (s *Fulfilment) applyReceiptDecision(txctx context.Context, orderID string, ids []string, p *auth.Principal, in ReceiptDecision, now time.Time) (string, error) {
	if in.Decision == "accept" {
		if err := s.stock.Transfer(txctx, orderID, ids, p.CompanyID, in.WarehouseID, p.UserID, now); err != nil {
			return "", err
		}
		return "delivered", nil
	}
	if err := s.stock.Release(txctx, orderID, ids, "rejected at receipt: "+in.Reason); err != nil {
		return "", err
	}
	return "rejected", nil
}

// receiptProgress reports how many shipment vehicles still await a decision
// and how many vehicles are now delivered per order line. A shipment is
// received once nothing in it awaits a decision.
func receiptProgress(allocs []model.Allocation, ids []string, inShipment func(model.Allocation) bool, status string) (pending int, delivered map[string]int) {
	delivered = map[string]int{}
	for _, a := range allocs {
		decided := inShipment(a) && slices.Contains(ids, a.VehicleID)
		if inShipment(a) && !decided {
			pending++
		}
		if a.Status == "delivered" || (decided && status == "delivered") {
			delivered[a.LineID]++
		}
	}
	return pending, delivered
}

// orderComplete reports whether every line of the order has reached its
// ordered quantity of delivered vehicles.
func orderComplete(o *model.Order, delivered map[string]int) bool {
	for _, l := range o.DecodeTerms().Lines {
		if delivered[l.LineID] < int(l.Quantity) {
			return false
		}
	}
	return true
}

type ShipmentView struct {
	Shipment   model.Shipment
	Milestones []model.Milestone
	Vehicles   []model.Allocation
}

func (s *Fulfilment) GetShipment(ctx context.Context, p *auth.Principal, id string) (*ShipmentView, error) {
	sh, _, err := s.shipment(ctx, s.store, p, id)
	if err != nil {
		return nil, err
	}
	return s.shipmentView(ctx, sh)
}

func (s *Fulfilment) shipmentView(ctx context.Context, sh *model.Shipment) (*ShipmentView, error) {
	ms, err := s.store.Fulfilment().Milestones(ctx, sh.ID)
	if err != nil {
		return nil, err
	}
	allocs, err := s.store.Fulfilment().Allocations(ctx, sh.OrderID)
	if err != nil {
		return nil, err
	}
	var vs []model.Allocation
	for _, a := range allocs {
		if a.ShipmentID != nil && *a.ShipmentID == sh.ID {
			vs = append(vs, a)
		}
	}
	return &ShipmentView{Shipment: *sh, Milestones: ms, Vehicles: vs}, nil
}
