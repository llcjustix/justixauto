package service

import (
	"context"
	"testing"
	"time"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
)

type contractCompanies struct{ CompanyRepository; company *model.Company }
func (r contractCompanies) GetIncludingDeleted(context.Context, string) (*model.Company, error) { return r.company, nil }
type contractStore struct{ Store; companies contractCompanies }

func (s contractStore) Companies() CompanyRepository { return s.companies }

func TestContractRequisites(t *testing.T) {
	c := &model.Company{ID: "11111111-1111-1111-1111-111111111111", Name: "Seller", LegalName: "Seller LLC", RegistrationNumber: "R-1", TaxID: "T-1", BankRequisites: "bank", Address: "address", Phone: "phone", Email: "mail@example.test"}
	s := NewCompany(NewDeps(contractStore{companies: contractCompanies{company: c}}, func() time.Time { return time.Time{} }))
	got, err := s.ContractRequisitesFor(context.Background(), c.ID, c.ID)
	if err != nil || got.BankRequisites != "bank" || got.TaxID != "T-1" { t.Fatalf("requisites=%+v err=%v", got, err) }
	if _, err := s.ContractRequisitesFor(context.Background(), c.ID, "22222222-2222-2222-2222-222222222222"); err != apperr.ErrNotFound { t.Fatalf("unrelated company error = %v", err) }
}
