# Final bootstrap validation

Date: 2026-10-15. Actor: `/root`. Status: PASS with the execution limits below.
Branch: `infra/admin-bootstrap-20261015`; dependency
`f91daa8e624da44f242a53c6677f127dcf0c87d8`. Completed source is pinned in
[source-manifest.md](source-manifest.md); version control attaches the local
commit after this report. No earlier unrelated test evidence is reused.

All requested runtime/source changes were completed before verification.
Root added one focused PostgreSQL integration regression in
`cmd/bootstrap-admin/main_test.go` to exercise the actual CLI as a child process,
its flags/stdin/exit status and the existing sign-in service. This is the only
source-path expansion beyond the six-file implementation packet. Children run
in an empty temporary directory with synthetic database configuration and do
not load the developer's dotenv. No broad test/build/lint/typecheck/vet/race or
independent review/QA chain ran.

## Behavior

The fixture used the repository-pinned PostgreSQL image
`docker.io/library/postgres:18.6-alpine3.24@sha256:d3e1620b530c944afa6e887d22eb899824da68e19c52024bf98f5220c88a65b2`,
an ephemeral loopback port and a new disposable database. Both containers were
removed by the runner's finally block. No existing database or app was changed.

Initial command, with synthetic `TEST_DATABASE_URL` and a temporary `GOCACHE`:

```sh
bash tools/go.sh test -count=1 -vet=off ./cmd/bootstrap-admin -run '^TestBootstrapDeployment$' -v
```

[Initial raw output](initial-behavior-check.txt) records four passes: missing
password, invalid password, missing identity fields, and database errors.
Three scenarios failed because the new test cleanup used `TRUNCATE ... CASCADE`:
the verification-actor FK reached companies and then removed the built-in roles.
That was a fixture error. Cleanup now deletes only fixture users/audit rows,
preserving the migration-seeded role. No application repair was needed.

Only the three failed scenarios were rerun in a fresh fixture:

```sh
bash tools/go.sh test -count=1 -vet=off ./cmd/bootstrap-admin -run '^TestBootstrapDeployment$/(create_login_skip_and_strict_repeat|collision_does_not_promote_or_reactivate|concurrent_seeds_create_once)$' -v
```

[Affected raw output](affected-behavior-check.txt): all three passed, 3.772s.
Together the seven cases establish:

- First creation stores one active platform admin and one bootstrap audit;
  the exact supplied password authenticates through the existing service.
- A credential-free rerun returns while stdin stays open without bytes.
  Account/password hash, role links, sessions and audit snapshots stay identical.
  Strict default mode still rejects another bootstrap.
- Missing/invalid initial input causes no writes; database errors fail.
- An existing login/email is not promoted or reactivated. A suspended admin
  does not count as active; a different valid admin can be created.
- Two processes are held at the existing admin-role lock until both reach it.
  Both then succeed, exactly one creates and one skips, with one admin/audit.
  This exercises the typed concurrent `already_bootstrapped` path.

Only formatting of the new test file changed after the successful rerun.
Application and deployment source stayed identical throughout the checks.

## Deployment wiring

The final Compose file was copied byte-for-byte into a temporary directory and
validated with synthetic dotenv values using `docker compose config --quiet`.
Its parsed configuration confirmed the same image/database/network, only
`DATABASE_URL` in the bootstrap container, explicit profile isolation, no ports
or restart, and stdin without a TTY. Default service selection excludes it.

The workflow was parsed with Ruby YAML. Assertions confirmed migration before
bootstrap before API, default failure gating, exact-image pull including the
new service, and secrets scoped only to the bootstrap step. The actual step's
Bash script ran against a recording Docker stub: literal dollar/backtick/quote/
backslash password bytes arrived unchanged on stdin, the arguments and optional
name default matched, missing secrets reached the command, and a stub exit 42
propagated. Dockerfile build/copy/entrypoint paths match the Compose invocation.

Both wiring checks passed before the behavior test. Local scratch evidence:
`/private/tmp/justixauto-admin-bootstrap-60dspzvl` and
`/private/tmp/justixauto-admin-bootstrap-retry-1sr31s8q`.

## Limits

No complete Docker image was built or launched; image packaging was inspected.
No GitHub workflow, remote server, real credential provisioning or deployment
was executed. The initial database on the test/dev server still requires the
named repository secrets when no active platform admin exists. No production
release/account recovery claim is made. Existing suites and unread
`test_creds.txt` are preserved. Source and disposable-fixture leases are released.
