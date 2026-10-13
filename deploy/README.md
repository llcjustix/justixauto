# Deployment and local services

The only infrastructure folder:

- `Dockerfile` (+ `Dockerfile.dockerignore`): the single image (API, migrate,
  one built React portal). Build from the repository root:
  `docker build -f deploy/Dockerfile .` or `make image`.
- `dev/compose.yml`: the dev server deployment used by
  `.github/workflows/deploy-dev.yml` (push to `dev` → build, push
  `poirot92/justix-auto:<sha>`, migrate, restart).
- `local/compose.yaml`: third-party services for local development.
- The Helm chart (`Chart.yaml`, `templates/`, `values*.yaml`) for Kubernetes.

Locally the backend and frontend run on your machine
(`make api`, `make web`). Their third-party services run in Docker.

The portal workspace is `web/apps/portal`; its build output is `dist` there.
The image copies that output to `/app/web` and sets `WEB_DIR=/app/web`; the
tracked dev workflow and Helm default use the same concrete directory. Existing
section URLs share this entry. `WEB_DIR` no longer points to a parent containing
four application directories. Existing private local configuration requires a
manual change to `WEB_DIR=web/apps/portal/dist`; an empty value stays API-only.
The one development web port is 5191. See
[the frontend ADR](../docs/justix-auto/adr-15-unified-frontend.md).

## Local services

`local/compose.yaml` runs the single PostgreSQL 18.6 database used by the
modular monolith ([ADR-14](../docs/justix-auto/adr-14-classic-modular-monolith.md)),
bound to loopback port 55432 with a named volume. Credentials come from the
git-ignored `.env` (see `.env.example`); never commit real secrets.

```sh
make db-up      # start and wait until ready
make db-down    # stop, keep data
make db-reset   # delete the volume and start again
```

## Servers (Kubernetes)

This folder is also the Helm chart (`Chart.yaml`, `templates/`), see
[ADR-15](../docs/justix-auto/adr-15-kubernetes-multi-replica-observability.md).
`values-dev.yaml` holds non-production server values; `values-prod.yaml` is a
template (registry, image digest, S3 bucket, IRSA role, ingress host). Rollout to
any cluster is human-only (`AGENTS.md`).
