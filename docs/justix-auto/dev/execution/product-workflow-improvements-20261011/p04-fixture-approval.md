# P04 affected fixture extension

Application hub grants the current P04 writer two exact additional test paths:

- internal/modules/retail/service/cash_sale_test.go
- internal/modules/retail/service/installment_finalization_test.go

The first affected unit run found existing successful invoice-issue fixtures
without the new retail.invoices.manage grant. Add that grant only to legitimate
issuers so the cases reach their existing behavior assertions; preserve denial
cases. No new product policy or broader production ownership.

Identity service unit package passed and must not be repeated for this repair.
Commerce initially failed to compile because of a duplicate permission constant
within the owned model files; the writer repairs it there. Rerun only the failed
retail/service and commerce/service selections after their relevant changes.
