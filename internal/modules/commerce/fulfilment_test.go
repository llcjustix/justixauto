package commerce_test

import (
	"net/http"
	"testing"
	"time"

	"justixauto/internal/e2e"
)

func warehouse(t *testing.T, c *e2e.Client, name, capacity string) string {
	w := c.Do(http.MethodPost, "/inventory/warehouses", map[string]any{
		"name": name, "country": map[string]string{"label": "Uzbekistan"},
		"city": "Tashkent", "address": "Yard 1", "capacity": capacity,
	})
	expect(t, w, http.StatusCreated)
	return str(w.Data(), "id")
}

// receive registers VINs of the model into a warehouse and returns their IDs.
func receive(t *testing.T, e *e2e.Env, c *e2e.Client, warehouseID, model string, vins ...string) []string {
	rev := c.Do(http.MethodGet, "/inventory/warehouses/"+warehouseID, nil).Revision()
	r := c.Do(http.MethodPost, "/inventory/warehouses/"+warehouseID+"/receipt-batches", map[string]any{
		"modelId":                   model,
		"modelSpecificationVersion": "1", "stock": map[string]any{"mode": "identified", "vins": vins},
		"receivedAt": e.Clock.Now().Format(time.RFC3339),
	}, ifMatch(rev)...)
	expect(t, r, http.StatusCreated)
	var ids []string
	for _, v := range r.Data()["vehicles"].([]any) {
		ids = append(ids, str(v.(map[string]any), "id"))
	}
	return ids
}

