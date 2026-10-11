# P04 — Accountant permission and invoice authority

Requirements: R10/R13. Dependencies: P03b. Read plan.md common execution rules and referenced source/contract slices. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 18k tokens (stop/split if exceeded).

## Exact owned paths

- `internal/modules/retail/model/permissions.go`
- `internal/modules/retail/service/deal.go`
- `internal/modules/retail/service/invoice_authority_test.go`
- `internal/modules/retail/service/installment_payment_test.go`
- `internal/modules/retail/handler/handler.go`
- `internal/modules/retail/module.go`
- `internal/modules/commerce/service/invoice.go`
- `internal/modules/commerce/service/invoice_authority_test.go`
- `internal/modules/commerce/model/permissions.go`
- `internal/modules/commerce/model/invoice.go`
- `internal/modules/commerce/handler/invoice.go`
- `migrations/000036_accountant_permissions.up.sql`
- `migrations/000036_accountant_permissions.down.sql`

## Behavior, interfaces and data flow

Add retail.invoices.manage and commerce.invoices.manage to module permission catalogs and persisted identity permission rows. Change retail IssueInvoice service plus HTTP route to its invoice permission, not retail.deals.manage; any internal deal lookup must allow legitimate invoice manager with retail.read without requiring deal management. Commerce currently uses commerce.trade for issue/void: add its separate invoice key and enforce in service and routes. Seed an ordinary prepared company role 'Бухгалтер' with retail.read, retail.invoices.manage, retail.payments.accept, commerce.read, commerce.invoices.manage, commerce.payments.accept and documents.read/upload/sensitive.download for evidence. Inspect exact existing payment keys before migration. Preserve company-admin all-company grants; platform admin receives no business bypass. R13 uses existing DecideInstallmentPayment atomic grouping; accepted evidence alone reduces debt. No recalculation or new payment ledger.

## Acceptance and affected units

Sales-manager-only cannot issue invoice/review payment; accountant can issue and review own-company records, cannot create sale merely through accountant preset. Company admin retains authority. Submitted/rejected evidence leaves debt unchanged; accepted grouped allocations reduce it once; stale/duplicate decisions rejected.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/retail/service ./internal/modules/commerce/service`. No other development verification or runtime actions.

Stop: Stop on any new product rule, ownership expansion or context ceiling; report exact missing decision/path. No unrelated refactoring.

