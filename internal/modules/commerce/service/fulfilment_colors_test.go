package service

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/auth"
)

type fulfilmentColorsStock struct {
	Stock
	vehicle    StockVehicle
	reserved   int
	deliveries []Delivery
}

func (s *fulfilmentColorsStock) Vehicle(context.Context, string, string) (*StockVehicle, error) {
	return &s.vehicle, nil
}
func (s *fulfilmentColorsStock) Reserve(context.Context, string, string, []string) error {
	s.reserved++
	return nil
}
func (s *fulfilmentColorsStock) Deliver(_ context.Context, d Delivery) (string, error) {
	s.deliveries = append(s.deliveries, d)
	return uuid.NewString(), nil
}

type fulfilmentColorsRepo struct {
	FulfilmentRepository
	allocations []model.Allocation
	lines       []model.ShipmentLine
	shipments   int
}

func (r *fulfilmentColorsRepo) Allocations(context.Context, string) ([]model.Allocation, error) {
	return r.allocations, nil
}
func (r *fulfilmentColorsRepo) ShipmentLines(context.Context, string) ([]model.ShipmentLine, error) {
	return r.lines, nil
}
func (r *fulfilmentColorsRepo) AddAllocations(_ context.Context, rows []model.Allocation) error {
	r.allocations = append(r.allocations, rows...)
	return nil
}
func (r *fulfilmentColorsRepo) CreateShipment(context.Context, *model.Shipment) error {
	r.shipments++
	return nil
}
func (r *fulfilmentColorsRepo) AddShipmentLines(_ context.Context, rows []model.ShipmentLine) error {
	r.lines = append(r.lines, rows...)
	return nil
}

type fulfilmentColorsStore struct {
	*colorsStore
	fulfilment *fulfilmentColorsRepo
}

func (s *fulfilmentColorsStore) InTx(_ context.Context, fn func(Store) error) error { return fn(s) }
func (s *fulfilmentColorsStore) Bind(ctx context.Context) context.Context           { return ctx }
func (s *fulfilmentColorsStore) Fulfilment() FulfilmentRepository                   { return s.fulfilment }

func fulfilmentColorsFixture() (*Fulfilment, *fulfilmentColorsStock, *fulfilmentColorsStore, *model.Order, *auth.Principal) {
	catalog, terms := colorsFixture()
	terms.Lines[0].ModelSpecificationVersion, terms.Lines[0].ExteriorColor, terms.Lines[0].InteriorColor = "1", "Blue", "Tan"
	raw, _ := json.Marshal(terms)
	warehouse := uuid.NewString()
	o := &model.Order{ID: uuid.NewString(), BuyerCompanyID: "buyer", SupplierCompanyID: "supplier", Version: 1,
		Status: model.OrderAccepted, Terms: raw, ReceivingWarehouseID: &warehouse}
	st := &fulfilmentColorsStore{colorsStore: &colorsStore{deals: &colorsDeals{order: o}}, fulfilment: &fulfilmentColorsRepo{}}
	stock := &fulfilmentColorsStock{vehicle: StockVehicle{ID: "vehicle", VIN: "VIN", ModelID: terms.Lines[0].ModelID,
		ModelSpecificationVersion: "2", ExteriorColor: "Blue", InteriorColor: "Tan"}}
	s := &Fulfilment{Deps{store: st, stock: stock, catalog: catalog, now: func() time.Time { return time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC) }}}
	return s, stock, st, o, &auth.Principal{CompanyID: "supplier", UserID: "user"}
}

func TestAllocationColorsMatchesKnownFactsBeforeReserve(t *testing.T) {
	for _, name := range []string{"same colors different version", "wrong body", "wrong interior", "unknown body", "unknown interior", "wrong model", "unknown ordered colors"} {
		t.Run(name, func(t *testing.T) {
			s, stock, st, o, p := fulfilmentColorsFixture()
			switch name {
			case "wrong body":
				stock.vehicle.ExteriorColor = "White"
			case "wrong interior":
				stock.vehicle.InteriorColor = "Black"
			case "unknown body":
				stock.vehicle.ExteriorColor = ""
			case "unknown interior":
				stock.vehicle.InteriorColor = ""
			case "wrong model":
				stock.vehicle.ModelID = "other"
			case "unknown ordered colors":
				terms := o.DecodeTerms()
				terms.Lines[0].ExteriorColor, terms.Lines[0].InteriorColor = "", ""
				o.Terms, _ = json.Marshal(terms)
			}
			before := bytes.Clone(o.Terms)
			_, err := s.Allocate(context.Background(), p, o.ID, 1, []AllocationItem{{OrderLineID: o.DecodeTerms().Lines[0].LineID, VehicleID: "vehicle"}})
			valid := name == "same colors different version" || name == "unknown ordered colors"
			if valid && (err != nil || stock.reserved != 1 || len(st.fulfilment.allocations) != 1) {
				t.Fatalf("valid allocation: %v, %+v", err, stock)
			}
			if !valid && (err == nil || stock.reserved != 0 || len(st.fulfilment.allocations) != 0) {
				t.Fatalf("invalid allocation reached reserve: %v, %+v", err, stock)
			}
			if !bytes.Equal(before, o.Terms) {
				t.Fatal("allocation rewrote historical terms")
			}
		})
	}
}