func TestAllocationShipmentAndReceipt(t *testing.T) {
	tr := newTrade(t)
	s, b := tr.supplier, tr.buyer
	sw := warehouse(t, s, "Supplier yard", "10")
	bw := warehouse(t, b, "Buyer yard", "2")
	v := receive(t, tr.e, s, sw, tr.model, "XTAAA11111A100001", "XTAAA11111A100002", "XTAAA11111A100003", "XTAAA11111A100004")
	buyerOwn := receive(t, tr.e, b, bw, tr.model, "XTAAA11111A200001")[0]

	_, version := tr.publishedOffer(t)
	lines := version["terms"].(map[string]any)["lines"].([]any)
	offerLine := str(lines[1].(map[string]any), "lineId")
	order := func() (string, string) {
		o := b.Do(http.MethodPost, "/commerce/orders", map[string]any{
			"offerVersionId": str(version, "id"), "warehouseId": bw,
			"lines": []map[string]string{{"offerLineId": offerLine, "quantity": "1"}},
		})
		expect(t, o, http.StatusCreated)
		id := str(o.Data(), "id")
		line := o.Data()["terms"].(map[string]any)["lines"].([]any)[0].(map[string]any)
		if str(line, "lineId") == offerLine || line["offerLineId"] != offerLine || o.Data()["receivingWarehouseId"] != bw {
			t.Fatalf("order line provenance and receiving warehouse: %v", o.Data())
		}
		expect(t, s.Do(http.MethodPost, "/commerce/orders/"+id+"/supplier-confirmations", map[string]any{}, ifMatch("1")...), http.StatusOK)
		return id, str(line, "lineId")
	}

	// RFQ acceptance still creates an accepted order without a receiving warehouse.
	// Keep its separate receipt/rejection path covered alongside direct delivery below.
	rfq := b.Do(http.MethodPost, "/commerce/rfqs", map[string]any{
		"supplierCompanyId": s.CompanyID,
		"lines":             []map[string]string{{"modelId": tr.model, "quantity": "3"}},
	})
	expect(t, rfq, http.StatusCreated)
	rfqID := str(rfq.Data(), "id")
	expect(t, b.Do(http.MethodPost, "/commerce/rfqs/"+rfqID+"/send", map[string]any{}, ifMatch("1")...), http.StatusOK)
	quoted := s.Do(http.MethodPost, "/commerce/rfqs/"+rfqID+"/quotation-versions", map[string]any{"terms": terms(tr.model)}, ifMatch("2")...)
	expect(t, quoted, http.StatusCreated)
	quotation := quoted.Data()["quotations"].([]any)[0].(map[string]any)
	accepted := b.Do(http.MethodPost, "/commerce/rfqs/"+rfqID+"/accept", map[string]string{
		"quotationVersionId": str(quotation, "id"), "quotationDigest": str(quotation, "digest"),
	}, ifMatch("3")...)
	expect(t, accepted, http.StatusCreated)
	if accepted.Data()["status"] != "accepted" || accepted.Data()["receivingWarehouseId"] != nil || accepted.Data()["hasReceivingWarehouse"] != false {
		t.Fatalf("RFQ order for separate receipt: %v", accepted.Data())
	}
	main := str(accepted.Data(), "id")
	orderLines := accepted.Data()["terms"].(map[string]any)["lines"].([]any)
	l1, l2 := str(orderLines[0].(map[string]any), "lineId"), str(orderLines[1].(map[string]any), "lineId")
	alloc := func(id, rev string, items ...[2]string) e2e.Response {
		var body []map[string]string
		for _, it := range items {
			body = append(body, map[string]string{"orderLineId": it[0], "vehicleId": it[1]})
		}
		return s.Do(http.MethodPost, "/commerce/orders/"+id+"/allocations", map[string]any{"items": body}, ifMatch(rev)...)
	}
	expect(t, alloc(main, "1", [2]string{l1, buyerOwn}), http.StatusUnprocessableEntity)                                       // not the supplier's
	expect(t, alloc(main, "1", [2]string{l1, v[0]}, [2]string{l1, v[1]}, [2]string{l1, v[2]}), http.StatusUnprocessableEntity) // over quantity
	expect(t, b.Do(http.MethodPost, "/commerce/orders/"+main+"/allocations", map[string]any{"items": []map[string]string{{"orderLineId": l1, "vehicleId": v[0]}}}, ifMatch("1")...), http.StatusForbidden)
	a := alloc(main, "1", [2]string{l1, v[0]}, [2]string{l1, v[1]}, [2]string{l2, v[2]})
	expect(t, a, http.StatusOK)
	if len(a.Data()["allocations"].([]any)) != 3 || !has(a.Data()["allowedActions"], "ship") {
		t.Fatalf("allocations: %v", a.Data())
	}

	// A VIN is held by one deal at a time; cancelling releases it.
	second, secondLine := order()
	expect(t, alloc(second, "2", [2]string{secondLine, v[0]}), http.StatusConflict, "vehicle_unavailable")
	expect(t, alloc(second, "2", [2]string{secondLine, v[3]}), http.StatusOK)
	expect(t, b.Do(http.MethodPost, "/commerce/orders/"+second+"/cancellations", map[string]string{"reason": "duplicate"}, ifMatch("3")...), http.StatusOK)
	third, thirdLine := order()
	expect(t, alloc(third, "2", [2]string{thirdLine, v[3]}), http.StatusOK)

	// Ship two vehicles; milestones are facts; the order is now fulfilling.
	sh := s.Do(http.MethodPost, "/commerce/orders/"+main+"/shipments", map[string]any{"vehicleIds": []string{v[0], v[1]}, "route": "local"}, ifMatch("2")...)
	expect(t, sh, http.StatusCreated)
	if sh.Data()["status"] != "in-transit" {
		t.Fatalf("shipment awaiting separate receipt: %v", sh.Data())
	}
	shipment := str(sh.Data(), "id")
	expect(t, s.Do(http.MethodPost, "/commerce/orders/"+main+"/shipments", map[string]any{"vehicleIds": []string{v[0]}, "route": "local"}, ifMatch("3")...), http.StatusUnprocessableEntity)
	expect(t, b.Do(http.MethodPost, "/commerce/orders/"+main+"/cancellations", map[string]string{"reason": "x"}, ifMatch("3")...), http.StatusConflict)
	now := tr.e.Clock.Now().Format(time.RFC3339)
	expect(t, s.Do(http.MethodPost, "/commerce/shipments/"+shipment+"/milestones", map[string]string{"milestoneType": "departed", "occurredAt": now, "location": "Asaka"}), http.StatusCreated)
	expect(t, b.Do(http.MethodPost, "/commerce/shipments/"+shipment+"/milestones", map[string]string{"milestoneType": "damage-reported", "occurredAt": now, "location": "Tashkent"}), http.StatusUnprocessableEntity)

	// Receipt: the buyer accepts one into its warehouse and rejects the other.
	decide := func(decision string, ids []string, rev, reason string) e2e.Response {
		return b.Do(http.MethodPost, "/commerce/shipments/"+shipment+"/receipt-decisions", map[string]any{
			"decision":   decision,
			"vehicleIds": ids, "warehouseId": bw, "reason": reason,
		}, ifMatch(rev)...)
	}
	expect(t, decide("accept", []string{v[0], v[1]}, "1", ""), http.StatusConflict, "capacity_exceeded") // yard has 1 free place
	expect(t, s.Do(http.MethodPost, "/commerce/shipments/"+shipment+"/receipt-decisions", map[string]any{"decision": "reject", "vehicleIds": []string{v[1]}, "reason": "x"}, ifMatch("1")...), http.StatusForbidden)
	expect(t, decide("accept", []string{v[0]}, "1", ""), http.StatusOK)
	expect(t, decide("reject", []string{v[1]}, "2", ""), http.StatusUnprocessableEntity)
	r := decide("reject", []string{v[1]}, "2", "scratched door")
	expect(t, r, http.StatusOK)
	if r.Data()["status"] != "received" {
		t.Fatalf("shipment: %v", r.Data())
	}

	// Ownership moved: the buyer now holds the vehicle, the supplier no longer sees it.
	got := b.Do(http.MethodGet, "/inventory/vehicle-units/"+v[0], nil)
	expect(t, got, http.StatusOK)
	if got.Data()["vehicle"].(map[string]any)["placement"].(map[string]any)["warehouseId"] != bw {
		t.Fatalf("buyer placement: %v", got.Data())
	}
	expect(t, s.Do(http.MethodGet, "/inventory/vehicle-units/"+v[0], nil), http.StatusNotFound)

	// Warehouse-backed orders check capacity at shipment, before immediate delivery.
	expect(t, s.Do(http.MethodPost, "/commerce/orders/"+third+"/shipments", map[string]any{"vehicleIds": []string{v[3]}, "route": "local"}, ifMatch("3")...), http.StatusConflict, "buyer_warehouse_full")
	blocked := s.Do(http.MethodGet, "/commerce/orders/"+third, nil)
	expect(t, blocked, http.StatusOK)
	if blocked.Revision() != "3" || blocked.Data()["status"] != "accepted" || len(blocked.Data()["shipments"].([]any)) != 0 {
		t.Fatalf("blocked shipment changed the order: %v", blocked.Data())
	}

	// The rejected vehicle is free again and can be re-allocated; full delivery completes the order.
	rev := s.Do(http.MethodGet, "/commerce/orders/"+main, nil).Revision()
	expect(t, alloc(main, rev, [2]string{l1, v[1]}), http.StatusOK)
	expect(t, b.Do(http.MethodPost, "/inventory/warehouses/"+bw+"/capacity-changes", map[string]any{"capacity": "5", "reason": "extended"},
		ifMatch(b.Do(http.MethodGet, "/inventory/warehouses/"+bw, nil).Revision())...), http.StatusOK)
	rev = s.Do(http.MethodGet, "/commerce/orders/"+main, nil).Revision()
	sh2 := s.Do(http.MethodPost, "/commerce/orders/"+main+"/shipments", map[string]any{"vehicleIds": []string{v[1], v[2]}, "route": "local"}, ifMatch(rev)...)
	expect(t, sh2, http.StatusCreated)
	shipment = str(sh2.Data(), "id")
	expect(t, decide("accept", []string{v[1], v[2]}, "1", ""), http.StatusOK)
	done := b.Do(http.MethodGet, "/commerce/orders/"+main, nil)
	if done.Data()["status"] != "completed" {
		t.Fatalf("order after full delivery: %v", done.Data()["status"])
	}

	// Current direct orders deliver to their chosen warehouse in the shipment command.
	direct := s.Do(http.MethodPost, "/commerce/orders/"+third+"/shipments", map[string]any{"vehicleIds": []string{v[3]}, "route": "local"}, ifMatch("3")...)
	expect(t, direct, http.StatusCreated)
	if direct.Data()["status"] != "received" || direct.Data()["vehicles"].([]any)[0].(map[string]any)["status"] != "delivered" {
		t.Fatalf("immediate delivery: %v", direct.Data())
	}
	directOrder := b.Do(http.MethodGet, "/commerce/orders/"+third, nil)
	expect(t, directOrder, http.StatusOK)
	if directOrder.Data()["status"] != "completed" || directOrder.Revision() != "4" {
		t.Fatalf("order after immediate delivery: %v", directOrder.Data())
	}
	delivered := b.Do(http.MethodGet, "/inventory/vehicle-units/"+v[3], nil)
	expect(t, delivered, http.StatusOK)
	vehicle := delivered.Data()["vehicle"].(map[string]any)
	if vehicle["ownerCompanyId"] != b.CompanyID || vehicle["placement"].(map[string]any)["warehouseId"] != bw {
		t.Fatalf("immediately delivered vehicle: %v", delivered.Data())
	}
	expect(t, s.Do(http.MethodGet, "/inventory/vehicle-units/"+v[3], nil), http.StatusNotFound)
	expect(t, b.Do(http.MethodPost, "/commerce/shipments/"+str(direct.Data(), "id")+"/receipt-decisions", map[string]any{
		"decision": "accept", "vehicleIds": []string{v[3]}, "warehouseId": bw,
	}, ifMatch("1")...), http.StatusUnprocessableEntity)
}
