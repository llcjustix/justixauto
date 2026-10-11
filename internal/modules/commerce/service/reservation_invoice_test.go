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

// invoiceStore retains all three ledgers in one fake transaction. Reservation
// money reads fail if an Invoice method reaches them before the source lock.
type invoiceStore struct {
	*paymentStore
	invoices map[string]model.Invoice
	evidence map[string]model.Evidence
	failCreate, failAttach, failOrderCAS, failInvoiceEvidence bool
	beforeSourceLock func(*invoiceStore)
	committedEvidence map[string]model.Evidence
}

func (s *invoiceStore) InTx(ctx context.Context, fn func(Store) error) error {
	raw, _ := json.Marshal(s.paymentStore.state)
	var before paymentState
	_ = json.Unmarshal(raw, &before)
	raw, _ = json.Marshal(s.invoices)
	var oldInvoices map[string]model.Invoice
	_ = json.Unmarshal(raw, &oldInvoices)
	raw, _ = json.Marshal(s.evidence)
	var oldEvidence map[string]model.Evidence
	_ = json.Unmarshal(raw, &oldEvidence)
	s.locked = false
	if err := fn(s); err != nil {
		s.paymentStore.state, s.invoices, s.evidence = before, oldInvoices, oldEvidence
		for id,e:=range s.committedEvidence{s.evidence[id]=e}
		s.locked = false
		return err
	}
	s.locked = false
	return nil
}
func (s *invoiceStore) Deals() DealRepository { return invoiceDeals{paymentDeals: paymentDeals{s:s.paymentStore}, s:s} }
func (s *invoiceStore) Invoices() InvoiceRepository { return invoiceRows{s:s} }
func (s *invoiceStore) ReservationPayments() ReservationPaymentRepository { return invoiceMoneyRows{paymentRows:paymentRows{s:s.paymentStore},s:s} }
func (s *invoiceStore) Reservations() ReservationRepository { return invoiceReservations{paymentReservations:paymentReservations{s:s.paymentStore},s:s} }

type invoiceReservations struct { paymentReservations; s *invoiceStore }
func (r invoiceReservations) ByOrderForUpdate(ctx context.Context, orderID string)(*model.Reservation,[]model.ReservationLine,error){
	if r.s.beforeSourceLock!=nil{fn:=r.s.beforeSourceLock;r.s.beforeSourceLock=nil;fn(r.s)}
	return r.paymentReservations.ByOrderForUpdate(ctx,orderID)
}

