# P12 — B2B contract versions and snapshots

Requirements: R9. Dependencies: P11. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/commerce/model/contract.go`
- `internal/modules/commerce/service/contract.go`
- `internal/modules/commerce/service/contract_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/repository/contract.go`
- `internal/modules/commerce/repository/store.go`
- `internal/modules/commerce/handler/contract.go`
- `internal/modules/commerce/handler/handler.go`
- `internal/modules/commerce/module.go`
- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`
- `migrations/000039_commerce_contract_versions.up.sql`
- `migrations/000039_commerce_contract_versions.down.sql`

## Behavior, interfaces and data flow

Same behavior as P11 but commerce-owned contracts tied to order, party-visible (buyer/supplier only). Exact versions snapshot both parties and accepted order terms/VINs or future-model lines. Identity requisite snapshots via small exported directory port/adaptor; never join identity tables. Templates company-owned; generated preview plain escaped content; external upload supported. Signed scans shared only to other order party via Files.Share transaction. Contract version/amendment is documentary record, not a bypass of mutual order-term acceptance. Existing economic addenda remain source of accepted terms. Both sides can read signed history; authorized recording side captures external paper signature fact, never claim browser click is e-signature. UI P13 consumes list/template/preview/sign endpoints. Adding reservation fields later requires snapshot to carry entire accepted terms representation without silent omissions.

## Acceptance and affected units

App seam unit: B2B party snapshot adapter validates the actual order parties and preserves exact snapshot fields. Add/use a meaningful pure fake-port case named `TestWorkflowPort12` in the owned app test file. Use a small app-local interface around a module's exported service where required, never import its internal layer or start DB. Run this affected unit once: `bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort12($|_)'`. This compiles app wiring while running only the new pure case; existing OpenAPI test uses e2e.DB and must not run. If no meaningful pure seam can be tested without widening source, report that exact limitation instead of substituting a build/integration command or claiming no-tests as evidence.

Outsider cannot read/write; own uploaded files shared only to counterparty. Accepted terms digest and original parties frozen; amendment preserves old version. Repeated/stale signing rejected. No direct cross-module table queries.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/commerce/service`. No other development verification/runtime actions.

Stop: Stop on new product policy or expanded ownership. Split before exceeding 20k context. No fixture-derived policies.

