package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const devAPIBin = "/repo/var/bin/dev-api"

// devRunner is a concurrency-safe fake. The API build and migrate finish at
// once with code 0 (buildCode for the build); processes matching exit end by
// themselves with code; everything else runs until its context is cancelled.
type devRunner struct {
	mu        sync.Mutex
	calls     []command
	exit      string // argv substring of a process that exits by itself
	code      int
	buildCode int
}

func (d *devRunner) run(ctx context.Context, c command) (string, int, error) {
	d.mu.Lock()
	d.calls = append(d.calls, c)
	d.mu.Unlock()
	s := argv(c)
	switch {
	case strings.Contains(s, "go.sh build"):
		return "", d.buildCode, nil
	case strings.Contains(s, "cmd/migrate"):
		return "", 0, nil
	case d.exit != "" && strings.Contains(s, d.exit):
		_, _ = c.stdout.Write([]byte("boom\n"))
		return "", d.code, nil
	}
	<-ctx.Done()
	return "", -1, context.Cause(ctx)
}

// count returns how many started commands satisfy match.
func (d *devRunner) count(match func(string) bool) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	n := 0
	for _, c := range d.calls {
		if match(argv(c)) {
			n++
		}
	}
	return n
}

func isAPIRun(s string) bool { return s == devAPIBin }
func isBuild(s string) bool  { return strings.Contains(s, "go.sh build") }
func has(sub string) func(string) bool {
	return func(s string) bool { return strings.Contains(s, sub) }
}

