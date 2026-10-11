# Final Git scope proposal

Status: moving path ledger only; not frozen candidate approval. Final
re-read is required after every remaining source writer releases.

## Identity and baseline

- Root: `/Users/bakhromachilov/startups/justixauto` (apphub).
- Branch: `feature/company-commerce-workflows-20261011`.
- Local milestone/base: `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, based on
  `4fa683ba4708bb71031534260ec5cb8742ce9977`.
- No branch switch, staging, commit, push, hook, Swagger generation, test, or
  source inspection was performed for this ledger.

The milestone record says the checkout was clean immediately after that commit.
Current status is therefore post-milestone work plus execution records, with a
moving source lease. This proposal does not claim byte-consistent ownership.

## Packet-owned source and migration paths

P22 is complete/released (`result-22.md`). Its exact eventual commit scope is
the following 13 paths, including the two accepted adapter additions:

- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`
- `internal/modules/commerce/model/consignment_return.go`
- `internal/modules/commerce/service/consignment_return.go`
- `internal/modules/commerce/service/consignment_return_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/repository/consignment_return.go`
- `internal/modules/commerce/repository/store.go`
- `internal/modules/commerce/handler/consignment_return.go`
- `internal/modules/commerce/handler/handler.go`
- `internal/modules/commerce/module.go`
- `migrations/000045_consignment_returns.up.sql`
- `migrations/000045_consignment_returns.down.sql`

The first two and the existing commerce paths are tracked modifications in the
moving checkout; the new return model/service/test/repository/handler and 000045
files are untracked. This is an ownership reconciliation, not authorization to
stage them. P22's migration was source-only and was not applied.

P23v is complete/released (`result-23v.md`). Its exact two-path scope is:

- `internal/modules/inventory/handler/dto.go` (currently modified)
- `internal/modules/inventory/handler/vehicle_custody_test.go` (currently
  untracked)

P23a is currently the sole source writer. Its exact five paths, all currently
modified, are:

- `web/apps/realization/src/sale-offer-source.ts`
- `web/apps/realization/src/sale-offer-source.test.ts`
- `web/apps/realization/src/sale-form.tsx`
- `web/apps/realization/src/sale-form.test.tsx`
- `web/apps/realization/src/sale-offer-form.test.tsx`

The original P23 remains last and has no source lease. Its approved eight paths
are:

- `web/apps/realization/src/consignment-workspace.tsx`
- `web/apps/realization/src/consignment-workspace.test.tsx`
- `web/apps/realization/src/data.ts`
- `web/apps/realization/src/order-fulfilment.tsx`
- `web/apps/realization/src/order-actions.tsx`
- `web/apps/realization/src/pages/trade.tsx`
- `web/apps/realization/src/pages/inventory.tsx`
- `web/apps/realization/src/pages/stock.tsx`

Of these, `data.ts` and `pages/trade.tsx` are already dirty; they must remain
outside any P23a or interim commit and be re-read when the original P23 lease is
granted. The other six are currently absent from status and must not be inferred
into a commit.

## Other current source paths

The remaining tracked modified and untracked source paths are predecessor or
shared moving changes. They are mapped to approved packets/results in the
machine-readable [candidate ledger](final-git-approved-path-candidates.txt).
Shared paths intentionally occur in several approved packets; that overlap is
feature ownership, not a user-change conflict. All post-milestone bytes must be
preserved.

- `cmd/api/main.go`
- `internal/modules/commerce/handler/dto.go`
- `internal/modules/commerce/model/offer.go`
- `internal/modules/commerce/model/order.go`
- `internal/modules/commerce/module.go` (P22-owned; see exact list above)
- `internal/modules/commerce/repository/store.go` (P22-owned; see exact list)
- `internal/modules/commerce/service/deal.go`
- `internal/modules/commerce/service/deps.go`
- `internal/modules/commerce/service/fulfilment.go`
- `internal/modules/commerce/service/invoice.go`
- `internal/modules/inventory/module.go`
- `internal/modules/inventory/repository/reservation.go`
- `internal/modules/inventory/repository/vehicle.go`
- `internal/modules/inventory/repository/vehicle_filter_test.go`
- `internal/modules/inventory/service/ports.go`
- `internal/modules/inventory/service/stock.go`
- `internal/modules/retail/handler/dto.go`
- `internal/modules/retail/handler/handler.go`
- `internal/modules/retail/module.go`
- `internal/modules/retail/repository/store.go`
- `internal/modules/retail/service/contract.go`
- `internal/modules/retail/service/contract_test.go`
- `internal/modules/retail/service/contract_test_helpers_test.go`
- `internal/modules/retail/service/deal.go`
- `internal/modules/retail/service/deal_colors_test.go`
- `internal/modules/retail/service/deps.go`
- `internal/modules/retail/service/installment_finalization_test.go`
- `internal/modules/retail/service/installment_test_helpers_test.go`
- `internal/modules/retail/service/listing.go`
- `internal/modules/retail/service/passport_test.go`
- `internal/modules/retail/service/ports.go`
- `web/apps/realization/src/data.ts` (original P23-owned, not P23a)
- `web/apps/realization/src/pages/billing.tsx`
- `web/apps/realization/src/pages/purchases.tsx`
- `web/apps/realization/src/pages/retail.tsx`
- `web/apps/realization/src/pages/trade.tsx` (original P23-owned)
- `web/packages/kit/src/applications.tsx`

The candidate ledger maps the overlapping commerce, inventory, retail,
reservation, stock, supply, and realization paths to their accepted
packet/result records. Final staging still requires a fresh status/path re-read
after P23a and original P23 release.

