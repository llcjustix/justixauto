# P08 — Private CRM passport profile backend

Requirements: R6. Dependencies: P07. Read plan.md common execution rules and referenced source/contract slices. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 18k tokens (stop/split if exceeded).

## Exact owned paths

- `internal/modules/retail/model/customer.go`
- `internal/modules/retail/service/crm.go`
- `internal/modules/retail/service/passport_test.go`
- `internal/modules/retail/service/ports.go`
- `internal/modules/documents/module.go`
- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`
- `internal/modules/retail/repository/crm.go`
- `internal/modules/retail/handler/dto.go`
- `internal/modules/documents/service/documents.go`
- `internal/modules/documents/service/passport_test.go`
- `migrations/000037_customer_passport.up.sql`
- `migrations/000037_customer_passport.down.sql`

## Behavior, interfaces and data flow

Extend existing CustomerInput.profile / customerDTO with optional passport series/number/issuer, issue/expiry/birth dates, personal ID, registered address and file binding IDs. Date-only format and sensible generic consistency; no country-specific regex or mandatory passport prerequisite. Persist fields and file IDs; bind own-company files through existing retail Files.Share(ctx, owner, file, same owner, retail.customer, customerID) using Store.Bind transaction. Add documents purpose passport-copy marked sensitive; existing MIME/size/private Open authorization reused. Do not include passport values or copy URLs in event details/logs; events may identify changed field names/opaque file IDs. Current CRM CreateCustomer/UpdateCustomer transaction shape must include binding failure rollback. Cross-company file ID fails.

## Acceptance and affected units

App seam unit: passport binding forwards owning tenant, required sensitive purpose and resource ID without widening file access. Add/use a meaningful pure fake-port case named `TestWorkflowPort08` in the owned app test file. Use a small app-local interface around a module's exported service where required, never import its internal layer or start DB. Run this affected unit once: `bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort08($|_)'`. This compiles app wiring while running only the new pure case; existing OpenAPI test uses e2e.DB and must not run. If no meaningful pure seam can be tested without widening source, report that exact limitation instead of substituting a build/integration command or claiming no-tests as evidence.

Optional empty profile valid; fields roundtrip/update; malformed date rejected; cross-tenant file bind fails; sensitive download needs documents permission; plain read returns metadata only, no storage key/public URL. No passport payload in audit events.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/retail/service ./internal/modules/documents/service`. No other development verification or runtime actions.

Stop: Stop on any new product rule, ownership expansion or context ceiling; report exact missing decision/path. No unrelated refactoring.

Privacy seam: passport binding must verify file purpose/sensitivity server-side, not trust uploader UI. Add a narrow Files.BindPassport port implemented by documents exported service via app adapter; it requires owned passport-copy/sensitive file before same-company resource binding. Do not allow a user to bind a nonsensitive vehicle-photo as a passport and thereby bypass sensitive-download permission. Reuse existing Share for the actual tenant grant after metadata validation.

