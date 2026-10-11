package service

import (
	"context"
	"testing"

	"justixauto/internal/modules/documents/model"
	"justixauto/internal/pkg/apperr"
)

type passportStore struct {
	owned  *model.File
	shared *model.Share
}

func (s *passportStore) Create(context.Context, *model.File) error { return nil }
func (s *passportStore) ByContent(context.Context, string, string, string) (*model.File, error) {
	return nil, apperr.ErrNotFound
}
func (s *passportStore) Readable(context.Context, string, string) (*model.File, error) {
	return s.owned, nil
}
func (s *passportStore) Owned(_ context.Context, companyID, id string) (*model.File, error) {
	if s.owned == nil || s.owned.CompanyID != companyID || s.owned.ID != id {
		return nil, apperr.ErrNotFound
	}
	return s.owned, nil
}
func (s *passportStore) Share(_ context.Context, ownerCompanyID, fileID string, sh *model.Share) (*model.File, error) {
	if s.owned == nil || s.owned.CompanyID != ownerCompanyID || s.owned.ID != fileID {
		return nil, apperr.ErrNotFound
	}
	s.shared = sh
	return s.owned, nil
}

func TestBindPassport(t *testing.T) {
	const owner = "11111111-1111-1111-1111-111111111111"
	const fileID = "22222222-2222-2222-2222-222222222222"
	const customerID = "33333333-3333-3333-3333-333333333333"

	t.Run("requires owned sensitive passport copy", func(t *testing.T) {
		for _, file := range []*model.File{
			{ID: fileID, CompanyID: owner, Purpose: "vehicle-photo", Sensitive: false},
			{ID: fileID, CompanyID: owner, Purpose: "passport-copy", Sensitive: false},
			{ID: fileID, CompanyID: "44444444-4444-4444-4444-444444444444", Purpose: "passport-copy", Sensitive: true},
		} {
			st := &passportStore{owned: file}
			s := New(st, nil, nil)
			if _, err := s.BindPassport(context.Background(), owner, fileID, "retail.customer", customerID); err == nil {
				t.Fatal("invalid passport binding succeeded")
			}
			if st.shared != nil {
				t.Fatal("binding was created before passport validation")
			}
		}
	})

	t.Run("shares only with owning tenant and customer resource", func(t *testing.T) {
		st := &passportStore{owned: &model.File{ID: fileID, CompanyID: owner, Purpose: "passport-copy", Sensitive: true}}
		s := New(st, nil, nil)
		if _, err := s.BindPassport(context.Background(), owner, fileID, "retail.customer", customerID); err != nil {
			t.Fatal(err)
		}
		if st.shared == nil || st.shared.CompanyID != owner || st.shared.ResourceType != "retail.customer" || st.shared.ResourceID != customerID {
			t.Fatalf("share = %+v", st.shared)
		}
	})
}
