package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// devApp is one React app served by its own Vite dev server with hot reload.
type devApp struct {
	name string // workspace directory under web/apps
	path string // URL prefix the app is served under (its Vite base)
	port int    // preferred port; the next free one is used when it is taken
}

// devApps start well above Vite's default 5173 range, which other local
// projects commonly occupy.
var devApps = []devApp{
	{"realization", "/", 5191},
	{"financing", "/finance/", 5192},
	{"insurance", "/insurance/", 5193},
	{"admin", "/admin/", 5194},
}

// devPortSearch bounds how far past a taken preferred port dev looks.
const devPortSearch = 50

// defaultAPIAddr matches the Makefile default when HTTP_ADDR is unset.
const defaultAPIAddr = "127.0.0.1:8080"

// runDev runs the API and the four Vite dev servers with hot reload until
// Ctrl-C or until a web app exits, then stops the rest. The API restarts by
// itself when Go sources or migrations change (migrations are applied first);
// a failing build leaves the web apps running. The database must already be
// up and migrated (`make dev` does that first). Vite origins are added to the
// API's ALLOWED_ORIGINS for this run only, so .env needs no edit.
func runDev(ctx context.Context, a *app, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("takes no arguments, got %q", strings.Join(args, " "))
	}
	portFree := a.portFree
	if portFree == nil {
		portFree = tcpPortFree
	}
	ports, err := pickDevPorts(portFree)
	if err != nil {
		return err
	}
	apiAddr := a.getenv("HTTP_ADDR")
	if apiAddr == "" {
		apiAddr = defaultAPIAddr
	}

	origins := make([]string, 0, len(devApps)+1)
	if existing := a.getenv("ALLOWED_ORIGINS"); existing != "" {
		origins = append(origins, existing)
	}
	for i := range devApps {
		origins = append(origins, "http://127.0.0.1:"+strconv.Itoa(ports[i]))
	}

	fmt.Fprintf(a.stdout, "API          http://%s\n", apiAddr)
	for i, app := range devApps {
		fmt.Fprintf(a.stdout, "%-12s http://127.0.0.1:%d%s\n", app.name, ports[i], app.path)
	}
	fmt.Fprintln(a.stdout, "Hot reload is on (the API restarts when Go files or migrations change); Ctrl-C stops everything.")

	// Later entries win in os/exec, so these override inherited values.
	// WEB_DIR is emptied: the API serves only /api, Vite serves the apps.
	apiEnv := append(os.Environ(), "ALLOWED_ORIGINS="+strings.Join(origins, ","), "WEB_DIR=")
	webEnv := append(os.Environ(), "JUSTIX_API=http://"+apiAddr)

	var mu sync.Mutex // one output line at a time across all processes
	devLog := &prefixWriter{w: a.stdout, prefix: "[dev] ", mu: &mu}
	logf := func(format string, args ...any) {
		fmt.Fprintf(devLog, format+"\n", args...)
	}

	runCtx, stop := context.WithCancelCause(ctx)
	defer stop(nil)

	type ended struct {
		label string
		code  int
		err   error
	}
	webEnded := make(chan ended, len(devApps))
	for i, app := range devApps {
		out := &prefixWriter{w: a.stdout, prefix: "[" + app.name + "] ", mu: &mu}
		errOut := &prefixWriter{w: a.stderr, prefix: "[" + app.name + "] ", mu: &mu}
		cmd := command{
			name: "npm",
			args: []string{"run", "dev", "--workspace", "web/apps/" + app.name, "--", "--port", strconv.Itoa(ports[i]), "--strictPort"},
			dir:  a.root, env: webEnv, stdout: out, stderr: errOut,
		}
		go func() {
			_, code, err := a.run.run(runCtx, cmd)
			out.flush()
			errOut.flush()
			webEnded <- ended{app.name, code, err}
		}()
	}

	changes := a.devChanges
	if changes == nil {
		changes = watchSources(runCtx, a.root, devPollInterval)
	}
	apiOut := &prefixWriter{w: a.stdout, prefix: "[api] ", mu: &mu}
	apiErr := &prefixWriter{w: a.stderr, prefix: "[api] ", mu: &mu}
	apiBin := filepath.Join(a.root, "var", "bin", "dev-api")
	build := command{
		name: "bash", args: []string{a.root + "/tools/go.sh", "build", "-o", apiBin, "./cmd/api"},
		dir: a.root, stdout: apiOut, stderr: apiErr,
	}
	api := command{name: apiBin, dir: a.root, env: apiEnv, stdout: apiOut, stderr: apiErr}
	migrate := command{
		name: "bash", args: []string{a.root + "/tools/go.sh", "run", "./cmd/migrate", "up"},
		dir: a.root, stdout: apiOut, stderr: apiErr,
	}
	apiStopped := make(chan struct{})
	go func() {
		defer close(apiStopped)
		superviseAPI(runCtx, a.run, build, api, migrate, changes, logf, func() { apiOut.flush(); apiErr.flush() })
	}()

	var first ended
	remaining := len(devApps)
	select {
	case first = <-webEnded:
		remaining--
		stop(fmt.Errorf("%s exited", first.label))
	case <-runCtx.Done():
	}
	<-apiStopped
	for range remaining {
		<-webEnded
	}
	if ctx.Err() != nil {
		return context.Cause(ctx) // Ctrl-C: realMain maps it to 130/143
	}
	if first.err != nil {
		return fmt.Errorf("%s could not run: %w; stopped the others", first.label, first.err)
	}
	return fmt.Errorf("%s exited with code %d; stopped the others", first.label, first.code)
}

// devChange is a batch of saved source changes; migrations is set when a
// migration file changed, so it is applied before the API restarts.
type devChange struct{ migrations bool }

