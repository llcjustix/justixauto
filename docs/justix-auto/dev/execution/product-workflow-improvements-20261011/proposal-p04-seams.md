# P04 seam proposal — company accountant preset

Status: read-only preparation for application hub; not a review/QA gate. Accepted plan/packet unchanged. No source edits, tests, Git or runtime actions. Task: product-workflow-improvements-20261011 / P04. Read current P04 and relevant role/provisioning/invoice slices only.

## Findings

1. **A SQL seed alone does not provide the accountant preset to future companies.** There is no ordinary-role provisioning mechanism today. `identity/service/company.go:129` CreateSeller and `:204` provision each create company/membership and attach the built-in CompanyAdminRoleID, but never create a private role. Only `identity/service/role.go:110` explicitly creates ordinary role records through the repository.

2. **One global ordinary “Бухгалтер” role is not assignable.** `identity/model/role.go:33` AssignableIn accepts company_admin system role or an ordinary role whose CompanyID equals the assigning company. A company-scoped role with NULL company_id fails. `identity/service/role.go:50` ForCompany filters by that rule; `identity/service/company_user.go:77` assignment uses it too. The seed must produce a private ordinary role per target company, scope=company, system_key=NULL, with grants in identity_role_permissions. Do not turn it into a new system role or loosen AssignableIn.

3. **Exact document key is correct:** `documents.sensitive.download`, defined in `documents/model/permissions.go:10` and persisted by migration 000023. Other exact keys are `documents.read` and `documents.upload`; “documents.read/upload/sensitive.download” in P04 is shorthand, never a literal SQL permission key. Content route requires documents.read; service Open additionally checks sensitive.download for sensitive files. The stale “fresh second factor” comment is not implemented and must not introduce MFA.

4. **Invoice lookup already works without sale-manage permission.** Retail GET deals/invoices use retail.read in `retail/handler/handler.go:106` and `:117`. `Deal.Invoice` (`deal.go:870`), `deal` (`:150`) and `actorInvoiceView` (`installment.go:316`) apply tenant/branch/state checks without requiring retail.deals.manage for reading. The latter limits submit-payment by PermDeals and review by PermPaymentsAccept, as desired. Retail IssueInvoice's initial PermDeals check (`deal.go:330`) and POST route (`handler.go:111`) are the planned changes. The nested finalizeInstallmentDraft path does not add another sale-manage check. Keep branch visibility and stale/status checks.

5. Commerce GET invoice/order-invoice routes use commerce.read (`commerce/handler/invoice.go:90`); `Invoice.Get/ForOrder` (`service/invoice.go:311/319`) check actual party through repositories, without commerce.trade. Issue/Void service+route permission changes remain within P04 owned files. Current commerce DTO allowedActions derive only from side/payment state (`handler/invoice.go:55`), so read-only callers can see void/review actions. P04/P05 should preserve state/party checks and make action availability match the new dedicated rights; this does not require broadening invoice lookup.

## Minimal proposed ownership additions

- `internal/modules/identity/service/company.go` — invoke one preset helper inside BOTH existing company-create transactions, using their existing transaction-bound Store. No role grants assigned automatically to ordinary employees; existing first administrator still gets company_admin.
- `internal/modules/identity/service/company_roles.go` (new) — create private accountant role and explicit grants using existing Roles().Create. Keep role scope, company ID and grant list in one helper; use the same target-kind predicate as the migration so future companies match backfill.
- `internal/modules/identity/service/company_roles_test.go` (new) — fake Store/RoleRepository tests for creation path coverage, tenant assignment and transaction failure.

No repository, port, model or handler expansion in identity is required: RoleRepository.Create already writes role+grants; Store.InTx already supplies its bound repository. Do not invoke Role.create from inside provisioning: it starts another transaction and validates an interactive user request; use the transaction-bound repository directly.

## Migration and runtime seam

000036 remains the owned migration pair. Insert the two new invoice permission catalog keys before role grants. Backfill private accountant roles for existing target companies; invoke the helper for new companies through self-service CreateSeller and shared Admin provision (seller/provider predicate consistent with migration). Keep the preset ordinary and editable by company role-management rules.

