# Backend packets: warehouse bulk movement

Task/base/hub/contract: `warehouse-bulk-move-20261013`; `4b92239779c5f3be1f7ea563123d8a829aee2fbe`; `/root`; `design.md` fixed proposed contracts after hub acceptance. Rules: root/internal AGENTS, agent-workflow final-only override, ADR-14 and http-domain inventory envelopes. No Git writes, tests or unit additions/runs, builds, lint, migrations, browser, runtime activation, shared canonical edits or children. Before application writes run `bash tools/check-git.sh`; root must be this project. Writer requires exact watcher pause confirmation and exclusive main-checkout writer lock. Preserve unrelated tests/credentials. Match source manifest before editing; approved upstream packet changes are expected.

## BE1 — Receipt location accounting compatibility

One behavior: keep receipt/global order facts intact while quantities reside and are identified at explicit warehouse locations. Requirements W5/W6/W7/W9. Dependencies: reviewed plan, accepted schema/HTTP contract and branch readiness. Context cap: 24k input tokens; source read in two stages, write a checkpoint before expanding. If exceeded, split within this task with hub approval; report RESLICE_REQUIRED only if canonical task scope changes.

Exact owned paths:

- `migrations/000046_inventory_receipt_locations.up.sql`
- `migrations/000046_inventory_receipt_locations.down.sql`
- `internal/modules/inventory/model/warehouse.go`
- `internal/modules/inventory/repository/warehouse.go`
- `internal/modules/inventory/service/ports.go`
- `internal/modules/inventory/service/receipt.go`
- `internal/modules/inventory/service/warehouse.go`
- `internal/modules/inventory/handler/handler.go`
- `internal/modules/inventory/handler/dto.go`

Read targeted `Receipt.receive/Identify/CorrectQuantity`, warehouse Occupied/OpenBatches/Batches/LockBatch/UpdateBatchCounts, Warehouse.Stock and handler receipt/stock DTOs. Read-only adjacency: service/stock.go `Deliver` and `Receipts` summaries, commerce/service/order_progress.go, app/app.go commerceStock. Do not edit commerce: root batch counts/IDs already provide its needed global contract.

Implement location schema/backfill, repository methods declared by service ports, global/local read projections and GET batch locations. Root batch model must not become a local-count DTO. Create location in the shared receive path so both manual and commerce receipts maintain it. Extend Identify and CorrectQuantity with optional warehouseId/default origin, root-lock serialization, selected location checks, original ID VIN placements and local occupancy calculations. Empty and unknown locations must fail accurately; no counterfeit completion. Quantity correction keeps global confirmed total semantics, audited reason and revision, applying delta only to chosen location. Keep nullable colours and original model/spec. Preserve standard response shape. Up/down are new numbered migrations only; down guard as design.

Acceptance by inspection for handoff, no execution: all creation/identification/correction paths preserve sum balances = global unidentified; occupancy uses balances; global receipt summaries unchanged; legacy request defaults unchanged; cross-company lookups scoped; no extra tests or unrelated changes. Actual checks wait until FINAL after FE. Provide compact changed-path/symbol handoff to hub for BE2; do not approve your own acceptance.

Stop: migration number collision; missing accepted contract; source hash mismatch outside predecessor; any need for foreign-module SQL, new receipt IDs, distributed correction policy or source paths outside ownership; runtime watcher still active; context over cap. Return gap to hub.

## BE2 — Atomic mixed relocation and compatible single-VIN move

One behavior: a permission-gated whole-command stock relocation preserving receipt provenance. Requirements W3/W4/W5/W6/W7/W9. Depends on BE1 complete (source only; no intermediate tests). Context cap: 20k input tokens. Resource lock: exclusive main-checkout writer; no runtime/DB/browser.

Exact owned paths:

- `internal/modules/inventory/service/warehouse_move.go` (new bulk operation on existing Vehicle service)
- `internal/modules/inventory/service/vehicle.go`
- `internal/modules/inventory/repository/vehicle.go`
- `internal/modules/inventory/service/ports.go`
- `internal/modules/inventory/repository/warehouse.go`
- `internal/modules/inventory/handler/handler.go`
- `internal/modules/inventory/handler/dto.go`

Read design transaction/HTTP sections; existing Vehicle.Move/lockMoveWarehouses; repository vehicle LockOwned/LockPlacement/UpsertPlacement/MovePlacement; custody and stock handover lock comments; model/fact.go, service/deps.go fact helper; model permissions and httpx.IfMatch. BE1 owns schema/read compatibility already. Do not add new permissions or change root module constructor: attach bulk method to existing Vehicle service/handler.

Implement `POST /warehouses/:id/stock-moves`, source If-Match and explicit destination/batch revisions. Validate all IDs, duplicates, selected limits and occurrence time; permission check at route and service. Mixed transaction follows design lock ordering and performs one full capacity calculation. Dedicated NOWAIT unit/placement/batch/warehouse methods avoid blocking cycles with target-first Stock.Transfer and placement-first Stock.Deliver. Translate SQLSTATE 55P03 locally to `stock_busy`, then leave transaction immediately; retain other callers' blocking lock semantics. Read company visibility after locks before returning details; source placement and location must still match. Move locked placement snapshots with UpsertPlacement preserving original receipt IDs. Decrement/increment balances under root locks; preserve root totals; bump batch/source/destination revisions once; append all movement facts transactionally. Whole command fails on any invalid item or persistence error. No per-item HTTP loop or separately committed calls.

Align single Move locking to unit/placement-before-warehouse using NOWAIT methods and preserve its receipt ID with the same placement helper, keeping existing HTTP compatibility. Do not modify MovePlacement used by cross-company custody. Existing ownership/custody/model/colour/reservation/payment records remain unchanged. Source revision makes exact command retry non-repeating, including unidentified-only moves.

Acceptance by inspection for handoff: one transaction and no writes before complete selection/capacity validation; compatible single endpoint; errors match design; empty/duplicate/fractional/over-limit requests reject; all facts and count changes share transaction. No verification runs until FINAL. Handoff accepted contract differences (if none, say none) and actual changed paths to hub/FE.

Stop: deadlock solution requires changing commerce/custody beyond paths; discovered policy ambiguity about reserved/consigned stock; required new repository production path beyond ownership; implementation cannot maintain preconditions or atomic rollback. Escalate, do not broaden task. Final checks cover behaviors in packet-final.md.
