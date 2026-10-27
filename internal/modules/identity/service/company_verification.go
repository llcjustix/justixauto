package service

import (
	"context"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

// SetSellerUserType records the platform-controlled type of a seller. Legacy
// sellers may be classified once; a classified company cannot be reclassified.
func (s *Company) SetSellerUserType(ctx context.Context, actor *auth.Principal, id string, expected int64, userType model.SellerUserType) (*model.Company, error) {
	if err := actor.Allow(model.PermPlatformCompaniesAccess); err != nil {
		return nil, err
	}
	if err := validID(id); err != nil {
		return nil, err
	}
	if !userType.Valid() {
		return nil, apperr.FieldError("userType", "must be dealer or distributor")
	}
	c, err := s.store.Companies().Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Kind != model.KindSeller {
		return nil, apperr.FieldError("userType", "applies only to seller companies")
	}
	if c.UserType != nil {
		return nil, apperr.New(apperr.ErrConflict, "company_user_type_set", "seller user type is already set")
	}
	if c.Version != expected {
		return nil, apperr.ErrStale
	}
	before := auditCompanyValues(c)
	c.UserType, c.UpdatedAt = &userType, s.clock()
	if err := s.store.InTx(ctx, func(st Store) error {
		if err := st.Companies().Update(ctx, c, expected); err != nil {
			return err
		}
		return s.auditChange(ctx, st, actor, "company.user_type_set", "company", c.ID, &c.ID, "", map[string]any{"userType": userType}, before, auditCompanyValues(c))
	}); err != nil {
		return nil, err
	}
	return c, nil
}

// SubmitVerification lets a company administrator submit or resubmit its
// optional profile information for platform review.
func (s *Company) SubmitVerification(ctx context.Context, actor *auth.Principal, id string, expected int64) (*model.Company, error) {
	c, err := s.companyVerificationMember(ctx, actor, id, model.PermCompanyEdit)
	if err != nil {
		return nil, err
	}
	if c.Version != expected {
		return nil, apperr.ErrStale
	}
	if !canSubmitVerification(c.Verification) {
		return nil, apperr.New(apperr.ErrConflict, "invalid_verification_transition", "company verification cannot be submitted from "+string(c.Verification))
	}
	before := c.Verification
	beforeValues := auditCompanyValues(c)
	c.Verification, c.VerificationReason, c.VerificationActorID, c.VerificationAt, c.UpdatedAt = model.VerificationSubmitted, "", nil, nil, s.clock()
	if err := s.store.InTx(ctx, func(st Store) error {
		if err := st.Companies().Update(ctx, c, expected); err != nil {
			return err
		}
		return s.auditChange(ctx, st, actor, "company.verification_submitted", "company", c.ID, &c.ID, "", map[string]any{"before": before, "after": c.Verification}, beforeValues, auditCompanyValues(c))
	}); err != nil {
		return nil, err
	}
	return c, nil
}

// SetVerification is the platform review action. Only submitted companies can
// be verified or returned; a return must explain what the company should fix.
func (s *Company) SetVerification(ctx context.Context, actor *auth.Principal, id string, expected int64, verification model.CompanyVerification, why string) (*model.Company, error) {
	if err := actor.Allow(model.PermPlatformCompaniesAccess); err != nil {
		return nil, err
	}
	if verification != model.VerificationVerified && verification != model.VerificationReturned {
		return nil, apperr.ErrNotFound
	}
	if err := validID(id); err != nil {
		return nil, err
	}
	c, err := s.store.Companies().Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if c.Version != expected {
		return nil, apperr.ErrStale
	}
	if !canReviewVerification(c.Verification, verification) {
		return nil, apperr.New(apperr.ErrConflict, "invalid_verification_transition", "company verification is not submitted")
	}
	var v apperr.Validation
	if verification == model.VerificationReturned {
		why = reason(&v, why)
	} else {
		why = ""
	}
	if err := v.Err(); err != nil {
		return nil, err
	}
	beforeValues := auditCompanyValues(c)
	before, now := c.Verification, s.clock()
	c.Verification, c.VerificationReason, c.VerificationActorID, c.VerificationAt, c.UpdatedAt = verification, why, &actor.UserID, &now, now
	if err := s.store.InTx(ctx, func(st Store) error {
		if err := st.Companies().Update(ctx, c, expected); err != nil {
			return err
		}
		return s.auditChange(ctx, st, actor, "company.verification_"+string(verification), "company", c.ID, &c.ID, why, map[string]any{"before": before, "after": verification}, beforeValues, auditCompanyValues(c))
	}); err != nil {
		return nil, err
	}
	return c, nil
}

func canSubmitVerification(status model.CompanyVerification) bool {
	return status == model.VerificationUnverified || status == model.VerificationReturned
}

func canReviewVerification(current, next model.CompanyVerification) bool {
	return current == model.VerificationSubmitted && (next == model.VerificationVerified || next == model.VerificationReturned)
}

func (s *Company) companyVerificationMember(ctx context.Context, actor *auth.Principal, id, permission string) (*model.Company, error) {
	if err := validID(id); err != nil {
		return nil, err
	}
	member, err := s.isMember(ctx, s.store, actor.UserID, id)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, apperr.New(apperr.ErrForbidden, "permission_denied", "missing permission "+permission)
	}
	if err := actor.Allow(permission); err != nil {
		return nil, err
	}
	return s.store.Companies().Get(ctx, id)
}
