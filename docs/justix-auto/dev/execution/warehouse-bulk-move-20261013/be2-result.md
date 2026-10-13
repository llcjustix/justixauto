# BE2 atomic stock move — result

Status: DONE (implementation handoff; no packet verification was run).

Implemented `POST /warehouses/:id/stock-moves` with source `If-Match`, explicit
destination and batch revisions, service and route permission checks, bounded
mixed selections, and one transaction. It locks selected units/placements,
batches, and warehouses in the accepted order using dedicated `FOR UPDATE
NOWAIT` paths; PostgreSQL `55P03` maps to `409 stock_busy`. It validates local
source balances and destination capacity before writing, preserves VIN receipt
batch placement links, moves only location balances, touches each selected batch
and both warehouses once, and appends per-VIN and per-batch facts. `ErrStale`
uses the existing HTTP `412 stale_revision` mapping; no `revision_conflict`
code exists in the accepted HTTP mapper.

Follow-up validation hardening rejects non-canonical UUID spellings (including
uppercase, braced, and compact aliases) for every command identifier, requires
all explicit revisions to be positive, and bounds each quantity before bounded
integer total accumulation. This prevents aliases of the same batch from
creating duplicate local balance snapshots.

Replay handling now reads the company-scoped source warehouse and compares its
revision inside the transaction before VIN placement validation. An old source
`If-Match` therefore returns the existing `412 stale_revision` mapping even
after the first command moved its selected VINs; the later NOWAIT locked
revision check remains authoritative for concurrent changes.

The successful `data` shape is `{sourceWarehouse, destinationWarehouse,
movedVehicleIds, movedUnidentified:[{receiptBatchId,quantity}], occurredAt}`.
Quantities and request revisions use decimal-string `jsonx.Quantity`; source
revision remains the envelope revision/ETag. Conflict codes are `stock_busy`,
`placement_changed`, `exceeds_unidentified`, and `capacity_exceeded`; unknown
VINs/batches and foreign invisible VINs return the existing 404 mapping.

The legacy single-VIN endpoint remains unchanged at the HTTP boundary, now uses
the same unit/placement-before-warehouse NOWAIT order and upserts its locked
placement to retain `receipt_batch_id`. Shared `MovePlacement` remains
unchanged for cross-company custody.

Changed paths and SHA-256:

- `internal/modules/inventory/service/warehouse_move.go` — `9afea906f51041ab274848dbd7214bd524481f4f0f02093e145424c9247ac0b2`
- `internal/modules/inventory/service/vehicle.go` — `2d5ac9ca1ea09a633a2cb0e284823aca0e23202b5b7b377524b97a32f9ec54ce`
- `internal/modules/inventory/service/ports.go` — `f23e80bcf1862b0921b36d3cb388b23e7fd9179c01072edf28f39f568aa3d1b9`
- `internal/modules/inventory/repository/vehicle.go` — `8db39cb96c53ebfb4187b153b0439530c10a952300245ac02454fb88ff716c20`
- `internal/modules/inventory/repository/warehouse.go` — `292a1efd32a42893dbf83a3cf7dfc51cb3aefa9859a344f4cd2a4a05ac3d8ab0`
- `internal/modules/inventory/handler/handler.go` — `54c13044302f65d3c754ba97b1c5f2d00b0114f89f305c3838e67958d379ba8f`
- `internal/modules/inventory/handler/dto.go` — `d4c5d936e45a0a3305f2679a5555f05dd4af70aa76bf7787995ba2c5688840bd`

No contract differences were introduced. No test, build, migration, database,
runtime, browser, lint, or typecheck command was run. `gofmt` was applied and
`git diff --check` was clean. Source-writer lease: released.
