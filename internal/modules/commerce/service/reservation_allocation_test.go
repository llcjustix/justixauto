package service

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/jsonx"
	"justixauto/internal/pkg/money"
)

type conversionStore struct {
	*reservationTestStore
	orders map[string]model.Order
	allocations []model.Allocation
	credits map[string]model.ReservationCredit
	evidence []model.ReservationEvidence
	failOrder, failAllocations, failCredit bool
	agreements map[string]model.ConsignmentAgreement
	units map[string]model.ConsignmentUnit
	failAgreement, failUnit bool
}
func (s *conversionStore) InTx(_ context.Context, fn func(Store) error) error {
	reservations,orders,credits:=maps.Clone(s.reservations),maps.Clone(s.orders),maps.Clone(s.credits)
	agreements,units:=maps.Clone(s.agreements),maps.Clone(s.units)
	holds:=maps.Clone(s.holds)
	claims:=map[string][]model.SupplyClaim{}
	for id,rows:=range s.claims { claims[id]=slices.Clone(rows) }
	associations,allocations,evidence,events:=slices.Clone(s.associations),slices.Clone(s.allocations),slices.Clone(s.evidence),s.events
	if err:=fn(s);err!=nil { s.reservations,s.orders,s.credits=reservations,orders,credits;s.agreements,s.units=agreements,units;s.holds,s.claims=holds,claims;s.associations,s.allocations,s.evidence,s.events=associations,allocations,evidence,events;return err }
	return nil
}
func (s *conversionStore) Deals() DealRepository {return conversionDeals{s:s}}
func (s *conversionStore) Fulfilment() FulfilmentRepository {return conversionFulfilment{s:s}}
func (s *conversionStore) ReservationPayments() ReservationPaymentRepository {return conversionMoney{emptyLifecycleMoney:emptyLifecycleMoney{},s:s}}
func (s *conversionStore) Invoices() InvoiceRepository {return conversionInvoices{}}
func (s *conversionStore) Consignments() ConsignmentRepository {return conversionConsignments{s:s}}
type conversionDeals struct {DealRepository;s *conversionStore}
func (r conversionDeals) CreateOrder(_ context.Context,o *model.Order) error {if r.s.failOrder{return errors.New("order create failed")};r.s.orders[o.ID]=*o;return nil}
func (r conversionDeals) Order(_ context.Context,companyID,id string)(*model.Order,error){o,ok:=r.s.orders[id];if !ok||o.Party(companyID)==""{return nil,apperr.ErrNotFound};return &o,nil}
func (r conversionDeals) UpdateOrder(_ context.Context,o *model.Order,expected int64)error{if r.s.failOrder{return errors.New("order update failed")};old,ok:=r.s.orders[o.ID];if !ok||old.Version!=expected{return apperr.ErrStale};o.Version++;r.s.orders[o.ID]=*o;return nil}
type conversionFulfilment struct {FulfilmentRepository;s *conversionStore}
func (r conversionFulfilment) AddAllocations(_ context.Context,rows []model.Allocation)error{if r.s.failAllocations{return errors.New("allocation write failed")};r.s.allocations=append(r.s.allocations,rows...);return nil}
func (r conversionFulfilment) Allocations(_ context.Context,orderID string)([]model.Allocation,error){var out []model.Allocation;for _,a:=range r.s.allocations{if a.OrderID==orderID{out=append(out,a)}};return out,nil}
func (r conversionFulfilment) ShipmentLines(context.Context,string)([]model.ShipmentLine,error){return nil,nil}
func (r conversionFulfilment) SetAllocationStatus(_ context.Context,orderID string,_ []string,status string,_ *string)error{for i:=range r.s.allocations{if r.s.allocations[i].OrderID==orderID{r.s.allocations[i].Status=status}};return nil}
type conversionMoney struct {emptyLifecycleMoney;s *conversionStore}
func (r conversionMoney) Evidence(_ context.Context,id string)([]model.ReservationEvidence,error){var out []model.ReservationEvidence;for _,e:=range r.s.evidence{if e.ReservationID==id{out=append(out,e)}};return out,nil}
func (r conversionMoney) CreditForReservation(_ context.Context,id string)(*model.ReservationCredit,error){c,ok:=r.s.credits[id];if !ok{return nil,apperr.ErrNotFound};return &c,nil}
func (r conversionMoney) CreateCredit(_ context.Context,c *model.ReservationCredit)error{if r.s.failCredit{return errors.New("credit write failed")};if _,ok:=r.s.credits[c.ReservationID];ok{return apperr.ErrConflict};r.s.credits[c.ReservationID]=*c;return nil}
type conversionInvoices struct{InvoiceRepository}
func (conversionInvoices) ForOrder(context.Context,string)([]model.Invoice,error){return nil,nil}
type conversionStock struct{reservationTestStock}
func (s conversionStock) Reserve(_ context.Context,_ string,orderID string,ids []string)error{if s.store.failStock{return errors.New("stock failed")};for _,id:=range ids{if s.store.holds[id]!=""{return apperr.ErrConflict};s.store.holds[id]=orderID};return nil}
func (s conversionStock) Release(_ context.Context,orderID string,_ []string,_ string)error{for id,holder:=range s.store.holds{if holder==orderID{delete(s.store.holds,id)}};return nil}

func conversionFixture(t *testing.T)(*Reservation,*conversionStore,*auth.Principal,*auth.Principal,string,string,[]string){
	t.Helper();s,base,buyer,supplier,id,line,ids:=activeAllocationFixture(t)
	st:=&conversionStore{reservationTestStore:base,orders:map[string]model.Order{},credits:map[string]model.ReservationCredit{},agreements:map[string]model.ConsignmentAgreement{},units:map[string]model.ConsignmentUnit{}}
	stock:=conversionStock{s.reservationStock.(reservationTestStock)}
	s.Deps.store=st;s.Deps.stock=stock;s.reservationStock=stock
	return s,st,buyer,supplier,id,line,ids
}
func conversionProposal(st *conversionStore,line string,partial bool) ReservationContinuation {
	lot:=st.lots[st.claimsLotID()]
	return ReservationContinuation{Mode:"purchase",AllowPartialPurchaseHandover:partial,HandoverTerms:"supplier handover on agreed receipt",Terms:model.Terms{Route:"local",PaymentSchedule:[]model.Installment{},Lines:[]model.Line{{OfferLineID:line,ModelID:lot.ModelID,ModelSpecificationVersion:lot.ModelSpecificationVersion,ExteriorColor:lot.ExteriorColor,InteriorColor:lot.InteriorColor,Quantity:4,UnitPrice:money.Money{AmountMinor:"2500",Currency:"USD"}}}}}
}

