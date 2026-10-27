// audit-verify consumes ordered identity audit export pages offline.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"justixauto/internal/pkg/auditverify"
)

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("audit-verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	checkpointPath := flags.String("checkpoint", "", "independently retained checkpoint JSON file")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: audit-verify [--checkpoint FILE] [PAGE.json|PAGES.jsonl|- ...]")
		fmt.Fprintln(stderr, "No files means stdin. Supply every page in export order; '-' reads stdin once.")
		fmt.Fprintln(stderr, "Output is a JSON summary with checkpoint. A local checkpoint is not external anchoring.")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return fail(stdout, err, 2)
	}
	paths := flags.Args()
	if len(paths) == 0 {
		paths = []string{"-"}
	}
	if len(paths) > 1024 {
		return fail(stdout, fmt.Errorf("%w: input file limit exceeded (1024)", auditverify.ErrLimit), 2)
	}
	var cp *auditverify.Checkpoint
	if *checkpointPath != "" {
		f, err := os.Open(filepath.Clean(*checkpointPath))
		if err != nil {
			return fail(stdout, fmt.Errorf("open checkpoint: %w", err), 2)
		}
		cp, err = auditverify.ReadCheckpoint(f)
		closeErr := f.Close()
		if err != nil {
			return fail(stdout, fmt.Errorf("read checkpoint: %w", err), 2)
		}
		if closeErr != nil {
			return fail(stdout, closeErr, 2)
		}
	}
	v, err := auditverify.New(cp)
	if err != nil {
		return fail(stdout, err, 2)
	}
	stdinRead := false
	for _, path := range paths {
		if path == "-" {
			if stdinRead {
				return fail(stdout, fmt.Errorf("stdin can only be consumed once"), 2)
			}
			stdinRead = true
			err = v.Consume(stdin)
		} else {
			f, openErr := os.Open(filepath.Clean(path))
			if openErr != nil {
				return fail(stdout, fmt.Errorf("open export: %w", openErr), 2)
			}
			err = v.Consume(f)
			closeErr := f.Close()
			if err == nil {
				err = closeErr
			}
		}
		if err != nil {
			return fail(stdout, err, 1)
		}
	}
	result, err := v.Finish()
	if err != nil {
		return fail(stdout, err, 1)
	}
	if err := json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, "write verification result:", err)
		return 2
	}
	return 0
}

func fail(w io.Writer, err error, code int) int {
	status := "invalid"
	if errors.Is(err, auditverify.ErrLimit) {
		status = "resource_limit_exceeded"
	}
	_ = json.NewEncoder(w).Encode(struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}{Status: status, Error: err.Error()})
	return code
}
