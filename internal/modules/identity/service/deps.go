// Package service holds identity's business rules. It imports model and
// internal/pkg only; it must never import echo, gorm, repository or handler.
package service

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"justixauto/internal/modules/identity/model"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/requestmeta"
	"justixauto/internal/pkg/validate"
)

// Deps is shared by all identity services.
type Deps struct {
	store              Store
	now                func() time.Time
	securityWriter     IndependentAuditWriter
	reportAuditFailure AuditPersistReporter
}

// NewDeps builds the shared dependencies every identity service embeds.
func NewDeps(store Store, now func() time.Time) Deps { return Deps{store: store, now: now} }

// WithSecurityAudit installs the independent root writer and a reporter which
// receives only fixed classifications, an action and the generated request ID.
func (d Deps) WithSecurityAudit(writer IndependentAuditWriter, report AuditPersistReporter) Deps {
	d.securityWriter, d.reportAuditFailure = writer, report
	return d
}

func (d Deps) clock() time.Time { return d.now().UTC() }

// audit appends an audit event inside the caller's transaction.
func (d Deps) audit(ctx context.Context, st Store, actor *auth.Principal, action, resourceType, resourceID string, companyID *string, reason string, details map[string]any) error {
	return d.auditChange(ctx, st, actor, action, resourceType, resourceID, companyID, reason, details, nil, nil)
}

// auditChange accepts only explicit projections of allowlisted domain fields.
// It compares serialized values and normalizes string sets so ordering and
// pointer identity never masquerade as changes. Callers snapshot before mutation
// and append after successful domain persistence in the same transaction.
func (d Deps) auditChange(ctx context.Context, st Store, actor *auth.Principal, action, resourceType, resourceID string, companyID *string, reason string, details, before, after map[string]any) error {
	beforeRaw, afterRaw, fields, err := auditFieldDiff(resourceType, before, after)
	if err != nil {
		return err
	}
	if before != nil || after != nil {
		copy := make(map[string]any, len(details)+1)
		for key, value := range details {
			copy[key] = value
		}
		copy["changedFields"] = fields
		details = copy
	}
	raw, err := json.Marshal(details)
	if err != nil {
		return errInvalidAuditEvent
	}
	if !safeAuditJSON(raw) || (actor != nil && actor.SessionID != "" &&
		(bytes.Contains(raw, []byte(actor.SessionID)) || strings.Contains(reason, actor.SessionID))) {
		return errInvalidAuditEvent
	}
	e, err := d.auditEvent(ctx, actor, action, resourceType, optionalAuditString(resourceID), companyID)
	if err != nil {
		return err
	}
	for _, diff := range [][]byte{beforeRaw, afterRaw} {
		if actor != nil && actor.SessionID != "" && bytes.Contains(diff, []byte(actor.SessionID)) {
			return errInvalidAuditEvent
		}
	}
	e.Reason, e.Details, e.BeforeValues, e.AfterValues = reason, raw, beforeRaw, afterRaw
	return st.Audit().Append(ctx, e)
}