func TestReservationConversionNoSomeAllVINs(t *testing.T) {
	ctx:=context.Background()
	for _,pre:=range []int{0,2,4} {t.Run(string(rune('0'+pre)),func(t *testing.T){
		s,st,buyer,supplier,id,line,ids:=conversionFixture(t)
		if pre>0 {items:=make([]ReservationAllocationItem,pre);for i:=range items{items[i]=ReservationAllocationItem{LineID:line,VehicleID:ids[i]}};if _,err:=s.Allocate(ctx,supplier,id,st.reservations[id].Version,items);err!=nil{t.Fatal(err)}}
		booking:=st.reservations[id].CurrentDigest
		proposal:=conversionProposal(st,line,false)
		v,err:=s.ProposeContinuation(ctx,buyer,id,st.reservations[id].Version,proposal);if err!=nil{t.Fatal(err)}
		v,err=s.AcceptContinuation(ctx,supplier,id,v.Reservation.Version,v.Reservation.ContinuationProposedDigest,v.Reservation.ContinuationProposedRevision);if err!=nil{t.Fatal(err)}
		converted,err:=s.Convert(ctx,buyer,id,v.Reservation.Version,v.Reservation.ContinuationAcceptedDigest,v.Reservation.ContinuationAcceptedRevision);if err!=nil{t.Fatal(err)}
		o:=converted.Order
		if converted.Reservation.Reservation.Status!=model.ReservationConverted||converted.Reservation.Reservation.CurrentDigest!=booking||len(st.allocations)!=pre||o.DecodeTerms().Reservation==nil||o.DecodeTerms().Reservation.Mode!="purchase"{t.Fatalf("conversion=%+v allocations=%d",converted,len(st.allocations))}
		if _,err:=s.Convert(ctx,buyer,id,converted.Reservation.Reservation.Version,v.Reservation.ContinuationAcceptedDigest,v.Reservation.ContinuationAcceptedRevision);err==nil{t.Fatal("second conversion accepted")}
		if pre<4 {items:=make([]AllocationItem,4-pre);for i:=range items{items[i]=AllocationItem{OrderLineID:o.DecodeTerms().Lines[0].LineID,VehicleID:ids[pre+i]}};f:=NewFulfilment(s.Deps);if _,err:=f.Allocate(ctx,supplier,o.ID,o.Version,items);err!=nil{t.Fatal(err)}}
		cap,err:=st.Supply().Capacity(ctx,st.claimsLotID());if err!=nil{t.Fatal(err)}
		if cap.Available!=6||cap.Held!=0||cap.Materialized!=4||len(st.allocations)!=4{t.Fatalf("capacity=%+v allocations=%d",cap,len(st.allocations))}
		for i:=0;i<4;i++{if st.holds[ids[i]]!=o.ID{t.Fatalf("holder[%d]=%s",i,st.holds[ids[i]])}}
	})}
}

func TestReservationConversionAgreementAndCancellation(t *testing.T) {
	ctx:=context.Background();s,st,buyer,supplier,id,line,_:=conversionFixture(t)
	proposal:=conversionProposal(st,line,true)
	proposal.MinimumPaid=&money.Money{AmountMinor:"2000",Currency:"USD"}
	proposal.HandoverConditions="supplier confirms partial handover in writing"
	v,err:=s.ProposeContinuation(ctx,buyer,id,2,proposal);if err!=nil{t.Fatal(err)}
	if _,err:=s.AcceptContinuation(ctx,buyer,id,v.Reservation.Version,v.Reservation.ContinuationProposedDigest,v.Reservation.ContinuationProposedRevision);err==nil{t.Fatal("proposer accepted")}
	if _,err:=s.AcceptContinuation(ctx,supplier,id,v.Reservation.Version,"wrong",v.Reservation.ContinuationProposedRevision);err==nil{t.Fatal("wrong digest accepted")}
	v,err=s.AcceptContinuation(ctx,supplier,id,v.Reservation.Version,v.Reservation.ContinuationProposedDigest,v.Reservation.ContinuationProposedRevision);if err!=nil{t.Fatal(err)}
	if _,err:=s.Convert(ctx,buyer,id,v.Reservation.Version,"wrong",v.Reservation.ContinuationAcceptedRevision);err==nil{t.Fatal("wrong conversion digest accepted")}
	converted,err:=s.Convert(ctx,buyer,id,v.Reservation.Version,v.Reservation.ContinuationAcceptedDigest,v.Reservation.ContinuationAcceptedRevision);if err!=nil{t.Fatal(err)}
	origin:=converted.Order.DecodeTerms().Reservation
	if origin==nil||!origin.AllowPartialPurchaseHandover||origin.MinimumPaid==nil||origin.MinimumPaid.AmountMinor!="2000"||origin.HandoverConditions!=proposal.HandoverConditions{t.Fatalf("origin=%+v",origin)}
	deal:=NewDeal(s.Deps,nil)
	order,err:=deal.CancelOrder(ctx,buyer,converted.Order.ID,converted.Order.Version,"agreed cancellation");if err!=nil{t.Fatal(err)}
	if order.Status!=model.OrderCancelled{t.Fatalf("order=%+v",order)}
	cap,_:=st.Supply().Capacity(ctx,st.claimsLotID())
	if cap.Available!=10||cap.Held!=0||cap.Materialized!=0{t.Fatalf("capacity=%+v",cap)}
	if _,err:=deal.CancelOrder(ctx,buyer,order.ID,order.Version,"again");err==nil{t.Fatal("repeat cancellation accepted")}
}

func TestReservationConversionCreditAndRollback(t *testing.T) {
	ctx:=context.Background();s,st,buyer,supplier,id,line,_:=conversionFixture(t)
	r:=st.reservations[id];snap,err:=decodeReservationSnapshot(r.CurrentTerms);if err!=nil{t.Fatal(err)}
	advance:=money.Money{AmountMinor:"2000",Currency:"USD"};snap.Terms.Advance=&advance
	due:=time.Date(2026,10,15,0,0,0,0,time.UTC);snap.Terms.AdvanceDueAt=&due
	r.CurrentDigest,r.CurrentTerms=reservationDigest(snap);r.AdvanceDueAt=&due;st.reservations[id]=r
	st.evidence=[]model.ReservationEvidence{{ID:uuid.NewString(),ReservationID:id,TermsDigest:r.CurrentDigest,TermsRevision:r.CurrentRevision,AmountMinor:"2000",Currency:"USD",Status:"accepted"}}
	v,err:=s.ProposeContinuation(ctx,buyer,id,2,conversionProposal(st,line,false));if err!=nil{t.Fatal(err)}
	v,err=s.AcceptContinuation(ctx,supplier,id,v.Reservation.Version,v.Reservation.ContinuationProposedDigest,v.Reservation.ContinuationProposedRevision);if err!=nil{t.Fatal(err)}
	st.failCredit=true
	if _,err:=s.Convert(ctx,buyer,id,v.Reservation.Version,v.Reservation.ContinuationAcceptedDigest,v.Reservation.ContinuationAcceptedRevision);err==nil{t.Fatal("credit failure accepted")}
	if len(st.orders)!=0||len(st.credits)!=0||st.reservations[id].OrderID!=nil{t.Fatalf("partial conversion orders=%d credits=%d source=%+v",len(st.orders),len(st.credits),st.reservations[id])}
	for _,c:=range st.claims[st.claimsLotID()]{if c.OrderID!=nil{t.Fatalf("claim moved despite rollback: %+v",c)}}
	st.failCredit=false
	converted,err:=s.Convert(ctx,buyer,id,v.Reservation.Version,v.Reservation.ContinuationAcceptedDigest,v.Reservation.ContinuationAcceptedRevision);if err!=nil{t.Fatal(err)}
	credit,ok:=st.credits[id];if !ok||credit.OrderID!=converted.Order.ID||credit.AmountMinor!="2000"||len(st.credits)!=1{t.Fatalf("credit=%+v exists=%v",credit,ok)}
	deal:=NewDeal(s.Deps,nil)
	if _,err:=deal.CancelOrder(ctx,buyer,converted.Order.ID,converted.Order.Version,"attempt");err==nil{t.Fatal("credited order cancellation accepted")}
}