Preserve existing live custom roles if “Бухгалтер” already exists; the partial unique index is (company_id, lower(name)) WHERE deleted_at IS NULL (000024). Never silently overwrite custom permissions or revive a soft-deleted role. Surface a same-name collision explicitly to hub if migration design cannot preserve the custom role while providing the requested preset. Down migration must identify its own seeded records/grants rather than delete unrelated roles by display name. No database inspection/execution was done here.

No permission shortcut: preset grants retail.read, retail.invoices.manage, retail.payments.accept, commerce.read, commerce.invoices.manage, commerce.payments.accept, documents.read, documents.upload, documents.sensitive.download. It does not grant retail.deals.manage or commerce.trade. Existing company administrator obtains new company permissions from the live catalog; platform admin gets no business bypass.

## Proposed affected unit additions (not run)

Once implemented, add identity/service to P04's existing one-shot service package command. Test both CreateSeller and CreateSellerWithAdmin/shared provision using fakes: distinct private role IDs and CompanyID for two companies; role assignable only in its company; preset key set exactly as above; no sale/trade authority; role-write failure aborts provisioning. Include no extra preset initialization on ordinary login/read or silent recreation of a deleted company role.

Within existing P04 invoice_authority_test.go files, exercise read+invoice-manage without sale/trade authority through IssueInvoice plus resulting invoice view, preserving branch/tenant isolation and first-installment finalization. Show sale-only cannot issue after permission split. Existing grouped review tests cover accepted-only accounting. No DB-backed unit, builds or automatic QA loop.

## Planning-time source fingerprints

```text
dc05343530c8ee7394442d9239da0927c043cc0e38ca3c7b08613d4bcdf38580  internal/modules/identity/model/role.go
67c64430c38be42f31f7254c9f73127c3d92d28d729900dcd9b42948b722b788  internal/modules/identity/service/role.go
85cb47466ab8969eab86133c9aca1be17c84756d855bb31b483acc8d6e4851ae  internal/modules/identity/service/company.go
ce1e345358e8a485c070e5c390f4fd7a1fdc53461d465635bfdd6d19277c9fa2  internal/modules/identity/service/ports.go
4c03f7f0ddead31c8bc4c72cc2e77e2d0f18fc94898e5cd6ddeb6846a53c91a1  internal/modules/identity/repository/role.go
4d1b1c7f4e245302820ec506d6d67dbd121020a3529c26b6e36c5d329c13c045  internal/modules/identity/service/company_user.go
dd0fe4b79c1964d607f5141802f6fad02036dad0570aaa62c64b2d0d37533c57  internal/modules/documents/model/permissions.go
5f2b69f92ee049a2aca582e4426aa0488df9e06a1fe656c0fda0262ea199005a  internal/modules/documents/service/documents.go
135cabb20fd8fed1e2c10a99bc5953bb0ba98865409ae462eab652005d015bd2  internal/modules/retail/service/deal.go
ce77df2681ffa9b4ab72abd7796a69547c562bf7efd73ee420abd9d97fecc537  internal/modules/retail/service/installment.go
0587421739a845c8f1a8225c6b58f50a8cc68b1286cfa985ebaace47efaf9515  internal/modules/retail/handler/handler.go
16538fa5a8082056a40cad295fafc2a947217c04d7351491beaba771ac5fad83  internal/modules/commerce/service/invoice.go
5782f8d9e1a058c1379cc1d43acddbbacc0303f3a6196e83409b04f47e2da382  internal/modules/commerce/handler/invoice.go
e07f8eb7df9b01c4a07b34de279fda4406f6176b9c8fc5c445b9b01d00834e13  migrations/000023_permission_catalog_company_roles.up.sql
31e517b08ce6f030ea3bbeb6577eaf4e8f9572ed09a179132f902b16b561e006  migrations/000024_soft_delete_permissions_roles.up.sql
```

