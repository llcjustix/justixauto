package app

import (
	"context"
	"errors"
	"testing"

	"justixauto/internal/modules/commerce"
	"justixauto/internal/modules/documents"
	"justixauto/internal/modules/identity"
	"justixauto/internal/pkg/apperr"
)

type workflowOrderLookup struct {
	order                 *commerce.Order
	err                   error
	seenViewer, seenOrder string
	marker                any
}

func (f *workflowOrderLookup) OrderPartiesFor(ctx context.Context, viewer, order string) (*commerce.Order, error) {
	f.seenViewer, f.seenOrder = viewer, order
	f.marker = ctx.Value("marker")
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
	ctx := context.WithValue(context.Background(), "marker", "retained")
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
	if _, err := a.ContractRequisitesFor(context.Background(), "seller", "other"); err != apperr.ErrNotFound {
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