func TestReservationConversionTransferFailuresRollback(t *testing.T) {
	ctx:=context.Background()
	for _,failure:=range []string{"stock","claim","allocations","order","event"}{t.Run(failure,func(t *testing.T){
		s,st,buyer,supplier,id,line,ids:=conversionFixture(t)
		if _,err:=s.Allocate(ctx,supplier,id,2,[]ReservationAllocationItem{{LineID:line,VehicleID:ids[0]}});err!=nil{t.Fatal(err)}
		v,err:=s.ProposeContinuation(ctx,buyer,id,3,conversionProposal(st,line,false));if err!=nil{t.Fatal(err)}
		v,err=s.AcceptContinuation(ctx,supplier,id,v.Reservation.Version,v.Reservation.ContinuationProposedDigest,v.Reservation.ContinuationProposedRevision);if err!=nil{t.Fatal(err)}
		switch failure {case "stock":st.failStock=true;case "claim":st.failClaim=true;case "allocations":st.failAllocations=true;case "order":st.failOrder=true;case "event":st.failEvent=true}
		if _,err:=s.Convert(ctx,buyer,id,v.Reservation.Version,v.Reservation.ContinuationAcceptedDigest,v.Reservation.ContinuationAcceptedRevision);err==nil{t.Fatal("failed stage converted")}
		cap,_:=st.Supply().Capacity(ctx,st.claimsLotID())
		if cap.Available!=6||cap.Held!=3||cap.Materialized!=1||len(st.orders)!=0||len(st.allocations)!=0||len(st.associations)!=1||st.holds[ids[0]]!=id||st.reservations[id].OrderID!=nil{t.Fatalf("partial rollback cap=%+v orders=%d allocs=%d associations=%d holder=%s",cap,len(st.orders),len(st.allocations),len(st.associations),st.holds[ids[0]])}
	})}
}

func TestReservationOrderAllocationOrdinaryOrderCompatibility(t *testing.T) {
	ctx:=context.Background();s,st,_,supplier,_,_,ids:=conversionFixture(t)
	lot:=st.lots[st.claimsLotID()];lineID:=uuid.NewString();orderID:=uuid.NewString()
	raw,_:=json.Marshal(model.Terms{Lines:[]model.Line{{LineID:lineID,ModelID:lot.ModelID,Quantity:1,ExteriorColor:lot.ExteriorColor,InteriorColor:lot.InteriorColor}}})
	st.orders[orderID]=model.Order{ID:orderID,Source:"direct",BuyerCompanyID:uuid.NewString(),SupplierCompanyID:supplier.CompanyID,Terms:raw,Status:model.OrderAccepted,Version:1}
	f:=NewFulfilment(s.Deps)
	if _,err:=f.Allocate(ctx,supplier,orderID,1,[]AllocationItem{{OrderLineID:lineID,VehicleID:ids[0]}});err!=nil{t.Fatal(err)}
	cap,_:=st.Supply().Capacity(ctx,st.claimsLotID())
	if cap.Available!=6||cap.Held!=4||cap.Materialized!=0||len(st.allocations)!=1||st.holds[ids[0]]!=orderID{t.Fatalf("ordinary allocation cap=%+v allocations=%d holder=%s",cap,len(st.allocations),st.holds[ids[0]])}
}

func TestReservationOrderAllocationCancelKeepsMaterialized(t *testing.T) {
	ctx:=context.Background();s,st,buyer,supplier,id,line,ids:=conversionFixture(t)
	if _,err:=s.Allocate(ctx,supplier,id,2,[]ReservationAllocationItem{{LineID:line,VehicleID:ids[0]}});err!=nil{t.Fatal(err)}
	v,err:=s.ProposeContinuation(ctx,buyer,id,3,conversionProposal(st,line,false));if err!=nil{t.Fatal(err)}
	v,err=s.AcceptContinuation(ctx,supplier,id,v.Reservation.Version,v.Reservation.ContinuationProposedDigest,v.Reservation.ContinuationProposedRevision);if err!=nil{t.Fatal(err)}
	converted,err:=s.Convert(ctx,buyer,id,v.Reservation.Version,v.Reservation.ContinuationAcceptedDigest,v.Reservation.ContinuationAcceptedRevision);if err!=nil{t.Fatal(err)}
	deal:=NewDeal(s.Deps,nil)
	if _,err:=deal.CancelOrder(ctx,buyer,converted.Order.ID,converted.Order.Version,"agreed cancellation");err!=nil{t.Fatal(err)}
	cap,_:=st.Supply().Capacity(ctx,st.claimsLotID())
	if cap.Available!=9||cap.Held!=0||cap.Materialized!=1||st.holds[ids[0]]!=""{t.Fatalf("cancelled cap=%+v holder=%s",cap,st.holds[ids[0]])}
	for _,a:=range st.associations{if a.Current{t.Fatalf("association still current: %+v",a)}}
}