func TestShipmentColorsKeepsRowsAndHistoricalPin(t *testing.T) {
	s, stock, st, o, p := fulfilmentColorsFixture()
	terms := o.DecodeTerms()
	first := terms.Lines[0]
	first.Quantity = 1
	second := first
	second.LineID = uuid.NewString()
	second.ExteriorColor, second.InteriorColor = "White", "Cream"
	terms.Lines = []model.Line{first, second}
	o.Terms, _ = json.Marshal(terms)
	before := bytes.Clone(o.Terms)
	sh, err := s.Ship(context.Background(), p, o.ID, 1, ShipmentInput{Route: "local", Lines: []ShipmentLineInput{
		{OrderLineID: first.LineID, Quantity: 1, VINs: []string{"1HGCM82633A004352"}}, {OrderLineID: second.LineID, Quantity: 1},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(stock.deliveries) != 2 || len(st.fulfilment.lines) != 2 || sh.Status != "received" || o.Status != model.OrderCompleted {
		t.Fatalf("shipment lost rows: %+v %+v", stock.deliveries, st.fulfilment.lines)
	}
	for i, want := range []model.Line{first, second} {
		d := stock.deliveries[i]
		if d.ModelSpecificationVersion != "1" || d.ExteriorColor != want.ExteriorColor || d.InteriorColor != want.InteriorColor || d.ModelID != want.ModelID || d.Quantity != 1 || d.ToCompanyID != "buyer" || d.ToWarehouseID != *o.ReceivingWarehouseID {
			t.Fatalf("delivery %d: %+v", i, d)
		}
		if st.fulfilment.lines[i].LineID != want.LineID {
			t.Fatal("lost order line identity")
		}
	}
	if len(stock.deliveries[0].VINs) != 1 || len(stock.deliveries[1].VINs) != 0 || !bytes.Equal(before, o.Terms) {
		t.Fatal("changed VINs or historical order")
	}
}

func TestShipmentColorsLegacyCompletionAndConflicts(t *testing.T) {
	for _, name := range []string{"explicit legacy", "partial legacy", "missing pin", "missing ambiguous body", "wrong historical palette", "missing exact version", "conflicting pin", "conflicting body", "conflicting interior"} {
		t.Run(name, func(t *testing.T) {
			s, stock, st, o, p := fulfilmentColorsFixture()
			terms := o.DecodeTerms()
			line := &terms.Lines[0]
			in := ShipmentLineInput{OrderLineID: line.LineID, Quantity: 1, ModelSpecificationVersion: "1", ExteriorColor: "Blue", InteriorColor: "Tan"}
			switch name {
			case "explicit legacy", "missing pin":
				line.ModelSpecificationVersion, line.ExteriorColor, line.InteriorColor = "", "", ""
			case "partial legacy", "missing ambiguous body":
				line.ExteriorColor = ""
			case "wrong historical palette":
				line.ExteriorColor = ""
				in.ExteriorColor = "Red"
			case "missing exact version":
				s.catalog.(*colorsCatalog).m.Specifications = s.catalog.(*colorsCatalog).m.Specifications[1:]
			case "conflicting pin":
				in.ModelSpecificationVersion = "2"
			case "conflicting body":
				in.ExteriorColor = "White"
			case "conflicting interior":
				in.InteriorColor = "Black"
			}
			if name == "missing pin" {
				in.ModelSpecificationVersion = ""
			}
			if name == "missing ambiguous body" {
				in.ExteriorColor = ""
			}
			o.Terms, _ = json.Marshal(terms)
			before := bytes.Clone(o.Terms)
			_, err := s.Ship(context.Background(), p, o.ID, 1, ShipmentInput{Route: "local", Lines: []ShipmentLineInput{in}})
			valid := name == "explicit legacy" || name == "partial legacy"
			if valid && (err != nil || len(stock.deliveries) != 1 || stock.deliveries[0].ModelSpecificationVersion != "1" || stock.deliveries[0].ExteriorColor != "Blue" || stock.deliveries[0].InteriorColor != "Tan") {
				t.Fatalf("legacy completion: %v %+v", err, stock.deliveries)
			}
			if !valid && (err == nil || len(stock.deliveries) != 0 || st.fulfilment.shipments != 0) {
				t.Fatalf("invalid selection wrote shipment: %v %+v", err, stock.deliveries)
			}
			if !bytes.Equal(before, o.Terms) {
				t.Fatal("shipment rewrote legacy JSON")
			}
		})
	}
}

func TestShipmentColorsProgressUsesActualBatchFacts(t *testing.T) {
	stock := &progressStock{batches: []ReceiptBatch{{ID: "batch", ModelID: "model-1", ModelSpecificationVersion: "1", ExteriorColor: "Blue", InteriorColor: "Tan"}}}
	s := &Deal{Deps: Deps{stock: stock}}
	o := progressOrder()
	before := bytes.Clone(o.Terms)
	refs, err := s.receiptBatchReferences(context.Background(), o, []model.ShipmentLine{{LineID: "line-1", ReceiptBatchID: "batch", Quantity: 2}})
	if err != nil || len(refs) != 1 || refs[0].ModelSpecificationVersion != "1" || refs[0].ExteriorColor != "Blue" || refs[0].InteriorColor != "Tan" || !bytes.Equal(before, o.Terms) {
		t.Fatalf("actual batch projection: %+v %v", refs, err)
	}
}