type invoiceDeals struct { paymentDeals; s *invoiceStore }
func (r invoiceDeals) UpdateOrder(_ context.Context, o *model.Order, expected int64) error {
	if r.s.failOrderCAS { return apperr.ErrStale }
	old := r.s.state.Orders[o.ID]
	if old.Version != expected { return apperr.ErrStale }
	o.Version = expected+1
	r.s.state.Orders[o.ID] = *o
	return nil
}
type invoiceMoneyRows struct { paymentRows; s *invoiceStore }
func (r invoiceMoneyRows) AttachInvoice(ctx context.Context, id, invoiceID string, at time.Time) error {
	if r.s.failAttach { return errors.New("attach failed") }
	return r.paymentRows.AttachInvoice(ctx,id,invoiceID,at)
}
type invoiceRows struct { InvoiceRepository; s *invoiceStore }
func (r invoiceRows) Create(_ context.Context, i *model.Invoice) error {
	if r.s.failCreate { return errors.New("create failed") }
	for _, old := range r.s.invoices { if old.OrderID==i.OrderID && old.Status=="issued" { return apperr.ErrConflict } }
	r.s.invoices[i.ID]=*i
	return nil
}
func (r invoiceRows) Get(_ context.Context, companyID,id string) (*model.Invoice,error) {
	i,ok:=r.s.invoices[id];if !ok||i.BuyerCompanyID!=companyID&&i.SupplierCompanyID!=companyID{return nil,apperr.ErrNotFound};return &i,nil
}
func (r invoiceRows) ForOrder(_ context.Context,id string) ([]model.Invoice,error) {
	out:=[]model.Invoice{};for _,i:=range r.s.invoices{if i.OrderID==id{out=append(out,i)}};return out,nil
}
func (r invoiceRows) Update(_ context.Context,i *model.Invoice,expected int64) error {
	old:=r.s.invoices[i.ID];if old.Version!=expected{return apperr.ErrStale};i.Version=expected+1;r.s.invoices[i.ID]=*i;return nil
}
func (r invoiceRows) AddEvidence(_ context.Context,e *model.Evidence) error {
	if r.s.failInvoiceEvidence { return errors.New("invoice evidence failed") };r.s.evidence[e.ID]=*e;return nil
}
func (r invoiceRows) Evidence(_ context.Context,id string) ([]model.Evidence,error) {
	if !r.s.locked {
		for _,i:=range r.s.invoices {if i.ID==id {if o,ok:=r.s.state.Orders[i.OrderID];ok&&o.Source=="reservation" {return nil,errors.New("invoice evidence read before reservation lock")}}}
	}
	out:=[]model.Evidence{};for _,e:=range r.s.evidence{if e.InvoiceID==id{out=append(out,e)}};return out,nil
}
func (r invoiceRows) GetEvidence(_ context.Context,id string) (*model.Evidence,error) {e,ok:=r.s.evidence[id];if !ok{return nil,apperr.ErrNotFound};return &e,nil}
func (r invoiceRows) UpdateEvidence(_ context.Context,e *model.Evidence,expected int64) error {old:=r.s.evidence[e.ID];if old.Version!=expected{return apperr.ErrStale};e.Version=expected+1;r.s.evidence[e.ID]=*e;return nil}

func invoiceFixture(t *testing.T, credited bool) (*Invoice,*Deal,*invoiceStore,*auth.Principal,*auth.Principal,string,string) {
	t.Helper()
	p,base,buyer,supplier,resID:=paymentFixture()
	orderID:=uuid.NewString()
	terms:=model.Terms{Lines:[]model.Line{{Quantity:1,UnitPrice:money.Money{AmountMinor:"100",Currency:"USD"}}}}
	raw,_:=json.Marshal(terms)
	o:=model.Order{ID:orderID,BuyerCompanyID:buyer.CompanyID,SupplierCompanyID:supplier.CompanyID,Source:"direct",Status:model.OrderAccepted,Terms:raw,Version:1}
	if credited {
		r:=base.state.Reservations[resID]
		snap,_:=decodeReservationSnapshot(r.CurrentTerms)
		advance:=money.Money{AmountMinor:"20",Currency:"USD"}
		snap.Terms.Advance=&advance
		r.CurrentTerms,_=json.Marshal(snap)
		base.state.Reservations[resID]=r
		v,err:=p.Submit(context.Background(),buyer,resID,r.Version,paymentInput("20"));if err!=nil{t.Fatal(err)}
		_,err=p.Decide(context.Background(),supplier,v.Evidence[0].ID,1,v.Revision,true,true,"");if err!=nil{t.Fatal(err)}
		r=base.state.Reservations[resID];r.OrderID=&orderID;base.state.Reservations[resID]=r
		o.Source="reservation"
		base.locked=true
		if _,err=p.LinkCreditLocked(context.Background(),base,&r,&o);err!=nil{t.Fatal(err)}
	}
	base.state.Orders[orderID]=o
	st:=&invoiceStore{paymentStore:base,invoices:map[string]model.Invoice{},evidence:map[string]model.Evidence{}}
	supplier.Permissions[model.PermInvoicesManage]=true
	d:=NewDeps(st,nil,nil,nil,paymentFiles{s:base,owner:buyer.CompanyID},func()time.Time{return time.Date(2026,10,11,12,0,0,0,time.UTC)})
	return NewInvoice(d),NewDeal(d,nil),st,buyer,supplier,resID,orderID
}

