package handler

import (
	"net/http"

	"github.com/labstack/echo/v5"

	"justixauto/internal/modules/identity/service"
)

func (c CookieConfig) deviceName() string {
	if c.Secure {
		return "__Host-justix_device"
	}
	return "justix_device"
}

func (c CookieConfig) deviceToken(ctx *echo.Context) string {
	return cookieValue(ctx, c.deviceName())
}

// DeviceHash returns only a validated, domain-separated correlation digest.
// Cookie possession is client-controlled evidence and never authenticates.
func (c CookieConfig) DeviceHash(ctx *echo.Context) string {
	return service.DeviceCorrelationHash(c.deviceToken(ctx))
}

func (c CookieConfig) setDevice(ctx *echo.Context, token string) {
	ctx.SetCookie(&http.Cookie{ //nolint:gosec // Secure follows the approved local HTTP/config distinction.
		Name: c.deviceName(), Value: token, Path: "/",
		Secure: c.Secure, HttpOnly: true, SameSite: http.SameSiteLaxMode,
		// No MaxAge/Expires: recognition lasts only while the browser retains it.
	})
}
