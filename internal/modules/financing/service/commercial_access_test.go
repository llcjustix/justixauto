package service

import (
	"context"
	"errors"
	"testing"

	"justixauto/internal/modules/financing/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

type commercialFinancingRepo struct {
	Repository
	created bool
}

func (r *commercialFinancingRepo) CreateApplication(context.Context, *model.Application) error {
	r.created = true
	return nil
}

func TestCommercialAccessBlocksNewFinancingApplicationBeforeWrites(t *testing.T) {
	repo := &commercialFinancingRepo{}
	_, err := (&Service{repo: repo}).Create(context.Background(), &auth.Principal{CompanyID: "seller"}, ApplicationInput{})
	var ae *apperr.Error
	if !errors.As(err, &ae) || ae.Code != "company_verification_required" || repo.created {
		t.Fatalf("Create error = %v, created = %v", err, repo.created)
	}
}