func auditFieldDiff(resource string, before, after map[string]any) ([]byte, []byte, []string, error) {
	var allowed []string
	switch resource {
	case "company":
		allowed = []string{"name", "legalName", "country", "countryKey", "region", "regionKey", "registration", "legalForm", "taxId", "bankRequisites", "email", "address", "phone", "kind", "status", "statusReason", "userType", "verification", "verificationReason", "verificationActorId", "deleted"}
	case "branch":
		allowed = []string{"name", "address"}
	case "user":
		allowed = []string{"login", "displayName", "email", "status", "statusReason", "roleIds"}
	case "membership":
		allowed = []string{"status", "statusReason", "branchAccess", "branchIds", "roleIds"}
	case "role":
		allowed = []string{"name", "permissionKeys", "deleted"}
	case "permission":
		allowed = []string{"name", "assignable", "deleted"}
	}
	left, right := map[string]json.RawMessage{}, map[string]json.RawMessage{}
	keys := make([]string, 0, len(before)+len(after))
	for _, values := range []map[string]any{before, after} {
		for key := range values {
			if !slices.Contains(allowed, key) {
				return nil, nil, nil, errInvalidAuditEvent
			}
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	keys = slices.Compact(keys)
	fields := []string{}
	for _, key := range keys {
		encode := func(values map[string]any) ([]byte, error) {
			value := values[key]
			if key == "roleIds" || key == "branchIds" || key == "permissionKeys" {
				if set, ok := value.([]string); ok {
					value = auditSet(set)
				} else if value != nil {
					return nil, errInvalidAuditEvent
				}
			}
			// Named string enums are safe; composite values (including
			// models) are rejected by the encoded scalar check below.
			raw, err := json.Marshal(value)
			if err != nil || !safeAuditJSON(raw) || (len(raw) > 0 && (raw[0] == '{' || (raw[0] == '[' && key != "roleIds" && key != "branchIds" && key != "permissionKeys"))) {
				return nil, errInvalidAuditEvent
			}
			return raw, nil
		}
		b, err := encode(before)
		if err != nil {
			return nil, nil, nil, err
		}
		a, err := encode(after)
		if err != nil {
			return nil, nil, nil, err
		}
		if bytes.Equal(b, a) {
			continue
		}
		fields = append(fields, key)
		left[key], right[key] = b, a
	}
	if len(fields) == 0 {
		return nil, nil, fields, nil
	}
	b, err := json.Marshal(left)
	if err != nil {
		return nil, nil, nil, errInvalidAuditEvent
	}
	a, err := json.Marshal(right)
	if err != nil {
		return nil, nil, nil, errInvalidAuditEvent
	}
	return b, a, fields, nil
}

func auditSet(values []string) []string {
	copy := append([]string{}, values...)
	slices.Sort(copy)
	return slices.Compact(copy)
}

func auditSetChanges(before, after []string) (removed, added []string) {
	removed, added = []string{}, []string{}
	for _, key := range auditSet(before) {
		if !slices.Contains(after, key) {
			removed = append(removed, key)
		}
	}
	for _, key := range auditSet(after) {
		if !slices.Contains(before, key) {
			added = append(added, key)
		}
	}
	return removed, added
}

func auditUserValues(u *model.User) map[string]any {
	var login any
	if u.Login != nil {
		login = *u.Login
	}
	return map[string]any{"login": login, "displayName": u.DisplayName, "email": u.Email, "status": u.Status, "statusReason": u.StatusReason}
}

func auditMembershipValues(m *model.Membership) map[string]any {
	return map[string]any{"status": m.Status, "statusReason": m.StatusReason, "branchAccess": m.BranchAccess, "branchIds": auditSet(m.BranchIDs)}
}

func auditRoleIDs(roles []model.Role) []string {
	ids := make([]string, len(roles))
	for i, r := range roles {
		ids[i] = r.ID
	}
	return auditSet(ids)
}

var (
	errInvalidAuditEvent = errors.New("invalid audit event")
	// ErrAuditPersistence is deliberately detached from any underlying SQL or
	// driver error. Independent failure recording must preserve the public denial.
	ErrAuditPersistence = errors.New("audit persistence failed")
)

type SecurityAction string

const (
	SecurityLoginSucceeded     SecurityAction = "auth.login_succeeded"
	SecurityLoginFailed        SecurityAction = "auth.login_failed"
	SecurityLogout             SecurityAction = "auth.logout"
	SecuritySessionExpired     SecurityAction = "auth.session_expired"
	SecuritySessionRevoked     SecurityAction = "auth.session_revoked"
	SecurityCompanyChanged     SecurityAction = "context.company_changed"
	SecurityBranchScopeChanged SecurityAction = "context.branch_scope_changed"
	SecurityRequestDenied      SecurityAction = "security.request_denied"
)

type AuditOutcome string

const (
	AuditSuccess AuditOutcome = "success"
	AuditDenied  AuditOutcome = "denied"
	AuditError   AuditOutcome = "error"
)

type LoginFailureReason string

const (
	LoginUnknown       LoginFailureReason = "unknown_login"
	LoginNoCredentials LoginFailureReason = "no_credentials"
	LoginBadPassword   LoginFailureReason = "bad_password"
	LoginLocked        LoginFailureReason = "locked"
	LoginSuspended     LoginFailureReason = "suspended"
	LoginInternalError LoginFailureReason = "internal_error"
)

// SecurityCode is a fixed internal/public code, never an error message.
type SecurityCode string

const (
	SecurityUnauthenticated        SecurityCode = "unauthenticated"
	SecurityPermissionDenied       SecurityCode = "permission_denied"
	SecurityOriginDenied           SecurityCode = "origin_denied"
	SecurityCSRFInvalid            SecurityCode = "csrf_invalid"
	SecurityPasswordChangeRequired SecurityCode = "password_change_required"
	SecurityInvalidCredentials     SecurityCode = "invalid_credentials"
	SecurityAccountLocked          SecurityCode = "account_locked"
	SecurityAccountSuspended       SecurityCode = "account_suspended"
	SecurityNotMember              SecurityCode = "not_a_member"
	SecurityCompanySuspended       SecurityCode = "company_suspended"
	SecurityNoActiveCompany        SecurityCode = "no_active_company"
	SecurityContextChanged         SecurityCode = "context_changed"
	SecurityBranchScopeDenied      SecurityCode = "branch_scope_denied"
	SecurityInternalError          SecurityCode = "internal_error"
)

// SecurityDetails is deliberately closed: no arbitrary map, raw session ID,
// cookie, token, password, request body or error object can be supplied here.
// Method and route are filled only from the captured request, never raw URL.
type SecurityDetails struct {
	FailureReason           LoginFailureReason `json:"failureReason,omitempty"`
	AttemptedLogin          string             `json:"attemptedLogin,omitempty"`
	AttemptedLoginSource    string             `json:"attemptedLoginSource,omitempty"`
	AttemptedLoginTruncated bool               `json:"attemptedLoginTruncated,omitempty"`
	TargetLogin             string             `json:"targetLogin,omitempty"`
	KnownDevice             *bool              `json:"knownDevice,omitempty"`
	SessionEndReason        string             `json:"sessionEndReason,omitempty"`
	Method                  string             `json:"method,omitempty"`
	Route                   string             `json:"route,omitempty"`
}

// SecurityContextValues is the allowlisted before/after snapshot for session
// company and branch changes. These contain no session identifier.
type SecurityContextValues struct {
	CompanyID   *string           `json:"companyId"`
	BranchScope *auth.BranchScope `json:"branchScope,omitempty"`
}

type SecurityEvent struct {
	Action    SecurityAction
	Outcome   AuditOutcome
	ErrorCode SecurityCode
	// Actor is a resolved, authenticated caller. A known failed-login target
	// belongs in ResourceID/TargetLogin; it must never be passed as Actor.
	Actor        *auth.Principal
	ResourceType string // empty, user, company, or session (resource ID always nil)
	ResourceID   *string
	CompanyID    *string
	Details      SecurityDetails
	Before       *SecurityContextValues
	After        *SecurityContextValues
	// Only precomputed domain-separated hashes are accepted. An override is
	// useful when a lifecycle event concerns a different session from the actor.
	SessionHash      string
	SessionStartedAt *time.Time
	DeviceHash       string
}

type AuditPersistFailure struct {
	RequestID      string
	Action         string
	Classification string
}

type AuditPersistReporter func(AuditPersistFailure)

type securityAuditKey struct{}
type securityAuditState struct{ attempted atomic.Bool }

// WithSecurityAuditState installs one mutable deduplication flag per request.
// It must wrap the entire identity middleware pipeline, before handlers run.
func WithSecurityAuditState(ctx context.Context) context.Context {
	if _, ok := ctx.Value(securityAuditKey{}).(*securityAuditState); ok {
		return ctx
	}
	return context.WithValue(ctx, securityAuditKey{}, &securityAuditState{})
}

func SecurityEventAttempted(ctx context.Context) bool {
	state, ok := ctx.Value(securityAuditKey{}).(*securityAuditState)
	return ok && state.attempted.Load()
}

// MarkSecurityEventAttempted marks the original request even if persistence
// fails, so a generic denial cannot obscure/duplicate its explicit event.
func MarkSecurityEventAttempted(ctx context.Context) {
	if state, ok := ctx.Value(securityAuditKey{}).(*securityAuditState); ok {
		state.attempted.Store(true)
	}
}

// AuditSecurity appends a successful explicit event in the caller's domain
// transaction. Any error must roll back the successful domain mutation.
func (d Deps) AuditSecurity(ctx context.Context, st Store, input SecurityEvent) error {
	event, err := d.securityEvent(ctx, input)
	if err != nil {
		return err
	}
	return st.Audit().Append(ctx, event)
}

// RecordSecurity independently records a failed/denied request after the domain
// transaction returns. No goroutine or ambient database context is retained.
func (d Deps) RecordSecurity(ctx context.Context, input SecurityEvent) error {
	MarkSecurityEventAttempted(ctx)
	event, err := d.securityEvent(ctx, input)
	classification := "invalid_event"
	if err == nil && d.securityWriter == nil {
		err, classification = ErrAuditPersistence, "writer_unavailable"
	}
	if err == nil {
		// WithoutCancel(ctx) alone would retain database.WithTx's private value.
		// Start from a fresh context and copy ONLY the defensive request snapshot.
		// This is cancellation-independent and contains no ambient transaction,
		// request body, cookie, principal pointer or unrelated context values.
		independent := context.WithoutCancel(context.Background())
		if len(event.Metadata) != 0 {
			var snapshot requestmeta.Snapshot
			if json.Unmarshal(event.Metadata, &snapshot) == nil {
				independent = requestmeta.WithSnapshot(independent, snapshot)
			}
		}
		bounded, cancel := context.WithTimeout(independent, 2*time.Second)
		err = d.securityWriter.AppendIndependent(bounded, event)
		cancel()
		classification = "storage_error"
		if errors.Is(err, context.DeadlineExceeded) {
			classification = "timeout"
		} else if errors.Is(err, context.Canceled) {
			classification = "canceled"
		}
	}
	if err == nil {
		return nil
	}
	if d.reportAuditFailure != nil {
		action := "security.invalid_event"
		if validSecurityAction(input.Action) {
			action = string(input.Action)
		}
		failure := AuditPersistFailure{Action: action, Classification: classification}
		if snapshot, ok := requestmeta.FromContext(ctx); ok && uuid.Validate(snapshot.RequestID) == nil {
			failure.RequestID = snapshot.RequestID
		}
		d.reportAuditFailure(failure)
	}
	return ErrAuditPersistence
}

func (d Deps) securityEvent(ctx context.Context, input SecurityEvent) (*model.AuditEvent, error) {
	if !validSecurityAction(input.Action) {
		return nil, errInvalidAuditEvent
	}
	if input.Outcome != AuditSuccess && input.Outcome != AuditDenied && input.Outcome != AuditError {
		return nil, errInvalidAuditEvent
	}
	switch input.ErrorCode {
	case "", SecurityUnauthenticated, SecurityPermissionDenied, SecurityOriginDenied, SecurityCSRFInvalid,
		SecurityPasswordChangeRequired, SecurityInvalidCredentials, SecurityAccountLocked,
		SecurityAccountSuspended, SecurityNotMember, SecurityCompanySuspended, SecurityNoActiveCompany,
		SecurityContextChanged, SecurityBranchScopeDenied, SecurityInternalError:
	default:
		return nil, errInvalidAuditEvent
	}
	switch input.Details.FailureReason {
	case "", LoginUnknown, LoginNoCredentials, LoginBadPassword, LoginLocked, LoginSuspended, LoginInternalError:
	default:
		return nil, errInvalidAuditEvent
	}
	switch input.Details.SessionEndReason {
	case "", "expired", "idle_timeout", "logout", "browser_login_rotated", "administrator_revoked", "password_changed", "inactive_user":
	default:
		return nil, errInvalidAuditEvent
	}
	switch input.ResourceType {
	case "", "user", "company":
	case "session":
		if input.ResourceID != nil {
			return nil, errInvalidAuditEvent
		}
	default:
		return nil, errInvalidAuditEvent
	}
	actor := input.Actor
	if input.Action == SecurityLoginFailed {
		// Even an authenticated browser retry or a known target account cannot
		// be mistaken for the submitter of failed credentials.
		actor = nil
		input.SessionHash, input.SessionStartedAt = "", nil
		if input.Outcome == AuditSuccess || input.Details.FailureReason == "" {
			return nil, errInvalidAuditEvent
		}
	}
	event, err := d.auditEvent(ctx, actor, string(input.Action), input.ResourceType, input.ResourceID, input.CompanyID)
	if err != nil {
		return nil, err
	}
	event.Outcome = optionalAuditString(string(input.Outcome))
	event.ErrorCode = optionalAuditString(string(input.ErrorCode))
	details := input.Details
	details.AttemptedLogin = safeAuditText(details.AttemptedLogin, 256)
	details.AttemptedLoginSource, details.AttemptedLoginTruncated = "", false
	if input.Details.AttemptedLogin != "" {
		details.AttemptedLoginSource = "client_supplied"
		details.AttemptedLoginTruncated = len(input.Details.AttemptedLogin) > 256
	}
	details.TargetLogin = safeAuditText(details.TargetLogin, 256)
	details.Method, details.Route = "", ""
	if snapshot, ok := requestmeta.FromContext(ctx); ok {
		details.Method, details.Route = snapshot.Method, snapshot.Route
	}
	event.Details, err = json.Marshal(details)
	if err != nil {
		return nil, errInvalidAuditEvent
	}
	for _, correlation := range []string{input.SessionHash, input.DeviceHash} {
		if correlation != "" && !isAuditHash(correlation) {
			return nil, errInvalidAuditEvent
		}
	}
	if input.SessionHash != "" {
		event.SessionHash = optionalAuditString(input.SessionHash)
	}
	if input.DeviceHash != "" {
		event.DeviceHash = optionalAuditString(input.DeviceHash)
	}
	if event.Metadata != nil {
		var snapshot requestmeta.Snapshot
		if json.Unmarshal(event.Metadata, &snapshot) != nil {
			return nil, errInvalidAuditEvent
		}
		if snapshot.Auth == nil && input.SessionHash != "" &&
			(input.Action == SecuritySessionExpired || input.Action == SecuritySessionRevoked || input.Action == SecurityLogout) {
			// A resolved session target is not an authenticated caller. Preserve
			// explicit safe lifecycle correlations without inventing actor or
			// authentication-method claims. Failed logins never enter this path.
			snapshot.Auth = &requestmeta.AuthFacts{}
		}
		if snapshot.Auth != nil {
			if input.SessionHash != "" {
				snapshot.Auth.SessionHash = input.SessionHash
			}
			if input.SessionStartedAt != nil {
				started := input.SessionStartedAt.UTC()
				snapshot.Auth.SessionStartedAt = &started
			}
			if input.DeviceHash != "" {
				snapshot.Auth.DeviceHash = input.DeviceHash
			}
			snapshot.Auth.KnownDevice = details.KnownDevice
		}
		// Reapply the total metadata cap after explicit auth enrichment.
		snapshot, _ = requestmeta.FromContext(requestmeta.WithSnapshot(context.Background(), snapshot))
		event.Metadata, err = json.Marshal(snapshot)
		if err != nil {
			return nil, errInvalidAuditEvent
		}
	}
	if event.BeforeValues, err = marshalSecurityContext(input.Before); err != nil {
		return nil, err
	}
	if event.AfterValues, err = marshalSecurityContext(input.After); err != nil {
		return nil, err
	}
	return event, nil
}

func validSecurityAction(action SecurityAction) bool {
	switch action {
	case SecurityLoginSucceeded, SecurityLoginFailed, SecurityLogout, SecuritySessionExpired,
		SecuritySessionRevoked, SecurityCompanyChanged, SecurityBranchScopeChanged, SecurityRequestDenied:
		return true
	default:
		return false
	}
}

func (d Deps) auditEvent(ctx context.Context, actor *auth.Principal, action, resourceType string, resourceID, companyID *string) (*model.AuditEvent, error) {
	for _, id := range []*string{resourceID, companyID} {
		if id != nil && uuid.Validate(*id) != nil {
			return nil, errInvalidAuditEvent
		}
	}
	// A session is correlated only through its domain-separated digest.
	if resourceType == "session" {
		resourceID = nil
	}
	event := &model.AuditEvent{
		ID: uuid.NewString(), OccurredAt: d.clock(), Action: action,
		ResourceType: resourceType, ResourceID: cloneAuditString(resourceID), CompanyID: cloneAuditString(companyID),
		Outcome: optionalAuditString(string(AuditSuccess)),
	}
	facts := requestmeta.AuthFacts{}
	if actor != nil {
		if uuid.Validate(actor.UserID) != nil {
			return nil, errInvalidAuditEvent
		}
		facts = actor.AuditFacts()
		facts.Login, facts.DisplayName = safeAuditText(facts.Login, 256), safeAuditText(facts.DisplayName, 256)
		if facts.AuthenticationMethod != "password" {
			facts.AuthenticationMethod = ""
		}
		event.ActorUserID = optionalAuditString(facts.ActorID)
		event.ActorLogin, event.ActorDisplayName = optionalAuditString(facts.Login), optionalAuditString(facts.DisplayName)
		event.SessionHash = optionalAuditString(facts.SessionHash)
	}
	if snapshot, ok := requestmeta.FromContext(ctx); ok {
		if uuid.Validate(snapshot.RequestID) == nil {
			event.RequestID = optionalAuditString(snapshot.RequestID)
		}
		event.ClientIP = cloneAuditString(snapshot.Network.ClientIP)
		if actor != nil {
			if previous := snapshot.Auth; previous != nil && previous.ActorID == facts.ActorID && previous.SessionHash == facts.SessionHash {
				facts.DeviceHash, facts.KnownDevice = previous.DeviceHash, previous.KnownDevice
			}
			snapshot.Auth = &facts
			event.DeviceHash = optionalAuditString(facts.DeviceHash)
		} else {
			// Never infer an actor from a failed-login target or prior session.
			snapshot.Auth = nil
		}
		snapshot, _ = requestmeta.FromContext(requestmeta.WithSnapshot(context.Background(), snapshot))
		var err error
		event.Metadata, err = json.Marshal(snapshot)
		if err != nil {
			return nil, errInvalidAuditEvent
		}
	}
	return event, nil
}

func optionalAuditString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func cloneAuditString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func isAuditHash(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func safeAuditText(value string, limit int) string {
	var out strings.Builder
	for _, r := range value {
		if r == utf8.RuneError || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			continue
		}
		if out.Len()+utf8.RuneLen(r) > limit {
			break
		}
		out.WriteRune(r)
	}
	return out.String()
}

// Existing domain callers remain compatible, but accidental whole-model or
// secret-bearing detail maps fail closed instead of persisting credentials.
func safeAuditJSON(raw []byte) bool {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return false
	}
	var safe func(any) bool
	safe = func(value any) bool {
		switch value := value.(type) {
		case map[string]any:
			for key, child := range value {
				key = strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(key))
				for _, forbidden := range []string{"password", "credential", "token", "cookie", "csrf", "sessionid", "authorization", "requestbody", "responsebody", "rawerror"} {
					if strings.Contains(key, forbidden) {
						return false
					}
				}
				if !safe(child) {
					return false
				}
			}
		case []any:
			for _, child := range value {
				if !safe(child) {
					return false
				}
			}
		}
		return true
	}
	return safe(value)
}

