package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"justixauto/internal/modules/commerce/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/money"
)

type paymentState struct {
	Reservations map[string]model.Reservation
	Orders map[string]model.Order
	Evidence map[string]model.ReservationEvidence
	Refunds map[string]model.ReservationRefund
	Credits map[string]model.ReservationCredit
	Shares int
	Events int
}
type paymentStore struct {
	Store
	state paymentState
	locked bool
	failShare, failEvidence, failRefund, failCredit, failRevision, failEvent bool
}
type paymentTxKey struct{}
func (s *paymentStore) InTx(_ context.Context, fn func(Store) error) error {
	raw, _ := json.Marshal(s.state)
	var before paymentState
	_ = json.Unmarshal(raw, &before)
	s.locked = false
	if err := fn(s); err != nil { s.state = before; s.locked = false; return err }
	s.locked = false
	return nil
}
func (s *paymentStore) Bind(ctx context.Context) context.Context { return context.WithValue(ctx, paymentTxKey{}, true) }
func (s *paymentStore) Reservations() ReservationRepository { return paymentReservations{s:s} }
func (s *paymentStore) ReservationPayments() ReservationPaymentRepository { return paymentRows{s:s} }
func (s *paymentStore) Deals() DealRepository { return paymentDeals{s:s} }
type paymentDeals struct { DealRepository; s *paymentStore }
func (r paymentDeals) Order(_ context.Context, companyID, id string) (*model.Order,error) { o,ok:=r.s.state.Orders[id];if !ok||o.Party(companyID)=="" {return nil,apperr.ErrNotFound};return &o,nil }
func (s *paymentStore) Events() EventRepository { return paymentEvents{s:s} }
func (s *paymentStore) Invoices() InvoiceRepository { return paymentInvoices{} }
type paymentInvoices struct{ InvoiceRepository }
func (paymentInvoices) Evidence(context.Context, string) ([]model.Evidence, error) { return []model.Evidence{}, nil }
type paymentEvents struct { EventRepository; s *paymentStore }
func (r paymentEvents) Append(_ context.Context, _ *model.Event) error { if r.s.failEvent { return errors.New("event failed") }; r.s.state.Events++; return nil }
type paymentReservations struct { ReservationRepository; s *paymentStore }
func (r paymentReservations) ForUpdate(_ context.Context, id string) (*model.Reservation, []model.ReservationLine, error) {
	x, ok := r.s.state.Reservations[id]; if !ok { return nil,nil,apperr.ErrNotFound }
	r.s.locked = true
	return &x,nil,nil
}
func (r paymentReservations) ByOrderForUpdate(ctx context.Context, orderID string) (*model.Reservation, []model.ReservationLine, error) {
	for id, x := range r.s.state.Reservations { if x.OrderID != nil && *x.OrderID == orderID { return r.ForUpdate(ctx,id) } }
	return nil,nil,apperr.ErrNotFound
}
func (r paymentReservations) Update(_ context.Context, x *model.Reservation, expected int64) error {
	if !r.s.locked { return errors.New("reservation row not locked") }
	if r.s.failRevision { return errors.New("revision write failed") }
	old := r.s.state.Reservations[x.ID]; if old.Version != expected { return apperr.ErrStale }
	x.Version = expected+1; r.s.state.Reservations[x.ID] = *x; return nil
}
type paymentRows struct { s *paymentStore }
func (r paymentRows) check() error { if !r.s.locked { return errors.New("money read before reservation lock") }; return nil }
func (r paymentRows) Evidence(_ context.Context, reservationID string) ([]model.ReservationEvidence,error) {
	if err:=r.check();err!=nil{return nil,err}
	out:=[]model.ReservationEvidence{}
	for _,e:=range r.s.state.Evidence {if e.ReservationID==reservationID {out=append(out,e)}}
	return out,nil
}
func (r paymentRows) GetEvidence(_ context.Context,id string) (*model.ReservationEvidence,error) {e,ok:=r.s.state.Evidence[id];if !ok{return nil,apperr.ErrNotFound};return &e,nil}
func (r paymentRows) AddEvidence(_ context.Context,e *model.ReservationEvidence) error {if err:=r.check();err!=nil{return err};if r.s.failEvidence{return errors.New("evidence write failed")};r.s.state.Evidence[e.ID]=*e;return nil}
func (r paymentRows) DecideEvidence(_ context.Context,e *model.ReservationEvidence,expected int64) error {if err:=r.check();err!=nil{return err};if r.s.failEvidence{return errors.New("decision write failed")};old:=r.s.state.Evidence[e.ID];if old.Version!=expected{return apperr.ErrStale};e.Version++;r.s.state.Evidence[e.ID]=*e;return nil}
func (r paymentRows) Refund(_ context.Context,id string) (*model.ReservationRefund,error) {if err:=r.check();err!=nil{return nil,err};x,ok:=r.s.state.Refunds[id];if !ok{return nil,apperr.ErrNotFound};return &x,nil}
func (r paymentRows) CreateRefund(_ context.Context,x *model.ReservationRefund) error {if err:=r.check();err!=nil{return err};if r.s.failRefund{return errors.New("refund write failed")};if _,ok:=r.s.state.Refunds[x.ReservationID];ok{return apperr.ErrConflict};r.s.state.Refunds[x.ReservationID]=*x;return nil}
func (r paymentRows) ConfirmRefund(_ context.Context,x *model.ReservationRefund) error {if err:=r.check();err!=nil{return err};if r.s.failRefund{return errors.New("refund write failed")};old:=r.s.state.Refunds[x.ReservationID];if old.Status!="requested"{return apperr.ErrStale};r.s.state.Refunds[x.ReservationID]=*x;return nil}
func (r paymentRows) CreditForReservation(_ context.Context,id string) (*model.ReservationCredit,error) {if err:=r.check();err!=nil{return nil,err};x,ok:=r.s.state.Credits[id];if !ok{return nil,apperr.ErrNotFound};return &x,nil}
func (r paymentRows) CreditForOrder(_ context.Context,id string) (*model.ReservationCredit,error) {if err:=r.check();err!=nil{return nil,err};for _,x:=range r.s.state.Credits{if x.OrderID==id{return &x,nil}};return nil,apperr.ErrNotFound}
func (r paymentRows) CreateCredit(_ context.Context,x *model.ReservationCredit) error {if err:=r.check();err!=nil{return err};if r.s.failCredit{return errors.New("credit write failed")};if _,ok:=r.s.state.Credits[x.ReservationID];ok{return apperr.ErrConflict};r.s.state.Credits[x.ReservationID]=*x;return nil}
func (r paymentRows) AttachInvoice(_ context.Context,id,invoiceID string,at time.Time) error {if err:=r.check();err!=nil{return err};if r.s.failCredit{return errors.New("credit write failed")};x,ok:=r.s.state.Credits[id];if !ok||x.InvoiceID!=nil{return apperr.ErrStale};x.InvoiceID=&invoiceID;x.InvoiceAttachedAt=&at;r.s.state.Credits[id]=x;return nil}
type paymentFiles struct { s *paymentStore; owner string }
func (f paymentFiles) Share(ctx context.Context,owner,file,to,typ,id string) error {
	if ctx.Value(paymentTxKey{})!=true {return errors.New("document share not bound to transaction")}
	if f.s.failShare {return errors.New("share failed")}
	if owner!=f.owner {return apperr.ErrNotFound}
	f.s.state.Shares++;return nil
}
func paymentFixture() (*ReservationPayment,*paymentStore,*auth.Principal,*auth.Principal,string) {
	id,buyerID,supplierID:=uuid.NewString(),uuid.NewString(),uuid.NewString()
	advance:=money.Money{AmountMinor:"100",Currency:"USD"}
	price:=money.Money{AmountMinor:"1000",Currency:"USD"}
	raw,_:=json.Marshal(ReservationSnapshot{Terms:ReservationTerms{UnitPrice:&price,Advance:&advance}})
	r:=model.Reservation{ID:id,BuyerCompanyID:buyerID,SupplierCompanyID:supplierID,Status:model.ReservationActive,CurrentTerms:raw,CurrentDigest:"digest-1",CurrentRevision:2,Version:4}
	st:=&paymentStore{state:paymentState{Reservations:map[string]model.Reservation{id:r},Orders:map[string]model.Order{},Evidence:map[string]model.ReservationEvidence{},Refunds:map[string]model.ReservationRefund{},Credits:map[string]model.ReservationCredit{}}}
	buyer:=&auth.Principal{CompanyID:buyerID,UserID:uuid.NewString(),Permissions:map[string]bool{model.PermRead:true,model.PermTrade:true}}
	accountant:=&auth.Principal{CompanyID:supplierID,UserID:uuid.NewString(),Permissions:map[string]bool{model.PermRead:true,model.PermPaymentsAccept:true}}
	d:=NewDeps(st,nil,nil,nil,paymentFiles{s:st,owner:buyerID},func()time.Time{return time.Date(2026,10,11,12,0,0,0,time.UTC)})
	return NewReservationPayment(d),st,buyer,accountant,id
}
func paymentInput(amount string) EvidenceInput {return EvidenceInput{ClaimedAmount:money.Money{AmountMinor:amount,Currency:"USD"},PaidOn:"2026-10-10",ExternalReference:"bank-reference",AttachmentIDs:[]string{uuid.NewString()}}}
func paymentAccepted(t *testing.T,s *ReservationPayment,st *paymentStore,buyer,accountant *auth.Principal,id string) string {
	t.Helper()
	v,err:=s.Submit(context.Background(),buyer,id,st.state.Reservations[id].Version,paymentInput("100"));if err!=nil{t.Fatal(err)}
	eid:=v.Evidence[0].ID
	_,err=s.Decide(context.Background(),accountant,eid,1,st.state.Reservations[id].Version,true,true,"");if err!=nil{t.Fatal(err)}
	return eid
}

