# BE1 receipt location accounting compatibility — result

Status: DONE (implementation handoff; no packet checks were run).

Implemented contract:

- Migration 46 creates `inventory_receipt_locations`, backfills each original
  receipt location (including zero balances), and blocks down migration when
  positive unidentified stock remains outside its original receiving warehouse.
- `Receipt.receive` creates the origin location in the same transaction.
  `Receipt.Identify` now accepts optional `warehouseId` (defaulting to the
  original receiving warehouse), locks its root batch then selected warehouse
  and location, places VINs there with the original receipt batch ID, and
  decrements both local and global unidentified balances.
- `Receipt.CorrectQuantity` accepts optional `warehouseId` (same default),
  applies its audited global-confirmed delta to exactly that location, rejects
  a negative local balance or local capacity overflow, and retains batch
  revision semantics.
- Warehouse occupancy now sums placements plus location balances. Warehouse
  inventory returns only positive local batches with
  `warehouseUnidentifiedCount`; receipt-wide batch fields keep their existing
  meanings. `GET /receipt-batches/:id` returns the existing global batch plus
  deterministic positive `{warehouseId, unidentifiedCount}` locations.

Changed paths and SHA-256:

- `migrations/000046_inventory_receipt_locations.up.sql` — `af845b1de029f60e84b05a92afd9c5027d923e4867befbe9070a271c47e0e188`
- `migrations/000046_inventory_receipt_locations.down.sql` — `23394109f286f010c6826f969a1a5fad058f7a9f22312c77ec026bdbd07370fe`
- `internal/modules/inventory/model/warehouse.go` — `3c4f52487e2bc5fb19c65632b9b7ed6d7df9666ba4544b5ae6dcc765f6b40ad8`
- `internal/modules/inventory/repository/warehouse.go` — `01433355a169292c3fd3e4f54e9ce9cd9c46f56378aeb03ccfa0d3d424079422`
- `internal/modules/inventory/service/ports.go` — `493170d21502e983dd4df85485ab9f99062379db70d44223b46c55b59c6c2670`
- `internal/modules/inventory/service/receipt.go` — `b5bbcda8a445311bfd9d854cc880730aa0842001572c418278b927e77f17f0c3`
- `internal/modules/inventory/service/warehouse.go` — `4d372e88272a3c10ba3333ac8ca677361f29bb66134142abe805997f7f38f6d8`
- `internal/modules/inventory/handler/handler.go` — `e27e6321f3766954adeee21a5eb8e26a2fcd2704acd84b1864aad8fdbb3a4cae`
- `internal/modules/inventory/handler/dto.go` — `603089d0389b34f0bda9a9036f114c5e5f928afd9e738f2c74a7762dc7c58194`
- `internal/modules/inventory/service/receipt_colors_test.go` — `58fb8065967a9ce7aac587773f2de19d27ab61924eb4220fbb58513a64c9f166`
  (hub-approved minimal existing test-double compatibility update; no cases added or run)

No migration/runtime/database/test/build/lint/typecheck command was run. `gofmt`
was applied to edited Go paths and `git diff --check` was clean. The unrelated
`internal/modules/identity/service/company_contract_test.go` SHA-256 remains
`a4d4b3f67107b5362810dcc96f68f1de8e59cae466dacafb7e100e29a59fe99f`.

Writer release: released the exclusive main-checkout source-writer lease. BE2
may extend the location repository methods for atomic relocation; no accepted
contract differences were introduced.
