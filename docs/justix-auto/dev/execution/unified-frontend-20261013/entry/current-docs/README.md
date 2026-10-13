# JustixAuto

Go backend (modular monolith: Echo + GORM + PostgreSQL) and four React apps
(Realization, Financing, Insurance, Admin), with the business rules and
runnable HTML mocks they are built from. Open **this folder**, not the
enclosing `startups` repository.

## Start here

- [Business logic](docs/justix-auto/business-logic.md)
- [Architecture](docs/justix-auto/architecture.md)
- [Infrastructure](docs/justix-auto/infrastructure.md)

## Preview the compact mocks

```sh
cd web
npm ci --ignore-scripts
npm run mocks:verify
npm run mocks:test
npm run mocks:serve
```

Open `http://127.0.0.1:4180/`, `/finance/`, `/insurance/`, `/admin/`.
The server binds only to loopback and exposes only runtime files in the mock
manifest. The four apps exchange **demo** data in the same browser/origin.
No real personal documents, passwords, financial actions or external APIs.
Use another `MOCK_PORT` if 4180 is occupied; do not kill an unrelated server.

## Quick start (Makefile)

```sh
make env                 # .env from .env.example (ports, database, origins: edit .env)
make hooks               # Git pre-commit hook (lefthook): regenerates the OpenAPI spec
make dev                 # PostgreSQL + migrations + API + all four web apps with hot reload
make stop                # Stop this checkout's API and Vite apps; keeps PostgreSQL data
```

`make dev` runs the API under [Air](https://github.com/air-verse/air) and the
four Vite apps side by side with
[concurrently](https://github.com/open-cli-tools/concurrently), prefixing each
log line with `[api]`, `[realization]`, … The apps use fixed ports:
realization `http://127.0.0.1:5191/`, financing `:5192/finance/`, insurance
`:5193/insurance/`, admin `:5194/admin/` (a taken port fails at start; the
`.env` `ALLOWED_ORIGINS` lists these). Vite proxies `/api` to `HTTP_ADDR` from
`.env`. Air (pinned in `tools/dev/go.mod`, configured in `.air.toml`) rebuilds
and restarts the API when Go files or migrations change, applying migrations
after a successful build; if the code does not compile, the previous binary is
restarted and the error shows under `[api]`. Ctrl-C stops everything; so does
any one of them exiting. For a single app use `make api` plus
`make web APP=realization` (or `financing`, `insurance`, `admin`);
`make api` alone serves the last `make web-build` output on one port. `make help` lists everything: tests
(`make test`, `make check`), database (`make db-psql`, `make db-reset`), and the
Docker image (`make image`).

## Run the backend

```sh
cp .env.example .env                    # local-only credentials
docker compose --env-file .env -f deploy/local/compose.yaml up -d
bash tools/go.sh run ./cmd/migrate up   # apply SQL migrations
bash tools/go.sh run ./cmd/api          # http://127.0.0.1:8080/healthz
```

`cmd/api` and `cmd/migrate` load `.env` from the working directory at start-up;
variables already set in the environment win, and deployments have no `.env`.

Sign in with `POST /api/v1/identity/session/login` `{"login","password"}`; the
session is an HttpOnly cookie. Every state-changing request sends the
`X-CSRF-Token` returned by login / `GET /api/v1/identity/session`. Repeating a
write is safe: unique business keys, state checks and `If-Match` reject it.
There is no second factor: the password alone signs in, and permissions alone
authorize sensitive actions.

## Run the web apps

Four cabinets share one sign-in and API: Realization (sellers, `/`), Financing
(banks and MFOs, `/finance/`), Insurance (`/insurance/`) and Admin (platform,
`/admin/`). Build them once and let the API serve them:

```sh
make web-install web-build                          # npm runs inside web/
bash tools/go.sh run ./cmd/api                      # WEB_DIR=web/apps in .env; http://127.0.0.1:8080/
```

For UI work run one app with hot reload instead; it proxies `/api/` to the
API (`JUSTIX_API`, else `HTTP_ADDR` from `.env`, else `http://127.0.0.1:8080`):

```sh
make web APP=realization      # also: financing, insurance, admin
```

An empty database has no users: create the first platform administrator once
with `printf '%s' "$PASSWORD" | make bootstrap-admin LOGIN=admin EMAIL=admin@example.com NAME=Admin`
(the command refuses once an administrator exists).

First steps after bootstrap: sign in at `/admin/`, create seller / bank / MFO /
insurance companies with their first administrator («Company administrator»:
every company permission) and activate them. That administrator creates the
company's own roles and employees in their cabinet under Настройки →
«Пользователи и роли». Admin's «Роли и разрешения» holds roles for JustixAuto
staff (platform permissions only). Permissions and their Russian names live in
PostgreSQL (`identity.permissions`) and are managed in Admin → «Разрешения». A user can belong to
several companies (company selector); roles are per company, and a company
admin can open a new company from Realization → Настройки.

Shared UI code lives in `web/packages/kit` (HTTP client with CSRF
and If-Match, session gate, shell, forms, tables and
the insurance/financing application views). Checks:

```sh
cd web && npm run typecheck && npm run lint && npm run test:unit && npm run build
```

Tests:

```sh
bash tools/go.sh vet ./...
bash tools/test-go.sh        # all tests incl. database and S3 tests (throwaway PostgreSQL + MinIO)
bash tools/go.sh test ./...  # without Docker: database tests are skipped
```

New migration: add the next numbered pair
`migrations/00000N_<name>.up.sql` / `.down.sql`; never edit an applied one.

## Layout

```text
cmd/api/                  HTTP API entry point (composition root)
cmd/migrate/              migration CLI (up / down [N] / version)
internal/modules/<name>/  one module: handler → service → repository, model
internal/pkg/             shared tech: config, database, HTTP server, errors
migrations/               versioned SQL migrations (embedded)
web/                      frontend: four React apps, shared packages (kit = shared UI),
                          its npm setup (package.json, node_modules), mock tooling and e2e harness
docs/justix-auto/         concise business, architecture and infrastructure docs; HTML mocks
tools/                    Go wrapper and dev tool (backend tooling)
deploy/                   server Helm chart; deploy/local/ = local services (Docker Compose)
AGENTS.md / CLAUDE.md     rules for coding agents
```

Mock rebuilding requires the original readable source path:
`cd web && npm run mocks:build -- /absolute/path/to/prototype`. The builder refuses an
existing output directory; preserve/review the old generated bundle first.
Original source and its hashes are recorded in `mocks/manifest.json`.
Original files were copied, not deleted/moved
out of spec-team. Shared dependencies remain single files and classic-script
order is preserved; only local variable names and whitespace are minified.
