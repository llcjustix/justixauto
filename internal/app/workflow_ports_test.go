package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"justixauto/internal/modules/commerce"
	"justixauto/internal/modules/documents"
	"justixauto/internal/modules/identity"
	"justixauto/internal/modules/inventory"
	"justixauto/internal/modules/retail"
	"justixauto/internal/pkg/apperr"
)

type workflowMarkerKey struct{}

type workflowOrderLookup struct {
	order                 *commerce.Order
	err                   error
	seenViewer, seenOrder string
	marker                any
}

func (f *workflowOrderLookup) OrderPartiesFor(ctx context.Context, viewer, order string) (*commerce.Order, error) {
	f.seenViewer, f.seenOrder = viewer, order
	f.marker = ctx.Value(workflowMarkerKey{})
	if f.err != nil {
		return nil, f.err
	}
	return f.order, nil
}

type workflowCommerceCompanies struct {
	calls [][2]string
	err   error
}

func (f *workflowCommerceCompanies) ContractRequisitesFor(_ context.Context, authorized, company string) (*identity.ContractRequisites, error) {
	f.calls = append(f.calls, [2]string{authorized, company})
	if f.err != nil {
		return nil, f.err
	}
	return &identity.ContractRequisites{ID: company, Name: company, LegalName: company + " LLC", RegistrationNumber: "reg", TaxID: "tax", BankRequisites: "bank", Address: "address", Phone: "phone", Email: "mail@example.test"}, nil
}

func TestWorkflowPort12(t *testing.T) {
	lookup := &workflowOrderLookup{order: &commerce.Order{ID: "actual", BuyerCompanyID: "buyer", SupplierCompanyID: "supplier"}}
	companies := &workflowCommerceCompanies{}
	adapter := commerceContractParties{orders: lookup, companies: companies}
	ctx := context.WithValue(context.Background(), workflowMarkerKey{}, "retained")
	buyer, supplier, err := adapter.ForOrder(ctx, "buyer", "actual")
	if err != nil {
		t.Fatal(err)
	}
	if lookup.seenViewer != "buyer" || lookup.seenOrder != "actual" || lookup.marker != "retained" {
		t.Fatalf("lookup=%+v", lookup)
	}
	if len(companies.calls) != 2 || companies.calls[0] != [2]string{"buyer", "buyer"} || companies.calls[1] != [2]string{"supplier", "supplier"} {
		t.Fatalf("private lookups=%+v", companies.calls)
	}
	if buyer.LegalName != "buyer LLC" || buyer.BankRequisites != "bank" || supplier.ID != "supplier" || supplier.Email != "mail@example.test" {
		t.Fatalf("requisites=%+v/%+v", buyer, supplier)
	}

	companies.calls = nil
	lookup.err = errors.New("outsider")
	if _, _, err = adapter.ForOrder(ctx, "outsider", "actual"); err == nil {
		t.Fatal("outsider accepted")
	}
	if len(companies.calls) != 0 {
		t.Fatalf("identity queried for outsider: %+v", companies.calls)
	}
}

type workflowReservationHolder struct {
	reserveCompany, reason                                 string
	reserveHolder, releaseHolder, reassignFrom, reassignTo inventory.Holder
	marker                                                 any
	ids                                                    []string
}

func (f *workflowReservationHolder) Reserve(ctx context.Context, company string, h inventory.Holder, ids []string) error {
	f.reserveCompany, f.reserveHolder, f.marker, f.ids = company, h, ctx.Value(workflowMarkerKey{}), append([]string(nil), ids...)
	return nil
}

func (f *workflowReservationHolder) Release(ctx context.Context, h inventory.Holder, ids []string, reason string) error {
	f.releaseHolder, f.reason, f.marker, f.ids = h, reason, ctx.Value(workflowMarkerKey{}), append([]string(nil), ids...)
	return nil
}