func marshalSecurityContext(value *SecurityContextValues) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	if value.CompanyID != nil && uuid.Validate(*value.CompanyID) != nil {
		return nil, errInvalidAuditEvent
	}
	copy := *value
	if value.BranchScope != nil {
		scope := *value.BranchScope
		if scope.Mode != model.ScopeAll && scope.Mode != model.ScopeSelected {
			return nil, errInvalidAuditEvent
		}
		scope.BranchIDs = slices.Clone(scope.BranchIDs)
		for _, id := range scope.BranchIDs {
			if uuid.Validate(id) != nil {
				return nil, errInvalidAuditEvent
			}
		}
		slices.Sort(scope.BranchIDs)
		scope.BranchIDs = slices.Compact(scope.BranchIDs)
		copy.BranchScope = &scope
	}
	raw, err := json.Marshal(copy)
	if err != nil {
		return nil, errInvalidAuditEvent
	}
	return raw, nil
}

// isMember reports whether the user has an active membership in the company.
func (d Deps) isMember(ctx context.Context, st Store, userID, companyID string) (bool, error) {
	_, err := st.Memberships().Active(ctx, userID, companyID)
	if errors.Is(err, apperr.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

// Shared input checks (see internal/pkg/validate).
var (
	validID   = validate.IDs
	text      = validate.Text
	email     = validate.Email
	reason    = validate.Reason
	uniqueIDs = validate.UniqueIDs
)

// revision formats an optimistic-locking version for an ETag/If-Match value.
func revision(v int64) string { return strconv.FormatInt(v, 10) }

// withPermissions resolves each role's grants against the live catalog in
// PostgreSQL: the company administrator holds every company permission, the
// platform administrator its fixed set, and custom roles their stored keys
// minus soft-deleted permissions (a deleted permission grants nothing).
func (d Deps) withPermissions(ctx context.Context, st Store, roles []model.Role) error {
	if len(roles) == 0 {
		return nil
	}
	catalog, err := st.Permissions().List(ctx)
	if err != nil {
		return err
	}
	resolveGrants(roles, catalog)
	return nil
}

// resolveGrants sets each role's grants from the live catalog (see
// withPermissions).
func resolveGrants(roles []model.Role, catalog []model.Permission) {
	live := make(map[string]bool, len(catalog))
	for _, p := range catalog {
		live[p.Key] = true
	}
	company := model.CompanyKeys(catalog)
	for i := range roles {
		r := &roles[i]
		switch {
		case r.SystemKey == nil:
			r.Permissions = slices.DeleteFunc(slices.Clone(r.Permissions), func(k string) bool { return !live[k] })
		case *r.SystemKey == model.RoleCompanyAdmin:
			r.Permissions = company
		default:
			r.Permissions = model.EffectivePermissions(*r)
		}
	}
}

// rolesIn returns the roles a user acts with in companyID: platform roles
// (held by the user) plus the company roles of the user's active membership
// in that company (user decision 2026-09-27: roles are per company). With no
// company, or no active membership, only platform roles count.
func (d Deps) rolesIn(ctx context.Context, st Store, userID, companyID string) ([]model.Role, error) {
	roles, err := st.Roles().UserRoles(ctx, userID)
	if err != nil {
		return nil, err
	}
	if companyID == "" {
		return roles, nil
	}
	m, err := st.Memberships().Active(ctx, userID, companyID)
	if errors.Is(err, apperr.ErrNotFound) {
		return roles, nil
	}
	if err != nil {
		return nil, err
	}
	company, err := st.Roles().MembershipRoles(ctx, m.ID)
	if err != nil {
		return nil, err
	}
	return append(roles, company...), nil
}
