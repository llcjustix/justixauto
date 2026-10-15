Status: DONE

Changed paths:

- `cmd/bootstrap-admin/main.go`: adds opt-in `-if-missing` preflight before stdin consumption and accepts only the typed `already_bootstrapped` race outcome as a successful skip; strict mode remains unchanged.
- `internal/modules/identity/service/user.go`: adds the read-only active platform-admin predicate used by the command preflight.
- `deploy/Dockerfile`, `deploy/dev/compose.yml`, and `.github/workflows/deploy-dev.yml`: package the command and run its profile-isolated, exact-image service after migrations and before API deployment, with credentials scoped to the bootstrap step and the password sent on stdin.
- `deploy/dev/README.md`: documents dev secret setup, reruns, failures, and manual operation without credentials.

Exclusions: no tests or local build/checks were run; no Git operation, credential access, live Docker/CI operation, canonical contract edit, or unowned source change was performed.
