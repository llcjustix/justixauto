package service

import (
	"context"

	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
)

// ReceiptBatchReference connects an order quantity shipment to its inventory
// receipt batch and current identification state.
type ReceiptBatchReference struct {
	OrderLineID, ShipmentID, ReceiptBatchID                 string
	WarehouseID, ModelID                                    string
	ModelSpecificationVersion, ExteriorColor, InteriorColor string
	ShippedQuantity, ConfirmedQuantity                      int
	IdentifiedCount, UnidentifiedCount                      int
	Revision                                                int64
}

// LineProgress separates completed fulfilment from live allocations. The
// identification counts describe only the shipped portion.
type LineProgress struct {
	Shipped, Allocated, Identified, Unidentified int
	ReceiptQuantityAdjusted                      bool
}

// OrderLineProgress calculates the public fulfilment counters for an order.
func OrderLineProgress(v *OrderView) map[string]LineProgress {
	allocations, shipmentLines, batches := v.Allocations, v.ShipmentLines, v.ReceiptBatches
	progress := map[string]LineProgress{}
	add := func(lineID string, update func(*LineProgress)) {
		p := progress[lineID]
		update(&p)
		progress[lineID] = p
	}
	for _, a := range allocations {
		switch a.Status {
		case "allocated":
			add(a.LineID, func(p *LineProgress) { p.Allocated++ })
		case "shipped", "delivered":
			// Concrete VIN allocations are identified by definition.
			add(a.LineID, func(p *LineProgress) { p.Shipped++; p.Identified++ })
		}
	}
	byLine := make(map[string]ReceiptBatchReference, len(batches))
	for _, b := range batches {
		byLine[b.OrderLineID+"\x00"+b.ShipmentID+"\x00"+b.ReceiptBatchID] = b
	}
	for _, l := range shipmentLines {
		add(l.LineID, func(p *LineProgress) { p.Shipped += l.Quantity })
		b := byLine[l.LineID+"\x00"+l.ShipmentID+"\x00"+l.ReceiptBatchID]
		add(l.LineID, func(p *LineProgress) {
			p.Identified += b.IdentifiedCount
			p.Unidentified += b.UnidentifiedCount
			p.ReceiptQuantityAdjusted = p.ReceiptQuantityAdjusted || b.ConfirmedQuantity != l.Quantity
		})
	}
	return progress
}

func (s *Deal) receiptBatchReferences(ctx context.Context, o *model.Order, lines []model.ShipmentLine) ([]ReceiptBatchReference, error) {
	lineModels := make(map[string]string, len(o.DecodeTerms().Lines))
	for _, line := range o.DecodeTerms().Lines {
		lineModels[line.LineID] = line.ModelID
	}
	ids := make([]string, 0, len(lines))
	seen := make(map[string]struct{}, len(lines))
	for _, l := range lines {
		if l.ReceiptBatchID == "" {
			return nil, apperr.New(apperr.ErrConflict, "shipment_batch_missing", "a quantity shipment has no receipt batch")
		}
		if _, ok := seen[l.ReceiptBatchID]; !ok {
			seen[l.ReceiptBatchID] = struct{}{}
			ids = append(ids, l.ReceiptBatchID)
		}
	}
	if len(ids) == 0 {
		return []ReceiptBatchReference{}, nil
	}
	bs, err := s.stock.ReceiptBatches(ctx, o.BuyerCompanyID, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]ReceiptBatch, len(bs))
	for _, b := range bs {
		byID[b.ID] = b
	}
	refs := make([]ReceiptBatchReference, 0, len(lines))
	for _, l := range lines {
		b, ok := byID[l.ReceiptBatchID]
		if !ok {
			return nil, apperr.ErrNotFound
		}
		if b.ModelID != lineModels[l.LineID] {
			return nil, apperr.New(apperr.ErrConflict, "shipment_batch_model_mismatch", "receipt batch model differs from the shipment line")
		}
		refs = append(refs, ReceiptBatchReference{OrderLineID: l.LineID, ShipmentID: l.ShipmentID, ReceiptBatchID: b.ID,
			ModelSpecificationVersion: b.ModelSpecificationVersion, ExteriorColor: b.ExteriorColor, InteriorColor: b.InteriorColor,
			WarehouseID: b.WarehouseID, ModelID: b.ModelID, ShippedQuantity: l.Quantity, ConfirmedQuantity: b.ConfirmedQuantity,
			IdentifiedCount:   b.IdentifiedCount,
			UnidentifiedCount: b.UnidentifiedCount, Revision: b.Revision})
	}
	return refs, nil
}
