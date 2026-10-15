# Test/dev administrator bootstrap — infrastructure decision

Date: 2026-10-15. Actor: `/root/admin_seed_devops`
(`devops_orchestrator`, gpt-6-astra/xhigh).
Observed source HEAD: `f91daa8e624da44f242a53c6677f127dcf0c87d8`.
Authority: user-requested administrator seeding during test-server Docker CI/CD;
application hub `/root` accepted the shared contract below on 2026-10-15.
This assigned execution artifact records the decision; the primary owns
promotion into `docs/justix-auto/adr-16-dev-admin-bootstrap.md` and other
canonical records.

## Findings

- `.github/workflows/deploy-dev.yml:4` triggers on `dev`; lines 18–19 select
  `deploy/dev/compose.yml`; lines 26–28 target the existing self-hosted runner.
  This is the tracked test/dev server delivery route. Lines 76–93 build and
  push `poirot92/justix-auto:<github.sha>`, lines 145–147 run migrations, and
  lines 154–159 replace the API. There is no bootstrap step.
- `deploy/dev/compose.yml:7` fixes project `justixauto`. The API uses port
  `2047:8080` (line 18). API and migration services share the image and external
  `justix-db` network (lines 10, 24, 38–40). Both currently consume the shared
  `.env` (lines 14–15 and 26–27).
- `deploy/test/compose.yaml:1` identifies disposable PostgreSQL/MinIO fixtures
  for `make test-go`. It is not the test-server deployment and needs no change.
  `.github/workflows/ci.yml:79` runs those existing CI fixtures; CI/release
  checks are separate from this task's final-only development verification.
- `deploy/Dockerfile:1` mentions bootstrap, but lines 21–22 compile only API
  and migrate. The distroless runtime copies `/out/` at line 26 and executes
  `/app/api` at line 31. The existing bootstrap binary is missing from it.
- `cmd/bootstrap-admin/main.go:43` accepts login, email and name flags, reads
  password only from stdin (line 48), then loads config and calls the identity
  service (line 64). Any error exits nonzero (lines 31–39), so repeats currently
  fail. `README.md:102` documents this strict manual bootstrap behavior.
- `internal/modules/identity/service/user.go:480` defines bootstrap in terms
  of an **active platform administrator**. Inside a transaction it locks the
  admin guard, counts active admins and returns `already_bootstrapped`
  (lines 499–508), then creates the password-hashed user, platform role and
  audit event (lines 517–523). Validation and hashing presently precede the
  guard. `internal/modules/identity/repository/role.go:164` implements the
  role-row lock; line 175 implements the active-admin count.
- `internal/config/config.go:41` documents `DATABASE_URL` as the only required
  configuration; remaining settings have defaults. A bootstrap service needs
  only that database configuration, without the API's complete environment.
- Approved ADR-14 defines the modular monolith and explicit migrations
  (`docs/justix-auto/adr-14-classic-modular-monolith.md:15`). Kubernetes remains
  the selected target, but this request concerns the existing Compose route.
  `deploy/values-dev.yaml:1` still requires explicit rollout configuration.
  `deploy/README.md:42` links an absent historical Kubernetes ADR; that link
  does not authorize a new deployment mechanism.

## Accepted shared contract

1. Build `cmd/bootstrap-admin` into the existing image alongside API/migrate.
   Keep its ordinary strict behavior; add explicit `-if-missing` automation
   behavior. The application hub owns its service implementation and the
   definition of an existing administrator.
2. `-if-missing` succeeds without mutation when any active platform admin
   exists, before requiring or validating supplied bootstrap credentials.
   Preserve the existing admin-guard transaction protection against concurrent
   creation. No password reset, re-enabling, role rewriting or default-account
   replacement occurs on repeat deployments. Credential conflicts, invalid
   initial input and database failures remain errors.
3. Add a dedicated one-shot `bootstrap-admin` Compose service under profile
   `bootstrap`: same `${IMAGE_TAG}`, entrypoint `/app/bootstrap-admin`, existing
   `justix-db` network, and `DATABASE_URL` only. It needs no exposed ports,
   restart policy or new infrastructure dependency. Profile exclusion prevents
   ordinary unqualified startup from seeding implicitly.
4. Run it explicitly after successful migrations and before API replacement:
   password stream into `docker compose run --rm -T --no-deps bootstrap-admin
   -if-missing`, with quoted login/email/name flags. The workflow step must
   propagate pipeline and command failure. The bootstrap transaction is the
   concurrency authority; deployment-level concurrency is not the only guard.
5. Obtain values through GitHub step environment bindings named
   `JUSTIX_AUTO_BOOTSTRAP_ADMIN_LOGIN`, `JUSTIX_AUTO_BOOTSTRAP_ADMIN_EMAIL`,
   `JUSTIX_AUTO_BOOTSTRAP_ADMIN_NAME`, and
   `JUSTIX_AUTO_BOOTSTRAP_ADMIN_PASSWORD`. Reference corresponding GitHub
   secrets rather than interpolating credential expressions into shell code.
   Name may default to `Platform Administrator`; login/email/password have
   no credential defaults and are mandatory only for initial creation.
6. Pass password with `printf '%s'` over stdin. Never place it in Docker build
   arguments/layers, command flags, the shared `deploy/dev/.env`, API container
   environment, Compose interpolation, log output or source-controlled files.
   Do not dump rendered Compose configuration; preserve quiet validation.
7. A successful create or existing-admin skip permits API deployment; any
   other bootstrap failure stops that deployment before API replacement.
   Changing configured secrets after successful creation does not rotate the
   stored administrator credentials.

## Scope, readiness and handoff

Source changes are ready for the primary's bounded implementation. No missing
input blocks coding. Runtime provisioning of the named GitHub secrets remains
required for a database with no active admin; their existence and values were
not inspected. The normal existing migration and database settings also remain
runtime prerequisites. No new schema, service, registry or Kubernetes component
is required.

This decision authorizes source preparation within the requested test/dev flow,
not live operations, Git promotion or production activation. No environment
secrets, private `.env`, keys or `test_creds.txt` were read. No tests, builds,
deployments, Git mutations or child dispatches were performed. The only write
is this assigned report. The primary will select the smallest useful final
behavior check after the complete requested change; no automatic review/QA
chain is requested.

Read lease released upon delivery of this report. The primary may schedule
version-control branch operations and the sole implementation writer.