func TestReservationConversionConsignmentAndChangedProposal(t *testing.T) {
	ctx:=context.Background();s,st,buyer,supplier,id,line,_:=conversionFixture(t)
	first:=conversionProposal(st,line,false);first.Mode="consignment"
	v,err:=s.ProposeContinuation(ctx,buyer,id,2,first);if err!=nil{t.Fatal(err)}
	v,err=s.AcceptContinuation(ctx,supplier,id,v.Reservation.Version,v.Reservation.ContinuationProposedDigest,v.Reservation.ContinuationProposedRevision);if err!=nil{t.Fatal(err)}
	oldDigest,oldRevision:=v.Reservation.ContinuationAcceptedDigest,v.Reservation.ContinuationAcceptedRevision
	changed:=first;changed.HandoverTerms="supplier retains title during custody"
	v,err=s.ProposeContinuation(ctx,buyer,id,v.Reservation.Version,changed);if err!=nil{t.Fatal(err)}
	if v.ContinuationAccepted!=nil||v.Reservation.ContinuationAcceptedDigest!=""{t.Fatal("old acceptance survived changed proposal")}
	if _,err:=s.Convert(ctx,buyer,id,v.Reservation.Version,oldDigest,oldRevision);err==nil{t.Fatal("old agreement converted")}
	v,err=s.AcceptContinuation(ctx,supplier,id,v.Reservation.Version,v.Reservation.ContinuationProposedDigest,v.Reservation.ContinuationProposedRevision);if err!=nil{t.Fatal(err)}
	converted,err:=s.Convert(ctx,buyer,id,v.Reservation.Version,v.Reservation.ContinuationAcceptedDigest,v.Reservation.ContinuationAcceptedRevision);if err!=nil{t.Fatal(err)}
	if len(st.allocations)!=0||converted.Order.DecodeTerms().Reservation.Mode!="consignment"||len(st.agreements)!=1||len(st.units)!=0{t.Fatalf("consignment conversion=%+v allocations=%d agreements=%d units=%d",converted.Order.DecodeTerms().Reservation,len(st.allocations),len(st.agreements),len(st.units))}
	cap,_:=st.Supply().Capacity(ctx,st.claimsLotID())
	if cap.Available!=6||cap.Held!=4||cap.Materialized!=0{t.Fatalf("future consignment capacity=%+v",cap)}
}

func TestReservationOrderAllocationRejectsWrongSource(t *testing.T) {
	ctx:=context.Background();s,st,buyer,supplier,id,line,ids:=conversionFixture(t)
	v,err:=s.ProposeContinuation(ctx,buyer,id,2,conversionProposal(st,line,false));if err!=nil{t.Fatal(err)}
	v,err=s.AcceptContinuation(ctx,supplier,id,v.Reservation.Version,v.Reservation.ContinuationProposedDigest,v.Reservation.ContinuationProposedRevision);if err!=nil{t.Fatal(err)}
	converted,err:=s.Convert(ctx,buyer,id,v.Reservation.Version,v.Reservation.ContinuationAcceptedDigest,v.Reservation.ContinuationAcceptedRevision);if err!=nil{t.Fatal(err)}
	order:=converted.Order
	stock:=s.reservationStock.(conversionStock)
	stock.vehicles[ids[0]].ModelSpecificationVersion="wrong"
	f:=NewFulfilment(s.Deps)
	if _,err:=f.Allocate(ctx,supplier,order.ID,order.Version,[]AllocationItem{{OrderLineID:order.DecodeTerms().Lines[0].LineID,VehicleID:ids[0]}});err==nil{t.Fatal("wrong source specification allocated")}
	cap,_:=st.Supply().Capacity(ctx,st.claimsLotID())
	if cap.Available!=6||cap.Held!=4||cap.Materialized!=0||len(st.allocations)!=0||st.holds[ids[0]]!=""{t.Fatalf("wrong source mutated state: cap=%+v allocations=%d holder=%s",cap,len(st.allocations),st.holds[ids[0]])}
	broken:=st.reservations[id];broken.OrderID=nil;st.reservations[id]=broken
	if _,err:=f.Allocate(ctx,supplier,order.ID,order.Version,[]AllocationItem{{OrderLineID:order.DecodeTerms().Lines[0].LineID,VehicleID:ids[1]}});err==nil{t.Fatal("missing reservation source allocated")}
}

func activeAllocationFixture(t *testing.T) (*Reservation,*reservationTestStore,*auth.Principal,*auth.Principal,string,string,[]string) {
	t.Helper()
	ctx:=context.Background()
	s,st,dealer,supplier,lot,_:=reservationFixture()
	st.lots[lot].TotalQuantity=10
	stock:=s.reservationStock.(reservationTestStock)
	ids:=[]string{uuid.NewString(),uuid.NewString(),uuid.NewString(),uuid.NewString(),uuid.NewString()}
	for _,id:=range ids {stock.vehicles[id]=&StockVehicle{ID:id,VIN:id,ModelID:st.lots[lot].ModelID,ModelSpecificationVersion:"2",ExteriorColor:"white",InteriorColor:"black"}}
	v,err:=s.Request(ctx,dealer,ReservationRequest{SupplierCompanyID:supplier.CompanyID,Lines:[]ReservationLineInput{{SupplyLotID:&lot,Quantity:4}},DesiredTerms:reservationTerms()})
	if err!=nil{t.Fatal(err)}
	v,err=s.Confirm(ctx,supplier,v.Reservation.ID,v.Reservation.Version,ReservationConfirmation{Terms:reservationTerms()})
	if err!=nil{t.Fatal(err)}
	if v.Reservation.Status!=model.ReservationActive{t.Fatalf("status=%s",v.Reservation.Status)}
	return s,st,dealer,supplier,v.Reservation.ID,v.Current.Lines[0].LineID,ids
}

func TestReservationAllocationConservesClaimsAndHistory(t *testing.T) {
	ctx:=context.Background()
	s,st,dealer,supplier,id,line,ids:=activeAllocationFixture(t)
	before,err:=st.Supply().Capacity(ctx,st.claimsLotID())
	if err!=nil{t.Fatal(err)}
	if before.Available!=6||before.Held!=4||before.Materialized!=0{t.Fatalf("before=%+v",before)}
	view,err:=s.Allocate(ctx,supplier,id,2,[]ReservationAllocationItem{{LineID:line,VehicleID:ids[0]},{LineID:line,VehicleID:ids[1]}})
	if err!=nil{t.Fatal(err)}
	cap,err:=st.Supply().Capacity(ctx,st.claimsLotID())
	if err!=nil{t.Fatal(err)}
	if cap.Available!=6||cap.Held!=2||cap.Materialized!=2||len(view.Allocations)!=2||view.Revision!=3||st.holds[ids[0]]!=id||st.holds[ids[1]]!=id{t.Fatalf("after=%+v view=%+v holds=%v",cap,view,st.holds)}
	for _,c:=range st.claims[st.claimsLotID()] {if c.State=="materialized"&&(c.MaterializationRef==nil||c.MaterializationUnitOrdinal==nil||*c.MaterializationUnitOrdinal!=1){t.Fatalf("provenance=%+v",c)}}
	read,err:=s.Allocations(ctx,dealer,id)
	if err!=nil||len(read.Allocations)!=2{t.Fatalf("read=%+v err=%v",read,err)}
	if _,err=s.Allocate(ctx,supplier,id,2,[]ReservationAllocationItem{{LineID:line,VehicleID:ids[2]}});!errors.Is(err,apperr.ErrStale){t.Fatalf("stale=%v",err)}
	if _,err=s.Allocate(ctx,supplier,id,3,[]ReservationAllocationItem{{LineID:line,VehicleID:ids[0]}});err==nil{t.Fatal("duplicate VIN accepted")}
	if _,err=s.Allocate(ctx,supplier,id,3,[]ReservationAllocationItem{{LineID:line,VehicleID:ids[2]},{LineID:line,VehicleID:ids[2]}});err==nil{t.Fatal("duplicate request VIN accepted")}
	if _,err=s.Allocate(ctx,supplier,id,3,[]ReservationAllocationItem{{LineID:line,VehicleID:ids[2]},{LineID:line,VehicleID:ids[3]},{LineID:line,VehicleID:ids[4]}});err==nil{t.Fatal("over allocated source")}
	cap,_=st.Supply().Capacity(ctx,st.claimsLotID())
	if cap.Available!=6||cap.Held!=2||cap.Materialized!=2||len(st.associations)!=2||st.reservations[id].Version!=3{t.Fatalf("rejections changed state: %+v associations=%d",cap,len(st.associations))}
}

