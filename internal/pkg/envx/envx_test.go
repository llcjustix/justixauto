package envx

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOr(t *testing.T) {
	t.Setenv("ENVX_SET", "value")
	t.Setenv("ENVX_EMPTY", "")
	for key, want := range map[string]string{"ENVX_SET": "value", "ENVX_EMPTY": "fallback", "ENVX_UNSET": "fallback"} {
		if got := Or(key, "fallback"); got != want {
			t.Errorf("Or(%q) = %q, want %q", key, got, want)
		}
	}
}

func TestLoadFileFillsOnlyUnsetVariables(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	body := "# comment\nENVX_FILE_NEW=from-file\nENVX_FILE_SET=from-file\nENVX_FILE_EMPTY=from-file\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENVX_FILE_SET", "from-env")
	t.Setenv("ENVX_FILE_EMPTY", "") // set but empty still wins, e.g. WEB_DIR= from make dev
	t.Setenv("ENVX_FILE_NEW", "")
	os.Unsetenv("ENVX_FILE_NEW") // t.Setenv above restores the original state afterwards

	if err := LoadFile(path); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{"ENVX_FILE_NEW": "from-file", "ENVX_FILE_SET": "from-env", "ENVX_FILE_EMPTY": ""} {
		if got := os.Getenv(key); got != want {
			t.Errorf("%s = %q, want %q", key, got, want)
		}
	}
}

func TestLoadFileIgnoresAMissingFile(t *testing.T) {
	if err := LoadFile(filepath.Join(t.TempDir(), ".env")); err != nil {
		t.Fatalf("err = %v, want nil for a missing file", err)
	}
}

func TestLoadFileReportsAMalformedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("NOT A VALID LINE\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LoadFile(path); err == nil {
		t.Fatal("want an error for a malformed file")
	}
}
