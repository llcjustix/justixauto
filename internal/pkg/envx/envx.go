// Package envx reads environment variables with defaults.
package envx

import (
	"errors"
	"io/fs"
	"os"

	"github.com/joho/godotenv"
)

// Or returns the value of key, or fallback when it is unset or empty.
func Or(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// LoadFile copies KEY=value lines from a dotenv file (local development's
// .env) into the process environment. Variables that are already set win, so
// the real environment (Kubernetes, CI, a shell or make dev) overrides the
// file. A missing file is not an error: deployed images have none.
func LoadFile(path string) error {
	if err := godotenv.Load(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