func (f *workflowReservationHolder) Reassign(ctx context.Context, from, to inventory.Holder, ids []string) error {
	f.reassignFrom, f.reassignTo, f.marker, f.ids = from, to, ctx.Value(workflowMarkerKey{}), append([]string(nil), ids...)
	return nil
}

func TestWorkflowPort17(t *testing.T) {
	f := &workflowReservationHolder{}
	a := commerceStock{reservations: f}
	ctx := context.WithValue(context.Background(), workflowMarkerKey{}, "bound transaction")
	if err := a.ReserveReservation(ctx, "supplier", "reservation", []string{"vin"}); err != nil {
		t.Fatal(err)
	}
	if f.reserveCompany != "supplier" || f.reserveHolder != (inventory.Holder{Type: "commerce-reservation", ID: "reservation"}) || f.marker != "bound transaction" || len(f.ids) != 1 || f.ids[0] != "vin" {
		t.Fatalf("reserve=%+v", f)
	}
	if err := a.ReleaseReservation(ctx, "reservation", []string{"vin"}, "cancel"); err != nil {
		t.Fatal(err)
	}
	if f.releaseHolder != (inventory.Holder{Type: "commerce-reservation", ID: "reservation"}) || f.reason != "cancel" || f.marker != "bound transaction" {
		t.Fatalf("release=%+v", f)
	}
	if err := a.ReassignReservationToOrder(ctx, "reservation", "order", []string{"vin"}); err != nil {
		t.Fatal(err)
	}
	if f.reassignFrom != (inventory.Holder{Type: "commerce-reservation", ID: "reservation"}) || f.reassignTo != (inventory.Holder{Type: "commerce-order", ID: "order"}) || f.marker != "bound transaction" || len(f.ids) != 1 || f.ids[0] != "vin" {
		t.Fatalf("reassign=%+v", f)
	}
}

type workflowCustody struct {
	input  inventory.CustodyInput
	marker any
	err    error
	facts  []string
}

func (f *workflowCustody) Custody(ctx context.Context, in inventory.CustodyInput) (*inventory.CustodyResult, error) {
	f.input, f.marker = in, ctx.Value(workflowMarkerKey{})
	if f.err != nil {
		return nil, f.err
	}
	return &inventory.CustodyResult{FactIDs: f.facts}, nil
}

func TestWorkflowPort21(t *testing.T) {
	f := &workflowCustody{facts: []string{"fact"}}
	a := commerceStock{custody: f}
	ctx := context.WithValue(context.Background(), workflowMarkerKey{}, "bound transaction")
	at := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	move := commerce.CustodyMove{OwnerCompanyID: "supplier", SourceCustodianCompanyID: "supplier", DestinationCompanyID: "dealer", DestinationWarehouseID: "warehouse", OrderID: "order", AgreementID: "agreement", OperationRef: "operation", ActorUserID: "actor", VehicleID: "vehicle", At: at}
	fact, err := a.Custody(ctx, move)
	if err != nil || fact != "fact" {
		t.Fatalf("fact=%s err=%v", fact, err)
	}
	in := f.input
	if f.marker != "bound transaction" || in.Purpose != "consignment" || in.OwnerCompanyID != move.OwnerCompanyID || in.SourceCustodianCompanyID != move.SourceCustodianCompanyID || in.DestinationCompanyID != move.DestinationCompanyID || in.DestinationWarehouseID != move.DestinationWarehouseID || in.ExpectedHolder != (inventory.Holder{Type: "commerce-order", ID: "order"}) || in.AgreementID != move.AgreementID || in.OperationRef != move.OperationRef || in.ActorUserID != move.ActorUserID || in.At != at || len(in.VehicleIDs) != 1 || in.VehicleIDs[0] != move.VehicleID {
		t.Fatalf("custody input=%+v marker=%v", in, f.marker)
	}
	want := errors.New("capacity failed")
	f.err = want
	if _, err := a.Custody(ctx, move); !errors.Is(err, want) {
		t.Fatalf("error=%v", err)
	}
	f.err = nil
	f.facts = nil
	if _, err := a.Custody(ctx, move); err == nil {
		t.Fatal("missing fact accepted")
	}
}

