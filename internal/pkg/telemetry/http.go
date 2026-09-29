package telemetry

import (
	"log/slog"
	"time"

	echootel "github.com/labstack/echo-otel/v5"
	"github.com/labstack/echo/v5"
)

// HTTP traces every request (a span per route; echootel also records
// http.server.request.duration) and logs it. Health probes are skipped.
func HTTP(log *slog.Logger) []echo.MiddlewareFunc {
	skip := func(c *echo.Context) bool { p := c.Request().URL.Path; return p == "/healthz" || p == "/readyz" }
	logged := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if skip(c) {
				return next(c)
			}
			start := time.Now()
			err := next(c)
			if err != nil {
				c.Echo().HTTPErrorHandler(c, err) // write the error response now so the status below is final
			}
			req := c.Request()
			res, status := echo.ResolveResponseStatus(c.Response(), nil)
			var size int64
			if res != nil {
				size = res.Size
			}
			route := c.Path()
			elapsed := time.Since(start)
			level := slog.LevelInfo
			if status >= 500 {
				level = slog.LevelError
			}
			log.Log(req.Context(), level, "request", "method", req.Method, "route", route, "path", req.URL.Path, "status", status,
				"duration_ms", elapsed.Milliseconds(), "request_id", c.Response().Header().Get(echo.HeaderXRequestID), "bytes", size)
			return nil
		}
	}
	// logged writes error responses itself, so the span sees the final status.
	return []echo.MiddlewareFunc{echootel.NewMiddlewareWithConfig(echootel.Config{Skipper: skip}), logged}
}