func TestReservationCredit_InvoiceAccounting(t *testing.T) {
	ctx:=context.Background()
	s,deal,st,buyer,supplier,resID,orderID:=invoiceFixture(t,true)
	if _,err:=deal.CancelOrder(ctx,buyer,orderID,1,"cancel");err==nil{t.Fatal("credited order cancelled before invoice")}
	issued,err:=s.Issue(ctx,supplier,orderID,1,"2026-11-11");if err!=nil{t.Fatal(err)}
	if issued.Invoice.TotalMinor!="100"||issued.Paid.AmountMinor!="20"||issued.Outstanding.AmountMinor!="80"||len(issued.Evidence)!=0{t.Fatalf("issue accounting: %#v",issued)}
	if st.state.Credits[resID].InvoiceID==nil||*st.state.Credits[resID].InvoiceID!=issued.Invoice.ID{t.Fatal("credit not attached")}
	if _,err=s.Void(ctx,supplier,issued.Invoice.ID,1,"change terms");err==nil{t.Fatal("credited invoice voided")}
	v,err:=s.SubmitEvidence(ctx,buyer,issued.Invoice.ID,paymentInput("30"));if err!=nil{t.Fatal(err)}
	if v.Paid.AmountMinor!="20"||v.Pending.AmountMinor!="30"||v.Outstanding.AmountMinor!="80"||len(v.Evidence)!=1{t.Fatalf("pending changed paid: %#v",v)}
	if _,err=s.SubmitEvidence(ctx,buyer,issued.Invoice.ID,paymentInput("51"));err==nil{t.Fatal("pending capacity exceeded")}
	v,err=s.DecideEvidence(ctx,supplier,v.Evidence[0].ID,1,true,true,"");if err!=nil{t.Fatal(err)}
	if v.Paid.AmountMinor!="50"||v.Pending.AmountMinor!="0"||v.Outstanding.AmountMinor!="50"||len(v.Evidence)!=1{t.Fatalf("accepted ordinary evidence: %#v",v)}
	if _,err=s.SubmitEvidence(ctx,buyer,issued.Invoice.ID,paymentInput("51"));err==nil{t.Fatal("accepted capacity exceeded")}
	for n:=0;n<2;n++ {got,err:=s.Get(ctx,buyer,issued.Invoice.ID);if err!=nil||got.Paid.AmountMinor!="50"{t.Fatalf("read %d: %#v %v",n,got,err)}}
	if _,err=s.Issue(ctx,supplier,orderID,st.state.Orders[orderID].Version,"");err!=nil{t.Fatalf("repeated issue: %v",err)}
	if len(st.invoices)!=1||st.state.Events!=5{t.Fatalf("repeated issue wrote invoice/event: %d/%d",len(st.invoices),st.state.Events)}
	void:=issued.Invoice;void.ID=uuid.NewString();void.Status="void";st.invoices[void.ID]=void
	list,err:=s.ForOrder(ctx,buyer,orderID);if err!=nil||len(list)!=2{t.Fatalf("invoice history: %v %#v",err,list)}
	paidCount:=0;for _,v:=range list {if v.Paid.AmountMinor=="50"{paidCount++} else if v.Paid.AmountMinor!="0"{t.Fatalf("void history received credit: %#v",v)}};if paidCount!=1{t.Fatal("credit counted more than once")}
}

