# ADR-16 — Conditional administrator bootstrap in dev deployment

Date: 2026-10-15. Status: accepted for the user-requested test/dev Docker
deployment change by application hub `/root` and infrastructure authority
`/root/admin_seed_devops`. [Infrastructure decision](dev/execution/admin-bootstrap-20261015/devops-decision.md).

The existing `dev` workflow uses `deploy/dev/compose.yml`. Its image must include
the existing `cmd/bootstrap-admin` alongside the API and migration commands.
After migrations and before API replacement, CI explicitly runs a one-shot
`bootstrap-admin` service from that same image, under the `bootstrap` profile.
The service uses the existing database/network, exposes no port and receives
only `DATABASE_URL` in its container environment. No new dependency is needed.

The command's opt-in `-if-missing` mode returns success without changing data
when an active platform administrator exists, before consuming credentials.
The existing identity service owns that predicate and the locked transaction
that creates the user, password hash, platform role and audit event. Only its
typed `already_bootstrapped` conflict is a successful concurrent skip. Other
errors stop deployment. The default manual command remains strict.

CI receives `JUSTIX_AUTO_BOOTSTRAP_ADMIN_LOGIN`, `_EMAIL`, `_NAME` and `_PASSWORD`
through step-scoped GitHub Actions secret bindings. Login, email and password
have no defaults; display name may default to `Platform Administrator`.
The password is passed on stdin, never through command flags, image layers,
shared dotenv, the API environment or log output. Initial creation requires
valid credentials; reruns with an existing active admin do not require secrets.

This creates a missing active administrator; it does not reset passwords,
change roles, reactivate accounts or resolve credential collisions. Changing
the secrets after creation does not rotate credentials. Pending/suspended users
do not satisfy the existing active-admin predicate, and colliding credentials
remain errors. Bootstrap stays a deployment command with no public HTTP route.

Scope is the existing test/dev Compose delivery route. `deploy/test` contains
disposable test fixtures. Production, Kubernetes rollout, account recovery,
actual secret provisioning and live deployment are outside this source change.
