# Architecture

Current architecture, consolidated 2026-10-03: a Go modular monolith and four
React/TypeScript applications. It replaces the earlier microservices/CQRS/event
sourcing design, preserved at Git tag `archive/cqrs-es-backend`.

## Backend

- `cmd/api` starts the HTTP server; `cmd/migrate` applies versioned SQL migrations.
- Echo handles HTTP and GORM persistence. One PostgreSQL uses the `public` schema
  with module-prefixed tables (`identity_users`, `commerce_orders`, etc.), following
  the recorded 2026-10-03 decision. Modules retain ownership of their tables.
- Each `internal/modules/<name>` contains handler → service → repository/model.
  Handlers parse/map requests; services own rules through interfaces;
  repositories own database access. `internal/app` wires module service ports.
- Modules never query another module's tables. Shared technical helpers live
  in `internal/pkg`. Transactions, row locks and database constraints protect
  stock, capacity, money and multi-object changes.
- A `version` column with ETag/If-Match prevents stale writes (412). Repeated
  creates/actions are guarded by business keys and state; there is no generic
  request-key replay ledger. Sensitive changes retain audit/history/snapshots.
- Money is integer minor units plus currency, serialized as decimal strings.
  Currency conversion requires an explicit policy. Never use floating point.

## HTTP and frontend

Routes use `/api/v1/<module>/…`. Session authentication uses HttpOnly cookies;
mutations require CSRF tokens. Company, branch, permissions and business-state
checks run on the server. Detail responses use `{data,revision}` and ETag;
lists use `{items,nextCursor,asOf}`; errors use
`{error:{code,message,fields,traceId}}`. DTOs hide internal persistence/auth fields.
The implemented API specification is in `internal/pkg/apidocs/swagger.json`.

`web/apps/{realization,financing,insurance,admin}` share UI, API/session helpers
and tokens under `web/packages`. Use one autocomplete input, logical dialog tabs,
visible valid actions, bottom close buttons and clear payment-row statuses.
Server results remain authoritative for financial totals and action eligibility.

HTML design references remain in `mocks/`: `/`, `/finance/`, `/insurance/`,
`/admin/`. Shared `app.js`, `realization*.js`, `partner-finance*.js`,
`finance-documents*.js` and `insurance*.js` demonstrate interactions; their
fixtures, credentials and calculation constants are not production policy.

Private files use API-authorized upload/download with size/type/hash checks;
local storage is for development, S3 for shared deployments. Schema changes use
new numbered up/down migrations; never edit applied files or use AutoMigrate.

See [business rules](business-logic.md), [infrastructure](infrastructure.md),
and scoped `internal/AGENTS.md` / `web/AGENTS.md` for implementation constraints.
