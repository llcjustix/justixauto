# P11 — Immutable retail contract versions

Requirements: R9. Dependencies: P10. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/retail/model/contract.go`
- `internal/modules/retail/service/contract.go`
- `internal/modules/retail/service/contract_test.go`
- `internal/modules/retail/service/deal.go`
- `internal/modules/retail/service/ports.go`
- `internal/modules/retail/repository/contract.go`
- `internal/modules/retail/repository/store.go`
- `internal/modules/retail/handler/contract.go`
- `internal/modules/retail/handler/handler.go`
- `internal/modules/retail/handler/dto.go`
- `internal/modules/retail/module.go`
- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`
- `internal/modules/identity/service/company.go`
- `internal/modules/identity/module.go`
- `migrations/000038_retail_contract_versions.up.sql`
- `migrations/000038_retail_contract_versions.down.sql`

## Behavior, interfaces and data flow

Create retail-owned contract template/version records, editable draft text and data-filled immutable snapshot (seller/customer, vehicle snapshot, price/currency, installment/payment terms). Template content configurable by authorized company staff; plain escaped placeholders, no executable expressions or legally approved default clauses. Support external uploaded document and paper signed scan using existing Files.Share private binding. Signed version stores reference/date/file IDs, content/party/terms snapshot, creator/time and revision; signed versions append-only. Amendment references prior signed version, never overwrites. Existing POST contract-records delegates to version append; legacy existing signed metadata migrates to a historical version without invented documents. Keep Deal current-reference compatibility fields for existing installment snapshot, but do not mutate saved schedule/plan when adding documentary amendment. Add list/detail/template draft/sign routes under deals, with If-Match and own-tenant checks. New contract drafts on existing obligations are servicing; new sale gate remains P03.

## Acceptance and affected units

App seam unit: retail party snapshot request cannot request an unrelated company. Add/use a meaningful pure fake-port case named `TestWorkflowPort11` in the owned app test file. Use a small app-local interface around a module's exported service where required, never import its internal layer or start DB. Run this affected unit once: `bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort11($|_)'`. This compiles app wiring while running only the new pure case; existing OpenAPI test uses e2e.DB and must not run. If no meaningful pure seam can be tested without widening source, report that exact limitation instead of substituting a build/integration command or claiming no-tests as evidence.

First sign captures immutable data; changing customer later does not rewrite snapshot. Retrying existing signed version conflicts, amendment adds history. Cross-tenant file/template denied; legacy signed fields retained. Retail saved installment schedule unchanged by documentary amendment. New signing requires scan; legacy missing scans preserved as legacy.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/retail/service`. No other development verification/runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

Party snapshot seam: extend retail Company port with the minimal company-requisite snapshot needed for its own contract. Add an exported identity service method and app adapter that authorizes the contract's company/party context; never place bank/tax/passport fields into public Directory/Profile merely to fill a contract. Customer snapshot comes from retail's own CRM repository. P12 reuses this identity export with validated order-party context. Legacy migration creates one known historical version from current signed fields only; no fabricated earlier history.

