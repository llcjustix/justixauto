# P04 result — accountant invoice authority

Status: DONE. Branch/head at dispatch: `feature/company-commerce-workflows-20261011` / `4fa683ba4708bb71031534260ec5cb8742ce9977`. No Git writes, runtime, migration execution, browser, DB fixture, lint, typecheck, build, review or QA action occurred.

## Implemented seams

- Added company permission keys `retail.invoices.manage` and `commerce.invoices.manage` to the module catalogs and migration `000036`.
- Retail `POST /api/v1/retail/deals/:id/invoices` and `Deal.IssueInvoice` require `retail.invoices.manage`; invoice reads remain `retail.read`. Retail payment-review routes and services remain `retail.payments.accept`.
- Commerce `POST /api/v1/commerce/orders/:id/invoices` and `POST /api/v1/commerce/invoices/:id/void`, plus `Invoice.Issue`/`Void`, require `commerce.invoices.manage`. Evidence decision service/route requires `commerce.payments.accept`; payment submission remains `commerce.trade`.
- Commerce invoice DTO `allowedActions` now exposes supplier `void` only with `commerce.invoices.manage`, and evidence `accept`/`reject` only with `commerce.payments.accept`; buyer `submit-payment` retains its existing party/state behavior.
- Seller creation through both `CreateSeller` and shared `provision` creates an ordinary, private, assignable-only-in-that-company `Бухгалтер` role. Its exact grants are `retail.read`, `retail.invoices.manage`, `retail.payments.accept`, `commerce.read`, `commerce.invoices.manage`, `commerce.payments.accept`, `documents.read`, `documents.upload`, and `documents.sensitive.download`; it grants neither sale management nor trade.
- Migration backfills live seller companies using deterministic IDs derived from `justixauto:p04:accountant:<company-id>` so down migration identifies only its own backfill. A live custom `Бухгалтер` is preserved and makes the seeded display name `Бухгалтер (счета и оплаты)`; a further same-name collision fails rather than overwriting custom data. Future-company roles have random IDs and are intentionally unaffected by migration rollback.

## Scope and evidence

- Approved test-fixture extension used: `internal/modules/retail/service/cash_sale_test.go` and `installment_finalization_test.go`; each now gives legitimate invoice issuers `retail.invoices.manage`, while the explicit denial case removes that new key.
- `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/identity/service` passed in the initial scoped run.
- After the authorized fixture extension and commerce constant repair: `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/service ./internal/modules/commerce/service` passed (`retail` 0.458s; `commerce` 0.847s). No further verification was run.
- R2 source-only observation requested by hub: no independent active-only setup guard was found. `identity/service/branch.go` `Branch.Create` uses `companyFor` permissions; `identity/service/company_user.go` `CompanyUser.Create` validates membership/roles; `inventory/service/warehouse.go` `Warehouse.Create`/`Update` contains no `CommercialAccess` or `AccessActive` condition. This is not new verification evidence.
