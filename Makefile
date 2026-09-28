# JustixAuto — local development. `make help` lists the targets.
#
# First run:   make env && make dev   → API + four web apps with hot reload
#              (the URLs are printed on start; Ctrl-C stops everything)
# One app:     make api (one terminal) + make web APP=realization (another)
#
# Settings (ports, database, origins) live in .env; the API, migrate and the
# Vite apps read it at start-up. Make itself does not read or export it.

SHELL := /bin/bash
.DEFAULT_GOAL := help

APP           ?= realization
# Test packages for make test-go.
PKG           ?= ./...
COMPOSE       := docker compose --env-file .env -f deploy/local/compose.yaml
TEST_COMPOSE  := docker compose -f deploy/test/compose.yaml
GO            := bash tools/go.sh
# The frontend (npm workspaces, node_modules) lives in web/.
NPM           := npm --prefix web
LINT          := $(GO) tool -modfile=tools/lint/go.mod golangci-lint
AIR           := $(GO) tool -modfile=tools/air/go.mod air
CONCURRENTLY  := web/node_modules/.bin/concurrently

.PHONY: help env db-up db-down db-reset db-psql migrate migrate-down \
        api web web-install web-build dev test test-go test-web lint typecheck check \
        openapi openapi-check doctor lint-go fmt deadcode image

help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# ---- setup ----

env: ## Create .env from .env.example (never overwrites)
	@if [ -f .env ]; then echo ".env exists — edit it or delete it first"; else cp .env.example .env && echo "created .env"; fi

web-install: ## Install web dependencies into web/node_modules (npm ci)
	$(NPM) ci --ignore-scripts

# ---- database ----

db-up: ## Start PostgreSQL (Docker) and wait until it accepts connections
	@test -f .env || { echo "run: make env"; exit 1; }
	$(COMPOSE) up -d --wait postgres

db-down: ## Stop PostgreSQL, keep data
	$(COMPOSE) stop

db-reset: ## Delete the local database volume (all data!) and start again
	$(COMPOSE) down -v
	$(MAKE) db-up migrate

db-psql: ## Open psql in the database container
	$(COMPOSE) exec postgres psql -U justixauto -d justixauto

migrate: ## Apply SQL migrations
	$(GO) run ./cmd/migrate up

migrate-down: ## Roll back the last migration
	$(GO) run ./cmd/migrate down 1

# ---- run ----

api: db-up migrate ## Run the API (http://$HTTP_ADDR); serves built web apps (WEB_DIR) if present
	$(GO) run ./cmd/api

web: ## Vite dev server with hot reload for one app: make web APP=realization|financing|insurance|admin
	$(NPM) run dev --workspace apps/$(APP)

web-build: ## Build the four web apps into web/apps/*/dist
	$(NPM) run build:apps

dev: db-up migrate ## API + four web apps with hot reload (Air restarts the API on Go/migration changes)
	@$(CONCURRENTLY) --kill-others --prefix-colors auto --names api,realization,financing,insurance,admin \
	  "$(AIR)" \
	  "$(NPM) run dev --workspace apps/realization" \
	  "$(NPM) run dev --workspace apps/financing" \
	  "$(NPM) run dev --workspace apps/insurance" \
	  "$(NPM) run dev --workspace apps/admin"

# ---- checks ----

test-go: ## Go tests incl. database and S3 (throwaway PostgreSQL + MinIO containers); PKG=./internal/...
	$(TEST_COMPOSE) up -d --wait || { $(TEST_COMPOSE) down -v; exit 1; }
	TEST_DATABASE_URL='postgres://postgres:justixtest@127.0.0.1:55442/justixauto_test?sslmode=disable' \
	  TEST_S3_ENDPOINT=http://127.0.0.1:59010 AWS_REGION=us-east-1 \
	  AWS_ACCESS_KEY_ID=justixtest AWS_SECRET_ACCESS_KEY=justixtest-secret \
	  $(GO) test -race -count=1 -p 1 $(PKG); \
	  status=$$?; $(TEST_COMPOSE) down -v; exit $$status

test-web: ## Web unit tests
	$(NPM) run test:unit

test: test-go test-web ## All tests

typecheck: ## TypeScript typecheck
	$(NPM) run typecheck

lint: openapi-check lint-go deadcode ## golangci-lint + deadcode + ESLint + Prettier + knip + OpenAPI spec up to date
	$(NPM) run lint
	$(NPM) run format:check
	$(NPM) run knip

lint-go: ## golangci-lint (config: .golangci.yml)
	$(LINT) run ./...

fmt: ## Format Go (gofumpt, goimports) and web (Prettier) sources
	$(LINT) fmt ./cmd/... ./internal/... ./migrations/...
	$(NPM) run format

deadcode: ## Fail on unreachable Go functions (tests count as callers)
	@out=$$($(GO) tool -modfile=tools/lint/go.mod deadcode -test ./...) || { echo "$$out"; exit 1; }; \
	  if [ -n "$$out" ]; then echo "$$out"; exit 1; fi

# ---- API docs ----

openapi: ## Regenerate the OpenAPI spec from handler annotations (served at /api/docs with API_DOCS=true)
	$(GO) tool swag fmt --dir ./cmd/api,./internal
	$(GO) tool swag init --quiet --dir ./cmd/api,./internal --generalInfo main.go \
	  --output internal/pkg/apidocs --outputTypes json --parseInternal --parseDependency --requiredByDefault

openapi-check: openapi ## Fail when the committed spec differs from the annotations
	@git diff --quiet -- internal/pkg/apidocs/swagger.json || \
	  { echo "OpenAPI spec is out of date: review and commit internal/pkg/apidocs/swagger.json"; exit 1; }

doctor: ## Check local development prerequisites (Go, Node, npm, Docker, Git)
	$(GO) version
	node --version
	npm --version
	docker compose version
	docker info --format 'Docker daemon: {{.ServerVersion}}'
	@bash tools/check-git.sh

check: lint typecheck test ## Everything CI would run

# ---- container image ----

image: ## Build the Docker image justixauto:dev
	docker build -f deploy/Dockerfile -t justixauto:dev --build-arg VERSION=$$(git rev-parse --short HEAD) .
