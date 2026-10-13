// Package webui serves the built portal next to the API.
package webui

import (
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v5"
)

// Mount serves one built portal directory. An absent index.html leaves the API
// unchanged. Entry fallback is only for explicit document navigation requests.
func Mount(e *echo.Echo, distDir string) {
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		return
	}
	dist := os.DirFS(distDir)
	h := func(c *echo.Context) error {
		raw, clean, ok := requestPath(c.Request())
		if !ok {
			return echo.ErrNotFound
		}
		if excluded(clean) {
			return echo.ErrNotFound
		}
		setSecurityHeaders(c)
		if clean != "/" && clean != "/index.html" {
			name := strings.TrimPrefix(clean, "/")
			if info, err := fs.Stat(dist, name); err == nil && !info.IsDir() {
				setCache(c, clean)
				return c.FileFS(name, dist)
			}
			if path.Ext(clean) != "" {
				return echo.ErrNotFound
			}
		}
		if clean == "/assets" || strings.HasPrefix(clean, "/assets/") {
			return echo.ErrNotFound
		}
		if clean != "/index.html" && strings.ContainsAny(clean, ".@%") { return echo.ErrNotFound }
		if !documentRequest(c.Request()) || (clean == "/index.html" && raw != "/index.html") {
			return echo.ErrNotFound
		}
		c.Response().Header().Set("Cache-Control", "no-cache")
		return c.FileFS("index.html", dist)
	}
	e.GET("/*", h)
	e.HEAD("/*", h)
}

func requestPath(r *http.Request) (string, string, bool) {
	raw := r.URL.EscapedPath()
	if raw == "" {
		raw = "/"
	}
	decoded, err := url.PathUnescape(raw)
	if err != nil || !safePath(raw) || !safePath(decoded) {
		return raw, "", false
	}
	return raw, path.Clean(decoded), true
}

func safePath(value string) bool {
	return strings.HasPrefix(value, "/") && !strings.Contains(value, "\\") && !containsDotDot(value)
}

func containsDotDot(value string) bool {
	for _, part := range strings.Split(value, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func excluded(clean string) bool {
	for _, prefix := range []string{"/api", "/healthz", "/readyz", "/src", "/node_modules", "/@"} {
		if clean == prefix || strings.HasPrefix(clean, prefix+"/") {
			return true
		}
	}
	return false
}

func documentRequest(r *http.Request) bool {
	if (r.Method != http.MethodGet && r.Method != http.MethodHead) || !acceptsHTML(r.Header.Values("Accept")) {
		return false
	}
	destination := r.Header.Get("Sec-Fetch-Dest")
	if destination != "" && destination != "document" && destination != "iframe" && destination != "frame" {
		return false
	}
	mode := r.Header.Get("Sec-Fetch-Mode")
	return mode == "" || mode == "navigate"
}

func acceptsHTML(values []string) bool {
	for _, value := range strings.Split(strings.Join(values, ","), ",") {
		parts := strings.Split(value, ";")
		if strings.TrimSpace(strings.ToLower(parts[0])) != "text/html" {
			continue
		}
		for _, parameter := range parts[1:] {
			parameter = strings.TrimSpace(parameter)
			if len(parameter) > 2 && strings.EqualFold(parameter[:2], "q=") {
				quality, err := strconv.ParseFloat(parameter[2:], 64)
				if err != nil || quality <= 0 || quality > 1 {
					return false
				}
			}
		}
		return true
	}
	return false
}

func setCache(c *echo.Context, clean string) {
	if strings.HasPrefix(clean, "/assets/") {
		c.Response().Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		return
	}
	c.Response().Header().Set("Cache-Control", "no-cache")
}

func setSecurityHeaders(c *echo.Context) {
	hdr := c.Response().Header()
	hdr.Set("X-Content-Type-Options", "nosniff")
	hdr.Set("X-Frame-Options", "DENY")
	hdr.Set("Referrer-Policy", "same-origin")
	hdr.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; "+
		"connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'; object-src 'none'")
}
