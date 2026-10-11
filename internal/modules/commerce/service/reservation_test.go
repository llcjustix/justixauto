package service

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

type reservationTestStore struct {
	*supplyTestStore
	reservations map[string]model.Reservation
	lines map[string][]model.ReservationLine
	holds map[string]string
	sold map[string]bool
	failUpdate bool
	associations []model.SupplyClaimAllocation
	failClaim, failAssociation, failStock, failEvent bool
	events int
}
func (s *reservationTestStore) InTx(_ context.Context, fn func(Store) error) error {
	beforeReservations := maps.Clone(s.reservations)
	beforeHolds := maps.Clone(s.holds)
	beforeClaims := make(map[string][]model.SupplyClaim, len(s.claims))
	for k, v := range s.claims { beforeClaims[k] = append([]model.SupplyClaim(nil), v...) }
	beforeAssociations := slices.Clone(s.associations)
	beforeEvents := s.events
	if err := fn(s); err != nil {
		s.reservations, s.holds, s.claims = beforeReservations, beforeHolds, beforeClaims
		s.associations, s.events = beforeAssociations, beforeEvents
		return err
	}
	return nil
}
func (s *reservationTestStore) Supply() SupplyRepository { return reservationSupplyRepo{SupplyRepository:supplyTestRepo{s.supplyTestStore},s:s} }
func (s *reservationTestStore) Events() EventRepository { return reservationEvents{s:s} }
type reservationEvents struct { EventRepository; s *reservationTestStore }
func (r reservationEvents) Append(context.Context,*model.Event) error { if r.s.failEvent { return errors.New("event failed") }; r.s.events++; return nil }
// supplyTestRepo also backs the older supply-only fixtures, which do not
// exercise physical associations.
func (r supplyTestRepo) ClaimAllocations(context.Context,string) ([]model.SupplyClaimAllocation,error) { return nil,nil }
func (r supplyTestRepo) AppendClaimAllocation(context.Context,model.SupplyClaimAllocation,string) error { return nil }
func (r supplyTestRepo) RetireClaimAllocation(context.Context,string,string) error { return nil }
type reservationSupplyRepo struct { SupplyRepository; s *reservationTestStore }
func (r reservationSupplyRepo) UpdateClaim(ctx context.Context,c *model.SupplyClaim) error { if r.s.failClaim { return errors.New("claim failed") }; return r.SupplyRepository.UpdateClaim(ctx,c) }
func (r reservationSupplyRepo) ClaimAllocations(_ context.Context,reservationID string) ([]model.SupplyClaimAllocation,error) {
	var out []model.SupplyClaimAllocation
	for _,a:=range r.s.associations { for _,claims:=range r.s.claims { for _,c:=range claims { if c.ID==a.ClaimID&&c.ReservationID!=nil&&*c.ReservationID==reservationID { out=append(out,a) } } } }
	return out,nil
}
func (r reservationSupplyRepo) AppendClaimAllocation(_ context.Context,a model.SupplyClaimAllocation,replaceID string) error {
	if r.s.failAssociation { return errors.New("association failed") }
	if replaceID!="" { found:=false; for i:=range r.s.associations { if r.s.associations[i].ID==replaceID&&r.s.associations[i].ClaimID==a.ClaimID&&r.s.associations[i].Current { r.s.associations[i].Current=false;found=true;break } };if !found{return apperr.ErrConflict} }
	for _,old:=range r.s.associations { if old.Current&&(old.ClaimID==a.ClaimID||old.VehicleID==a.VehicleID) {return apperr.ErrConflict} }
	r.s.associations=append(r.s.associations,a);return nil
}
func (r reservationSupplyRepo) RetireClaimAllocation(_ context.Context,claimID,associationID string) error {for i:=range r.s.associations {a:=&r.s.associations[i];if a.ID==associationID&&a.ClaimID==claimID&&a.Current {a.Current=false;return nil}};return apperr.ErrConflict}
func (s *reservationTestStore) Reservations() ReservationRepository { return reservationTestRepo{s: s} }
type reservationTestRepo struct { ReservationRepository; s *reservationTestStore }
func (r reservationTestRepo) Create(_ context.Context, x *model.Reservation, lines []model.ReservationLine) error {
	r.s.reservations[x.ID] = *x; r.s.lines[x.ID] = append([]model.ReservationLine(nil), lines...); return nil
}
func (r reservationTestRepo) ForUpdate(_ context.Context, id string) (*model.Reservation, []model.ReservationLine, error) {
	x, ok := r.s.reservations[id]; if !ok { return nil, nil, apperr.ErrNotFound }
	return &x, append([]model.ReservationLine(nil), r.s.lines[id]...), nil
}
func (r reservationTestRepo) ByOrderForUpdate(ctx context.Context,orderID string) (*model.Reservation,[]model.ReservationLine,error) { for id,x:=range r.s.reservations { if x.OrderID!=nil&&*x.OrderID==orderID{return r.ForUpdate(ctx,id)} };return nil,nil,apperr.ErrNotFound }
func (r reservationTestRepo) Update(_ context.Context, x *model.Reservation, expected int64) error {
	if r.s.failUpdate { return errors.New("write failed") }
	old := r.s.reservations[x.ID]; if old.Version != expected { return apperr.ErrStale }
	x.Version++; r.s.reservations[x.ID] = *x; return nil
}
func (r reservationTestRepo) Get(ctx context.Context, companyID, id string) (*model.Reservation, []model.ReservationLine, error) {
	x, lines, err := r.ForUpdate(ctx, id); if err != nil { return nil,nil,err }
	if x.BuyerCompanyID != companyID && x.SupplierCompanyID != companyID { return nil,nil,apperr.ErrNotFound }
	return x,lines,nil
}
func (r reservationTestRepo) List(_ context.Context, companyID string, _, _ int) ([]model.Reservation,error) {
	var out []model.Reservation; for _, x := range r.s.reservations { if x.BuyerCompanyID == companyID || x.SupplierCompanyID == companyID { out = append(out,x) } }; return out,nil
}
func (r reservationTestRepo) Lines(_ context.Context, id string) ([]model.ReservationLine,error) { return r.s.lines[id],nil }

