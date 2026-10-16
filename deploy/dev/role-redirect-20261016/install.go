// Static-only installer for the pinned legacy frontend.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"syscall"
	"time"
)

const sourceSHA = "3cc124c5a5858a5eaa2991edafc2771ee7232966"
const recoveryRoot = "/app/.justixauto-hotfix/role-redirect-20261016"

var operationSHA, patchSHA string
var names = []string{"realization", "financing", "insurance", "admin"}

type Manifest struct {
	Operation string            `json:"operation"`
	Source    string            `json:"source"`
	Patch     string            `json:"patch"`
	Files     map[string]string `json:"files"`
	Binaries  map[string]string `json:"binaries"`
}
type Entry struct {
	Before string `json:"before"`
	After  string `json:"after"`
	Mode   uint32 `json:"mode"`
}
type Receipt struct {
	Manifest Manifest         `json:"manifest"`
	State    string           `json:"state"`
	Updated  string           `json:"updated"`
	Entries  map[string]Entry `json:"entries"`
}
type Owner struct {
	PID       int
	Operation string
}

func stage() string           { return "/tmp/justixauto-role-redirect-" + operationSHA }
func backup() string          { return recoveryRoot + "/" + operationSHA }
func live(name string) string { return "/app/web/apps/" + name + "/dist" }
func safe(path string) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("unclean path: %s", path)
	}
	current := "/"
	for _, part := range strings.Split(strings.TrimPrefix(path, "/"), "/") {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink: %s", current)
		}
		if current != path && !info.IsDir() {
			return fmt.Errorf("non-directory: %s", current)
		}
	}
	return nil
}
func sum(path string) (string, error) {
	if err := safe(path); err != nil {
		return "", err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not regular: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func expect(path, hash string) error {
	got, err := sum(path)
	if err != nil {
		return err
	}
	if got != hash {
		return fmt.Errorf("hash drift: %s", path)
	}
	return nil
}
func mkdir(path string, mode os.FileMode) error {
	if err := safe(path); err != nil {
		return err
	}
	return os.MkdirAll(path, mode)
}
func syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
func copyNew(from, to string, mode os.FileMode) error {
	if _, err := sum(from); err != nil {
		return err
	}
	if err := safe(to); err != nil {
		return err
	}
	input, err := os.Open(from)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	_, err = io.Copy(output, input)
	if err == nil {
		err = output.Sync()
	}
	closeErr := output.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return syncDir(filepath.Dir(to))
}
func atomicBytes(path string, data []byte, mode os.FileMode) error {
	if err := safe(path); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".role-redirect-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(mode); err == nil {
		_, err = f.Write(data)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(tmp, path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}
func replace(from, to, wanted string, mode os.FileMode) error {
	if err := expect(from, wanted); err != nil {
		return err
	}
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}
	return atomicBytes(to, data, mode)
}
func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return atomicBytes(path, append(data, '\n'), 0600)
}
func readJSON(path string, value any) error {
	if _, err := sum(path); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, value)
}
func save(r *Receipt) error {
	r.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	return writeJSON(backup()+"/receipt.json", r)
}
func valid(m Manifest) error {
	if m.Operation != operationSHA || m.Source != sourceSHA || m.Patch != patchSHA {
		return errors.New("provenance mismatch")
	}
	hashRE := regexp.MustCompile("^[a-f0-9]{64}$")
	for _, name := range names {
		for _, key := range []string{name + "/index.html", "baseline/" + name + "/index.html"} {
			if !hashRE.MatchString(m.Files[key]) {
				return errors.New("missing entry hash")
			}
		}
	}
	if len(m.Binaries) != 2 || !hashRE.MatchString(m.Binaries["api"]) || !hashRE.MatchString(m.Binaries["migrate"]) {
		return errors.New("missing binary hashes")
	}
	for path, hash := range m.Files {
		if filepath.Clean(path) != path || strings.HasPrefix(path, "/") || strings.Contains(path, "..") || !hashRE.MatchString(hash) {
			return errors.New("unsafe manifest path")
		}
		allowed := false
		for _, name := range names {
			if strings.HasPrefix(path, name+"/") || path == "baseline/"+name+"/index.html" {
				allowed = true
			}
		}
		if !allowed {
			return fmt.Errorf("unexpected manifest file: %s", path)
		}
	}
	return nil
}
func binaries(m Manifest) error {
	for _, name := range []string{"api", "migrate"} {
		if err := expect("/app/"+name, m.Binaries[name]); err != nil {
			return err
		}
	}
	return nil
}
func sortedFiles(m Manifest) []string {
	keys := make([]string, 0, len(m.Files))
	for key := range m.Files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func checkPayload(m Manifest) error {
	if err := valid(m); err != nil {
		return err
	}
	seen := make(map[string]bool)
	for _, top := range append(append([]string{}, names...), "baseline") {
		err := filepath.Walk(stage()+"/"+top, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if err = safe(path); err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			rel, err := filepath.Rel(stage(), path)
			if err != nil {
				return err
			}
			hash, ok := m.Files[rel]
			if !ok {
				return fmt.Errorf("unlisted payload: %s", rel)
			}
			seen[rel] = true
			return expect(path, hash)
		})
		if err != nil {
			return err
		}
	}
	if len(seen) != len(m.Files) {
		return errors.New("missing payload files")
	}
	return nil
}
func assetTarget(path string) (string, bool) {
	for _, name := range names {
		if strings.HasPrefix(path, name+"/") && path != name+"/index.html" {
			return live(name) + "/" + strings.TrimPrefix(path, name+"/"), true
		}
	}
	return "", false
}
func checkAssets(m Manifest, require bool) error {
	for _, path := range sortedFiles(m) {
		dst, ok := assetTarget(path)
		if !ok {
			continue
		}
		if err := safe(dst); err != nil {
			return err
		}
		_, err := os.Lstat(dst)
		if os.IsNotExist(err) && !require {
			continue
		}
		if err != nil {
			return err
		}
		if err = expect(dst, m.Files[path]); err != nil {
			return err
		}
	}
	return nil
}
func acquire(mode string) (func() error, error) {
	if err := mkdir(recoveryRoot, 0700); err != nil {
		return nil, err
	}
	lock := recoveryRoot + "/lock"
	if err := safe(lock); err != nil {
		return nil, err
	}
	err := os.Mkdir(lock, 0700)
	if os.IsExist(err) && mode == "rollback" {
		var owner Owner
		if e := readJSON(lock+"/owner.json", &owner); e != nil {
			return nil, errors.New("unreadable lock; manual recovery required")
		}
		if owner.Operation != operationSHA || owner.PID <= 0 {
			return nil, errors.New("foreign lock")
		}
		if e := syscall.Kill(owner.PID, 0); e != syscall.ESRCH {
			return nil, errors.New("lock owner may still run")
		}
		if e := os.Remove(lock + "/owner.json"); e != nil {
			return nil, e
		}
		if e := os.Remove(lock); e != nil {
			return nil, e
		}
		err = os.Mkdir(lock, 0700)
	}
	if err != nil {
		return nil, fmt.Errorf("operation locked: %w", err)
	}
	if err = writeJSON(lock+"/owner.json", Owner{os.Getpid(), operationSHA}); err != nil {
		return nil, err
	}
	return func() error {
		if err := os.Remove(lock + "/owner.json"); err != nil {
			return err
		}
		return os.Remove(lock)
	}, nil
}
func loadReceipt() (Receipt, error) {
	var r Receipt
	err := readJSON(backup()+"/receipt.json", &r)
	if err != nil {
		return r, err
	}
	if err = valid(r.Manifest); err != nil {
		return r, err
	}
	if len(r.Entries) != len(names) {
		return r, errors.New("incomplete receipt")
	}
	for _, name := range names {
		e, ok := r.Entries[name]
		if !ok || e.Before != r.Manifest.Files["baseline/"+name+"/index.html"] || e.After != r.Manifest.Files[name+"/index.html"] {
			return r, errors.New("entry receipt mismatch")
		}
		if err = expect(backup()+"/"+name+".html", e.Before); err != nil {
			return r, err
		}
	}
	return r, nil
}
func rollback(r *Receipt) error {
	if err := binaries(r.Manifest); err != nil {
		return err
	}
	// Inspect every current entry before replacing any entry.
	for _, name := range names {
		got, err := sum(live(name) + "/index.html")
		if err != nil {
			return err
		}
		e := r.Entries[name]
		if got != e.Before && got != e.After {
			r.State = "drifted"
			return errors.Join(fmt.Errorf("unknown entry content: %s", name), save(r))
		}
		if err = expect(backup()+"/"+name+".html", e.Before); err != nil {
			return err
		}
	}
	r.State = "rolling-back"
	if err := save(r); err != nil {
		return err
	}
	for _, name := range names {
		e := r.Entries[name]
		if err := replace(backup()+"/"+name+".html", live(name)+"/index.html", e.Before, os.FileMode(e.Mode)); err != nil {
			return err
		}
	}
	r.State = "rolled-back"
	return save(r)
}
func status(r *Receipt) error {
	if err := binaries(r.Manifest); err != nil {
		return err
	}
	if r.State != "applied" && r.State != "rolled-back" {
		return fmt.Errorf("incomplete state: %s", r.State)
	}
	for _, name := range names {
		e := r.Entries[name]
		want := e.Before
		if r.State == "applied" {
			want = e.After
		}
		if err := expect(live(name)+"/index.html", want); err != nil {
			return err
		}
	}
	if r.State == "applied" {
		if err := checkAssets(r.Manifest, true); err != nil {
			return err
		}
	}
	return nil
}
func apply() (err error) {
	if _, e := os.Lstat(backup()); e == nil {
		r, e := loadReceipt()
		if e != nil {
			return fmt.Errorf("incomplete operation; explicit recovery required: %w", e)
		}
		if r.State != "applied" {
			return fmt.Errorf("operation exists in state %s", r.State)
		}
		return status(&r)
	} else if !os.IsNotExist(e) {
		return e
	}
	var m Manifest
	if err = readJSON(stage()+"/manifest.json", &m); err != nil {
		return err
	}
	if err = checkPayload(m); err != nil {
		return err
	}
	if err = binaries(m); err != nil {
		return err
	}
	if err = checkAssets(m, false); err != nil {
		return err
	}
	r := Receipt{Manifest: m, State: "prepared", Entries: map[string]Entry{}}
	for _, name := range names {
		path := live(name) + "/index.html"
		if err = expect(path, m.Files["baseline/"+name+"/index.html"]); err != nil {
			return err
		}
		info, e := os.Lstat(path)
		if e != nil {
			return e
		}
		r.Entries[name] = Entry{m.Files["baseline/"+name+"/index.html"], m.Files[name+"/index.html"], uint32(info.Mode().Perm())}
	}
	if err = mkdir(backup(), 0700); err != nil {
		return err
	}
	if err = copyNew(stage()+"/installer", backup()+"/installer", 0700); err != nil {
		return err
	}
	if err = writeJSON(backup()+"/manifest.json", m); err != nil {
		return err
	}
	for _, name := range names {
		if err = copyNew(live(name)+"/index.html", backup()+"/"+name+".html", os.FileMode(r.Entries[name].Mode)); err != nil {
			return err
		}
		if err = expect(backup()+"/"+name+".html", r.Entries[name].Before); err != nil {
			return err
		}
	}
	if err = save(&r); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Join(err, rollback(&r))
		}
	}()
	for _, path := range sortedFiles(m) {
		dst, ok := assetTarget(path)
		if !ok {
			continue
		}
		if err = mkdir(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		if _, e := os.Lstat(dst); e == nil {
			if err = expect(dst, m.Files[path]); err != nil {
				return err
			}
		} else if os.IsNotExist(e) {
			if err = copyNew(stage()+"/"+path, dst, 0644); err != nil {
				return err
			}
		} else {
			return e
		}
	}
	if err = checkAssets(m, true); err != nil {
		return err
	}
	r.State = "applying"
	if err = save(&r); err != nil {
		return err
	}
	for _, name := range names {
		if err = expect(live(name)+"/index.html", r.Entries[name].Before); err != nil {
			return err
		}
		if err = replace(stage()+"/"+name+"/index.html", live(name)+"/index.html", r.Entries[name].After, 0644); err != nil {
			return err
		}
	}
	r.State = "applied"
	if err = save(&r); err != nil {
		return err
	}
	return status(&r)
}
func buildManifest() error {
	m := Manifest{Operation: operationSHA, Source: sourceSHA, Patch: patchSHA, Files: map[string]string{}, Binaries: map[string]string{}}
	for _, name := range []string{"api", "migrate"} {
		raw, err := os.ReadFile("/out/payload/baseline/" + name + ".sha256")
		if err != nil {
			return err
		}
		fields := strings.Fields(string(raw))
		if len(fields) == 0 {
			return errors.New("empty binary digest")
		}
		m.Binaries[name] = fields[0]
		if err = os.Remove("/out/payload/baseline/" + name + ".sha256"); err != nil {
			return err
		}
	}
	for _, top := range append(append([]string{}, names...), "baseline") {
		err := filepath.Walk("/out/payload/"+top, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			hash, err := sum(path)
			if err != nil {
				return err
			}
			rel, err := filepath.Rel("/out/payload", path)
			if err != nil {
				return err
			}
			m.Files[rel] = hash
			return nil
		})
		if err != nil {
			return err
		}
	}
	if err := valid(m); err != nil {
		return err
	}
	return writeJSON("/out/payload/manifest.json", m)
}
func run() (err error) {
	if !regexp.MustCompile("^[a-f0-9]{40}$").MatchString(operationSHA) || !regexp.MustCompile("^[a-f0-9]{64}$").MatchString(patchSHA) {
		return errors.New("invalid compiled provenance")
	}
	if len(os.Args) != 2 {
		return errors.New("usage: installer apply|status|rollback|cleanup-stage")
	}
	mode := os.Args[1]
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if mode == "manifest" && exe == "/out/installer" {
		return buildManifest()
	}
	if exe != stage()+"/installer" && exe != backup()+"/installer" {
		return errors.New("unexpected executable path")
	}
	if err = safe(exe); err != nil {
		return err
	}
	if mode != "apply" && mode != "status" && mode != "rollback" && mode != "cleanup-stage" {
		return errors.New("invalid mode")
	}
	unlock, err := acquire(mode)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, unlock()) }()
	if mode == "cleanup-stage" {
		if exe != backup()+"/installer" {
			return errors.New("cleanup requires retained helper")
		}
		r, e := loadReceipt()
		if e != nil {
			return e
		}
		if e = status(&r); e != nil {
			return e
		}
		if e = safe(stage()); e != nil {
			return e
		}
		return os.RemoveAll(stage())
	}
	if mode == "apply" {
		err = apply()
	} else {
		var r Receipt
		r, err = loadReceipt()
		if err == nil {
			if mode == "rollback" {
				err = rollback(&r)
			} else {
				err = status(&r)
			}
		}
	}
	if err != nil {
		return err
	}
	data, err := os.ReadFile(backup() + "/receipt.json")
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(data)
	return err
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
