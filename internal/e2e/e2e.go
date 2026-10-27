// Package e2e runs module tests end to end: a real PostgreSQL database
// (TEST_DATABASE_URL, emptied per test), the identity module for sign-in, and
// HTTP clients that behave like a browser (cookies, CSRF).
// Use it from _test.go files only.
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"justixauto/internal/app"
	"justixauto/internal/modules/documents"
	"justixauto/internal/modules/identity"
	"justixauto/internal/pkg/database"
)

// Clock is a controllable time source shared by the server and the test.
type Clock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *Clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.t }

// DB migrates TEST_DATABASE_URL and empties all application tables. The test
// is skipped when the variable is not set.
func DB(t *testing.T) *gorm.DB {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database (make test-go)")
	}
	m, err := database.NewMigrator(url)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		t.Fatal(err)
	}
	m.Close()
	db, err := database.Open(url)
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	err = db.Raw(`SELECT quote_ident(schemaname) || '.' || quote_ident(tablename) FROM pg_tables
		WHERE schemaname = 'public'
		AND tablename ~ '^(identity|inventory|commerce|retail|documents|financing|insurance|platform)_'
		AND tablename <> 'identity_permissions'
		ORDER BY tablename`).Scan(&tables).Error
	if err != nil {
		t.Fatal(err)
	}
	if len(tables) == 0 {
		t.Fatal("no application tables found in disposable test database")
	}
	// Company truncation cascades to roles even when roles are omitted from
	// the table list. Keep the exact builtin rows, including migration columns.
	if err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`CREATE TEMP TABLE test_builtin_roles ON COMMIT DROP AS
			SELECT * FROM public.identity_roles WHERE system_key IS NOT NULL`).Error; err != nil {
			return err
		}
		// The audit tables are append-only; lift their guards only for this reset.
		if err := tx.Exec(auditGuardsOff).Error; err != nil {
			return err
		}
		if err := tx.Exec("TRUNCATE " + strings.Join(tables, ", ") + " CASCADE").Error; err != nil {
			return err
		}
		if err := tx.Exec(auditGenesisAndGuardsOn).Error; err != nil {
			return err
		}
		return tx.Exec("INSERT INTO public.identity_roles SELECT * FROM test_builtin_roles").Error
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	})
	return db
}

// auditGuardsOff disables the append-only audit triggers until the end of the
// reset. ALTER TABLE is transactional, so a failed reset keeps them enabled.
const auditGuardsOff = `
ALTER TABLE public.identity_audit_events DISABLE TRIGGER USER;
ALTER TABLE public.identity_audit_chain DISABLE TRIGGER USER;`

// auditGenesisAndGuardsOn recreates the empty chain head exactly as migration
// 000047 seeds it, then re-enables the audit triggers.
const auditGenesisAndGuardsOn = `
WITH boundary AS (
    SELECT pg_catalog.gen_random_uuid() AS stream_id,
        COALESCE(pg_catalog.max(seq), 0) AS legacy_max_seq,
        pg_catalog.count(*) AS legacy_count,
        pg_catalog.clock_timestamp() AS created_at
    FROM public.identity_audit_events
), payload AS (
    SELECT boundary.*, public.identity_audit_boundary_v1(
        stream_id, legacy_max_seq, legacy_count, created_at) AS bytes FROM boundary
), genesis AS (
    SELECT payload.*, pg_catalog.sha256(
        pg_catalog.convert_to(E'justixauto-audit-genesis-v1\n', 'UTF8') || bytes) AS hash
    FROM payload
)
INSERT INTO public.identity_audit_chain (
    singleton, stream_id, chain_version, last_index, last_hash, genesis_hash,
    legacy_max_seq, legacy_count, created_at, boundary_payload
)
SELECT true, stream_id, 1, 0, hash, hash, legacy_max_seq, legacy_count, created_at, bytes
FROM genesis;
ALTER TABLE public.identity_audit_events ENABLE TRIGGER USER;
ALTER TABLE public.identity_audit_chain ENABLE TRIGGER USER;`

// ResetAudit empties the append-only audit history of a disposable test
// database and restores a fresh chain head. Never use it outside tests.
func ResetAudit(db *gorm.DB) error {
	return db.Transaction(func(tx *gorm.DB) error {
		return tx.Exec(auditGuardsOff +
			"\nTRUNCATE public.identity_audit_events, public.identity_audit_chain;" +
			auditGenesisAndGuardsOn).Error
	})
}