func TestWorkflowPort22(t *testing.T) {
	f := &workflowCustody{facts: []string{"return-fact"}}
	a := commerceStock{custody: f}
	ctx := context.WithValue(context.Background(), workflowMarkerKey{}, "bound transaction")
	at := time.Date(2026, 10, 11, 1, 2, 3, 0, time.UTC)
	move := commerce.ReturnCustodyMove{OwnerCompanyID: "supplier", SourceCustodianCompanyID: "dealer", DestinationWarehouseID: "supplier-warehouse", AgreementID: "agreement", RequestID: "request", ActorUserID: "actor", VehicleID: "vin", At: at}
	fact, err := a.ReturnCustody(ctx, move)
	if err != nil || fact != "return-fact" {
		t.Fatalf("fact=%s err=%v", fact, err)
	}
	in := f.input
	if f.marker != "bound transaction" || in.Purpose != "return" || in.ExpectedHolder != (inventory.Holder{}) || in.OwnerCompanyID != "supplier" || in.SourceCustodianCompanyID != "dealer" || in.DestinationCompanyID != "supplier" || in.DestinationWarehouseID != "supplier-warehouse" || in.AgreementID != "agreement" || in.OperationRef != "request" || in.ActorUserID != "actor" || !in.At.Equal(at) || len(in.VehicleIDs) != 1 || in.VehicleIDs[0] != "vin" {
		t.Fatalf("return input=%+v marker=%v", in, f.marker)
	}
	want := errors.New("capacity failed")
	f.err = want
	if _, err := a.ReturnCustody(ctx, move); !errors.Is(err, want) {
		t.Fatalf("error=%v", err)
	}
	f.err = nil
	f.facts = nil
	if _, err := a.ReturnCustody(ctx, move); err == nil {
		t.Fatal("missing fact accepted")
	}
}

type workflowRetailCommerce struct {
	authority                                 *commerce.ConsignmentSaleAuthority
	err                                       error
	method, seller, vehicle, deal, actor, ref string
	at                                        time.Time
	marker                                    any
}

func (f *workflowRetailCommerce) record(ctx context.Context, method, seller, vehicle, deal, actor, ref string, at time.Time) {
	f.method, f.seller, f.vehicle, f.deal, f.actor, f.ref, f.at, f.marker = method, seller, vehicle, deal, actor, ref, at, ctx.Value(workflowMarkerKey{})
}

func (f *workflowRetailCommerce) CheckConsignmentSale(ctx context.Context, seller, vehicle string) (*commerce.ConsignmentSaleAuthority, error) {
	f.record(ctx, "check", seller, vehicle, "", "", "", time.Time{})
	return f.authority, f.err
}

func (f *workflowRetailCommerce) ReserveConsignmentSale(ctx context.Context, seller, vehicle, deal, actor string) (*commerce.ConsignmentSaleAuthority, error) {
	f.record(ctx, "reserve", seller, vehicle, deal, actor, "", time.Time{})
	return f.authority, f.err
}

func (f *workflowRetailCommerce) ConsignmentSaleForDeal(ctx context.Context, seller, vehicle, deal string) (*commerce.ConsignmentSaleAuthority, error) {
	f.record(ctx, "for-deal", seller, vehicle, deal, "", "", time.Time{})
	return f.authority, f.err
}

func (f *workflowRetailCommerce) CancelConsignmentSale(ctx context.Context, seller, vehicle, deal, actor string) error {
	f.record(ctx, "cancel", seller, vehicle, deal, actor, "", time.Time{})
	return f.err
}

func (f *workflowRetailCommerce) CompleteConsignmentSale(ctx context.Context, seller, vehicle, deal, ref, actor string, at time.Time) error {
	f.record(ctx, "complete", seller, vehicle, deal, actor, ref, at)
	return f.err
}