func (d *devRunner) waitFor(t *testing.T, match func(string) bool, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for d.count(match) < n {
		if time.Now().After(deadline) {
			t.Fatalf("waited for %d matching calls, got %d", n, d.count(match))
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (d *devRunner) env(t *testing.T, match func(string) bool, key string) string {
	t.Helper()
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, c := range d.calls {
		if match(argv(c)) {
			v := ""
			for _, e := range c.env {
				if strings.HasPrefix(e, key+"=") {
					v = strings.TrimPrefix(e, key+"=") // last one wins, as in os/exec
				}
			}
			return v
		}
	}
	t.Fatal("no matching call")
	return ""
}

// lockedBuilder is a strings.Builder safe to read while dev writes to it.
type lockedBuilder struct {
	mu sync.Mutex
	b  strings.Builder
}

func (l *lockedBuilder) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuilder) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func devTestApp(r runner, env map[string]string, busy ...int) (*app, *lockedBuilder) {
	out := &lockedBuilder{}
	return &app{
		root: "/repo", run: r, stdout: out, stderr: &lockedBuilder{},
		getenv:   func(k string) string { return env[k] },
		portFree: func(p int) bool { return !contains(busy, p) },
	}, out
}

func contains(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

// startDev runs runDev in the background with a scripted change channel.
func startDev(t *testing.T, r *devRunner, env map[string]string) (chan devChange, context.CancelCauseFunc, <-chan error, *lockedBuilder) {
	t.Helper()
	changes := make(chan devChange, 4)
	a, out := devTestApp(r, env)
	a.devChanges = changes
	ctx, cancel := context.WithCancelCause(context.Background())
	done := make(chan error, 1)
	go func() { done <- runDev(ctx, a, nil) }()
	t.Cleanup(func() { cancel(errors.New("test over")) })
	return changes, cancel, done, out
}

func TestDevBuildsAndRunsTheAPIBinaryAndFourApps(t *testing.T) {
	r := &devRunner{}
	_, cancel, done, out := startDev(t, r, map[string]string{"HTTP_ADDR": "127.0.0.1:8090", "ALLOWED_ORIGINS": "http://x.test"})
	r.waitFor(t, isAPIRun, 1)
	r.waitFor(t, has("npm run dev"), 4)

	if got := r.env(t, isAPIRun, "ALLOWED_ORIGINS"); got !=
		"http://x.test,http://127.0.0.1:5191,http://localhost:5191,http://127.0.0.1:5192,http://localhost:5192,"+
			"http://127.0.0.1:5193,http://localhost:5193,http://127.0.0.1:5194,http://localhost:5194" {
		t.Errorf("API ALLOWED_ORIGINS = %q", got)
	}
	if got := r.env(t, isAPIRun, "WEB_DIR"); got != "" {
		t.Errorf("API WEB_DIR = %q, want empty (Vite serves the apps)", got)
	}
	if got := r.env(t, has("apps/admin"), "JUSTIX_API"); got != "http://127.0.0.1:8090" {
		t.Errorf("JUSTIX_API = %q", got)
	}
	for _, want := range []string{
		"npm run dev --workspace apps/realization -- --port 5191 --strictPort",
		"npm run dev --workspace apps/admin -- --port 5194 --strictPort",
		"bash /repo/tools/go.sh build -o " + devAPIBin + " ./cmd/api",
	} {
		if r.count(func(s string) bool { return s == want }) != 1 {
			t.Errorf("missing process %q", want)
		}
	}
	stopped := errors.New("stop")
	cancel(stopped)
	if err := <-done; !errors.Is(err, stopped) {
		t.Fatalf("err = %v, want the cancel cause", err)
	}
	if s := out.String(); !strings.Contains(s, "admin        http://127.0.0.1:5194/admin/") {
		t.Errorf("output missing app URL:\n%s", s)
	}
}

func TestDevStopsEverythingWhenAWebAppExits(t *testing.T) {
	r := &devRunner{exit: "apps/insurance", code: 1}
	a, out := devTestApp(r, map[string]string{})
	a.devChanges = make(chan devChange)
	err := runDev(context.Background(), a, nil)
	if err == nil || !strings.Contains(err.Error(), "insurance exited with code 1") {
		t.Fatalf("err = %v, want insurance exit reported", err)
	}
	if !strings.Contains(out.String(), "[insurance] boom\n") {
		t.Errorf("missing prefixed log line:\n%s", out.String())
	}
}

func TestDevRebuildsAndRestartsOnlyTheAPIWhenSourcesChange(t *testing.T) {
	r := &devRunner{}
	changes, _, _, out := startDev(t, r, map[string]string{})
	r.waitFor(t, isAPIRun, 1)
	changes <- devChange{}
	r.waitFor(t, isAPIRun, 2)
	if n := r.count(isBuild); n != 2 {
		t.Errorf("builds = %d, want 2", n)
	}
	if n := r.count(has("npm run dev")); n != 4 {
		t.Errorf("web apps started %d times, want 4 (never restarted)", n)
	}
	if r.count(has("cmd/migrate")) != 0 {
		t.Error("migrations must not run for a Go-only change")
	}
	if !strings.Contains(out.String(), "[dev] source changed, rebuilding the API") {
		t.Errorf("missing restart message:\n%s", out.String())
	}
}

func TestDevAppliesMigrationsBeforeStartingTheNewAPI(t *testing.T) {
	r := &devRunner{}
	changes, _, _, _ := startDev(t, r, map[string]string{})
	r.waitFor(t, isAPIRun, 1)
	changes <- devChange{migrations: true}
	r.waitFor(t, isAPIRun, 2)
	r.mu.Lock()
	var order []string
	for _, c := range r.calls {
		if s := argv(c); !strings.Contains(s, "npm") {
			order = append(order, s)
		}
	}
	r.mu.Unlock()
	if len(order) != 5 || !isBuild(order[2]) || !strings.Contains(order[3], "cmd/migrate up") || !isAPIRun(order[4]) {
		t.Fatalf("want build, api, build, migrate, api; got %v", order)
	}
}

func TestDevKeepsTheRunningAPIWhenABuildFails(t *testing.T) {
	r := &devRunner{}
	changes, _, _, out := startDev(t, r, map[string]string{})
	r.waitFor(t, isAPIRun, 1)
	r.mu.Lock()
	r.buildCode = 1 // the saved code does not compile
	r.mu.Unlock()
	changes <- devChange{}
	r.waitFor(t, isBuild, 2)
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), "build failed") && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if n := r.count(isAPIRun); n != 1 {
		t.Errorf("API started %d times, want 1 (the old one keeps running)", n)
	}
}

func TestDevKeepsWebAppsRunningWhenTheAPIExits(t *testing.T) {
	r := &devRunner{exit: devAPIBin, code: 1} // e.g. the API panics at start-up
	changes, cancel, done, out := startDev(t, r, map[string]string{})
	deadline := time.Now().Add(5 * time.Second)
	for !strings.Contains(out.String(), "exited with code 1") && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	select {
	case err := <-done:
		t.Fatalf("dev stopped after the API exited: %v", err)
	default:
	}
	changes <- devChange{}
	r.waitFor(t, isAPIRun, 2)
	stopped := errors.New("stop")
	cancel(stopped)
	if err := <-done; !errors.Is(err, stopped) {
		t.Fatalf("err = %v, want the cancel cause", err)
	}
}

func TestDevDefaultsTheAPIAddress(t *testing.T) {
	r := &devRunner{}
	startDev(t, r, map[string]string{})
	r.waitFor(t, has("apps/realization"), 1)
	if got := r.env(t, has("apps/realization"), "JUSTIX_API"); got != "http://127.0.0.1:8080" {
		t.Errorf("JUSTIX_API = %q, want the default API address", got)
	}
}

func TestDevRejectsArguments(t *testing.T) {
	a, _ := devTestApp(&devRunner{}, nil)
	if err := runDev(context.Background(), a, []string{"x"}); err == nil {
		t.Fatal("want an error for unexpected arguments")
	}
}

func TestDevSkipsBusyPortsWithoutSharingOne(t *testing.T) {
	ports, err := pickDevPorts(func(p int) bool { return p != 5191 && p != 5192 })
	if err != nil {
		t.Fatal(err)
	}
	// realization moves past 5191/5192 to 5193; financing must not reuse it.
	want := []int{5193, 5194, 5195, 5196}
	for i := range want {
		if ports[i] != want[i] {
			t.Fatalf("ports = %v, want %v", ports, want)
		}
	}
}

func TestPrefixWriterKeepsLinesWholeAndFlushesTail(t *testing.T) {
	out := &strings.Builder{}
	w := &prefixWriter{w: out, prefix: "[a] ", mu: &sync.Mutex{}}
	_, _ = w.Write([]byte("one\ntw"))
	_, _ = w.Write([]byte("o\nthree"))
	w.flush()
	if got := out.String(); got != "[a] one\n[a] two\n[a] three\n" {
		t.Fatalf("got %q", got)
	}
}

func TestWatchSourcesReportsGoAndMigrationChanges(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"internal/x", "migrations"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write := func(rel, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, rel), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("internal/x/a.go", "package x")
	write("migrations/0001_a.up.sql", "select 1;")
	write("internal/x/notes.txt", "ignored")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	changes := watchSources(ctx, root, 20*time.Millisecond)

	receive := func() devChange {
		t.Helper()
		select {
		case c := <-changes:
			return c
		case <-time.After(3 * time.Second):
			t.Fatal("no change reported")
			return devChange{}
		}
	}
	write("internal/x/a.go", "package x // edited")
	if c := receive(); c.migrations {
		t.Error("a Go change must not be reported as a migration change")
	}
	write("migrations/0002_b.up.sql", "select 2;")
	if c := receive(); !c.migrations {
		t.Error("a new migration must be reported as a migration change")
	}
	write("internal/x/notes.txt", "still ignored")
	select {
	case c := <-changes:
		t.Errorf("non-source file reported as a change: %+v", c)
	case <-time.After(150 * time.Millisecond):
	}
}