func TestReservationPayment(t *testing.T) {
	ctx:=context.Background()
	t.Run("free and evidence lifecycle",func(t *testing.T){
		s,st,buyer,accountant,id:=paymentFixture()
		r:=st.state.Reservations[id];snap,_:=decodeReservationSnapshot(r.CurrentTerms);snap.Terms.Advance=nil;raw,_:=json.Marshal(snap);r.CurrentTerms=raw;st.state.Reservations[id]=r
		v,err:=s.Get(ctx,buyer,id);if err!=nil||v.AgreedAdvance.AmountMinor!="0"||v.EvidenceEverSubmitted{t.Fatalf("free view: %#v %v",v,err)}
		if _,err=s.Submit(ctx,buyer,id,r.Version,paymentInput("1"));err==nil{t.Fatal("fabricated advance accepted")}
		_,st,buyer,accountant,id=paymentFixture();s=NewReservationPayment(NewDeps(st,nil,nil,nil,paymentFiles{s:st,owner:buyer.CompanyID},func()time.Time{return time.Date(2026,10,11,12,0,0,0,time.UTC)}))
		v,err=s.Submit(ctx,buyer,id,4,paymentInput("60"));if err!=nil{t.Fatal(err)}
		if v.Pending.AmountMinor!="60"||v.Accepted.AmountMinor!="0"||!v.EvidenceEverSubmitted{t.Fatal("pending counted as paid or missing history")}
		if _,err=s.Submit(ctx,buyer,id,v.Revision,paymentInput("41"));err==nil{t.Fatal("pending capacity bypassed")}
		eid:=v.Evidence[0].ID
		v,err=s.Decide(ctx,accountant,eid,1,v.Revision,false,false,"wrong proof");if err!=nil{t.Fatal(err)}
		if v.PendingCount!=0||v.Accepted.AmountMinor!="0"||!v.EvidenceEverSubmitted||v.Evidence[0].ExternalReference!="bank-reference"{t.Fatal("rejected evidence lost or changed")}
		if _,err=s.Decide(ctx,accountant,eid,1,v.Revision,true,true,"");err==nil{t.Fatal("duplicate decision")}
		v,err=s.Submit(ctx,buyer,id,v.Revision,paymentInput("100"));if err!=nil{t.Fatal(err)}
		newID:="";for _,e:=range v.Evidence{if e.Status=="submitted"{newID=e.ID}}
		v,err=s.Decide(ctx,accountant,newID,1,v.Revision,true,true,"");if err!=nil{t.Fatal(err)}
		if v.Accepted.AmountMinor!="100"||v.PendingCount!=0{t.Fatal("accepted money incorrect")}
	})
	t.Run("authority and rollback",func(t *testing.T){
		s,st,buyer,accountant,id:=paymentFixture()
		wrong:=*accountant;wrong.CompanyID=uuid.NewString()
		if _,err:=s.Get(ctx,&wrong,id);err==nil{t.Fatal("foreign read")}
		bad:=paymentInput("1");bad.ClaimedAmount.Currency="EUR"
		if _,err:=s.Submit(ctx,buyer,id,4,bad);err==nil{t.Fatal("wrong currency")}
		bad=paymentInput("1.5");if _,err:=s.Submit(ctx,buyer,id,4,bad);err==nil{t.Fatal("fractional money")}
		st.failShare=true;if _,err:=s.Submit(ctx,buyer,id,4,paymentInput("10"));err==nil{t.Fatal("share failure")};st.failShare=false
		st.failEvidence=true;if _,err:=s.Submit(ctx,buyer,id,4,paymentInput("10"));err==nil{t.Fatal("write failure")};st.failEvidence=false
		st.failRevision=true;if _,err:=s.Submit(ctx,buyer,id,4,paymentInput("10"));err==nil{t.Fatal("revision failure")};st.failRevision=false
		st.failEvent=true;if _,err:=s.Submit(ctx,buyer,id,4,paymentInput("10"));err==nil{t.Fatal("event failure")};st.failEvent=false
		if len(st.state.Evidence)!=0||st.state.Shares!=0||st.state.Events!=0||st.state.Reservations[id].Version!=4{t.Fatal("partial transaction survived")}
		v,err:=s.Submit(ctx,buyer,id,4,paymentInput("10"));if err!=nil{t.Fatal(err)}
		if _,err=s.Decide(ctx,buyer,v.Evidence[0].ID,1,v.Revision,true,true,"");err==nil{t.Fatal("buyer decided payment")}
		if _,err=s.Decide(ctx,accountant,v.Evidence[0].ID,1,4,true,true,"");err==nil{t.Fatal("stale reservation decision")}
	})
}