func (s *reservationTestStore) claimsLotID() string {for id:=range s.lots{return id};return ""}

func TestReservationAllocationRejectsWrongSourcesAndRollsBack(t *testing.T) {
	ctx:=context.Background()
	for _,name:=range []string{"outsider","line","model","specification","exterior","interior","owner","stock","claim","association","update","event","converted"} {
		t.Run(name,func(t *testing.T){
			s,st,_,supplier,id,line,ids:=activeAllocationFixture(t)
			stock:=s.reservationStock.(reservationTestStock)
			p:=supplier
			item:=ReservationAllocationItem{LineID:line,VehicleID:ids[0]}
			switch name {
			case "outsider": copy:=*supplier;copy.CompanyID=uuid.NewString();p=&copy
			case "line":item.LineID=uuid.NewString()
			case "model":stock.vehicles[ids[0]].ModelID=uuid.NewString()
			case "specification":stock.vehicles[ids[0]].ModelSpecificationVersion="3"
			case "exterior":stock.vehicles[ids[0]].ExteriorColor="red"
			case "interior":stock.vehicles[ids[0]].InteriorColor="tan"
			case "owner":stock.owners=map[string]string{ids[0]:uuid.NewString()};s.reservationStock=stock
			case "stock":st.failStock=true
			case "claim":st.failClaim=true
			case "association":st.failAssociation=true
			case "update":st.failUpdate=true
			case "event":st.failEvent=true
			case "converted":x:=st.reservations[id];orderID:=uuid.NewString();x.OrderID=&orderID;st.reservations[id]=x
			}
			if _,err:=s.Allocate(ctx,p,id,2,[]ReservationAllocationItem{item});err==nil{t.Fatal("invalid allocation accepted")}
			cap,err:=st.Supply().Capacity(ctx,st.claimsLotID())
			if err!=nil{t.Fatal(err)}
			if cap.Available!=6||cap.Held!=4||cap.Materialized!=0||len(st.associations)!=0||st.holds[ids[0]]!=""||st.reservations[id].Version!=2||st.events!=2{t.Fatalf("failure mutated state: cap=%+v holders=%v associations=%v revision=%d events=%d",cap,st.holds,st.associations,st.reservations[id].Version,st.events)}
		})
	}
}

func TestReservationAllocationTransferKeepsClaimAndHolderHistory(t *testing.T) {
	ctx:=context.Background()
	s,st,_,supplier,id,line,ids:=activeAllocationFixture(t)
	if _,err:=s.Allocate(ctx,supplier,id,2,[]ReservationAllocationItem{{LineID:line,VehicleID:ids[0]},{LineID:line,VehicleID:ids[1]}});err!=nil{t.Fatal(err)}
	refs:=map[string]string{}
	for _,c:=range st.claims[st.claimsLotID()] {if c.MaterializationRef!=nil{refs[c.ID]=*c.MaterializationRef}}
	orderID,orderLine:=uuid.NewString(),uuid.NewString()
	var allocations []model.Allocation
	err:=st.InTx(ctx,func(store Store)error{
		r,_,err:=store.Reservations().ForUpdate(ctx,id);if err!=nil{return err}
		allocations,err=s.TransferObligations(ctx,store,r,orderID,map[string]string{line:orderLine});if err!=nil{return err}
		r.OrderID=&orderID
		return store.Reservations().Update(ctx,r,r.Version)
	})
	if err!=nil{t.Fatal(err)}
	if len(allocations)!=2||st.holds[ids[0]]!=orderID||st.holds[ids[1]]!=orderID||len(st.associations)!=4{t.Fatalf("transfer allocations=%d holders=%v history=%v",len(allocations),st.holds,st.associations)}
	for _,c:=range st.claims[st.claimsLotID()] {
		if c.OrderID==nil||*c.OrderID!=orderID||c.OrderLineID==nil||*c.OrderLineID!=orderLine {t.Fatalf("claim owner=%+v",c)}
		if ref,ok:=refs[c.ID];ok&&(c.MaterializationRef==nil||*c.MaterializationRef!=ref){t.Fatalf("first materialization changed: %+v",c)}
	}
	retired,current:=0,0
	for _,a:=range st.associations {if a.Current {current++;if a.HolderType!="commerce-order"{t.Fatalf("current=%+v",a)}} else {retired++;if a.HolderType!="commerce-reservation"{t.Fatalf("retired=%+v",a)}}}
	if retired!=2||current!=2{t.Fatalf("history retired/current=%d/%d",retired,current)}
	cap,_:=st.Supply().Capacity(ctx,st.claimsLotID())
	if cap.Available!=6||cap.Held!=2||cap.Materialized!=2{t.Fatalf("capacity after transfer=%+v",cap)}
	if err:=st.InTx(ctx,func(store Store)error{r,_,err:=store.Reservations().ForUpdate(ctx,id);if err!=nil{return err};_,err=s.TransferObligations(ctx,store,r,uuid.NewString(),map[string]string{line:uuid.NewString()});return err});err==nil{t.Fatal("second transfer accepted")}
}

func TestReservationAllocationTransferStockFailureRollsBack(t *testing.T) {
	ctx:=context.Background()
	s,st,_,supplier,id,line,ids:=activeAllocationFixture(t)
	if _,err:=s.Allocate(ctx,supplier,id,2,[]ReservationAllocationItem{{LineID:line,VehicleID:ids[0]}});err!=nil{t.Fatal(err)}
	st.failStock=true
	err:=st.InTx(ctx,func(store Store)error{r,_,err:=store.Reservations().ForUpdate(ctx,id);if err!=nil{return err};_,err=s.TransferObligations(ctx,store,r,uuid.NewString(),map[string]string{line:uuid.NewString()});return err})
	if err==nil{t.Fatal("stock failure accepted")}
	if st.holds[ids[0]]!=id||len(st.associations)!=1||st.reservations[id].OrderID!=nil{t.Fatalf("rollback holders=%v associations=%v reservation=%+v",st.holds,st.associations,st.reservations[id])}
	for _,c:=range st.claims[st.claimsLotID()]{if c.OrderID!=nil{t.Fatalf("claim transferred despite rollback: %+v",c)}}
}

