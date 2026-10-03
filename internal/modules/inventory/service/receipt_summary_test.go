package service

import (
	"context"
	"testing"

	"justixauto/internal/modules/inventory/model"
)

type summaryWarehouseRepo struct {
	WarehouseRepository
	companyID string
	ids       []string
	batches   []model.ReceiptBatch
}

func (r *summaryWarehouseRepo) Batches(_ context.Context, companyID string, ids []string) ([]model.ReceiptBatch, error) {
	r.companyID, r.ids = companyID, append([]string(nil), ids...)
	return r.batches, nil
}

type summaryStore struct {
	Store
	warehouses WarehouseRepository
}

func (s summaryStore) Warehouses() WarehouseRepository { return s.warehouses }

func TestReceiptSummaryIncludesFullyIdentifiedBatch(t *testing.T) {
	repo := &summaryWarehouseRepo{batches: []model.ReceiptBatch{{ID: "batch-1", WarehouseID: "warehouse-1", ModelID: "model-1", ConfirmedQuantity: 3, IdentifiedCount: 3, UnidentifiedCount: 0, Version: 5}}}
	stock := &Stock{Deps: Deps{store: summaryStore{warehouses: repo}}}
	summaries, err := stock.ReceiptSummaries(context.Background(), "buyer", []string{"batch-1", "batch-1"})
	if err != nil {
		t.Fatal(err)
	}
	if repo.companyID != "buyer" || len(repo.ids) != 1 || repo.ids[0] != "batch-1" {
		t.Fatalf("lookup company=%q ids=%v", repo.companyID, repo.ids)
	}
	if len(summaries) != 1 || summaries[0].ConfirmedQuantity != 3 || summaries[0].IdentifiedCount != 3 || summaries[0].UnidentifiedCount != 0 || summaries[0].Revision != 5 {
		t.Fatalf("summaries=%+v", summaries)
	}
}