func TestReservationRefund(t *testing.T) {
	ctx:=context.Background()
	s,st,buyer,accountant,id:=paymentFixture()
	v,err:=s.Submit(ctx,buyer,id,4,paymentInput("40"));if err!=nil{t.Fatal(err)}
	v,err=s.RequestRefund(ctx,buyer,id,v.Revision,"cancel");if err!=nil{t.Fatal(err)}
	if v.RefundStatus!="requested"||v.RefundObligation.AmountMinor!="0"{t.Fatal("intent changed accepted money")}
	st.locked=true;current:=st.state.Reservations[id]
	eligible,err:=s.ReleaseEligibleLocked(ctx,st,&current);if err!=nil||eligible{t.Fatalf("pending allowed release: %v",err)}
	if _,err=s.ConfirmRefund(ctx,accountant,id,v.Revision,paymentInput("40"),true);err==nil{t.Fatal("pending evidence settled")}
	v,err=s.Decide(ctx,accountant,v.Evidence[0].ID,1,v.Revision,true,true,"");if err!=nil{t.Fatal(err)}
	if v.RefundObligation.AmountMinor!="40"||v.Accepted.AmountMinor!="40"{t.Fatal("accepted obligation not dynamic")}
	if _,err=s.ConfirmRefund(ctx,accountant,id,v.Revision,paymentInput("39"),true);err==nil{t.Fatal("partial refund")}
	if _,err=s.ConfirmRefund(ctx,buyer,id,v.Revision,paymentInput("40"),true);err==nil{t.Fatal("buyer confirmed refund")}
	foreign:=*accountant;foreign.CompanyID=uuid.NewString()
	if _,err=s.ConfirmRefund(ctx,&foreign,id,v.Revision,paymentInput("40"),true);err==nil{t.Fatal("foreign accountant confirmed refund")}
	wrong:=paymentInput("40");wrong.ClaimedAmount.Currency="EUR"
	if _,err=s.ConfirmRefund(ctx,accountant,id,v.Revision,wrong,true);err==nil{t.Fatal("wrong currency")}
	st.failRefund=true;if _,err=s.ConfirmRefund(ctx,accountant,id,v.Revision,paymentInput("40"),true);err==nil{t.Fatal("refund write failure")};st.failRefund=false
	st.failEvent=true;if _,err=s.ConfirmRefund(ctx,accountant,id,v.Revision,paymentInput("40"),true);err==nil{t.Fatal("refund event failure")};st.failEvent=false
	if st.state.Refunds[id].Status!="requested"{t.Fatal("refund confirmed after rollback")}
	st.failShare=false;st.state.Shares=0
	s.files=paymentFiles{s:st,owner:accountant.CompanyID}
	v,err=s.ConfirmRefund(ctx,accountant,id,v.Revision,paymentInput("40"),true);if err!=nil{t.Fatal(err)}
	if v.Accepted.AmountMinor!="40"||v.Refunded.AmountMinor!="40"||st.state.Shares!=1{t.Fatal("refund not fully recorded")}
	if _,err=s.ConfirmRefund(ctx,accountant,id,v.Revision,paymentInput("40"),true);err==nil{t.Fatal("duplicate refund")}
	st.locked=true;current=st.state.Reservations[id];eligible,err=s.ReleaseEligibleLocked(ctx,st,&current);if err!=nil||!eligible{t.Fatalf("confirmed refund did not permit release: %v",err)}
	orderID:=uuid.NewString();current.OrderID=&orderID
	eligible,err=s.ReleaseEligibleLocked(ctx,st,&current);if err!=nil||eligible{t.Fatal("converted source permitted release")}
}