// devPollInterval is how often dev looks for saved Go and migration files.
const devPollInterval = 700 * time.Millisecond

// superviseAPI builds and runs the API until ctx ends. On every change it
// builds first; only a successful build replaces the running API (after
// applying migrations when they changed), so a compile error leaves the last
// good API serving. The binary is run directly, so stopping it reaches the
// API itself (go run would not forward the signal).
func superviseAPI(ctx context.Context, r runner, build, api, migrate command, changes <-chan devChange, logf func(string, ...any), flush func()) {
	type result struct {
		code int
		err  error
	}
	var (
		running bool
		cancel  context.CancelFunc
		done    chan result
	)
	stopAPI := func() {
		if running {
			cancel()
			<-done
			running = false
		}
	}
	startAPI := func() {
		var apiCtx context.Context
		apiCtx, cancel = context.WithCancel(ctx)
		done = make(chan result, 1)
		go func(done chan result) {
			_, code, err := r.run(apiCtx, api)
			flush()
			done <- result{code, err}
		}(done)
		running = true
	}
	// deploy builds and, if the build succeeds, swaps in the new API.
	deploy := func(c devChange) {
		if _, code, err := r.run(ctx, build); err != nil || code != 0 {
			if ctx.Err() == nil {
				logf("build failed; fix the code and save (the previous API keeps running if it was up)")
			}
			return
		}
		stopAPI()
		if c.migrations {
			logf("migrations changed, applying them")
			if _, code, err := r.run(ctx, migrate); err == nil && code != 0 {
				logf("migrate exited with code %d", code)
			}
		}
		startAPI()
	}
	// merge folds changes already queued behind c into it.
	merge := func(c devChange) devChange {
		for {
			select {
			case more := <-changes:
				c.migrations = c.migrations || more.migrations
			default:
				return c
			}
		}
	}
	defer stopAPI()
	deploy(devChange{})
	for {
		var apiDone chan result
		if running {
			apiDone = done
		}
		select {
		case <-ctx.Done():
			return
		case c := <-changes:
			logf("source changed, rebuilding the API")
			deploy(merge(c))
		case res := <-apiDone:
			running = false
			cancel()
			if ctx.Err() != nil {
				return
			}
			if res.err != nil {
				logf("the API could not start: %v; save a file to try again", res.err)
			} else {
				logf("the API exited with code %d; save a file to restart it", res.code)
			}
		}
	}
}

// watchSources polls the backend sources (Go files under cmd/ and
// internal/, SQL migrations) and reports saved changes. It waits until files
// stop changing for one interval, so a burst of saves restarts only once.
func watchSources(ctx context.Context, root string, interval time.Duration) <-chan devChange {
	out := make(chan devChange, 1)
	goState, sqlState := sourceState(root) // baseline before returning: no save is missed
	go func() {
		pending := devChange{}
		dirty := false
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
			}
			g, q := sourceState(root)
			changed := g != goState || q != sqlState
			if changed {
				pending.migrations = pending.migrations || q != sqlState
				goState, sqlState, dirty = g, q, true
				continue // wait for one quiet interval
			}
			if dirty {
				select {
				case out <- pending:
				case <-ctx.Done():
					return
				}
				pending, dirty = devChange{}, false
			}
		}
	}()
	return out
}

// sourceState fingerprints Go sources and migrations by path, size and
// modification time.
func sourceState(root string) (goState, sqlState string) {
	var g, q strings.Builder
	for _, dir := range []string{"cmd", "internal", "migrations"} {
		_ = filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil //nolint:nilerr // unreadable entries are skipped, not fatal
			}
			var b *strings.Builder
			switch {
			case strings.HasSuffix(path, ".go"):
				b = &g
			case strings.HasSuffix(path, ".sql"):
				b = &q
			default:
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return nil //nolint:nilerr // a file removed mid-walk is skipped
			}
			fmt.Fprintf(b, "%s|%d|%d\n", path, info.Size(), info.ModTime().UnixNano())
			return nil
		})
	}
	return g.String(), q.String()
}

// pickDevPorts returns one free port per dev app, starting at its preferred
// port and never handing the same port to two apps.
func pickDevPorts(portFree func(int) bool) ([]int, error) {
	used := map[int]bool{}
	ports := make([]int, len(devApps))
	for i, app := range devApps {
		found := false
		for p := app.port; p < app.port+devPortSearch; p++ {
			if !used[p] && portFree(p) {
				ports[i], used[p], found = p, true, true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("no free port for %s in %d-%d", app.name, app.port, app.port+devPortSearch-1)
		}
	}
	return ports, nil
}

// tcpPortFree reports whether 127.0.0.1:port can be listened on right now.
func tcpPortFree(port int) bool {
	l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(port))
	if err != nil {
		return false
	}
	return l.Close() == nil
}

// prefixWriter writes complete lines to w with a prefix, holding a partial
// line until its newline (or flush) so output from several processes does
// not interleave mid-line.
type prefixWriter struct {
	w      io.Writer
	prefix string
	mu     *sync.Mutex
	buf    []byte
}

func (p *prefixWriter) Write(b []byte) (int, error) {
	p.buf = append(p.buf, b...)
	for {
		i := bytes.IndexByte(p.buf, '\n')
		if i < 0 {
			return len(b), nil
		}
		if err := p.emit(p.buf[:i+1]); err != nil {
			return len(b), err
		}
		p.buf = p.buf[i+1:]
	}
}

func (p *prefixWriter) flush() {
	if len(p.buf) > 0 {
		_ = p.emit(append(p.buf, '\n'))
		p.buf = nil
	}
}

func (p *prefixWriter) emit(line []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, err := io.WriteString(p.w, p.prefix+string(line))
	if err != nil && !errors.Is(err, io.ErrClosedPipe) {
		return err
	}
	return nil
}
