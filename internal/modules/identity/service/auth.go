package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
)

type SessionConfig struct {
	IdleTimeout      time.Duration
	AbsoluteTimeout  time.Duration
	LockoutThreshold int
	LockoutDuration  time.Duration
}

var DefaultSessionConfig = SessionConfig{
	IdleTimeout:      30 * time.Minute,
	AbsoluteTimeout:  12 * time.Hour,
	LockoutThreshold: 5,
	LockoutDuration:  15 * time.Minute,
}

// Auth manages login, sessions and the working company/branch context.
type Auth struct {
	Deps
	cfg SessionConfig
}

// NewAuth builds the auth service.
func NewAuth(d Deps, cfg SessionConfig) *Auth { return &Auth{Deps: d, cfg: cfg} }

// AbsoluteTimeout returns how long a session cookie lives once issued.
func (s *Auth) AbsoluteTimeout() time.Duration { return s.cfg.AbsoluteTimeout }

var errInvalidCredentials = apperr.New(apperr.ErrUnauthenticated, "invalid_credentials", "login or password is incorrect")

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func tokenHash(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// DeviceCorrelationHash accepts only canonical base64url of exactly 32 random
// bytes. It is correlation evidence, never proof of an authenticated identity.
func DeviceCorrelationHash(cookie string) string {
	if len(cookie) != 43 {
		return ""
	}
	decoded, err := base64.RawURLEncoding.Strict().DecodeString(cookie)
	if err != nil || len(decoded) != 32 || base64.RawURLEncoding.EncodeToString(decoded) != cookie {
		return ""
	}
	hash := sha256.Sum256([]byte("justixauto:device:v1\x00" + cookie))
	return hex.EncodeToString(hash[:])
}

// Login verifies credentials and starts a session, returning its raw cookie
// token. A previous session of the same browser (previousToken) is revoked,
// so the cookie always rotates.
func (s *Auth) Login(ctx context.Context, login, password, previousToken string) (string, error) {
	token, _, err := s.LoginWithDevice(ctx, login, password, previousToken, "")
	return token, err
}

// LoginWithDevice also recognizes this account's browser-cookie digest. A raw
// device candidate is returned only when its successful transaction committed;
// HTTP callers may then issue it. It is independent of the session/CSRF token.
func (s *Auth) LoginWithDevice(ctx context.Context, login, password, previousToken, deviceCookie string) (token, newDeviceCookie string, resultErr error) {
	deviceHash := DeviceCorrelationHash(deviceCookie)
	// Preserve the original supplied-cookie evidence for failure recording.
	failureDeviceHash := deviceHash
	failure := SecurityDetails{FailureReason: LoginInternalError, AttemptedLogin: login}
	failureCode, failureOutcome := SecurityInternalError, AuditError
	var targetID *string
	defer func() {
		if resultErr != nil {
			_ = s.RecordSecurity(ctx, SecurityEvent{
				Action: SecurityLoginFailed, Outcome: failureOutcome, ErrorCode: failureCode,
				ResourceType: "user", ResourceID: targetID, Details: failure, DeviceHash: failureDeviceHash,
			})
		}
	}()
	now := s.clock()
	u, err := s.store.Users().FindByLogin(ctx, strings.TrimSpace(login))
	if err == nil {
		targetID = &u.ID
		if u.Login != nil {
			failure.TargetLogin = *u.Login
		}
	}
	if errors.Is(err, apperr.ErrNotFound) || (err == nil && u.PasswordHash == nil) {
		_, _ = verifyPassword(password, dummyHash) // equalize timing
		failure.FailureReason, failureCode, failureOutcome = LoginUnknown, SecurityInvalidCredentials, AuditDenied
		if err == nil {
			failure.FailureReason = LoginNoCredentials
		}
		return "", "", errInvalidCredentials
	}
	if err != nil {
		return "", "", err
	}
	if u.LockedUntil != nil && u.LockedUntil.After(now) {
		failure.FailureReason, failureCode, failureOutcome = LoginLocked, SecurityAccountLocked, AuditDenied
		return "", "", &apperr.RateLimitedError{RetryAfter: u.LockedUntil.Sub(now)}
	}
	ok, err := verifyPassword(password, *u.PasswordHash)
	if err != nil {
		return "", "", err
	}
	if !ok {
		if err := s.recordFailure(ctx, u); err != nil {
			return "", "", err
		}
		failure.FailureReason, failureCode, failureOutcome = LoginBadPassword, SecurityInvalidCredentials, AuditDenied
		return "", "", errInvalidCredentials
	}
	if u.Status != model.UserActive {
		failure.FailureReason, failureCode, failureOutcome = LoginSuspended, SecurityAccountSuspended, AuditDenied
		return "", "", apperr.New(apperr.ErrForbidden, "account_suspended", "this account is suspended")
	}
	if deviceHash == "" {
		newDeviceCookie, err = randomToken()
		if err != nil {
			return "", "", err
		}
		deviceHash = DeviceCorrelationHash(newDeviceCookie)
	}
	token, _, err = s.startSession(ctx, u, previousToken, deviceHash)
	if err != nil {
		return "", "", err
	}
	MarkSecurityEventAttempted(ctx)
	return token, newDeviceCookie, nil
}

// recordFailure counts a failed password and locks the
// account for a while after too many in a row.
func (s *Auth) recordFailure(ctx context.Context, u *model.User) error {
	return s.store.Users().RecordFailure(ctx, u.ID, s.cfg.LockoutThreshold, s.clock().Add(s.cfg.LockoutDuration))
}

// startSession creates a session and returns its raw cookie token, which is
// never stored.
func (s *Auth) startSession(ctx context.Context, u *model.User, previousToken, deviceHash string) (string, *model.Session, error) {
	now := s.clock()
	token, err := randomToken()
	if err != nil {
		return "", nil, err
	}
	csrf, err := randomToken()
	if err != nil {
		return "", nil, err
	}
	sess := &model.Session{
		ID: uuid.NewString(), TokenHash: tokenHash(token), CSRFToken: csrf, UserID: u.ID,
		BranchScopeMode: model.ScopeAll, BranchIDs: []string{}, ContextRevision: 1,
		CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(s.cfg.AbsoluteTimeout),
	}
	err = s.store.InTx(ctx, func(st Store) error {
		if err := st.Users().SetLoginState(ctx, u.ID, 0, nil); err != nil {
			return err
		}
		if previousToken != "" {
			if prev, err := st.Sessions().FindLive(ctx, tokenHash(previousToken)); err == nil {
				if err := s.endSession(ctx, st, prev, nil, SecuritySessionRevoked, "browser_login_rotated", deviceHash, now); err != nil {
					return err
				}
			}
		}
		if err := st.Sessions().Create(ctx, sess); err != nil {
			return err
		}
		known, err := st.KnownDevices().Recognize(ctx, u.ID, deviceHash, now)
		if err != nil {
			return err
		}
		// The principal deliberately describes the new session, even when the
		// request arrived with an older authenticated browser session.
		p := sessionPrincipal(u, sess)
		return s.AuditSecurity(ctx, st, SecurityEvent{
			Action: SecurityLoginSucceeded, Outcome: AuditSuccess, Actor: p,
			ResourceType: "user", ResourceID: &u.ID, DeviceHash: deviceHash,
			SessionHash: auth.SessionCorrelationHash(sess.ID), SessionStartedAt: &sess.CreatedAt,
			Details: SecurityDetails{KnownDevice: &known},
		})
	})
	if err != nil {
		return "", nil, err
	}
	return token, sess, nil
}

// Authenticate resolves a cookie token into the caller. Expired, revoked or
// unknown sessions and inactive users return ErrUnauthenticated. A company
// context whose membership was revoked is ignored.
func (s *Auth) Authenticate(ctx context.Context, token string) (*auth.Principal, *model.Session, error) {
	return s.AuthenticateWithDevice(ctx, token, "")
}

// AuthenticateWithDevice accepts only a precomputed cookie digest for lifecycle
// correlation; authentication itself continues to depend on the session token.
func (s *Auth) AuthenticateWithDevice(ctx context.Context, token, deviceHash string) (*auth.Principal, *model.Session, error) {
	if token == "" {
		return nil, nil, apperr.ErrUnauthenticated
	}
	sess, err := s.store.Sessions().FindLive(ctx, tokenHash(token))
	if errors.Is(err, apperr.ErrNotFound) {
		return nil, nil, apperr.ErrUnauthenticated
	}
	if err != nil {
		return nil, nil, err
	}
	now := s.clock()
	if now.After(sess.ExpiresAt) || now.Sub(sess.LastSeenAt) > s.cfg.IdleTimeout {
		reason := "expired"
		if !now.After(sess.ExpiresAt) {
			reason = "idle_timeout"
		}
		// As before, an expired token always denies even if persistence fails.
		// RevokeLive + append either commit together or both roll back.
		err := s.store.InTx(ctx, func(st Store) error {
			return s.endSession(ctx, st, sess, nil, SecuritySessionExpired, reason, deviceHash, now)
		})
		if err == nil {
			MarkSecurityEventAttempted(ctx)
		}
		return nil, nil, apperr.ErrUnauthenticated
	}
	u, err := s.store.Users().Get(ctx, sess.UserID)
	if err != nil {
		return nil, nil, err
	}
	if u.Status != model.UserActive {
		return nil, nil, apperr.ErrUnauthenticated
	}
	p := sessionPrincipal(u, sess)
	if err := s.applyActiveCompany(ctx, p, u.ID, sess); err != nil {
		return nil, nil, err
	}
	// Permissions: platform roles plus the roles held in the selected company.
	roles, err := s.rolesIn(ctx, s.store, u.ID, p.CompanyID)
	if err != nil {
		return nil, nil, err
	}
	if err := s.withPermissions(ctx, s.store, roles); err != nil {
		return nil, nil, err
	}
	for _, r := range roles {
		for _, perm := range r.Permissions {
			p.Permissions[perm] = true
		}
	}
	if now.Sub(sess.LastSeenAt) > time.Minute {
		if err := s.store.Sessions().Touch(ctx, sess.ID, now); err != nil {
			return nil, nil, err
		}
	}
	return p, sess, nil
}

// applyActiveCompany scopes the principal to the session's active company while
// the user is still a member and the company is not suspended.
func (s *Auth) applyActiveCompany(ctx context.Context, p *auth.Principal, userID string, sess *model.Session) error {
	if sess.ActiveCompanyID == nil {
		return nil
	}
	member, err := s.isMember(ctx, s.store, userID, *sess.ActiveCompanyID)
	if err != nil || !member {
		return err
	}
	company, err := s.store.Companies().Get(ctx, *sess.ActiveCompanyID)
	if err != nil {
		return err
	}
	if company.Status != model.AccessSuspended {
		p.CompanyID = *sess.ActiveCompanyID
		p.BranchScope = auth.BranchScope{Mode: sess.BranchScopeMode, BranchIDs: sess.BranchIDs}
		p.CommercialAccess = company.Kind != model.KindSeller || (company.UserType != nil && company.Verification == model.VerificationVerified)
	}
	return nil
}

func sessionPrincipal(u *model.User, sess *model.Session) *auth.Principal {
	p := &auth.Principal{
		UserID: u.ID, SessionID: sess.ID, Permissions: map[string]bool{},
		DisplayName: u.DisplayName, SessionStartedAt: sess.CreatedAt, AuthenticationMethod: "password",
		ContextRevision: sess.ContextRevision, BranchScope: auth.BranchScope{Mode: model.ScopeAll, BranchIDs: []string{}},
		PasswordChangeRequired: u.PasswordChangeRequired,
	}
	if u.Login != nil {
		p.Login = *u.Login
	}
	return p
}

func (s *Auth) LogoutActor(ctx context.Context, p *auth.Principal) error {
	return s.logout(ctx, p.SessionID, p)
}

func (s *Auth) logout(ctx context.Context, sessionID string, p *auth.Principal) error {
	err := s.store.InTx(ctx, func(st Store) error {
		sess, err := st.Sessions().FindLiveByID(ctx, sessionID)
		if errors.Is(err, apperr.ErrNotFound) {
			return nil // preserve idempotent logout; no transition to audit
		}
		if err != nil {
			return err
		}
		return s.endSession(ctx, st, sess, p, SecurityLogout, "logout", "", s.clock())
	})
	if err == nil {
		MarkSecurityEventAttempted(ctx)
	}
	return err
}

// endSession is called inside a domain transaction. Targets of expired/rotated
// sessions are not attributed as authenticated callers. Concurrent requests
// append only when the conditional update actually changed the row.
func (s *Auth) endSession(ctx context.Context, st Store, sess *model.Session, actor *auth.Principal, action SecurityAction, reason, deviceHash string, at time.Time) error {
	changed, err := st.Sessions().RevokeLive(ctx, sess.ID, at)
	if err != nil || !changed {
		return err
	}
	return s.AuditSecurity(ctx, st, SecurityEvent{
		Action: action, Outcome: AuditSuccess, Actor: actor,
		ResourceType: "user", ResourceID: &sess.UserID,
		SessionHash: auth.SessionCorrelationHash(sess.ID), SessionStartedAt: &sess.CreatedAt,
		DeviceHash: deviceHash, Details: SecurityDetails{SessionEndReason: reason},
	})
}

// SessionView is the contract's GET /session data.
type SessionView struct {
	User                SessionUser         `json:"user"`
	Roles               []RoleRef           `json:"roles"`
	Permissions         []string            `json:"permissions"`
	Context             ContextView         `json:"context"`
	AccessibleCompanies []AccessibleCompany `json:"accessibleCompanies"`
	Setup               SetupView           `json:"setup"`
}

type SessionUser struct {
	ID                     string           `json:"id"`
	DisplayName            string           `json:"displayName"`
	Status                 model.UserStatus `json:"status"`
	PasswordChangeRequired bool             `json:"passwordChangeRequired"`
}

type RoleRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ContextView struct {
	Revision    string           `json:"revision"`
	CompanyID   *string          `json:"companyId"`
	BranchScope auth.BranchScope `json:"branchScope"`
}

type AccessibleCompany struct {
	ID                 string                    `json:"id"`
	Name               string                    `json:"name"`
	Kind               model.CompanyKind         `json:"kind"`
	UserType           *model.SellerUserType     `json:"userType"`
	Access             model.CompanyAccess       `json:"access"`
	Verification       model.CompanyVerification `json:"verification"`
	VerificationReason string                    `json:"verificationReason"`
}

type SetupView struct {
	Next                      string `json:"next"` // company | branch | none
	PartnershipRequiredForB2B bool   `json:"partnershipRequiredForB2B"`
}

func (s *Auth) View(ctx context.Context, p *auth.Principal) (*SessionView, error) {
	u, err := s.store.Users().Get(ctx, p.UserID)
	if err != nil {
		return nil, err
	}
	roles, err := s.rolesIn(ctx, s.store, u.ID, p.CompanyID)
	if err != nil {
		return nil, err
	}
	memberships, err := s.store.Memberships().ListByUser(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	var companyIDs []string
	for _, m := range memberships {
		if m.Status == model.MembershipActive {
			companyIDs = append(companyIDs, m.CompanyID)
		}
	}
	companies, err := s.store.Companies().GetMany(ctx, companyIDs)
	if err != nil {
		return nil, err
	}
	view := &SessionView{
		User:                SessionUser{ID: u.ID, DisplayName: u.DisplayName, Status: u.Status, PasswordChangeRequired: u.PasswordChangeRequired},
		Roles:               make([]RoleRef, len(roles)),
		Permissions:         p.PermissionList(),
		Context:             ContextView{Revision: revision(p.ContextRevision), BranchScope: p.BranchScope},
		AccessibleCompanies: make([]AccessibleCompany, len(companies)),
		Setup:               SetupView{Next: "none", PartnershipRequiredForB2B: true},
	}
	for i, r := range roles {
		view.Roles[i] = RoleRef{ID: r.ID, Name: r.Name}
	}
	for i, c := range companies {
		view.AccessibleCompanies[i] = AccessibleCompany{ID: c.ID, Name: c.Name, Kind: c.Kind, UserType: c.UserType, Access: c.Status, Verification: c.Verification, VerificationReason: c.VerificationReason}
	}
	if p.CompanyID != "" {
		id := p.CompanyID
		view.Context.CompanyID = &id
	}
	// Setup shows only the first missing prerequisite: company, then branch.
	switch {
	case len(companies) == 0:
		view.Setup.Next = "company"
	case p.CompanyID != "":
		branches, err := s.store.Branches().List(ctx, p.CompanyID)
		if err != nil {
			return nil, err
		}
		if len(branches) == 0 {
			view.Setup.Next = "branch"
		}
	}
	return view, nil
}

// SetCompany switches the working company (nil clears it) and resets the
// branch scope to ALL. The user must be an active member of the company.
func (s *Auth) SetCompany(ctx context.Context, p *auth.Principal, expected int64, companyID *string) (resultErr error) {
	var denialCode SecurityCode
	defer func() { s.recordContextDenial(ctx, p, resultErr, denialCode) }()
	if companyID != nil {
		if err := validID(*companyID); err != nil {
			return apperr.FieldError("companyId", "must be a valid ID")
		}
		member, err := s.isMember(ctx, s.store, p.UserID, *companyID)
		if err != nil {
			return err
		}
		if !member {
			denialCode = SecurityNotMember
			return apperr.New(apperr.ErrForbidden, "not_a_member", "you are not a member of this company")
		}
		company, err := s.store.Companies().Get(ctx, *companyID)
		if err != nil {
			return err
		}
		if company.Status == model.AccessSuspended {
			denialCode = SecurityCompanySuspended
			return apperr.New(apperr.ErrForbidden, "company_suspended", "this company is suspended")
		}
	}
	sess := &model.Session{ID: p.SessionID, ActiveCompanyID: companyID, BranchScopeMode: model.ScopeAll, BranchIDs: []string{}}
	return s.updateContext(ctx, p, sess, expected, SecurityCompanyChanged)
}

// SetBranchScope narrows the working branches inside the active company. ALL
// requires an empty list; SELECTED requires branches the membership allows.
func (s *Auth) SetBranchScope(ctx context.Context, p *auth.Principal, expected int64, scope auth.BranchScope) (result auth.BranchScope, resultErr error) {
	var denialCode SecurityCode
	defer func() { s.recordContextDenial(ctx, p, resultErr, denialCode) }()
	if p.CompanyID == "" {
		denialCode = SecurityNoActiveCompany
		return auth.BranchScope{}, apperr.New(apperr.ErrConflict, "no_active_company", "select a company first")
	}
	var v apperr.Validation
	ids := uniqueIDs(&v, "branchIds", scope.BranchIDs)
	switch scope.Mode {
	case model.ScopeAll:
		if len(ids) != 0 {
			v.Add("branchIds", "must be empty for ALL")
		}
	case model.ScopeSelected:
		if len(ids) == 0 {
			v.Add("branchIds", "select at least one branch")
		}
	default:
		v.Add("mode", "must be ALL or SELECTED")
	}
	if err := v.Err(); err != nil {
		return auth.BranchScope{}, err
	}
	if len(ids) > 0 {
		n, err := s.store.Branches().CountInCompany(ctx, p.CompanyID, ids)
		if err != nil {
			return auth.BranchScope{}, err
		}
		m, err := s.store.Memberships().Active(ctx, p.UserID, p.CompanyID)
		if err != nil {
			return auth.BranchScope{}, err
		}
		allowed := int(n) == len(ids)
		if m.BranchAccess == model.SelectedBranches {
			for _, id := range ids {
				allowed = allowed && slices.Contains(m.BranchIDs, id)
			}
		}
		if !allowed {
			denialCode = SecurityBranchScopeDenied
			return auth.BranchScope{}, apperr.FieldError("branchIds", "contains branches you cannot access")
		}
	}
	sess := &model.Session{ID: p.SessionID, ActiveCompanyID: &p.CompanyID, BranchScopeMode: scope.Mode, BranchIDs: ids}
	return auth.BranchScope{Mode: scope.Mode, BranchIDs: ids}, s.updateContext(ctx, p, sess, expected, SecurityBranchScopeChanged)
}

func (s *Auth) updateContext(ctx context.Context, p *auth.Principal, next *model.Session, expected int64, action SecurityAction) error {
	err := s.store.InTx(ctx, func(st Store) error {
		before, err := st.Sessions().FindByID(ctx, p.SessionID)
		if errors.Is(err, apperr.ErrNotFound) {
			return apperr.ErrStale // same missing-record response as UpdateContext
		}
		if err != nil {
			return err
		}
		// Keep the original revision predicate and public error. In particular,
		// this does not add a new revoked/expired session predicate to the CAS.
		if err := st.Sessions().UpdateContext(ctx, next, expected); err != nil {
			return err
		}
		if sameSessionContext(before, next) {
			return nil
		}
		return s.AuditSecurity(ctx, st, SecurityEvent{
			Action: action, Outcome: AuditSuccess, Actor: p, ResourceType: "session",
			Before: sessionContextValues(before), After: sessionContextValues(next),
		})
	})
	if err == nil {
		MarkSecurityEventAttempted(ctx)
	}
	return err
}

func sessionContextValues(sess *model.Session) *SecurityContextValues {
	return &SecurityContextValues{
		CompanyID:   sess.ActiveCompanyID,
		BranchScope: &auth.BranchScope{Mode: sess.BranchScopeMode, BranchIDs: slices.Clone(sess.BranchIDs)},
	}
}

func sameSessionContext(a, b *model.Session) bool {
	if (a.ActiveCompanyID == nil) != (b.ActiveCompanyID == nil) ||
		(a.ActiveCompanyID != nil && *a.ActiveCompanyID != *b.ActiveCompanyID) || a.BranchScopeMode != b.BranchScopeMode {
		return false
	}
	left, right := slices.Clone(a.BranchIDs), slices.Clone(b.BranchIDs)
	slices.Sort(left)
	slices.Sort(right)
	return slices.Equal(left, right)
}

func (s *Auth) recordContextDenial(ctx context.Context, p *auth.Principal, err error, code SecurityCode) {
	if err == nil {
		return
	}
	if errors.Is(err, apperr.ErrStale) {
		code = SecurityContextChanged
	}
	if code == "" {
		return // validation and ordinary business/storage errors are not auth events
	}
	_ = s.RecordSecurity(ctx, SecurityEvent{
		Action: SecurityRequestDenied, Outcome: AuditDenied, ErrorCode: code, Actor: p,
		ResourceType: "session",
	})
}

// ChangePassword replaces the signed-in user's password after checking the
// current one. Other sessions end; this one stays signed in.
func (s *Auth) ChangePassword(ctx context.Context, p *auth.Principal, current, next, confirmation string) error {
	u, err := s.store.Users().Get(ctx, p.UserID)
	if err != nil {
		return err
	}
	now := s.clock()
	if u.LockedUntil != nil && u.LockedUntil.After(now) {
		return &apperr.RateLimitedError{RetryAfter: u.LockedUntil.Sub(now)}
	}
	ok, err := verifyPassword(current, *u.PasswordHash)
	if err != nil {
		return err
	}
	if !ok {
		if err := s.recordFailure(ctx, u); err != nil {
			return err
		}
		return apperr.FieldError("currentPassword", "is incorrect")
	}
	var v apperr.Validation
	validatePassword(&v, "newPassword", next, confirmation)
	if next == current {
		v.Add("newPassword", "must differ from the current password")
	}
	if err := v.Err(); err != nil {
		return err
	}
	hash, err := hashPassword(next)
	if err != nil {
		return err
	}
	return s.store.InTx(ctx, func(st Store) error {
		u.PasswordHash, u.PasswordChangeRequired, u.UpdatedAt = &hash, false, now
		if err := st.Users().Update(ctx, u, u.Version); err != nil {
			return err
		}
		if err := st.Users().SetLoginState(ctx, u.ID, 0, nil); err != nil {
			return err
		}
		if err := st.Sessions().RevokeUser(ctx, u.ID, p.SessionID, now); err != nil {
			return err
		}
		return s.audit(ctx, st, p, "user.password_changed", "user", u.ID, nil, "", nil)
	})
}
