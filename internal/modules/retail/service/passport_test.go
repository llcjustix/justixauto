package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"justixauto/internal/modules/retail/model"
	"justixauto/internal/pkg/auth"
)

const (
	passportCompany = "11111111-1111-1111-1111-111111111111"
	passportActor   = "22222222-2222-2222-2222-222222222222"
	passportFile    = "33333333-3333-3333-3333-333333333333"
)

type passportCRM struct {
	CRMRepository
	customer *model.Customer
}

func (r *passportCRM) CreateCustomer(_ context.Context, c *model.Customer) error {
	copy := *c
	r.customer = &copy
	return nil
}

func (r *passportCRM) Customer(_ context.Context, companyID, id string) (*model.Customer, error) {
	if r.customer == nil || r.customer.CompanyID != companyID || r.customer.ID != id {
		return nil, errors.New("customer not found")
	}
	copy := *r.customer
	return &copy, nil
}

func (r *passportCRM) UpdateCustomer(_ context.Context, c *model.Customer, expected int64) error {
	if r.customer == nil || r.customer.Version != expected {
		return errors.New("stale customer")
	}
	copy := *c
	copy.Version = expected + 1
	r.customer = &copy
	c.Version = copy.Version
	return nil
}

type passportEvents struct{ events []*model.Event }

func (e *passportEvents) Append(_ context.Context, event *model.Event) error {
	e.events = append(e.events, event)
	return nil
}
func (*passportEvents) For(context.Context, string, string) ([]model.Event, error) { return nil, nil }

type passportStoreRetail struct {
	crm    *passportCRM
	events *passportEvents
}

func (s *passportStoreRetail) CRM() CRMRepository                     { return s.crm }
func (*passportStoreRetail) Listings() ListingRepository              { return nil }
func (*passportStoreRetail) Deals() DealRepository                    { return nil }
func (*passportStoreRetail) Contracts() ContractRepository            { return nil }
func (s *passportStoreRetail) Events() EventRepository                { return s.events }
func (*passportStoreRetail) Bind(ctx context.Context) context.Context { return ctx }
func (s *passportStoreRetail) InTx(_ context.Context, fn func(Store) error) error {
	var before *model.Customer
	if s.crm.customer != nil {
		copy := *s.crm.customer
		before = &copy
	}
	events := append([]*model.Event(nil), s.events.events...)
	if err := fn(s); err != nil {
		s.crm.customer, s.events.events = before, events
		return err
	}
	return nil
}

type passportFiles struct {
	bound []string
	fail  bool
}

func (*passportFiles) Share(context.Context, string, string, string, string, string) error {
	return nil
}
func (f *passportFiles) BindPassport(_ context.Context, owner, fileID, resourceType, resourceID string) error {
	if f.fail {
		return errors.New("passport file cannot bind")
	}
	f.bound = append(f.bound, strings.Join([]string{owner, fileID, resourceType, resourceID}, ":"))
	return nil
}

func passportInput() CustomerInput {
	var in CustomerInput
	in.Profile.DisplayName = "Buyer"
	in.Profile.PassportSeries = "AA"
	in.Profile.PassportNumber = "1234567"
	in.Profile.PassportIssuer = "Authority"
	in.Profile.PassportIssuedOn = "2020-02-03"
	in.Profile.PassportExpiresOn = "2030-02-03"
	in.Profile.BirthDate = "1990-02-03"
	in.Profile.PersonalID = "PID-1"
	in.Profile.RegisteredAddress = "Tashkent"
	in.Profile.PassportCopyFileIDs = []string{passportFile}
	return in
}

func TestPassportProfile(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	principal := &auth.Principal{UserID: passportActor, CompanyID: passportCompany}
	newCRM := func(files *passportFiles) (*CRM, *passportStoreRetail) {
		st := &passportStoreRetail{crm: &passportCRM{}, events: &passportEvents{}}
		return NewCRM(NewDeps(st, nil, nil, files, func() time.Time { return now })), st
	}

	t.Run("roundtrips and updates optional profile", func(t *testing.T) {
		files := &passportFiles{}
		s, st := newCRM(files)
		customer, err := s.CreateCustomer(context.Background(), principal, passportInput())
		if err != nil {
			t.Fatal(err)
		}
		if customer.PassportIssuedOn == nil || customer.PassportIssuedOn.Format(time.DateOnly) != "2020-02-03" || len(files.bound) != 1 {
			t.Fatalf("customer=%+v bindings=%v", customer, files.bound)
		}
		update := passportInput()
		update.Profile.RegisteredAddress = "Samarkand"
		updated, err := s.UpdateCustomer(context.Background(), principal, customer.ID, customer.Version, update)
		if err != nil || updated.RegisteredAddress != "Samarkand" || st.crm.customer.RegisteredAddress != "Samarkand" {
			t.Fatalf("updated=%+v err=%v", updated, err)
		}
		for _, event := range st.events.events {
			if strings.Contains(string(event.Details), "1234567") || strings.Contains(string(event.Details), "Tashkent") {
				t.Fatalf("passport payload leaked into event: %s", event.Details)
			}
		}
	})

	t.Run("rejects malformed dates", func(t *testing.T) {
		s, _ := newCRM(&passportFiles{})
		in := passportInput()
		in.Profile.PassportIssuedOn = "03-02-2020"
		if _, err := s.CreateCustomer(context.Background(), principal, in); err == nil {
			t.Fatal("malformed date accepted")
		}
	})

	t.Run("rolls back customer when binding fails", func(t *testing.T) {
		s, st := newCRM(&passportFiles{fail: true})
		if _, err := s.CreateCustomer(context.Background(), principal, passportInput()); err == nil {
			t.Fatal("binding failure accepted")
		}
		if st.crm.customer != nil || len(st.events.events) != 0 {
			t.Fatalf("customer or event escaped rollback: %+v %+v", st.crm.customer, st.events.events)
		}
	})
}
