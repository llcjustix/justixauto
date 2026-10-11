# P03a — Guard new inventory and retail operations

Requirements: R2. Dependencies: P01/P02. Read plan.md common execution rules and referenced source/contract slices. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 18k tokens (stop/split if exceeded).

## Exact owned paths

- `internal/modules/inventory/service/receipt.go`
- `internal/modules/inventory/service/commercial_access_test.go`
- `internal/modules/inventory/service/receipt_colors_test.go`
- `internal/modules/retail/service/deal.go`
- `internal/modules/retail/service/listing.go`
- `internal/modules/retail/service/commercial_access_test.go`
- `internal/modules/retail/service/deal_colors_test.go`
- `internal/modules/retail/service/cash_sale_test.go`
- `internal/modules/retail/service/installment_test_helpers_test.go`

## Behavior, interfaces and data flow

Wire P01 explicit commercial helper into new receipt/vehicle creation, new retail deals, new listing and publication entry points. Inspect Receipt.Create/Confirm/Identify before classifying: receipt against already-accepted obligation and later VIN identification must remain serviceable; direct new-stock receipt needs verification. No blanket check in shared resource-load methods or ordinary permission Allow. Preserve servicing signed/preexisting deals: read, invoice/evidence, accept/reject payment, existing contract/registration/delivery and cancellation subject to their existing state rules. UI handoff records operation matrix. If one existing creation method services both new stock and established receipt, distinguish the origin in service using an existing origin field; do not add a bypass flag supplied by browser.

## Acceptance and affected units

Unverified/untyped seller direct new-stock, listing publish and new-sale blocked before writes; verified same role permitted. Old sale payment/obligation path remains accessible. Wrong permission still denied independently. Existing receipt identity/finalization not broken.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/inventory/service ./internal/modules/retail/service`. No other development verification or runtime actions.

Stop: Receipt origin cannot be safely distinguished: stop only that entry point and return exact symbol for hub classification. Never infer history-servicing bypass from client boolean.