func TestReservationOrderAllocationReusesConsumedClaim(t *testing.T) {
	ctx:=context.Background()
	s,st,_,supplier,id,line,ids:=activeAllocationFixture(t)
	if _,err:=s.Allocate(ctx,supplier,id,2,[]ReservationAllocationItem{{LineID:line,VehicleID:ids[0]}});err!=nil{t.Fatal(err)}
	orderID,orderLine:=uuid.NewString(),uuid.NewString()
	err:=st.InTx(ctx,func(store Store)error{r,_,err:=store.Reservations().ForUpdate(ctx,id);if err!=nil{return err};_,err=s.TransferObligations(ctx,store,r,orderID,map[string]string{line:orderLine});if err!=nil{return err};r.OrderID=&orderID;r.Status=model.ReservationConverted;return store.Reservations().Update(ctx,r,r.Version)})
	if err!=nil{t.Fatal(err)}
	var consumedID,firstRef,currentID string
	for _,c:=range st.claims[st.claimsLotID()]{if c.State=="materialized"{consumedID=c.ID;firstRef=*c.MaterializationRef}}
	for _,a:=range st.associations{if a.ClaimID==consumedID&&a.Current{currentID=a.ID}}
	if err:=st.Supply().RetireClaimAllocation(ctx,consumedID,currentID);err!=nil{t.Fatal(err)}
	delete(st.holds,ids[0])
	provenance:=&model.ReservationOrderTerms{ReservationID:id,BookingDigest:st.reservations[id].CurrentDigest}
	raw,_:=json.Marshal(model.Terms{Reservation:provenance,Lines:[]model.Line{{LineID:orderLine,OfferLineID:line}}})
	order:=&model.Order{ID:orderID,Source:"reservation",SupplierCompanyID:supplier.CompanyID,BuyerCompanyID:st.reservations[id].BuyerCompanyID,Terms:raw}
	newAllocation:=model.Allocation{ID:uuid.NewString(),OrderID:orderID,LineID:orderLine,VehicleID:ids[1],VIN:ids[1],Status:"allocated"}
	if err:=st.InTx(ctx,func(store Store)error{return s.Deps.materializeOrderClaims(ctx,store,order,[]model.Allocation{newAllocation})});err!=nil{t.Fatal(err)}
	cap,_:=st.Supply().Capacity(ctx,st.claimsLotID())
	if cap.Available!=6||cap.Held!=3||cap.Materialized!=1{t.Fatalf("reassociation changed capacity: %+v",cap)}
	for _,c:=range st.claims[st.claimsLotID()]{if c.ID==consumedID&&(c.MaterializationRef==nil||*c.MaterializationRef!=firstRef){t.Fatalf("first provenance changed: %+v",c)}}
	if len(st.associations)!=3||!st.associations[2].Current||st.associations[2].ClaimID!=consumedID||st.associations[2].VehicleID!=ids[1]{t.Fatalf("replacement history=%+v",st.associations)}
	next:=model.Allocation{ID:uuid.NewString(),OrderID:orderID,LineID:orderLine,VehicleID:ids[2],VIN:ids[2],Status:"allocated"}
	if err:=st.InTx(ctx,func(store Store)error{return s.Deps.materializeOrderClaims(ctx,store,order,[]model.Allocation{next})});err!=nil{t.Fatal(err)}
	cap,_=st.Supply().Capacity(ctx,st.claimsLotID())
	if cap.Available!=6||cap.Held!=2||cap.Materialized!=2{t.Fatalf("new order VIN capacity=%+v",cap)}
	if err:=st.InTx(ctx,func(store Store)error{return s.Deps.materializeOrderClaims(ctx,store,order,[]model.Allocation{next})});err==nil{t.Fatal("duplicate order allocation accepted")}
}

