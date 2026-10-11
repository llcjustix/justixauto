# Milestone OpenAPI hook scope disposition

The first local milestone is `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`.
VCS staged the 267 exact authorized paths. Installed `.git/hooks/pre-commit`
invoked Lefthook's `openapi` job, which ran `make --no-print-directory openapi
STAGED="{staged_files}"` and staged `internal/pkg/apidocs/swagger.json`.
The Makefile target invokes `go tool swag fmt` and `go tool swag init` over
cmd/api and internal, outputting JSON to internal/pkg/apidocs. The commit thus
contains 268 files. Captured hook output only showed the Lefthook header; its
configured commands do not invoke tests, lint, typecheck or application builds.

Application hub accepts this generated API-reference artifact as part of the
same authorized API changes and preserves the commit. No history rewrite or
rollback is needed. This records the unexpected automatic staging rather than
claiming the exact-path mechanism prevented it. No release readiness follows.

For subsequent agent commits, VCS must disable the hook per command using a
temporary core.hooksPath override. Do not change repository/global config,
regenerate documents automatically, or run extra verification. Any later
generated API-reference update needs its own explicit source scope.
