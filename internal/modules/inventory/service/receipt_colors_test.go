package service

import (
	"context"
	"testing"
	"time"

	"justixauto/internal/modules/inventory/model"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/jsonx"
)

const (
	colorModelID     = "11111111-1111-1111-1111-111111111111"
	colorWarehouseID = "22222222-2222-2222-2222-222222222222"
	colorBatchID     = "33333333-3333-3333-3333-333333333333"
)

type colorModelRepo struct {
	ModelRepository
	specs       map[int]*model.Specification
	lastVersion int
}

func (r *colorModelRepo) Spec(_ context.Context, _ string, version int) (*model.Specification, error) {
	r.lastVersion = version
	return r.specs[version], nil
}

type colorWarehouseRepo struct {
	WarehouseRepository
	batch   *model.ReceiptBatch
	created *model.ReceiptBatch
}

func (r *colorWarehouseRepo) Lock(_ context.Context, _, _ string) (*model.Warehouse, error) {
	return &model.Warehouse{ID: colorWarehouseID, Capacity: 10, Version: 1}, nil
}
func (r *colorWarehouseRepo) Occupied(_ context.Context, _ []string) (map[string]int, error) {
	return map[string]int{colorWarehouseID: 0}, nil
}
func (r *colorWarehouseRepo) CreateBatch(_ context.Context, b *model.ReceiptBatch) error {
	r.created = b
	return nil
}
func (r *colorWarehouseRepo) LockBatch(_ context.Context, _, _ string) (*model.ReceiptBatch, error) {
	return r.batch, nil
}
func (r *colorWarehouseRepo) UpdateBatchCounts(_ context.Context, b *model.ReceiptBatch) error {
	b.Version++
	return nil
}
func (r *colorWarehouseRepo) Touch(_ context.Context, _ *model.Warehouse, _ time.Time) error {
	return nil
}

type colorVehicleRepo struct {
	VehicleRepository
	created []model.VehicleUnit
}

func (r *colorVehicleRepo) ExistingVINs(_ context.Context, _ []string) ([]string, error) {
	return nil, nil
}
func (r *colorVehicleRepo) Create(_ context.Context, units []model.VehicleUnit, _ []model.Placement) error {
	r.created = append(r.created, units...)
	return nil
}

type colorFactsRepo struct{ FactRepository }

func (*colorFactsRepo) Append(context.Context, ...model.Fact) error { return nil }

type colorStore struct {
	Store
	models     ModelRepository
	warehouses WarehouseRepository
	vehicles   VehicleRepository
	facts      FactRepository
}

func (s *colorStore) Models() ModelRepository         { return s.models }
func (s *colorStore) Warehouses() WarehouseRepository { return s.warehouses }
func (s *colorStore) Vehicles() VehicleRepository     { return s.vehicles }
func (s *colorStore) Facts() FactRepository           { return s.facts }
func (s *colorStore) InTx(_ context.Context, fn func(Store) error) error {
	return fn(s)
}

func colorService(specs map[int]*model.Specification, warehouses *colorWarehouseRepo, vehicles *colorVehicleRepo) (*Receipt, *colorModelRepo) {
	models := &colorModelRepo{specs: specs}
	receipt := NewReceipt(NewDeps(&colorStore{models: models, warehouses: warehouses, vehicles: vehicles, facts: &colorFactsRepo{}}, func() time.Time {
		return time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	}, nil))
	return receipt, models
}

func colorSpec(version int, exterior, interior []string) *model.Specification {
	return &model.Specification{ModelID: colorModelID, SpecVersion: version, ExteriorColors: exterior, InteriorColors: interior}
}

func colorPrincipal() *auth.Principal { return &auth.Principal{CompanyID: "company", UserID: "user"} }

