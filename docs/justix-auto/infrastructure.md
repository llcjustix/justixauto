# Infrastructure

Repository configuration, consolidated 2026-10-03. This describes the setup;
it does not certify a running environment or authorize deployment.

## Local development

Go is pinned by `tools/go.sh`; Node/npm by `web/package.json`. Optional downloaded
toolchains live in the ignored `.local/toolchains/` directory, outside docs.
Configuration comes from ignored `.env` (start with `.env.example`); deployed
environment variables take precedence. Never commit credentials.

`deploy/local/compose.yaml` provides PostgreSQL with a persistent named volume.
The API and four Vite apps run on the laptop. No local Kubernetes is required.

```sh
make env                     # create local configuration; review it
make dev                     # database, migrations, Air API and four Vite apps
make stop                    # stop this checkout's app processes; retain DB
```

Vite ports: Realization 5191, Financing 5192, Insurance 5193, Admin 5194.
They proxy `/api` to the configured backend. Air rebuilds/restarts on Go/SQL
changes and applies migrations; starting it can change the local database.
For separate processes use `make api` and `make web APP=realization` (or another
app). `make db-reset` destroys local data; it is not an ordinary restart.

Revalidate actual migration/process state before activation, preserve a database
backup and do not resume an intentionally held watcher without authorization.
Migration 034 backfills vehicle colors; its down migration can refuse populated
values. The 2026-10-03 schema decision records a database recreation; do not rely
on earlier migration-state notes. This docs cleanup performs no runtime changes.

## Delivery

- `deploy/Dockerfile` produces one image containing API, migration CLI and four
  built web apps. Private uploaded files require durable storage.
- `.github/workflows/ci.yml` checks pushes/PRs for `dev` and `main`: lint,
  typecheck, web units and Go tests with database/S3 fixtures.
- `.github/workflows/deploy-dev.yml` separately builds/pushes a SHA-tagged image,
  runs migrations and restarts the dev server through `deploy/dev/compose.yml`.
  A push to `dev` therefore deploys. Production/main promotion is human-only.
- `deploy/` also contains a Helm chart with dev/prod value files; its presence
  is not evidence of a live Kubernetes rollout. Default: two API replicas,
  rolling updates, readiness/liveness probes, disruption budget and optional HPA.
  Several replicas require S3; sessions live in PostgreSQL.
- A Helm pre-install/upgrade migration Job runs before new pods. Migrations must
  remain compatible with old pods: expand → deploy → contract. Use immutable
  images, external secrets and adequate database connection-pool capacity.
- `/healthz` checks the process, `/readyz` database connectivity. The API emits
  JSON logs with trace IDs and optional OpenTelemetry OTLP traces/metrics.
  Production Grafana/Loki/trace storage and cluster/registry/GitOps choices need
  an environment-specific decision; do not assume they are provisioned.

## Development policy

Run only affected unit tests once per relevant change. Extra build, lint,
typecheck, browser/E2E/integration or review/QA checks need an explicit request;
CI/release checks remain separate. Applied migrations are immutable. No live
deployment, secret changes or production action follows from a docs edit.
