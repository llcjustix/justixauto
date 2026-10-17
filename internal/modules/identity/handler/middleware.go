// Package handler holds identity's Echo routes, request/response DTOs,
// mappers and the Echo authentication middleware. It imports service, model
// and internal/pkg only; it must never import repository or gorm.
package handler

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/identity/service"
	"justixauto/internal/pkg/apperr"
	"justixauto/internal/pkg/auth"
	"justixauto/internal/pkg/requestmeta"
)

// CookieConfig controls the session cookie. Secure must be true outside local
// HTTP development; the __Host- prefix is only valid for Secure cookies.
type CookieConfig struct {
	Secure         bool
	AllowedOrigins []string // extra origins allowed for state-changing requests
}

func (c CookieConfig) name() string {
	if c.Secure {
		return "__Host-justix_session"
	}
	return "justix_session"
}

func (c CookieConfig) set(ctx *echo.Context, token string, maxAge int) {
	ctx.SetCookie(&http.Cookie{ //nolint:gosec // Secure comes from CookieConfig.Secure (config-driven, true outside local dev); HttpOnly/SameSite are set below
		Name: c.name(), Value: token, Path: "/", MaxAge: maxAge,
		Secure: c.Secure, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

func (c CookieConfig) token(ctx *echo.Context) string { return cookieValue(ctx, c.name()) }

func cookieValue(ctx *echo.Context, name string) string {
	cookie, err := ctx.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

const sessionKey = "identity.session"

// Authenticator loads the session from the cookie for every request. Anonymous
// requests pass through without a principal; routes decide with auth.Require.
// State-changing requests must come from an allowed Origin and, when signed
// in, carry the session's X-CSRF-Token.
type Authenticator struct {
	auth   *service.Auth
	cookie CookieConfig
}

// NewAuthenticator builds the Echo authentication middleware.
func NewAuthenticator(a *service.Auth, cookie CookieConfig) *Authenticator {
	return &Authenticator{auth: a, cookie: cookie}
}

func unsafeMethod(m string) bool {
	return m != http.MethodGet && m != http.MethodHead && m != http.MethodOptions
}

func (a *Authenticator) originAllowed(c *echo.Context) bool {
	origin := c.Request().Header.Get("Origin")
	if origin == "" {
		return true // non-browser clients; the CSRF token still applies
	}
	if slices.Contains(a.cookie.AllowedOrigins, origin) {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host == c.Request().Host
}

// Middleware authenticates requests. csrfExempt lists route suffixes that run
// before a session exists (login).
func (a *Authenticator) Middleware(csrfExempt ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) (resultErr error) {
			// Observe the entire identity/route pipeline, including guards in
			// other modules. This state must exist before any explicit event.
			r := c.Request()
			c.SetRequest(r.WithContext(service.WithSecurityAuditState(r.Context())))
			var auditActor *auth.Principal
			defer func() {
				// next and its domain transaction have returned before the root
				// audit writer runs. Its failure never replaces the public result.
				a.auth.RecordRequestDenial(c.Request().Context(), auditActor, requestDenialCode(c, resultErr))
			}()
			unsafe := unsafeMethod(c.Request().Method)
			if unsafe && !a.originAllowed(c) {
				return apperr.New(apperr.ErrForbidden, "origin_denied", "request origin is not allowed")
			}
			deviceHash := a.cookie.DeviceHash(c)
			p, sess, err := a.auth.AuthenticateWithDevice(c.Request().Context(), a.cookie.token(c), deviceHash)
			switch {
			case errors.Is(err, apperr.ErrUnauthenticated):
				return next(c)
			case err != nil:
				return err
			}
			// Capture trusted audit facts before the remaining rejection points,
			// while keeping auth.Get unavailable until every guard has passed.
			auditActor = p
			facts := p.AuditFacts()
			facts.DeviceHash = deviceHash
			r = c.Request()
			c.SetRequest(r.WithContext(requestmeta.WithAuth(r.Context(), facts)))
			exempt := slices.ContainsFunc(csrfExempt, func(suffix string) bool { return strings.HasSuffix(c.Path(), suffix) })
			if unsafe && !exempt {
				sent := c.Request().Header.Get("X-CSRF-Token")
				if subtle.ConstantTimeCompare([]byte(sent), []byte(sess.CSRFToken)) != 1 {
					return apperr.New(apperr.ErrForbidden, "csrf_invalid", "missing or invalid X-CSRF-Token")
				}
			}
			if p.PasswordChangeRequired && !passwordChangeAllowed(c) {
				return apperr.New(apperr.ErrForbidden, "password_change_required", "choose a new password first")
			}
			auth.Set(c, p)
			c.Set(sessionKey, sess)
			return next(c)
		}
	}
}

// requestDenialCode uses only fixed classifications, never an error's message,
// a raw URL or a response body. Explicit context denials also have non-401/403
// public statuses; ordinary conflicts/validation/storage failures stay out.
func requestDenialCode(c *echo.Context, err error) service.SecurityCode {
	var fallback service.SecurityCode
	if res, unwrapErr := echo.UnwrapResponse(c.Response()); unwrapErr == nil && res.Committed {
		switch res.Status {
		case http.StatusUnauthorized:
			fallback = service.SecurityUnauthenticated
		case http.StatusForbidden:
			fallback = service.SecurityPermissionDenied
		}
	}
	if err == nil {
		return fallback
	}
	// Match the public error mapper's precedence without inspecting payloads.
	var validation *apperr.ValidationError
	var limited *apperr.RateLimitedError
	if errors.As(err, &validation) || errors.As(err, &limited) {
		return fallback
	}
	if status := echo.StatusCode(err); status != 0 {
		switch status {
		case http.StatusUnauthorized:
			return service.SecurityUnauthenticated
		case http.StatusForbidden:
			return service.SecurityPermissionDenied
		default:
			return fallback
		}
	}
	if errors.Is(err, apperr.ErrUnauthenticated) {
		fallback = service.SecurityUnauthenticated
	} else if errors.Is(err, apperr.ErrForbidden) {
		fallback = service.SecurityPermissionDenied
	}
	var coded *apperr.Error
	if !errors.As(err, &coded) {
		return fallback
	}
	if fallback != "" {
		switch coded.Code {
		case "origin_denied":
			return service.SecurityOriginDenied
		case "csrf_invalid":
			return service.SecurityCSRFInvalid
		case "password_change_required":
			return service.SecurityPasswordChangeRequired
		case "invalid_credentials":
			return service.SecurityInvalidCredentials
		case "account_locked":
			return service.SecurityAccountLocked
		case "account_suspended":
			return service.SecurityAccountSuspended
		case "not_a_member":
			return service.SecurityNotMember
		case "company_suspended":
			return service.SecurityCompanySuspended
		case "branch_scope_denied":
			return service.SecurityBranchScopeDenied
		}
	}
	if errors.Is(err, apperr.ErrConflict) {
		switch coded.Code {
		case "no_active_company":
			return service.SecurityNoActiveCompany
		case "context_changed":
			return service.SecurityContextChanged
		}
	}
	return fallback
}

// passwordChangeAllowed: with an administrator-set password, the session may
// only read itself, change the password or sign out.
func passwordChangeAllowed(c *echo.Context) bool {
	path, method := c.Path(), c.Request().Method
	return (method == http.MethodGet && strings.HasSuffix(path, "/identity/session")) ||
		strings.HasSuffix(path, "/identity/session/password") || strings.HasSuffix(path, "/identity/session/logout")
}
