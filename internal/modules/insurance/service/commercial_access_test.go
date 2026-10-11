package service

import (
	"context"
	"errors"
	"testing"

	"justixauto/internal/modules/insurance/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

type commercialInsuranceRepo struct {
	Repository
	created bool
}

func (r *commercialInsuranceRepo) Create(context.Context, *model.Application) error {
	r.created = true
	return nil
}

func TestCommercialAccessBlocksNewInsuranceApplicationBeforeWrites(t *testing.T) {
	repo := &commercialInsuranceRepo{}
	_, err := (&Service{repo: repo}).Create(context.Background(), &auth.Principal{CompanyID: "seller"}, CreateInput{})
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != "company_verification_required" || repo.created {
		t.Fatalf("Create error = %v, created = %v", err, repo.created)
	}
}