type workflowRetailStock struct {
	in           inventory.ConsignmentSaleInput
	method, fact string
	err          error
	marker       any
}

func (f *workflowRetailStock) ReserveConsignmentSale(ctx context.Context, in inventory.ConsignmentSaleInput) error {
	f.in, f.method, f.marker = in, "hold", ctx.Value(workflowMarkerKey{})
	return f.err
}

func (f *workflowRetailStock) DeliverConsignmentSale(ctx context.Context, in inventory.ConsignmentSaleInput) (string, error) {
	f.in, f.method, f.marker = in, "deliver", ctx.Value(workflowMarkerKey{})
	return f.fact, f.err
}

func TestWorkflowPort21_RetailAuthorityAndStock(t *testing.T) {
	c := &workflowRetailCommerce{authority: &commerce.ConsignmentSaleAuthority{
		UnitID: "unit", AgreementID: "agreement", OrderID: "accepted-order", OrderLineID: "line", VehicleID: "vehicle", VIN: "VIN",
		SupplierCompanyID: "supplier", DealerCompanyID: "dealer", HandoverFactID: "handover-fact", Revision: 8,
	}}
	s := &workflowRetailStock{fact: "customer-delivery-fact"}
	a := retailConsignment{commerce: c, stock: s}
	var _ retail.Consignment = a
	ctx := context.WithValue(context.Background(), workflowMarkerKey{}, "bound transaction")
	at := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	u, err := a.CheckConsignmentSale(ctx, "dealer", "vehicle")
	if err != nil || u == nil || u.OrderID != "accepted-order" || u.SupplierCompanyID != "supplier" || u.DealerCompanyID != "dealer" || u.AgreementID != "agreement" || u.VIN != "VIN" || u.Revision != 8 || c.marker != "bound transaction" {
		t.Fatalf("checked authority=%+v call=%+v err=%v", u, c, err)
	}
	u, err = a.ReserveConsignmentSale(ctx, "dealer", "vehicle", "retail-deal", "actor")
	if err != nil || c.method != "reserve" || c.seller != "dealer" || c.vehicle != "vehicle" || c.deal != "retail-deal" || c.actor != "actor" {
		t.Fatalf("reserve call=%+v err=%v", c, err)
	}
	if err = a.HoldConsignmentSale(ctx, u, "retail-deal", "actor", at); err != nil {
		t.Fatal(err)
	}
	in := s.in
	if s.method != "hold" || s.marker != "bound transaction" || in.UnitID != "unit" || in.AgreementID != "agreement" || in.OrderID != "accepted-order" || in.OrderLineID != "line" || in.VehicleID != "vehicle" || in.VIN != "VIN" || in.SupplierCompanyID != "supplier" || in.DealerCompanyID != "dealer" || in.HandoverFactID != "handover-fact" || in.Revision != 8 || in.RetailDealID != "retail-deal" || in.ActorUserID != "actor" || !in.At.Equal(at) {
		t.Fatalf("hold input=%+v", in)
	}
	if _, err = a.ConsignmentSaleForDeal(ctx, "dealer", "vehicle", "retail-deal"); err != nil || c.method != "for-deal" {
		t.Fatalf("for deal=%+v err=%v", c, err)
	}
	fact, err := a.DeliverConsignmentSale(ctx, u, "retail-deal", "actor", at)
	if err != nil || fact != "customer-delivery-fact" || s.method != "deliver" || s.in != in {
		t.Fatalf("delivery=%s %+v err=%v", fact, s, err)
	}
	if err = a.CompleteConsignmentSale(ctx, "dealer", "vehicle", "retail-deal", fact, "actor", at); err != nil || c.ref != fact || c.method != "complete" || c.marker != "bound transaction" || !c.at.Equal(at) {
		t.Fatalf("completion=%+v err=%v", c, err)
	}
	if err = a.CancelConsignmentSale(ctx, "dealer", "vehicle", "retail-deal", "actor"); err != nil || c.method != "cancel" || c.actor != "actor" {
		t.Fatalf("cancellation=%+v err=%v", c, err)
	}
}