type reservationTestStock struct { Stock; store *reservationTestStore; vehicles map[string]*StockVehicle; owners map[string]string }
func (s reservationTestStock) Vehicle(_ context.Context, companyID,id string) (*StockVehicle,error) {
	x := s.vehicles[id]; if x == nil || s.owners != nil && s.owners[id] != companyID { return nil,apperr.ErrNotFound }; return x,nil
}
func (s reservationTestStock) ReserveReservation(_ context.Context, companyID,reservationID string, ids []string) error {
	if s.store.failStock { return errors.New("stock failed") }
	for _, id := range ids { if s.store.holds[id] != "" || s.store.sold[id] { return apperr.New(apperr.ErrConflict,"stock_unavailable","VIN unavailable") }; s.store.holds[id] = reservationID }; return nil
}
func (s reservationTestStock) ReassignReservationToOrder(_ context.Context,reservationID,orderID string,ids []string) error {
	if s.store.failStock { return errors.New("stock failed") }
	for _,id:=range ids { if s.store.holds[id]!=reservationID { return apperr.ErrConflict } }
	for _,id:=range ids { s.store.holds[id]=orderID };return nil
}
func (s reservationTestStock) ReleaseReservation(_ context.Context, reservationID string, ids []string, reason string) error {
	for _, id := range ids { if s.store.holds[id] == reservationID { delete(s.store.holds,id) } }; return nil
}
func reservationFixture() (*Reservation,*reservationTestStore,*auth.Principal,*auth.Principal,string,string) {
	dealerID,supplierID,lotID,vinID,modelID := uuid.NewString(),uuid.NewString(),uuid.NewString(),uuid.NewString(),uuid.NewString()
	base := &supplyTestStore{lots: map[string]*model.SupplyLot{lotID:{ID:lotID,SupplierCompanyID:supplierID,ModelID:modelID,ModelSpecificationVersion:"2",ExteriorColor:"white",InteriorColor:"black",TotalQuantity:3}},claims:map[string][]model.SupplyClaim{},active:true}
	st := &reservationTestStore{supplyTestStore:base,reservations:map[string]model.Reservation{},lines:map[string][]model.ReservationLine{},holds:map[string]string{},sold:map[string]bool{}}
	dir := supplyDirectory{companies:map[string]*Company{dealerID:{ID:dealerID,Name:"Dealer",Kind:"seller",UserType:supplyString("dealer"),Active:true},supplierID:{ID:supplierID,Name:"Supplier",Kind:"seller",UserType:supplyString("distributor"),Active:true}}}
	stock := reservationTestStock{store:st,vehicles:map[string]*StockVehicle{vinID:{ID:vinID,ModelID:modelID,ModelSpecificationVersion:"2",ExteriorColor:"white",InteriorColor:"black"}}}
	d := NewDeps(st,dir,nil,stock,nil,func()time.Time{return time.Date(2026,10,11,0,0,0,0,time.UTC)})
	return NewReservation(d,stock),st,&auth.Principal{CompanyID:dealerID,UserID:uuid.NewString(),CommercialAccess:true,Permissions:map[string]bool{model.PermRead:true,model.PermTrade:true}},&auth.Principal{CompanyID:supplierID,UserID:uuid.NewString(),CommercialAccess:true,Permissions:map[string]bool{model.PermRead:true,model.PermTrade:true}},lotID,vinID
}
func reservationTerms() ReservationTerms { expiry:=time.Date(2026,10,20,0,0,0,0,time.UTC); return ReservationTerms{UnitPrice:&money.Money{AmountMinor:"10000",Currency:"USD"},ExpiresAt:&expiry} }
func TestReservationTransitions(t *testing.T) {
	ctx:=context.Background()
	t.Run("trade permission and party type are required",func(t *testing.T){
		s,_,dealer,supplier,lot,_:=reservationFixture()
		input:=ReservationRequest{SupplierCompanyID:supplier.CompanyID,Lines:[]ReservationLineInput{{SupplyLotID:&lot,Quantity:1}}}
		dealer.Permissions[model.PermTrade]=false
		if _,err:=s.Request(ctx,dealer,input);err==nil{t.Fatal("request without trade permission")}
		dealer.Permissions[model.PermTrade]=true
		if _,err:=s.Request(ctx,supplier,input);err==nil{t.Fatal("distributor requested as dealer")}
	})
	t.Run("confirmation rejects invalid economic terms",func(t *testing.T){
		s,_,dealer,supplier,lot,_:=reservationFixture()
		v,err:=s.Request(ctx,dealer,ReservationRequest{SupplierCompanyID:supplier.CompanyID,Lines:[]ReservationLineInput{{SupplyLotID:&lot,Quantity:1}}});if err!=nil{t.Fatal(err)}
		bad:=reservationTerms();bad.UnitPrice.AmountMinor="0"
		if _,err:=s.Confirm(ctx,supplier,v.Reservation.ID,1,ReservationConfirmation{Terms:bad});err==nil{t.Fatal("zero price accepted")}
		bad=reservationTerms();past:=time.Date(2026,10,10,0,0,0,0,time.UTC);bad.ExpiresAt=&past
		if _,err:=s.Confirm(ctx,supplier,v.Reservation.ID,1,ReservationConfirmation{Terms:bad});err==nil{t.Fatal("past expiry accepted")}
	})
	t.Run("exact VIN confirms once and outsider is hidden",func(t *testing.T){
		s,st,dealer,supplier,_,vin:=reservationFixture()
		v,err:=s.Request(ctx,dealer,ReservationRequest{SupplierCompanyID:supplier.CompanyID,Lines:[]ReservationLineInput{{VehicleID:&vin,Quantity:1}},DesiredTerms:reservationTerms()}); if err!=nil{t.Fatal(err)}
		if st.holds[vin]!=""{t.Fatal("request held VIN")}
		if _,err=s.Get(ctx,&auth.Principal{CompanyID:uuid.NewString(),Permissions:map[string]bool{model.PermRead:true}},v.Reservation.ID);!errors.Is(err,apperr.ErrNotFound){t.Fatalf("outsider=%v",err)}
		v,err=s.Confirm(ctx,supplier,v.Reservation.ID,1,ReservationConfirmation{Terms:reservationTerms()});if err!=nil{t.Fatal(err)}
		if v.Reservation.Status!=model.ReservationActive||st.holds[vin]!=v.Reservation.ID||v.Reservation.RequestedRevision!=1||v.Reservation.OfferedRevision!=2||v.Reservation.CurrentRevision!=2{t.Fatalf("status/hold/revision=%+v/%v",v.Reservation,st.holds)}
		if _,err=s.Confirm(ctx,supplier,v.Reservation.ID,v.Reservation.Version,ReservationConfirmation{Terms:reservationTerms()});err==nil{t.Fatal("repeat activation succeeded")}
	})
	t.Run("changed future quantity waits for exact acceptance",func(t *testing.T){
		s,st,dealer,supplier,lot,_:=reservationFixture()
		v,err:=s.Request(ctx,dealer,ReservationRequest{SupplierCompanyID:supplier.CompanyID,Lines:[]ReservationLineInput{{SupplyLotID:&lot,Quantity:2}}});if err!=nil{t.Fatal(err)}
		if len(st.claims[lot])!=0{t.Fatal("request held future capacity")}
		lineID:=v.Requested.Lines[0].LineID
		v,err=s.Confirm(ctx,supplier,v.Reservation.ID,1,ReservationConfirmation{Terms:reservationTerms(),Lines:[]ReservationLineQuantity{{LineID:lineID,Quantity:3}}});if err!=nil{t.Fatal(err)}
		if v.Reservation.Status!=model.ReservationOffered||len(st.claims[lot])!=0||v.Requested.Lines[0].Quantity!=2||v.Offered.Lines[0].Quantity!=3{t.Fatalf("offer=%+v",v)}
		if _,err=s.Accept(ctx,dealer,v.Reservation.ID,v.Reservation.Version,"wrong");err==nil{t.Fatal("stale digest accepted")}
		v,err=s.Accept(ctx,dealer,v.Reservation.ID,v.Reservation.Version,v.Reservation.OfferedDigest);if err!=nil{t.Fatal(err)}
		if v.Reservation.Status!=model.ReservationActive||v.Reservation.CurrentRevision!=3||len(st.claims[lot])!=3{t.Fatalf("activation=%+v claims=%d",v.Reservation,len(st.claims[lot]))}
		for _,c:=range st.claims[lot]{if c.ReservationID==nil||*c.ReservationID!=v.Reservation.ID||c.ReservationLineID==nil||*c.ReservationLineID!=lineID{t.Fatalf("claim linkage=%+v",c)}}
	})
	t.Run("exhaustion and failed update roll back",func(t *testing.T){
		s,st,dealer,supplier,lot,_:=reservationFixture()
		v,err:=s.Request(ctx,dealer,ReservationRequest{SupplierCompanyID:supplier.CompanyID,Lines:[]ReservationLineInput{{SupplyLotID:&lot,Quantity:3}},DesiredTerms:reservationTerms()});if err!=nil{t.Fatal(err)}
		st.claims[lot]=[]model.SupplyClaim{{ID:uuid.NewString(),SupplyLotID:lot,State:"held"}}
		if _,err=s.Confirm(ctx,supplier,v.Reservation.ID,1,ReservationConfirmation{Terms:reservationTerms()});err==nil{t.Fatal("oversell accepted")}
		if len(st.claims[lot])!=1||st.reservations[v.Reservation.ID].Status!=model.ReservationRequested{t.Fatal("oversell changed state")}
		st.claims[lot]=nil;st.failUpdate=true
		if _,err=s.Confirm(ctx,supplier,v.Reservation.ID,1,ReservationConfirmation{Terms:reservationTerms()});err==nil{t.Fatal("failed update accepted")}
		if len(st.claims[lot])!=0||st.reservations[v.Reservation.ID].Status!=model.ReservationRequested{t.Fatal("failed update did not roll back")}
	})
	t.Run("held or sold VIN cannot activate",func(t *testing.T){
		for _, unavailable := range []string{"held","sold"} {
			t.Run(unavailable,func(t *testing.T){
				s,st,dealer,supplier,_,vin:=reservationFixture()
				v,err:=s.Request(ctx,dealer,ReservationRequest{SupplierCompanyID:supplier.CompanyID,Lines:[]ReservationLineInput{{VehicleID:&vin,Quantity:1}},DesiredTerms:reservationTerms()});if err!=nil{t.Fatal(err)}
				if unavailable=="held" { st.holds[vin]="other" } else { st.sold[vin]=true }
				if _,err=s.Confirm(ctx,supplier,v.Reservation.ID,1,ReservationConfirmation{Terms:reservationTerms()});err==nil{t.Fatal("unavailable VIN activated")}
				if st.reservations[v.Reservation.ID].Status!=model.ReservationRequested{t.Fatal("failed activation changed reservation")}
			})
		}
	})
}