func colorIdentify(exteriorColor string, vins ...string) IdentifyInput {
	in := IdentifyInput{ExteriorColor: exteriorColor, Atomic: true}
	for _, vin := range vins {
		in.Items = append(in.Items, struct {
			VIN     string `json:"vin"`
			ModelID string `json:"modelId"`
		}{VIN: vin, ModelID: colorModelID})
	}
	return in
}

func TestReceiptColorsIdentifiedAndUnidentifiedBatches(t *testing.T) {
	warehouses, vehicles := &colorWarehouseRepo{}, &colorVehicleRepo{}
	receipt, models := colorService(map[int]*model.Specification{1: colorSpec(1, []string{"Red", "Blue"}, []string{"Black", "Tan"}), 2: colorSpec(2, []string{"Green"}, []string{"White"})}, warehouses, vehicles)
	result, err := receipt.Receive(context.Background(), colorPrincipal(), colorWarehouseID, 1, ReceiptInput{
		ModelID: colorModelID, ModelSpecificationVersion: jsonx.Quantity(1), ExteriorColor: "blue", InteriorColor: "Tan",
		Stock: StockInput{Mode: "identified", VINs: []string{"1HGCM82633A004352"}}, ReceivedAt: time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	if models.lastVersion != 1 {
		t.Fatalf("validated specification version %d, want pinned version 1", models.lastVersion)
	}
	if result.Batch.ExteriorColor == nil || *result.Batch.ExteriorColor != "Blue" || result.Batch.InteriorColor == nil || *result.Batch.InteriorColor != "Tan" {
		t.Fatalf("batch colors = %+v", result.Batch)
	}
	if len(vehicles.created) != 1 || vehicles.created[0].ExteriorColor == nil || *vehicles.created[0].ExteriorColor != "Blue" || vehicles.created[0].InteriorColor == nil || *vehicles.created[0].InteriorColor != "Tan" {
		t.Fatalf("units = %+v", vehicles.created)
	}
}

func TestIdentifyColorsResolvesLegacyBatchThenReusesPersistedPair(t *testing.T) {
	batch := &model.ReceiptBatch{ID: colorBatchID, WarehouseID: colorWarehouseID, ModelID: colorModelID, SpecVersion: 1, UnidentifiedCount: 2}
	warehouses, vehicles := &colorWarehouseRepo{batch: batch}, &colorVehicleRepo{}
	receipt, _ := colorService(map[int]*model.Specification{1: colorSpec(1, []string{"Red", "Blue"}, []string{"Black"})}, warehouses, vehicles)
	first := colorIdentify("Blue", "1HGCM82633A004352")
	if _, err := receipt.Identify(context.Background(), colorPrincipal(), colorBatchID, first); err != nil {
		t.Fatal(err)
	}
	second := colorIdentify("", "1HGCM82633A004353")
	if _, err := receipt.Identify(context.Background(), colorPrincipal(), colorBatchID, second); err != nil {
		t.Fatal(err)
	}
	if batch.ExteriorColor == nil || *batch.ExteriorColor != "Blue" || batch.InteriorColor == nil || *batch.InteriorColor != "Black" {
		t.Fatalf("persisted batch colors = %+v", batch)
	}
	if len(vehicles.created) != 2 || *vehicles.created[0].ExteriorColor != "Blue" || *vehicles.created[1].ExteriorColor != "Blue" || *vehicles.created[0].InteriorColor != "Black" || *vehicles.created[1].InteriorColor != "Black" {
		t.Fatalf("identified units = %+v", vehicles.created)
	}
}

func TestIdentifyColorsRejectsKnownBatchMismatchWithoutCreatingVIN(t *testing.T) {
	exterior, interior := "Blue", "Black"
	batch := &model.ReceiptBatch{ID: colorBatchID, WarehouseID: colorWarehouseID, ModelID: colorModelID, SpecVersion: 1, ExteriorColor: &exterior, InteriorColor: &interior, UnidentifiedCount: 1}
	warehouses, vehicles := &colorWarehouseRepo{batch: batch}, &colorVehicleRepo{}
	receipt, _ := colorService(map[int]*model.Specification{1: colorSpec(1, []string{"Red", "Blue"}, []string{"Black"})}, warehouses, vehicles)
	in := colorIdentify("Red", "1HGCM82633A004352")
	if _, err := receipt.Identify(context.Background(), colorPrincipal(), colorBatchID, in); err == nil {
		t.Fatal("Identify succeeded with a mismatched known batch color")
	}
	if len(vehicles.created) != 0 || *batch.ExteriorColor != "Blue" || *batch.InteriorColor != "Black" {
		t.Fatalf("mismatch changed batch or units: batch=%+v units=%+v", batch, vehicles.created)
	}
}

func TestReceiptColorsNormalizeOnlySingletonPalettes(t *testing.T) {
	warehouses, vehicles := &colorWarehouseRepo{}, &colorVehicleRepo{}
	receipt, _ := colorService(map[int]*model.Specification{1: colorSpec(1, []string{"Red"}, []string{"Black"})}, warehouses, vehicles)
	_, err := receipt.Receive(context.Background(), colorPrincipal(), colorWarehouseID, 1, ReceiptInput{
		ModelID: colorModelID, ModelSpecificationVersion: jsonx.Quantity(1), Stock: StockInput{Mode: "unidentified", Quantity: 1}, ReceivedAt: time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC),
	})
	if err != nil || warehouses.created.ExteriorColor == nil || *warehouses.created.ExteriorColor != "Red" || warehouses.created.InteriorColor == nil || *warehouses.created.InteriorColor != "Black" {
		t.Fatalf("singleton normalization err=%v batch=%+v", err, warehouses.created)
	}
	_, err = receipt.Receive(context.Background(), colorPrincipal(), colorWarehouseID, 1, ReceiptInput{
		ModelID: colorModelID, ModelSpecificationVersion: jsonx.Quantity(1), Stock: StockInput{Mode: "unidentified", Quantity: 1}, ReceivedAt: time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	warehouses.created = nil
	receipt, _ = colorService(map[int]*model.Specification{1: colorSpec(1, []string{"Red", "Blue"}, []string{"Black"})}, warehouses, vehicles)
	_, err = receipt.Receive(context.Background(), colorPrincipal(), colorWarehouseID, 1, ReceiptInput{
		ModelID: colorModelID, ModelSpecificationVersion: jsonx.Quantity(1), Stock: StockInput{Mode: "unidentified", Quantity: 1}, ReceivedAt: time.Date(2026, 10, 2, 11, 0, 0, 0, time.UTC),
	})
	if err == nil || warehouses.created != nil {
		t.Fatalf("multi-choice receipt defaulted colors: err=%v batch=%+v", err, warehouses.created)
	}
}

func TestIdentifyColorsPartialLegacyCompletionPreservesHistory(t *testing.T) {
	for _, name := range []string{"body known", "interior known"} {
		t.Run(name, func(t *testing.T) {
			body, interior := "blue", "tan"
			batch := &model.ReceiptBatch{ID: colorBatchID, WarehouseID: colorWarehouseID, ModelID: colorModelID, SpecVersion: 1, UnidentifiedCount: 2, IdentifiedCount: 1}
			in := colorIdentify("Blue", "1HGCM82633A004352")
			in.InteriorColor = "Tan"
			if name == "body known" {
				batch.ExteriorColor = &body
			} else {
				batch.InteriorColor = &interior
			}
			// This pre-existing VIN is deliberately partial; completing the batch
			// must not retroactively claim a color for that historical vehicle.
			old := model.VehicleUnit{ID: "old", ExteriorColor: batch.ExteriorColor, InteriorColor: batch.InteriorColor}
			warehouses, vehicles := &colorWarehouseRepo{batch: batch}, &colorVehicleRepo{created: []model.VehicleUnit{old}}
			receipt, models := colorService(map[int]*model.Specification{1: colorSpec(1, []string{"Blue", "Red"}, []string{"Tan", "Black"}), 2: colorSpec(2, []string{"Green"}, []string{"White"})}, warehouses, vehicles)
			if _, err := receipt.Identify(context.Background(), colorPrincipal(), colorBatchID, in); err != nil {
				t.Fatal(err)
			}
			if _, err := receipt.Identify(context.Background(), colorPrincipal(), colorBatchID, colorIdentify("", "1HGCM82633A004353")); err != nil {
				t.Fatal(err)
			}
			if models.lastVersion != 1 || batch.SpecVersion != 1 || batch.Version != 2 || batch.IdentifiedCount != 3 || batch.UnidentifiedCount != 0 {
				t.Fatalf("pin/counts: %+v version lookup %d", batch, models.lastVersion)
			}
			if (name == "body known" && batch.ExteriorColor != &body) || (name == "interior known" && batch.InteriorColor != &interior) {
				t.Fatal("known color rewritten")
			}
			if vehicles.created[0].ExteriorColor != old.ExteriorColor || vehicles.created[0].InteriorColor != old.InteriorColor {
				t.Fatal("existing VIN history rewritten")
			}
			if len(vehicles.created) != 3 {
				t.Fatalf("VIN count %d", len(vehicles.created))
			}
			for _, unit := range vehicles.created[1:] {
				if unit.SpecVersion != 1 || unit.ExteriorColor == nil || *unit.ExteriorColor != *batch.ExteriorColor || unit.InteriorColor == nil || *unit.InteriorColor != *batch.InteriorColor {
					t.Fatalf("new VIN differs from batch: %+v", unit)
				}
			}
		})
	}
}

func TestIdentifyColorsPartialLegacyValidationIsAtomic(t *testing.T) {
	for _, name := range []string{"body conflict", "interior conflict", "missing ambiguous body", "wrong historical interior", "missing exact version", "empty pinned palette"} {
		t.Run(name, func(t *testing.T) {
			body, interior := "Blue", "Tan"
			batch := &model.ReceiptBatch{ID: colorBatchID, WarehouseID: colorWarehouseID, ModelID: colorModelID, SpecVersion: 1, UnidentifiedCount: 1}
			in := colorIdentify("Blue", "1HGCM82633A004352")
			in.InteriorColor = "Tan"
			spec := colorSpec(1, []string{"Blue", "Red"}, []string{"Tan", "Black"})
			switch name {
			case "body conflict":
				batch.ExteriorColor = &body
				in.ExteriorColor = "Red"
			case "interior conflict":
				batch.InteriorColor = &interior
				in.InteriorColor = "Black"
			case "missing ambiguous body":
				batch.InteriorColor = &interior
				in.ExteriorColor = ""
			case "wrong historical interior":
				batch.ExteriorColor = &body
				in.InteriorColor = "White"
			case "missing exact version":
				batch.ExteriorColor = &body
				spec = nil
			case "empty pinned palette":
				batch.ExteriorColor = &body
				spec.InteriorColors = nil
			}
			before := *batch
			warehouses, vehicles := &colorWarehouseRepo{batch: batch}, &colorVehicleRepo{}
			receipt, _ := colorService(map[int]*model.Specification{1: spec, 2: colorSpec(2, []string{"Green"}, []string{"White"})}, warehouses, vehicles)
			if _, err := receipt.Identify(context.Background(), colorPrincipal(), colorBatchID, in); err == nil {
				t.Fatal("invalid partial completion accepted")
			}
			if batch.ExteriorColor != before.ExteriorColor || batch.InteriorColor != before.InteriorColor || batch.SpecVersion != before.SpecVersion || batch.UnidentifiedCount != before.UnidentifiedCount || batch.IdentifiedCount != before.IdentifiedCount || batch.Version != before.Version || len(vehicles.created) != 0 {
				t.Fatalf("validation changed batch or VINs: %+v %+v", batch, vehicles.created)
			}
		})
	}
}