type receiptTestStore struct {
	*conversionStore
	shipments map[string]model.Shipment
	shipmentLines []model.ShipmentLine
	batches map[string]ReceiptBatch
	deliveries []Delivery
	failDelivery, failShipment, failShipmentLines bool
}
type receiptTxKey struct{}
func (s *receiptTestStore) Bind(ctx context.Context) context.Context { return context.WithValue(ctx, receiptTxKey{}, true) }
func (s *receiptTestStore) InTx(ctx context.Context, fn func(Store) error) error {
	shipments, lines, batches, deliveries := maps.Clone(s.shipments), slices.Clone(s.shipmentLines), maps.Clone(s.batches), slices.Clone(s.deliveries)
	err := s.conversionStore.InTx(ctx, func(Store) error { return fn(s) })
	if err != nil { s.shipments, s.shipmentLines, s.batches, s.deliveries = shipments, lines, batches, deliveries }
	return err
}
func (s *receiptTestStore) Fulfilment() FulfilmentRepository { return receiptFulfilment{conversionFulfilment: conversionFulfilment{s: s.conversionStore}, s: s} }
type receiptFulfilment struct { conversionFulfilment; s *receiptTestStore }
func (r receiptFulfilment) CreateShipment(_ context.Context, sh *model.Shipment) error {
	if r.s.failShipment { return errors.New("shipment write failed") }
	r.s.shipments[sh.ID] = *sh
	return nil
}
func (r receiptFulfilment) AddShipmentLines(_ context.Context, rows []model.ShipmentLine) error {
	if r.s.failShipmentLines { return errors.New("shipment lines write failed") }
	r.s.shipmentLines = append(r.s.shipmentLines, rows...)
	return nil
}
func (r receiptFulfilment) ShipmentLines(_ context.Context, orderID string) ([]model.ShipmentLine, error) {
	var out []model.ShipmentLine
	for _, row := range r.s.shipmentLines { if row.OrderID == orderID { out = append(out, row) } }
	return out, nil
}
type receiptStock struct { conversionStock; s *receiptTestStore }
func (s receiptStock) Deliver(ctx context.Context, d Delivery) (string, error) {
	if ctx.Value(receiptTxKey{}) != true { return "", errors.New("delivery is not bound to commerce transaction") }
	if s.s.failDelivery { return "", errors.New("delivery failed") }
	id := uuid.NewString()
	s.s.batches[id] = ReceiptBatch{ID: id, WarehouseID: d.ToWarehouseID, ModelID: d.ModelID, ModelSpecificationVersion: d.ModelSpecificationVersion, ExteriorColor: d.ExteriorColor, InteriorColor: d.InteriorColor, ConfirmedQuantity: d.Quantity, UnidentifiedCount: d.Quantity, Revision: 1}
	s.s.deliveries = append(s.s.deliveries, d)
	return id, nil
}
type receiptCatalog struct { line model.Line }
func (c receiptCatalog) Model(_ context.Context, id string) (*Model, error) {
	if id != c.line.ModelID { return nil, apperr.ErrNotFound }
	return &Model{ID: id, CurrentSpecificationVersion: c.line.ModelSpecificationVersion, Specifications: []ModelSpecification{{Version: c.line.ModelSpecificationVersion, ExteriorColors: []string{c.line.ExteriorColor}, InteriorColors: []string{c.line.InteriorColor}}}}, nil
}
func receiptFixture(t *testing.T, preallocated int) (*Fulfilment, *receiptTestStore, *auth.Principal, *model.Order, []string) {
	t.Helper()
	ctx := context.Background()
	reservation, base, buyer, supplier, id, sourceLine, ids := conversionFixture(t)
	if preallocated > 0 {
		items := make([]ReservationAllocationItem, preallocated)
		for i := range items { items[i] = ReservationAllocationItem{LineID: sourceLine, VehicleID: ids[i]} }
		if _, err := reservation.Allocate(ctx, supplier, id, base.reservations[id].Version, items); err != nil { t.Fatal(err) }
	}
	v, err := reservation.ProposeContinuation(ctx, buyer, id, base.reservations[id].Version, conversionProposal(base, sourceLine, true))
	if err != nil { t.Fatal(err) }
	v, err = reservation.AcceptContinuation(ctx, supplier, id, v.Reservation.Version, v.Reservation.ContinuationProposedDigest, v.Reservation.ContinuationProposedRevision)
	if err != nil { t.Fatal(err) }
	converted, err := reservation.Convert(ctx, buyer, id, v.Reservation.Version, v.Reservation.ContinuationAcceptedDigest, v.Reservation.ContinuationAcceptedRevision)
	if err != nil { t.Fatal(err) }
	order := converted.Order
	warehouse := uuid.NewString()
	order.ReceivingWarehouseID = &warehouse
	base.orders[order.ID] = *order
	st := &receiptTestStore{conversionStore: base, shipments: map[string]model.Shipment{}, batches: map[string]ReceiptBatch{}}
	stock := reservation.reservationStock.(conversionStock)
	f := NewFulfilment(reservation.Deps)
	f.Deps.store = st
	f.Deps.stock = receiptStock{conversionStock: stock, s: st}
	f.Deps.catalog = receiptCatalog{line: order.DecodeTerms().Lines[0]}
	return f, st, supplier, order, ids
}
func receiptInput(order *model.Order, quantity int) ShipmentInput {
	return ShipmentInput{Route: "local", Lines: []ShipmentLineInput{{OrderLineID: order.DecodeTerms().Lines[0].LineID, Quantity: jsonx.Quantity(quantity)}}}
}
func receiptCapacity(t *testing.T, st *receiptTestStore, held, materialized int) {
	t.Helper()
	cap, err := st.Supply().Capacity(context.Background(), st.claimsLotID())
	if err != nil { t.Fatal(err) }
	if cap.Available != 6 || cap.Held != held || cap.Materialized != materialized { t.Fatalf("capacity=%+v, want 6/%d/%d", cap, held, materialized) }
}
func TestReservationQuantityReceiptPartialFullAndIdentify(t *testing.T) {
	ctx := context.Background()
	f, st, supplier, order, ids := receiptFixture(t, 0)
	first, err := f.Ship(ctx, supplier, order.ID, order.Version, receiptInput(order, 2))
	if err != nil { t.Fatal(err) }
	if first.Status != "received" || len(st.shipmentLines) != 1 || len(st.batches) != 1 { t.Fatalf("first receipt=%+v lines=%v batches=%v", first, st.shipmentLines, st.batches) }
	receiptCapacity(t, st, 2, 2)
	row := st.shipmentLines[0]
	if row.ShipmentID != first.ID || row.Quantity != 2 || row.ReceiptBatchID == "" { t.Fatalf("receipt line=%+v", row) }
	ordinals := map[int32]bool{}
	for _, claim := range st.claims[st.claimsLotID()] {
		if claim.MaterializationKind != "quantity-receipt" { continue }
		if claim.MaterializationRef == nil || *claim.MaterializationRef != row.ID || claim.MaterializationUnitOrdinal == nil || claim.OrderID == nil || *claim.OrderID != order.ID || claim.OrderLineID == nil || *claim.OrderLineID != row.LineID { t.Fatalf("claim provenance=%+v", claim) }
		ordinals[*claim.MaterializationUnitOrdinal] = true
	}
	if len(ordinals) != 2 || !ordinals[1] || !ordinals[2] { t.Fatalf("receipt ordinals=%v", ordinals) }
	// Inventory Identify changes the existing batch's identity counts. Commerce
	// keeps the original line/batch claim provenance and consumes nothing again.
	batch := st.batches[row.ReceiptBatchID]
	batch.IdentifiedCount, batch.UnidentifiedCount, batch.Revision = 2, 0, 2
	st.batches[row.ReceiptBatchID] = batch
	if st.shipmentLines[0].ReceiptBatchID != batch.ID { t.Fatal("identified batch lost receipt line") }
	updated, err := f.Allocate(ctx, supplier, order.ID, st.orders[order.ID].Version, []AllocationItem{{OrderLineID: row.LineID, VehicleID: ids[0]}})
	if err != nil { t.Fatal(err) }
	receiptCapacity(t, st, 1, 3)
	if len(st.associations) != 1 { t.Fatalf("associations=%v", st.associations) }
	for _, claim := range st.claims[st.claimsLotID()] { if claim.ID == st.associations[0].ClaimID && claim.MaterializationKind != "vin-allocation" { t.Fatalf("receipt claim reused: %+v", claim) } }
	if _, err := f.Ship(ctx, supplier, order.ID, updated.Version, receiptInput(order, 1)); err != nil { t.Fatal(err) }
	receiptCapacity(t, st, 0, 4)
	if len(st.shipmentLines) != 2 || st.shipmentLines[0].ReceiptBatchID != batch.ID { t.Fatalf("receipt history=%v", st.shipmentLines) }
	if _, err := f.Ship(ctx, supplier, order.ID, updated.Version, receiptInput(order, 1)); !errors.Is(err, apperr.ErrStale) { t.Fatalf("stale ship=%v", err) }
	if _, err := f.Allocate(ctx, supplier, order.ID, st.orders[order.ID].Version, []AllocationItem{{OrderLineID: row.LineID, VehicleID: ids[1]}}); err == nil { t.Fatal("fulfilled quantity allocated again") }
}

func TestReservationQuantityReceiptFullBatch(t *testing.T) {
	f, st, supplier, order, _ := receiptFixture(t, 0)
	if _, err := f.Ship(context.Background(), supplier, order.ID, order.Version, receiptInput(order, 4)); err != nil { t.Fatal(err) }
	receiptCapacity(t, st, 0, 4)
	if st.orders[order.ID].Status != model.OrderCompleted || len(st.shipmentLines) != 1 || st.shipmentLines[0].Quantity != 4 { t.Fatalf("full receipt order=%+v lines=%v", st.orders[order.ID], st.shipmentLines) }
	ordinals := map[int32]bool{}
	for _, claim := range st.claims[st.claimsLotID()] {
		if claim.MaterializationKind != "quantity-receipt" || claim.MaterializationRef == nil || *claim.MaterializationRef != st.shipmentLines[0].ID || claim.MaterializationUnitOrdinal == nil { t.Fatalf("full receipt claim=%+v", claim) }
		ordinals[*claim.MaterializationUnitOrdinal] = true
	}
	for unit := int32(1); unit <= 4; unit++ { if !ordinals[unit] { t.Fatalf("missing receipt ordinal %d: %v", unit, ordinals) } }
}

