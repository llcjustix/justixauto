package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/jsonx"
)

type progressStock struct {
	Stock
	batches   []ReceiptBatch
	err       error
	requested []string
}

func (s *progressStock) ReceiptBatches(_ context.Context, _ string, ids []string) ([]ReceiptBatch, error) {
	s.requested = append([]string(nil), ids...)
	return s.batches, s.err
}

func progressOrder() *model.Order {
	terms, _ := json.Marshal(model.Terms{Lines: []model.Line{
		{LineID: "line-1", ModelID: "model-1", Quantity: jsonx.Quantity(10)},
		{LineID: "line-2", ModelID: "model-1", Quantity: jsonx.Quantity(10)},
	}})
	return &model.Order{BuyerCompanyID: "buyer", Terms: terms}
}

func TestOrderReceiptBatchReferencesUseOneBoundedRead(t *testing.T) {
	stock := &progressStock{batches: []ReceiptBatch{{ID: "batch-1", WarehouseID: "warehouse-1", ModelID: "model-1", ConfirmedQuantity: 3, IdentifiedCount: 1, UnidentifiedCount: 2, Revision: 7}}}
	d := &Deal{Deps: Deps{stock: stock}}
	refs, err := d.receiptBatchReferences(context.Background(), progressOrder(), []model.ShipmentLine{
		{LineID: "line-1", ShipmentID: "shipment-1", ReceiptBatchID: "batch-1", Quantity: 3},
		{LineID: "line-2", ShipmentID: "shipment-2", ReceiptBatchID: "batch-1", Quantity: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(stock.requested) != 1 || stock.requested[0] != "batch-1" || len(refs) != 2 || refs[0].WarehouseID != "warehouse-1" || refs[0].Revision != 7 {
		t.Fatalf("requested=%v refs=%+v", stock.requested, refs)
	}
}

func TestOrderReceiptBatchReferencesPropagateReadErrors(t *testing.T) {
	want := apperr.ErrNotFound
	d := &Deal{Deps: Deps{stock: &progressStock{err: want}}}
	_, err := d.receiptBatchReferences(context.Background(), progressOrder(), []model.ShipmentLine{{LineID: "line-1", ShipmentID: "shipment-1", ReceiptBatchID: "batch-1", Quantity: 1}})
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestOrderReceiptBatchReferencesExposeQuantityCorrection(t *testing.T) {
	d := &Deal{Deps: Deps{stock: &progressStock{batches: []ReceiptBatch{{ID: "batch-1", ModelID: "model-1", ConfirmedQuantity: 3, IdentifiedCount: 1, UnidentifiedCount: 2}}}}}
	line := model.ShipmentLine{LineID: "line-1", ShipmentID: "shipment-1", ReceiptBatchID: "batch-1", Quantity: 2}
	refs, err := d.receiptBatchReferences(context.Background(), progressOrder(), []model.ShipmentLine{line})
	if err != nil || len(refs) != 1 || refs[0].ShippedQuantity != 2 || refs[0].ConfirmedQuantity != 3 {
		t.Fatalf("refs=%+v err=%v", refs, err)
	}
	p := OrderLineProgress(&OrderView{ShipmentLines: []model.ShipmentLine{line}, ReceiptBatches: refs})["line-1"]
	if p.Shipped != 2 || p.Identified != 1 || p.Unidentified != 2 || !p.ReceiptQuantityAdjusted {
		t.Fatalf("progress=%+v", p)
	}
}

func TestOrderReceiptBatchReferencesRejectModelMismatch(t *testing.T) {
	d := &Deal{Deps: Deps{stock: &progressStock{batches: []ReceiptBatch{{ID: "batch-1", ModelID: "model-2", ConfirmedQuantity: 1}}}}}
	_, err := d.receiptBatchReferences(context.Background(), progressOrder(), []model.ShipmentLine{{LineID: "line-1", ShipmentID: "shipment-1", ReceiptBatchID: "batch-1", Quantity: 1}})
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != "shipment_batch_model_mismatch" {
		t.Fatalf("got %v", err)
	}
}