func TestReservationCredit_InvoiceRollbackAndCompatibility(t *testing.T) {
	ctx:=context.Background()
	for name,fail:=range map[string]func(*invoiceStore){
		"create":func(s *invoiceStore){s.failCreate=true},
		"attach":func(s *invoiceStore){s.failAttach=true},
		"order CAS":func(s *invoiceStore){s.failOrderCAS=true},
		"event":func(s *invoiceStore){s.failEvent=true},
	} {t.Run(name,func(t *testing.T){s,_,st,_,supplier,resID,orderID:=invoiceFixture(t,true);fail(st)
		if _,err:=s.Issue(ctx,supplier,orderID,1,"2026-11-11");err==nil{t.Fatal("injected failure succeeded")}
		if len(st.invoices)!=0||st.state.Credits[resID].InvoiceID!=nil||st.state.Orders[orderID].Version!=1||st.state.Events!=2{t.Fatalf("partial issue survived: %#v %#v",st.invoices,st.state.Credits[resID])}
	})}
	t.Run("ordinary order",func(t *testing.T){s,_,st,buyer,supplier,_,orderID:=invoiceFixture(t,false)
		v,err:=s.Issue(ctx,supplier,orderID,1,"2026-11-11");if err!=nil||v.Paid.AmountMinor!="0"||v.Outstanding.AmountMinor!="100"{t.Fatalf("ordinary issue: %#v %v",v,err)}
		if _,err=s.Void(ctx,supplier,v.Invoice.ID,1,"change terms");err!=nil{t.Fatalf("ordinary void: %v",err)}
		if _,err=s.SubmitEvidence(ctx,buyer,v.Invoice.ID,paymentInput("1"));err==nil{t.Fatal("void invoice took payment")}
		if len(st.state.Credits)!=0{t.Fatal("ordinary order acquired credit")}
	})
	t.Run("ordinary accepted payment still blocks cancellation",func(t *testing.T){s,deal,st,buyer,supplier,_,orderID:=invoiceFixture(t,false)
		v,err:=s.Issue(ctx,supplier,orderID,1,"2026-11-11");if err!=nil{t.Fatal(err)}
		claim,err:=s.SubmitEvidence(ctx,buyer,v.Invoice.ID,paymentInput("10"));if err!=nil{t.Fatal(err)}
		if _,err=s.DecideEvidence(ctx,supplier,claim.Evidence[0].ID,1,true,true,"");err!=nil{t.Fatal(err)}
		if _,err=deal.CancelOrder(ctx,buyer,orderID,2,"cancel");err==nil{t.Fatal("ordinary accepted payment allowed cancellation")}
		if st.state.Orders[orderID].Status!=model.OrderAccepted{t.Fatal("failed cancellation mutated order")}
	})
	t.Run("existing issued invoice receives one association",func(t *testing.T){s,_,st,_,supplier,resID,orderID:=invoiceFixture(t,true)
		i:=model.Invoice{ID:uuid.NewString(),OrderID:orderID,BuyerCompanyID:st.state.Orders[orderID].BuyerCompanyID,SupplierCompanyID:supplier.CompanyID,Status:"issued",Currency:"USD",TotalMinor:"100",Version:1}
		st.invoices[i.ID]=i
		v,err:=s.Issue(ctx,supplier,orderID,1,"");if err!=nil||v.Invoice.ID!=i.ID||v.Paid.AmountMinor!="20"{t.Fatalf("reconcile: %#v %v",v,err)}
		if len(st.invoices)!=1||*st.state.Credits[resID].InvoiceID!=i.ID||st.state.Events!=3{t.Fatal("reconcile made a second invoice or skipped event")}
		if _,err=s.Issue(ctx,supplier,orderID,2,"");err!=nil||st.state.Events!=3{t.Fatalf("repeat reconcile mutated: %v",err)}
	})
	t.Run("existing pending claim cannot overfill after association",func(t *testing.T){s,_,st,_,supplier,resID,orderID:=invoiceFixture(t,true)
		i:=model.Invoice{ID:uuid.NewString(),OrderID:orderID,BuyerCompanyID:st.state.Orders[orderID].BuyerCompanyID,SupplierCompanyID:supplier.CompanyID,Status:"issued",Currency:"USD",TotalMinor:"100",Version:1}
		st.invoices[i.ID]=i
		e:=model.Evidence{ID:uuid.NewString(),InvoiceID:i.ID,AmountMinor:"90",Currency:"USD",Status:"submitted",Version:1}
		st.evidence[e.ID]=e
		if _,err:=s.Issue(ctx,supplier,orderID,1,"");err==nil{t.Fatal("overfilled association accepted")}
		if st.state.Credits[resID].InvoiceID!=nil||st.state.Orders[orderID].Version!=1||st.state.Events!=2{t.Fatal("overfilled association partially committed")}
	})
	t.Run("invalid source and duplicate issued fail closed",func(t *testing.T){s,_,st,_,supplier,resID,orderID:=invoiceFixture(t,true)
		o:=st.state.Orders[orderID];o.Source="direct";st.state.Orders[orderID]=o
		if _,err:=s.Issue(ctx,supplier,orderID,1,"");err==nil{t.Fatal("wrong source issued")}
		o.Source="reservation";o.BuyerCompanyID=uuid.NewString();st.state.Orders[orderID]=o
		if _,err:=s.Issue(ctx,supplier,orderID,1,"");err==nil{t.Fatal("wrong party source issued")}
		o.BuyerCompanyID=st.state.Reservations[resID].BuyerCompanyID;o.Terms,_=json.Marshal(model.Terms{Lines:[]model.Line{{Quantity:1,UnitPrice:money.Money{AmountMinor:"100",Currency:"EUR"}}}});st.state.Orders[orderID]=o
		if _,err:=s.Issue(ctx,supplier,orderID,1,"");err==nil{t.Fatal("wrong currency source issued")}
		o.Terms,_=json.Marshal(model.Terms{Lines:[]model.Line{{Quantity:1,UnitPrice:money.Money{AmountMinor:"100",Currency:"USD"}}}});st.state.Orders[orderID]=o
		for j:=0;j<2;j++{i:=model.Invoice{ID:uuid.NewString(),OrderID:orderID,BuyerCompanyID:o.BuyerCompanyID,SupplierCompanyID:o.SupplierCompanyID,Status:"issued",Currency:"USD",TotalMinor:"100"};st.invoices[i.ID]=i}
		if _,err:=s.Issue(ctx,supplier,orderID,1,"");err==nil||st.state.Credits[resID].InvoiceID!=nil{t.Fatal("ambiguous invoice associated")}
	})
}