The exact untracked non-document paths outside the P22/P23v lists are:

- `internal/app/reservation_jobs.go`, `internal/app/reservation_jobs_test.go`,
  `internal/app/reservation_stock.go`, `internal/app/reservation_stock_test.go`
- `internal/modules/commerce/handler/consignment.go`,
  `internal/modules/commerce/handler/consignment_test.go`,
  `internal/modules/commerce/handler/reservation.go`,
  `internal/modules/commerce/handler/reservation_payment.go`,
  `internal/modules/commerce/handler/reservation_payment_history_test.go`,
  `internal/modules/commerce/handler/reservation_payment_test.go`,
  `internal/modules/commerce/handler/reservation_stock.go`,
  `internal/modules/commerce/handler/reservation_stock_test.go`,
  `internal/modules/commerce/handler/reservation_test.go`,
  `internal/modules/commerce/handler/supply.go`
- `internal/modules/commerce/model/consignment.go`,
  `internal/modules/commerce/model/reservation.go`,
  `internal/modules/commerce/model/reservation_payment.go`,
  `internal/modules/commerce/model/supply.go`
- `internal/modules/commerce/repository/consignment.go`,
  `internal/modules/commerce/repository/reservation.go`,
  `internal/modules/commerce/repository/reservation_payment.go`,
  `internal/modules/commerce/repository/supply.go`
- `internal/modules/commerce/reservation_stock.go`,
  `internal/modules/commerce/service/consignment.go`,
  `internal/modules/commerce/service/consignment_sale_test.go`,
  `internal/modules/commerce/service/consignment_test.go`,
  `internal/modules/commerce/service/reservation.go`,
  `internal/modules/commerce/service/reservation_allocation.go`,
  `internal/modules/commerce/service/reservation_allocation_test.go`,
  `internal/modules/commerce/service/reservation_invoice_test.go`,
  `internal/modules/commerce/service/reservation_lifecycle.go`,
  `internal/modules/commerce/service/reservation_lifecycle_test.go`,
  `internal/modules/commerce/service/reservation_payment.go`,
  `internal/modules/commerce/service/reservation_payment_test.go`,
  `internal/modules/commerce/service/reservation_stock.go`,
  `internal/modules/commerce/service/reservation_stock_test.go`,
  `internal/modules/commerce/service/reservation_test.go`,
  `internal/modules/commerce/service/supply.go`,
  `internal/modules/commerce/service/supply_test.go`
- `internal/modules/inventory/reservation_stock.go`,
  `internal/modules/inventory/service/consignment_sale_test.go`,
  `internal/modules/inventory/service/custody.go`,
  `internal/modules/inventory/service/custody_test.go`,
  `internal/modules/inventory/service/reservation_holder_test.go`,
  `internal/modules/inventory/service/reservation_stock.go`,
  `internal/modules/inventory/service/reservation_stock_test.go`
- `internal/modules/retail/handler/scoring_dto_test.go`,
  `internal/modules/retail/model/scoring.go`,
  `internal/modules/retail/repository/scoring.go`,
  `internal/modules/retail/service/consignment_test.go`,
  `internal/modules/retail/service/scoring.go`,
  `internal/modules/retail/service/scoring_test.go`
- `migrations/000040_retail_demo_scoring.down.sql`,
  `migrations/000040_retail_demo_scoring.up.sql`,
  `migrations/000041_commerce_future_supply.down.sql`,
  `migrations/000041_commerce_future_supply.up.sql`,
  `migrations/000042_commerce_reservations.down.sql`,
  `migrations/000042_commerce_reservations.up.sql`,
  `migrations/000043_reservation_payments.down.sql`,
  `migrations/000043_reservation_payments.up.sql`,
  `migrations/000044_commerce_consignments.down.sql`,
  `migrations/000044_commerce_consignments.up.sql`
- `web/apps/realization/src/contract-workspace.test.tsx`,
  `web/apps/realization/src/contract-workspace.tsx`,
  `web/apps/realization/src/reservation-payment.test.tsx`,
  `web/apps/realization/src/reservation-payment.tsx`,
  `web/apps/realization/src/reservation-workspace.test.tsx`,
  `web/apps/realization/src/reservation-workspace.tsx`,
  `web/apps/realization/src/sale-insurance.test.tsx`,
  `web/apps/realization/src/sale-insurance.tsx`

These are preserved predecessor/moving paths, all mapped in the candidate
ledger, with no current final-commit grant. They are not unexplained deletions
and must not be swept into a commit.

## Execution artifacts

All additions below the execution directory
`docs/justix-auto/dev/execution/product-workflow-improvements-20261011/` are
root-owned orchestration records: dispatch manifests/checkpoints, accepted
scope/approval packets, result reports, delivery maps, snapshots, and this
proposal (including the earlier `final-git-preparation.md`). They are not
application source. Their exact inclusion in a later documentation commit must
be decided by the root coordinator after the record freeze; do not use a glob or
stage the subtree wholesale while writers are active. The root
`checkpoint.md` is actively changing and remains especially excluded until the
final handoff.

## Remaining operation

After P23a releases, freeze the source and re-read status against this ledger.
Then reconcile P23a's five paths, the released P23v two paths, released P22's
13 paths, the final original-P23 eight paths, and explicitly selected root
records. Any path absent from the candidate ledger or later approved P23
result/path record is an unexplained-path blocker and must be reported rather
than silently staged. Shared paths are not blockers merely because multiple
approved packets map to them. Only then
may version control stage explicit paths and commit with
`git -c core.hooksPath=/dev/null commit` and a conventional packet-identity
message. This proposal itself does not approve a commit or push.