func TestReservationCredit(t *testing.T) {
	ctx:=context.Background()
	s,st,buyer,accountant,id:=paymentFixture()
	eid:=paymentAccepted(t,s,st,buyer,accountant,id)
	r:=st.state.Reservations[id]
	o:=&model.Order{ID:uuid.NewString(),BuyerCompanyID:buyer.CompanyID,SupplierCompanyID:accountant.CompanyID,Source:"reservation",Status:model.OrderAccepted}
	terms:=model.Terms{Lines:[]model.Line{{Quantity:1,UnitPrice:money.Money{AmountMinor:"1000",Currency:"USD"}}}}
	o.Terms,_=json.Marshal(terms)
	st.locked=true
	credit,err:=s.LinkCreditLocked(ctx,st,&r,o);if err!=nil{t.Fatal(err)}
	if credit==nil||credit.AmountMinor!="100"||credit.InvoiceID!=nil{t.Fatal("credit identity or amount")}
	var ids []string;_ = json.Unmarshal(credit.EvidenceIDs,&ids);if len(ids)!=1||ids[0]!=eid{t.Fatal("accepted evidence provenance missing")}
	if _,err=s.LinkCreditLocked(ctx,st,&r,o);err==nil{t.Fatal("duplicate credit")}
	if _,err=s.RequestRefund(ctx,buyer,id,r.Version,"cancel");err==nil{t.Fatal("credited money entered refund")}
	r.OrderID=&o.ID;st.state.Reservations[id]=r
	st.locked=false
	got,err:=s.CreditForOrderLocked(ctx,st,o);if err!=nil||got==nil||got.OrderID!=o.ID{t.Fatalf("credit lookup: %v",err)}
	i:=&model.Invoice{ID:uuid.NewString(),OrderID:o.ID,BuyerCompanyID:buyer.CompanyID,SupplierCompanyID:accountant.CompanyID,Status:"issued",Currency:"USD",TotalMinor:"1000"}
	wrongInvoice:=*i;wrongInvoice.Currency="EUR"
	if err=s.AttachCreditInvoiceLocked(ctx,st,got,&wrongInvoice);err==nil{t.Fatal("wrong-currency invoice attached")}
	wrongInvoice=*i;wrongInvoice.Status="void"
	if err=s.AttachCreditInvoiceLocked(ctx,st,got,&wrongInvoice);err==nil{t.Fatal("void invoice attached")}
	if err=s.AttachCreditInvoiceLocked(ctx,st,got,i);err!=nil{t.Fatal(err)}
	if st.state.Credits[id].InvoiceID==nil{t.Fatal("nullable invoice association not attached")}
	if err=s.AttachCreditInvoiceLocked(ctx,st,got,i);err==nil{t.Fatal("duplicate invoice association")}
	zero,other,_,_,otherID:=paymentFixture();ordinary:=&model.Order{ID:uuid.NewString(),Source:"direct"};other.locked=true
	if got,err=zero.CreditForOrderLocked(ctx,other,ordinary);err!=nil||got!=nil{t.Fatal("ordinary order fabricated credit")}
	_ = otherID
	third,ts,tbuyer,taccountant,tid:=paymentFixture();paymentAccepted(t,third,ts,tbuyer,taccountant,tid)
	ts.locked=true;ts.failCredit=true
	if err=ts.InTx(ctx,func(store Store) error {rr,_,e:=third.LockMoney(ctx,store,tid);if e!=nil{return e};returnCredit,e:=third.LinkCreditLocked(ctx,store,rr,&model.Order{ID:uuid.NewString(),BuyerCompanyID:tbuyer.CompanyID,SupplierCompanyID:taccountant.CompanyID,Source:"reservation",Status:model.OrderAccepted,Terms:o.Terms});_ = returnCredit;return e});err==nil{t.Fatal("credit failure")}
	if len(ts.state.Credits)!=0{t.Fatal("credit survived rollback")}
	for name, mutate:=range map[string]func(*model.Order){
		"wrong buyer":func(o *model.Order){o.BuyerCompanyID=uuid.NewString()},
		"wrong source":func(o *model.Order){o.Source="direct"},
		"wrong currency":func(o *model.Order){o.Terms,_=json.Marshal(model.Terms{Lines:[]model.Line{{Quantity:1,UnitPrice:money.Money{AmountMinor:"1000",Currency:"EUR"}}}})},
		"too small":func(o *model.Order){o.Terms,_=json.Marshal(model.Terms{Lines:[]model.Line{{Quantity:1,UnitPrice:money.Money{AmountMinor:"99",Currency:"USD"}}}})},
	}{
		t.Run(name,func(t *testing.T){next,store,b,a,sourceID:=paymentFixture();paymentAccepted(t,next,store,b,a,sourceID)
			candidate:=*o;candidate.BuyerCompanyID=b.CompanyID;candidate.SupplierCompanyID=a.CompanyID;candidate.ID=uuid.NewString();mutate(&candidate)
			locked:=store.state.Reservations[sourceID];store.locked=true
			if _,err:=next.LinkCreditLocked(ctx,store,&locked,&candidate);err==nil{t.Fatal("invalid source credited")}
			if len(store.state.Credits)!=0{t.Fatal("failed source wrote credit")}
		})
	}
	t.Run("pending and refund conflict",func(t *testing.T){next,store,b,a,sourceID:=paymentFixture()
		_,err:=next.Submit(ctx,b,sourceID,4,paymentInput("20"));if err!=nil{t.Fatal(err)}
		locked:=store.state.Reservations[sourceID];candidate:=*o;candidate.BuyerCompanyID=b.CompanyID;candidate.SupplierCompanyID=a.CompanyID;candidate.ID=uuid.NewString();store.locked=true
		if _,err:=next.LinkCreditLocked(ctx,store,&locked,&candidate);err==nil{t.Fatal("pending payment credited")}
		v,err:=next.RequestRefund(ctx,b,sourceID,locked.Version,"cancel");if err!=nil{t.Fatal(err)}
		v,err=next.Decide(ctx,a,v.Evidence[0].ID,1,v.Revision,false,false,"unpaid");if err!=nil{t.Fatal(err)}
		locked=store.state.Reservations[sourceID];store.locked=true
		if _,err:=next.LinkCreditLocked(ctx,store,&locked,&candidate);err==nil{t.Fatal("refunding source credited")}
	})
	t.Run("caller transaction rolls back credit on event failure",func(t *testing.T){next,store,b,a,sourceID:=paymentFixture();paymentAccepted(t,next,store,b,a,sourceID)
		candidate:=*o;candidate.ID=uuid.NewString();candidate.BuyerCompanyID=b.CompanyID;candidate.SupplierCompanyID=a.CompanyID
		store.failEvent=true
		err:=store.InTx(ctx,func(tx Store) error {locked,_,err:=next.LockMoney(ctx,tx,sourceID);if err!=nil{return err};if _,err=next.LinkCreditLocked(ctx,tx,locked,&candidate);err!=nil{return err};return next.event(ctx,tx,a,"reservation.converted","reservation",sourceID,"",nil)})
		if err==nil||len(store.state.Credits)!=0{t.Fatal("event failure left credit")}
	})
	t.Run("invoice view includes assigned credit exactly once",func(t *testing.T){
		next,store,b,a,sourceID:=paymentFixture();paymentAccepted(t,next,store,b,a,sourceID)
		orderID:=uuid.NewString(); r:=store.state.Reservations[sourceID]; r.OrderID=&orderID; store.state.Reservations[sourceID]=r
		o:=&model.Order{ID:orderID,BuyerCompanyID:b.CompanyID,SupplierCompanyID:a.CompanyID,Source:"reservation",Status:model.OrderAccepted}; o.Terms,_=json.Marshal(model.Terms{Lines:[]model.Line{{Quantity:1,UnitPrice:money.Money{AmountMinor:"1000",Currency:"USD"}}}})
		store.state.Orders[orderID]=*o; store.locked=true; credit,err:=next.LinkCreditLocked(ctx,store,&r,o); if err!=nil{t.Fatal(err)}
		invoice:=&model.Invoice{ID:uuid.NewString(),OrderID:orderID,BuyerCompanyID:b.CompanyID,SupplierCompanyID:a.CompanyID,Status:"issued",Currency:"USD",TotalMinor:"1000"}
		if err=next.AttachCreditInvoiceLocked(ctx,store,credit,invoice);err!=nil{t.Fatal(err)}
		view,err:=NewInvoice(NewDeps(store,nil,nil,nil,nil,func()time.Time{return time.Now()})).view(ctx,store,invoice);if err!=nil{t.Fatal(err)}
		if view.Paid.AmountMinor!="100"||view.Outstanding.AmountMinor!="900"{t.Fatalf("credit not counted once: %#v",view)}
		void:=*invoice;void.Status="void";view,err=NewInvoice(NewDeps(store,nil,nil,nil,nil,func()time.Time{return time.Now()})).view(ctx,store,&void);if err!=nil{t.Fatal(err)}
		if view.Paid.AmountMinor!="0"{t.Fatalf("void invoice repeated credit: %#v",view)}
	})
}