func TestReservationCredit_InvoiceLockedCapacityRecheck(t *testing.T) {
	ctx:=context.Background()
	s,_,st,buyer,supplier,_,orderID:=invoiceFixture(t,true)
	issued,err:=s.Issue(ctx,supplier,orderID,1,"2026-11-11");if err!=nil{t.Fatal(err)}
	// A payment accepted between the locator read and source lock consumes
	// capacity. The attempted 80 claim must use the post-lock 50 remainder.
	st.beforeSourceLock=func(s *invoiceStore){e:=model.Evidence{ID:uuid.NewString(),InvoiceID:issued.Invoice.ID,AmountMinor:"30",Currency:"USD",Status:"accepted",Version:2};s.evidence[e.ID]=e;s.committedEvidence=map[string]model.Evidence{e.ID:e}}
	if _,err=s.SubmitEvidence(ctx,buyer,issued.Invoice.ID,paymentInput("80"));err==nil{t.Fatal("stale capacity admitted payment")}
	if len(st.evidence)!=1||st.state.Shares!=1{t.Fatal("rejected claim left write behind")}
	v,err:=s.Get(ctx,buyer,issued.Invoice.ID);if err!=nil||v.Paid.AmountMinor!="50"||v.Outstanding.AmountMinor!="50"{t.Fatalf("post-lock accounting: %#v %v",v,err)}
	if _,err=s.SubmitEvidence(ctx,buyer,issued.Invoice.ID,paymentInput("50"));err!=nil{t.Fatalf("valid remaining capacity: %v",err)}
}
