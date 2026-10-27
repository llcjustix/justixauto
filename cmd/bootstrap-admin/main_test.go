package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"

	"justixauto/internal/modules/identity"
	"justixauto/internal/pkg/database"
)

// Run the actual CLI in a child process, including flag parsing and exit codes.
func TestMain(m *testing.M) {
	if os.Getenv("JUSTIX_BOOTSTRAP_TEST_PROCESS") == "1" {
		main()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TEST_DATABASE_URL must name a disposable database: identity rows are emptied.
func TestBootstrapDeployment(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("set TEST_DATABASE_URL to a disposable PostgreSQL database")
	}
	m, err := database.NewMigrator(dbURL)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		m.Close()
		t.Fatal(err)
	}
	m.Close()
	db, err := database.Open(dbURL)
	if err != nil {
		t.Fatal(err)
	}
	pool, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pool.Close() })
	mod, err := identity.New(db, identity.Config{Session: identity.DefaultSessionConfig})
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// The child must never load the developer's .env.
	workingDir := t.TempDir()
	const password = "fixture-only-$dollar-`literal`-password"
	args := []string{"-login", "bootstrap-test", "-email", "bootstrap@example.test", "-name", "Bootstrap Test"}
	conditionalArgs := append([]string{"-if-missing"}, args...)
	command := func(ctx context.Context, input io.Reader, targetURL string, arguments ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, executable, arguments...)
		cmd.Env = []string{"JUSTIX_BOOTSTRAP_TEST_PROCESS=1", "DATABASE_URL=" + targetURL}
		cmd.Dir, cmd.Stdin = workingDir, input
		return cmd
	}
	run := func(t *testing.T, input io.Reader, targetURL string, arguments ...string) (string, error) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		out, err := command(ctx, input, targetURL, arguments...).CombinedOutput()
		if ctx.Err() != nil {
			t.Fatal("bootstrap command did not finish promptly")
		}
		if bytes.Contains(out, []byte(password)) {
			t.Fatal("bootstrap printed the password")
		}
		return string(out), err
	}
	reset := func(t *testing.T) {
		t.Helper()
		// TRUNCATE CASCADE reaches companies through verification_actor_id,
		// then removes even built-in roles. Delete only our fixture accounts;
		// their sessions and user-role links use ON DELETE CASCADE.
		if err := db.Exec("DELETE FROM identity_audit_events; DELETE FROM identity_users").Error; err != nil {
			t.Fatal(err)
		}
	}
	counts := func(t *testing.T, users, admins, audits int64) {
		t.Helper()
		for query, want := range map[string]int64{
			"SELECT count(*) FROM identity_users": users,
			"SELECT count(*) FROM identity_user_roles ur JOIN identity_users u ON u.id=ur.user_id WHERE ur.role_id='00000000-0000-4000-8000-000000000001' AND u.status='active'": admins,
			"SELECT count(*) FROM identity_audit_events WHERE action='user.bootstrapped'":                                                                                        audits,
		} {
			var got int64
			if err := db.Raw(query).Scan(&got).Error; err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Fatalf("unexpected fixture count: got %d, want %d", got, want)
			}
		}
	}
	snapshot := func(t *testing.T) string {
		t.Helper()
		var state strings.Builder
		for _, table := range []string{"identity_users", "identity_user_roles", "identity_sessions", "identity_audit_events"} {
			var rows string
			if err := db.Raw("SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY to_jsonb(r)::text), '[]'::jsonb)::text FROM " + table + " r").Scan(&rows).Error; err != nil {
				t.Fatal(err)
			}
			state.WriteString(rows)
		}
		return state.String()
	}
	seed := func(t *testing.T) {
		t.Helper()
		out, err := run(t, strings.NewReader(password), dbURL, conditionalArgs...)
		if err != nil || !strings.Contains(out, "created platform administrator") {
			t.Fatalf("initial bootstrap failed: %v; %s", err, out)
		}
	}

	t.Run("create_login_skip_and_strict_repeat", func(t *testing.T) {
		reset(t)
		seed(t)
		counts(t, 1, 1, 1)
		if _, err := mod.Auth.Login(context.Background(), "bootstrap-test", password, ""); err != nil {
			t.Fatalf("seeded credentials cannot sign in: %v", err)
		}
		before := snapshot(t)
		reader, writer, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		defer reader.Close()
		defer writer.Close()
		// Keep stdin open with no bytes: a read would hang until the timeout.
		out, err := run(t, reader, dbURL, "-if-missing")
		if err != nil || !strings.Contains(out, "skipping bootstrap") {
			t.Fatalf("existing-admin skip failed: %v; %s", err, out)
		}
		if out, err := run(t, strings.NewReader(password), dbURL, args...); err == nil || !strings.Contains(out, "already exists") {
			t.Fatalf("strict repeat did not fail as expected: %v; %s", err, out)
		}
		if snapshot(t) != before {
			t.Fatal("repeat changed account, password hash, roles, sessions or audit")
		}
	})

	for _, tc := range []struct {
		name, input string
		arguments   []string
	}{
		{"missing_password", "", conditionalArgs},
		{"invalid_password", "short", conditionalArgs},
		{"missing_identity", password, []string{"-if-missing"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reset(t)
			if _, err := run(t, strings.NewReader(tc.input), dbURL, tc.arguments...); err == nil {
				t.Fatal("invalid initial credentials were accepted")
			}
			counts(t, 0, 0, 0)
		})
	}

	t.Run("collision_does_not_promote_or_reactivate", func(t *testing.T) {
		reset(t)
		seed(t)
		if err := db.Exec("DELETE FROM identity_user_roles").Error; err != nil {
			t.Fatal(err)
		}
		before := snapshot(t)
		out, err := run(t, strings.NewReader(password), dbURL, conditionalArgs...)
		if err == nil || !strings.Contains(out, "login or email already exists") || snapshot(t) != before {
			t.Fatal("login collision was not rejected without account mutation")
		}
		reset(t)
		seed(t)
		if err := db.Exec("UPDATE identity_users SET status='suspended'").Error; err != nil {
			t.Fatal(err)
		}
		before = snapshot(t)
		out, err = run(t, strings.NewReader(password), dbURL, conditionalArgs...)
		if err == nil || !strings.Contains(out, "login or email already exists") || snapshot(t) != before {
			t.Fatal("suspended admin was skipped, reactivated or changed")
		}
		out, err = run(t, strings.NewReader(password), dbURL, "-if-missing", "-login", "new-admin", "-email", "new@example.test", "-name", "New Admin")
		if err != nil {
			t.Fatalf("no-active-admin bootstrap failed: %v; %s", err, out)
		}
		counts(t, 2, 1, 2)
	})

	t.Run("concurrent_seeds_create_once", func(t *testing.T) {
		reset(t)
		// Hold the guard until both commands pass preflight and reach the lock.
		tx := db.Begin()
		if tx.Error != nil {
			t.Fatal(tx.Error)
		}
		defer tx.Rollback()
		if err := tx.Exec("SELECT id FROM identity_roles WHERE id='00000000-0000-4000-8000-000000000001' FOR UPDATE").Error; err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		type result struct {
			output []byte
			err    error
		}
		results := make(chan result, 2)
		for range 2 {
			go func() {
				out, err := command(ctx, strings.NewReader(password), dbURL, conditionalArgs...).CombinedOutput()
				results <- result{out, err}
			}()
		}
		deadline := time.Now().Add(10 * time.Second)
		for {
			var waiting int64
			if err := db.Raw("SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%identity_roles%'").Scan(&waiting).Error; err != nil {
				t.Fatal(err)
			}
			if waiting >= 2 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("concurrent commands did not both reach the admin guard")
			}
			time.Sleep(25 * time.Millisecond)
		}
		if err := tx.Commit().Error; err != nil {
			t.Fatal(err)
		}
		created, skipped := 0, 0
		for range 2 {
			r := <-results
			if bytes.Contains(r.output, []byte(password)) {
				t.Fatal("concurrent bootstrap printed the password")
			}
			if r.err != nil {
				t.Fatalf("concurrent bootstrap failed: %v; %s", r.err, r.output)
			}
			if bytes.Contains(r.output, []byte("created platform administrator")) {
				created++
			}
			if bytes.Contains(r.output, []byte("skipping bootstrap")) {
				skipped++
			}
		}
		if created != 1 || skipped != 1 {
			t.Fatalf("concurrent results: created=%d skipped=%d", created, skipped)
		}
		counts(t, 1, 1, 1)
	})

	t.Run("database_error_is_not_a_skip", func(t *testing.T) {
		u, err := url.Parse(dbURL)
		if err != nil {
			t.Fatal(err)
		}
		u.Path = "/bootstrap_database_does_not_exist"
		if out, err := run(t, nil, u.String(), "-if-missing"); err == nil || strings.Contains(out, "skipping bootstrap") {
			t.Fatal("database failure was treated as a successful skip")
		}
	})
}