func TestReservationQuantityReceiptMixedAllocated(t *testing.T) {
	f, st, supplier, order, _ := receiptFixture(t, 2)
	if _, err := f.Ship(context.Background(), supplier, order.ID, order.Version, receiptInput(order, 2)); err != nil { t.Fatal(err) }
	receiptCapacity(t, st, 0, 4)
	if len(st.shipmentLines) != 1 || len(st.allocations) != 2 { t.Fatalf("mixed progress lines=%d allocations=%d", len(st.shipmentLines), len(st.allocations)) }
	if _, err := f.Ship(context.Background(), supplier, order.ID, st.orders[order.ID].Version, receiptInput(order, 1)); err == nil { t.Fatal("mixed fulfilled quantity shipped twice") }
}

func TestReservationQuantityReceiptRetiredVINDoesNotReuseReceipt(t *testing.T) {
	ctx := context.Background()
	f, st, supplier, order, ids := receiptFixture(t, 0)
	if _, err := f.Ship(ctx, supplier, order.ID, order.Version, receiptInput(order, 2)); err != nil { t.Fatal(err) }
	lineID := order.DecodeTerms().Lines[0].LineID
	first, err := f.Allocate(ctx, supplier, order.ID, st.orders[order.ID].Version, []AllocationItem{{OrderLineID: lineID, VehicleID: ids[0]}})
	if err != nil { t.Fatal(err) }
	vinClaimID := st.associations[0].ClaimID
	var firstRef string
	for _, claim := range st.claims[st.claimsLotID()] { if claim.ID == vinClaimID { firstRef = *claim.MaterializationRef } }
	if err := st.Supply().RetireClaimAllocation(ctx, vinClaimID, st.associations[0].ID); err != nil { t.Fatal(err) }
	st.allocations[0].Status = "rejected"
	delete(st.holds, ids[0])
	if _, err := f.Allocate(ctx, supplier, order.ID, first.Version, []AllocationItem{{OrderLineID: lineID, VehicleID: ids[1]}}); err != nil { t.Fatal(err) }
	receiptCapacity(t, st, 1, 3)
	if len(st.associations) != 2 || st.associations[1].ClaimID != vinClaimID { t.Fatalf("retired VIN association=%v", st.associations) }
	for _, claim := range st.claims[st.claimsLotID()] {
		if claim.ID == vinClaimID && (claim.MaterializationRef == nil || *claim.MaterializationRef != firstRef) { t.Fatalf("VIN first provenance changed: %+v", claim) }
		if claim.MaterializationKind == "quantity-receipt" && (claim.MaterializationRef == nil || *claim.MaterializationRef != st.shipmentLines[0].ID) { t.Fatalf("receipt provenance changed: %+v", claim) }
	}
}

func TestReservationQuantityReceiptOrdinaryOrderUnchanged(t *testing.T) {
	f, st, supplier, converted, _ := receiptFixture(t, 0)
	terms := converted.DecodeTerms()
	terms.Reservation = nil
	terms.Lines[0].LineID = uuid.NewString()
	raw, err := json.Marshal(terms)
	if err != nil { t.Fatal(err) }
	ordinary := model.Order{ID: uuid.NewString(), Source: "direct", BuyerCompanyID: converted.BuyerCompanyID, SupplierCompanyID: converted.SupplierCompanyID, ReceivingWarehouseID: converted.ReceivingWarehouseID, Terms: raw, Status: model.OrderAccepted, Version: 1}
	st.orders[ordinary.ID] = ordinary
	if _, err := f.Ship(context.Background(), supplier, ordinary.ID, ordinary.Version, receiptInput(&ordinary, 1)); err != nil { t.Fatal(err) }
	receiptCapacity(t, st, 4, 0)
	if len(st.shipmentLines) != 1 || st.shipmentLines[0].OrderID != ordinary.ID { t.Fatalf("ordinary receipt=%v", st.shipmentLines) }
}

func TestReservationQuantityReceiptFailuresRollback(t *testing.T) {
	for _, failure := range []string{"delivery", "shipment", "shipment-lines", "claim", "order", "event", "lot-model", "lot-spec", "lot-exterior", "lot-interior", "line-source", "claim-link", "wrong-party", "missing-source", "stale"} {
		t.Run(failure, func(t *testing.T) {
			f, st, supplier, order, _ := receiptFixture(t, 0)
			expected := order.Version
			initialEvents := st.events
			switch failure {
			case "delivery": st.failDelivery = true
			case "shipment": st.failShipment = true
			case "shipment-lines": st.failShipmentLines = true
			case "claim": st.failClaim = true
			case "order": st.failOrder = true
			case "event": st.failEvent = true
			case "lot-model", "lot-spec", "lot-exterior", "lot-interior":
				lot := st.lots[st.claimsLotID()]
				switch failure { case "lot-model": lot.ModelID = uuid.NewString(); case "lot-spec": lot.ModelSpecificationVersion = "changed"; case "lot-exterior": lot.ExteriorColor = "changed"; case "lot-interior": lot.InteriorColor = "changed" }
				st.lots[lot.ID] = lot
			case "line-source":
				changed := st.orders[order.ID]
				terms := changed.DecodeTerms()
				terms.Lines[0].OfferLineID = uuid.NewString()
				changed.Terms, _ = json.Marshal(terms)
				st.orders[order.ID] = changed
			case "claim-link":
				lotID := st.claimsLotID()
				for i := range st.claims[lotID] { claim := st.claims[lotID][i]; wrong := uuid.NewString(); claim.OrderLineID = &wrong; st.claims[lotID][i] = claim }
			case "wrong-party":
				r := st.reservations[order.DecodeTerms().Reservation.ReservationID]
				r.SupplierCompanyID = uuid.NewString()
				st.reservations[r.ID] = r
			case "missing-source": r := st.reservations[order.DecodeTerms().Reservation.ReservationID]; r.OrderID = nil; st.reservations[r.ID] = r
			case "stale": expected--
			}
			if _, err := f.Ship(context.Background(), supplier, order.ID, expected, receiptInput(order, 2)); err == nil { t.Fatal("invalid receipt accepted") }
			receiptCapacity(t, st, 4, 0)
			if len(st.shipments) != 0 || len(st.shipmentLines) != 0 || len(st.batches) != 0 || len(st.deliveries) != 0 || st.orders[order.ID].Version != order.Version || st.events != initialEvents { t.Fatalf("failure left state: shipments=%d lines=%d batches=%d deliveries=%d order=%+v events=%d", len(st.shipments), len(st.shipmentLines), len(st.batches), len(st.deliveries), st.orders[order.ID], st.events) }
		})
	}
}
