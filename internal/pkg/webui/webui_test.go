package webui

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/labstack/echo/v5"
)

func write(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMount(t *testing.T) {
	dist := t.TempDir()
	write(t, filepath.Join(dist, "index.html"), "portal")
	write(t, filepath.Join(dist, "assets", "app-1.js"), "js")
	e := echo.New()
	e.GET("/api/v1/ping", func(c *echo.Context) error { return c.String(http.StatusOK, "pong") })
	e.GET("/healthz", func(c *echo.Context) error { return c.String(http.StatusOK, "healthy") })
	e.GET("/readyz", func(c *echo.Context) error { return c.String(http.StatusOK, "ready") })
	Mount(e, dist)

	cases := []struct {
		path, accept, body, cache string
		status                    int
	}{
		{"/", "text/html", "portal", "no-cache", 200},
		{"/sales/123", "text/html", "portal", "no-cache", 200},
		{"/finance/applications", "text/html", "portal", "no-cache", 200},
		{"/insurance/quote", "text/html", "portal", "no-cache", 200},
		{"/admin/users", "text/html", "portal", "no-cache", 200},
		{"/assets/app-1.js", "*/*", "js", "public, max-age=31536000, immutable", 200},
		{"/assets/missing.js", "*/*", "", "", 404},
		{"/assets/missing", "text/html", "", "", 404},
		{"/api/v1/ping", "text/html", "pong", "", 200},
		{"/healthz", "text/html", "healthy", "", 200},
		{"/readyz", "text/html", "ready", "", 200},
		{"/missing.html", "text/html", "", "", 404},
		{"/src/main.tsx", "text/html", "", "", 404},
		{"/%2e%2e/etc/passwd", "text/html", "", "", 404},
		{"/%252e%252e/etc/passwd", "text/html", "", "", 404},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, http.NoBody)
		req.Header.Set("Accept", tc.accept)
		e.ServeHTTP(rec, req)
		if rec.Code != tc.status {
			t.Fatalf("%s: got status %d", tc.path, rec.Code)
		}
		if tc.body != "" && rec.Body.String() != tc.body {
			t.Fatalf("%s: got body %q", tc.path, rec.Body.String())
		}
		if tc.cache != "" && rec.Header().Get("Cache-Control") != tc.cache {
			t.Fatalf("%s: got cache %q", tc.path, rec.Header().Get("Cache-Control"))
		}
	}
}

func TestPortalDocumentIntent(t *testing.T) {
	dist := t.TempDir()
	write(t, filepath.Join(dist, "index.html"), "portal")
	e := echo.New()
	Mount(e, dist)
	for _, tc := range []struct {
		name, method, accept, dest, mode string
		status                           int
	}{
		{"get", http.MethodGet, "text/html", "document", "navigate", 200},
		{"head", http.MethodHead, "text/html", "iframe", "navigate", 200},
		{"wildcard", http.MethodGet, "*/*", "", "", 404},
		{"zero-quality", http.MethodGet, "text/html;q=0", "", "", 404},
		{"decimal-zero-quality", http.MethodGet, "text/html;q=0.0", "", "", 404},
		{"image", http.MethodGet, "text/html", "image", "", 404},
		{"cors", http.MethodGet, "text/html", "", "cors", 404},
		{"post", http.MethodPost, "text/html", "", "", 405},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/finance", http.NoBody)
			req.Header.Set("Accept", tc.accept)
			req.Header.Set("Sec-Fetch-Dest", tc.dest)
			req.Header.Set("Sec-Fetch-Mode", tc.mode)
			e.ServeHTTP(rec, req)
			if rec.Code != tc.status {
				t.Fatalf("status %d", rec.Code)
			}
			if tc.method == http.MethodHead && rec.Body.Len() != 0 {
				t.Fatal("HEAD has body")
			}
			if tc.status == http.StatusOK && rec.Header().Get("Content-Security-Policy") == "" {
				t.Fatal("missing CSP")
			}
		})
	}
}

func TestMountMissingIndex(t *testing.T) {
	e := echo.New()
	Mount(e, t.TempDir())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestPortalRejectsMalformedPath(t *testing.T) {
	dist := t.TempDir()
	write(t, filepath.Join(dist, "index.html"), "portal")
	e := echo.New()
	Mount(e, dist)
	rec := httptest.NewRecorder()
	req := &http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/%zz"}, Header: make(http.Header)}
	req.Header.Set("Accept", "text/html")
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d", rec.Code)
	}
}