func TestWorkflowPort21_RetailErrorsAndAbsence(t *testing.T) {
	c, s := &workflowRetailCommerce{}, &workflowRetailStock{}
	a := retailConsignment{commerce: c, stock: s}
	ctx := context.WithValue(context.Background(), workflowMarkerKey{}, "bound transaction")
	if u, err := a.CheckConsignmentSale(ctx, "dealer", "vehicle"); err != nil || u != nil {
		t.Fatalf("absence=%+v err=%v", u, err)
	}
	want := errors.New("trusted service failed")
	c.err = want
	if _, err := a.ReserveConsignmentSale(ctx, "dealer", "vehicle", "deal", "actor"); !errors.Is(err, want) {
		t.Fatalf("commerce error=%v", err)
	}
	c.err = nil
	s.err = want
	u := &retail.ConsignmentSale{UnitID: "unit"}
	if err := a.HoldConsignmentSale(ctx, u, "deal", "actor", time.Now()); !errors.Is(err, want) {
		t.Fatalf("hold error=%v", err)
	}
	if _, err := a.DeliverConsignmentSale(ctx, u, "deal", "actor", time.Now()); !errors.Is(err, want) {
		t.Fatalf("delivery error=%v", err)
	}
	// The app constructor accepts the same adapter used above; legacy callers
	// remain valid through the optional final argument.
	if m := retail.New(nil, time.Now, nil, nil, nil, nil, a); m == nil || m.Deals == nil {
		t.Fatal("retail constructor dropped consignment adapter")
	}
}

type workflowPassportBinder struct {
	owner, fileID, resourceType, resourceID string
}

type workflowContractCompanies struct{ gotAuthorized, gotCompany string }

func (f *workflowContractCompanies) IsMember(context.Context, string, string) (bool, error) {
	return false, nil
}

func (f *workflowContractCompanies) BranchOf(context.Context, string, string) (bool, error) {
	return false, nil
}

func (f *workflowContractCompanies) ContractRequisitesFor(_ context.Context, authorized, company string) (*identity.ContractRequisites, error) {
	f.gotAuthorized, f.gotCompany = authorized, company
	return &identity.ContractRequisites{ID: company, TaxID: "private"}, nil
}

func TestWorkflowPort11(t *testing.T) {
	f := &workflowContractCompanies{}
	a := retailCompanies{s: f}
	if _, err := a.ContractRequisitesFor(context.Background(), "seller", "other"); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("unrelated=%v", err)
	}
	if f.gotAuthorized != "" || f.gotCompany != "" {
		t.Fatalf("unexpected identity lookup=%s/%s", f.gotAuthorized, f.gotCompany)
	}
	if _, err := a.ContractRequisitesFor(context.Background(), "seller", "seller"); err != nil {
		t.Fatal(err)
	}
}

func (f *workflowPassportBinder) Share(context.Context, string, string, string, string, string) (*documents.File, error) {
	return &documents.File{}, nil
}

func (f *workflowPassportBinder) BindPassport(_ context.Context, owner, fileID, resourceType, resourceID string) (*documents.File, error) {
	f.owner, f.fileID, f.resourceType, f.resourceID = owner, fileID, resourceType, resourceID
	return &documents.File{}, nil
}

func TestWorkflowPort08(t *testing.T) {
	binder := &workflowPassportBinder{}
	files := fileShares{s: binder}
	if err := files.BindPassport(context.Background(), "seller", "passport-file", "retail.customer", "customer"); err != nil {
		t.Fatal(err)
	}
	if binder.owner != "seller" || binder.fileID != "passport-file" || binder.resourceType != "retail.customer" || binder.resourceID != "customer" {
		t.Fatalf("passport bind = %+v", binder)
	}
}
