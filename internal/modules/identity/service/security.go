package service

import (
	"context"

	"justixauto/internal/pkg/auth"
)

// RecordRequestDenial observes a completed request pipeline. Only fixed denial
// codes are accepted; explicit login/context/session events take precedence,
// including a failed persistence attempt. The independent writer is bounded
// and the original denial remains authoritative regardless of its result.
func (s *Auth) RecordRequestDenial(ctx context.Context, actor *auth.Principal, code SecurityCode) {
	if SecurityEventAttempted(ctx) {
		return
	}
	switch code {
	case SecurityUnauthenticated, SecurityPermissionDenied, SecurityOriginDenied,
		SecurityCSRFInvalid, SecurityPasswordChangeRequired, SecurityInvalidCredentials,
		SecurityAccountLocked, SecurityAccountSuspended, SecurityNotMember,
		SecurityCompanySuspended, SecurityNoActiveCompany, SecurityContextChanged,
		SecurityBranchScopeDenied:
	default:
		return
	}
	var companyID *string
	if actor != nil && actor.CompanyID != "" {
		id := actor.CompanyID
		companyID = &id
	}
	_ = s.RecordSecurity(ctx, SecurityEvent{
		Action: SecurityRequestDenied, Outcome: AuditDenied, ErrorCode: code,
		Actor: actor, CompanyID: companyID,
	})
}
