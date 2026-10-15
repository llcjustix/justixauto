# Dev administrator bootstrap

The dev deployment workflow runs a one-shot administrator bootstrap after
migrations and before replacing the API. It uses the same immutable image and
database network as the deployment, but the isolated `bootstrap-admin` Compose
profile receives only `DATABASE_URL`.

Configure these repository GitHub Actions secrets for the first deployment
(`Settings → Secrets and variables → Actions`):

- `JUSTIX_AUTO_BOOTSTRAP_ADMIN_LOGIN`
- `JUSTIX_AUTO_BOOTSTRAP_ADMIN_EMAIL`
- `JUSTIX_AUTO_BOOTSTRAP_ADMIN_NAME` (optional; defaults to `Platform Administrator`)
- `JUSTIX_AUTO_BOOTSTRAP_ADMIN_PASSWORD`

[GitHub's secret setup instructions](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-secrets)
describe the settings. The current job does not select a GitHub Environment,
so environment-only secrets are not available to it.
The password is sent to the bootstrap command through standard input; do not
put it in this directory's `.env`, Docker arguments, or API configuration.
Use a single-line password of 8–128 characters. Login, email and password have
no defaults and must pass the existing identity validation for initial creation.
If an active admin already exists, the secrets may be absent. Updating the
secrets after creation does not update the stored credentials.

On its first run, the command creates one active user with the existing
`platform_admin` role. Later deployments succeed without consuming the
credential fields when an active platform administrator already exists. A
pending or suspended administrator does not satisfy that predicate. The
bootstrap command never resets credentials, changes roles, reactivates an
account, or promotes an account that collides with the requested login or
email; those failures stop deployment for authorized operator resolution.

For a manual run from the repository root, use Bash and the existing dev `.env`
with `IMAGE_TAG` and `DATABASE_URL`. Migrations must already have succeeded:

```bash
set +x
set -o pipefail
read -r -p 'Admin login: ' bootstrap_login
read -r -p 'Admin email: ' bootstrap_email
read -r -s -p 'Admin password: ' bootstrap_password
printf '\n'
printf '%s' "$bootstrap_password" | docker compose -f deploy/dev/compose.yml \
  run --rm -T --no-deps bootstrap-admin -if-missing \
  -login "$bootstrap_login" -email "$bootstrap_email" -name 'Platform Administrator'
bootstrap_status=$?
unset bootstrap_password
test "$bootstrap_status" -eq 0
```

[Docker's one-off run reference](https://docs.docker.com/reference/cli/docker/compose/run/)
describes the container invocation. The command without `-if-missing` retains
strict single-use behavior. A bootstrap error stops CI before API replacement;
correct the initial configuration or resolve the collision, then rerun CI.
Use account management for password changes. This change targets the test/dev
server; `deploy/test/compose.yaml` is only a disposable local test fixture.