// Env is a running API with identity plus the modules under test.
type Env struct {
	T        *testing.T
	DB       *gorm.DB
	Clock    *Clock
	Identity *identity.Module
	srv      *httptest.Server
}

// New starts the full API (all modules, wired as in production) on a clean
// database with a controllable clock.
func New(t *testing.T) *Env {
	t.Helper()
	db := DB(t)
	clock := &Clock{t: time.Date(2026, 9, 22, 9, 0, 0, 0, time.UTC)}
	e, idm, err := app.New(db, app.Config{
		Files: fileStorage(t), Session: identity.DefaultSessionConfig, Now: clock.Now,
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(e)
	t.Cleanup(srv.Close)
	return &Env{T: t, DB: db, Clock: clock, Identity: idm, srv: srv}
}

// Client is one browser session.
type Client struct {
	env  *Env
	http *http.Client
	csrf string
	// UserID and CompanyID are filled by the helpers that create them.
	UserID, CompanyID string
}

func (e *Env) Browser() *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{env: e, http: &http.Client{Jar: jar}}
}

// Response is a decoded API response.
type Response struct {
	Status int
	Header http.Header
	Body   map[string]any
}

func (r Response) Data() map[string]any { d, _ := r.Body["data"].(map[string]any); return d }
func (r Response) Items() []any         { i, _ := r.Body["items"].([]any); return i }
func (r Response) Revision() string     { s, _ := r.Body["revision"].(string); return s }
func (r Response) Code() string {
	e, _ := r.Body["error"].(map[string]any)
	c, _ := e["code"].(string)
	return c
}

// Do sends a request to /api/v1+path. headers are name/value pairs.
func (c *Client) Do(method, path string, body any, headers ...string) Response {
	t := c.env.T
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, c.env.srv.URL+"/api/v1"+path, reader)
	req.Header.Set("Content-Type", "application/json")
	if c.csrf != "" {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	for i := 0; i+1 < len(headers); i += 2 {
		req.Header.Set(headers[i], headers[i+1])
	}
	res, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out := Response{Status: res.StatusCode, Header: res.Header, Body: map[string]any{}}
	_ = json.NewDecoder(res.Body).Decode(&out.Body)
	if token := res.Header.Get("X-CSRF-Token"); token != "" {
		c.csrf = token
	}
	return out
}

// IfMatch builds the If-Match header pair for Do.
func IfMatch(revision string) []string { return []string{"If-Match", `"` + revision + `"`} }

// Expect fails the test unless the status (and optional error code) match.
func Expect(t *testing.T, r Response, status int, code ...string) {
	t.Helper()
	if r.Status != status || (len(code) > 0 && r.Code() != code[0]) {
		t.Fatalf("want %d %v, got %d %v", status, code, r.Status, r.Body)
	}
}

// SignIn logs in with login and password.
func (c *Client) SignIn(login, password string) Response {
	return c.Do(http.MethodPost, "/identity/session/login", map[string]string{"login": login, "password": password})
}

const adminPassword = "admin-password-123"

// Admin bootstraps and signs in the platform administrator (once per Env).
func (e *Env) Admin() *Client {
	t := e.T
	t.Helper()
	_, err := e.Identity.Users.Bootstrap(context.Background(), identity.BootstrapInput{
		DisplayName: "Platform Admin", Login: "admin", Email: "admin@justix.test", Password: adminPassword,
	})
	if err != nil {
		t.Fatal(err)
	}
	admin := e.Browser()
	Expect(t, admin.SignIn("admin", adminPassword), http.StatusOK)
	return admin
}

// CompanyUser creates an active, verified dealer and returns an employee
// holding exactly perms through a private company role.
func (e *Env) CompanyUser(admin *Client, name string, perms ...string) *Client {
	e.T.Helper()
	return e.KindUser(admin, "seller", name, perms...)
}

// KindUser is CompanyUser for any company kind: seller, bank, mfo or insurance.
func (e *Env) KindUser(admin *Client, kind, name string, perms ...string) *Client {
	t := e.T
	t.Helper()
	login := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	path, body := "/identity/admin/seller-companies", map[string]any{"userType": "dealer"}
	if kind != "seller" {
		path, body = "/identity/admin/provider-companies", map[string]any{"kind": kind}
	}
	body["company"] = map[string]any{
		"name": name, "country": map[string]string{"key": "UZ", "label": "Uzbekistan"},
		"registration": "REG-" + login, "email": "office@" + login + ".test",
	}
	body["firstAdmin"] = map[string]string{
		"displayName": name + " admin", "login": login + "-admin", "email": login + "-admin@company.test",
		"password": "company-password-1", "passwordConfirmation": "company-password-1",
	}
	created := admin.Do(http.MethodPost, path, body)
	Expect(t, created, http.StatusCreated)
	companyID := created.Data()["company"].(map[string]any)["id"].(string)
	activated := admin.Do(http.MethodPost, "/identity/admin/companies/"+companyID+"/activate",
		map[string]string{"reason": "onboarded"}, IfMatch("1")...)
	Expect(t, activated, http.StatusOK)

	owner := e.Browser()
	session := owner.SignIn(login+"-admin", "company-password-1")
	Expect(t, session, http.StatusOK)
	Expect(t, owner.Do(http.MethodPut, "/identity/session/context", map[string]any{"companyId": companyID}, IfMatch(session.Revision())...), http.StatusOK)
	if kind == "seller" {
		submitted := owner.Do(http.MethodPost, "/identity/companies/"+companyID+"/submit-verification", nil, IfMatch(activated.Revision())...)
		Expect(t, submitted, http.StatusOK)
		Expect(t, admin.Do(http.MethodPost, "/identity/admin/companies/"+companyID+"/verify", nil, IfMatch(submitted.Revision())...), http.StatusOK)
	}
	role := owner.Do(http.MethodPost, "/identity/companies/"+companyID+"/roles", map[string]any{"name": name + " staff", "permissionKeys": perms})
	Expect(t, role, http.StatusCreated)
	staff := owner.Do(http.MethodPost, "/identity/companies/"+companyID+"/users", map[string]any{
		"displayName": name + " user", "login": login, "email": login + "@company.test",
		"password": "temporary-company-pass-1", "roleIds": []string{role.Data()["id"].(string)},
	})
	Expect(t, staff, http.StatusCreated)
	c := e.Browser()
	Expect(t, c.SignIn(login, "temporary-company-pass-1"), http.StatusOK)
	session = c.Do(http.MethodPost, "/identity/session/password", map[string]string{
		"currentPassword": "temporary-company-pass-1", "newPassword": "company-password-1", "newPasswordConfirmation": "company-password-1",
	})
	Expect(t, session, http.StatusOK)
	Expect(t, c.Do(http.MethodPut, "/identity/session/context", map[string]any{"companyId": companyID}, IfMatch(session.Revision())...), http.StatusOK)
	c.UserID, c.CompanyID = staff.Data()["id"].(string), companyID
	return c
}

// Upload sends a file as multipart/form-data to /api/v1/documents/files.
func (c *Client) Upload(purpose, name string, content []byte) Response {
	t := c.env.T
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("purpose", purpose)
	part, _ := w.CreateFormFile("file", name)
	_, _ = part.Write(content)
	_ = w.Close()
	req, _ := http.NewRequest(http.MethodPost, c.env.srv.URL+"/api/v1/documents/files", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-CSRF-Token", c.csrf)
	res, err := c.http.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out := Response{Status: res.StatusCode, Header: res.Header, Body: map[string]any{}}
	_ = json.NewDecoder(res.Body).Decode(&out.Body)
	return out
}

// Raw performs a GET and returns status and body bytes (e.g. file downloads).
func (c *Client) Raw(path string) (int, []byte) {
	res, err := c.http.Get(c.env.srv.URL + "/api/v1" + path)
	if err != nil {
		c.env.T.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, b
}

// PDF is a minimal file detected as application/pdf.
var PDF = []byte("%PDF-1.4\n1 0 obj << /Type /Catalog >> endobj\ntrailer << /Root 1 0 R >>\n%%EOF\n")

// fileStorage uses S3 (e.g. MinIO) when TEST_S3_ENDPOINT is set, else a
// temporary directory.
func fileStorage(t *testing.T) documents.Storage {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		return documents.DirStorage{Root: t.TempDir()}
	}
	s, err := documents.NewS3Storage(context.Background(), documents.S3Config{
		Bucket: "justixauto-test", Region: "us-east-1",
		Prefix: "t/" + uuid.NewString() + "/", Endpoint: endpoint, PathStyle: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.EnsureBucket(context.Background()); err != nil {
		t.Fatal(err)
	}
	return s
}
