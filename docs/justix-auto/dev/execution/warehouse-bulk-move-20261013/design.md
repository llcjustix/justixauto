# Warehouse bulk movement design

Status: proposed decomposition; hub acceptance required before dispatch.
Task: `docs/justix-auto/dev/warehouse-bulk-move-20261013.md`.
Authority: application hub `/root`. Base: `4b92239779c5f3be1f7ea563123d8a829aee2fbe`.
Branch supplied by hub: `fix/warehouse-bulk-move-20261013`; local commit only.

## Intermediate source findings

- `ReceiptBatch` combines global receipt counts and original warehouse. Its SQL constraint requires identified + unidentified = confirmed. Partial relocation cannot decrement confirmed or create a second receipt without changing receipt meaning.
- Commerce `ShipmentLine.ReceiptBatchID` and `Stock.Receipts`/receipt summaries use the original batch. Order progress reads that batch's global counts. Keep these IDs and totals stable.
- `WarehouseRepository.Occupied` currently sums batches by original warehouse. Introduce location balances before enabling partial relocation; identification and correction must consume the same balances.
- `Receipt.Identify` locks batch then warehouse. `CorrectQuantity` does likewise. Receipt creation also serves commerce quantity delivery through `Receipt.receive`.
- Existing `VehicleRepository.MovePlacement` clears `receipt_batch_id`; this loses provenance even in same-company moves. Preserve provenance for that operation.
- Existing single-VIN movement locks warehouses before placement; custody locks vehicle/placement before warehouses. New bulk work must align these overlapping paths rather than introduce an additional inversion.
- Shared `ReceiptIdentification` currently loads only the original warehouse and treats absence there as complete. It needs receipt location discovery and an explicit selected location, including when opened from an order.

No source, Git, runtime, DB, browser or tests have been changed/run by the slicer. Unrelated `company_contract_test.go` and unread `test_creds.txt` remain outside scope.

## Requirements and acceptance coverage

These are local acceptance labels, not new backlog tasks.

| ID | Approved behavior | Implementation | Final evidence |
|---|---|---|---|
| W1 | Visible warehouse movement; accessible VIN navigation; single modal flow | FE | F-UI |
| W2 | Manual/all-loaded/first-N ascending VIN selection; visible editable VIN/model/colour list | FE | F-UI |
| W3 | Mixed VINs and positive partial quantities from multiple batches in one atomic command | BE2, FE | F-HTTP, F-UI |
| W4 | Same company, move permission, distinct destination, occurrence date, whole-command rejection on invalid/stale/capacity | BE2, FE | F-HTTP, F-UI |
| W5 | Preserve original receipt totals, exact specification/colours, ownership/custody, original order association | BE1, BE2 | F-HTTP |
| W6 | Occupancy and remaining identification agree after moves and later destination identification | BE1, BE2, FE | F-HTTP, F-UI |
| W7 | Compatible existing single-VIN move and existing receipt/identify/correction clients | BE1, BE2, FE | F-HTTP |
| W8 | Clear no-access/no-destination/loading/error/success states, editable drafts, safe retry, keyboard/mobile | FE | F-UI |
| W9 | No purchases/payments/ownership policy changes, new migration only, final-only verification | All | F-HTTP plus execution scope |

Dependency order: **BE1 → BE2 → FE → FINAL**. BE1/BE2 are two bounded sections in `packet-be.md`; no intermediate tests or automatic review/QA loop. Independent plan review and owning-hub contract acceptance occur before execution. One independent final QA packet contains small HTTP and UI cases on the same finished SHA. Final failures trigger only affected repairs/rechecks.

## Fixed proposed data contract

Add migration `000046_inventory_receipt_locations.up.sql` and `.down.sql` (stop if number is taken). Table `inventory_receipt_locations` has composite primary key `(receipt_batch_id, warehouse_id)`, FKs to inventory batch/warehouse, and `unidentified_count integer NOT NULL CHECK >= 0`. Backfill one row for every existing batch at its original warehouse using its unidentified count. Keep zero balances so drained allocations are explicit. Application transactions guarantee same-company locations and `SUM(location.unidentified_count) = batch.unidentified_count`.

`ReceiptBatch` remains the original receipt: `warehouse_id` is original receiving warehouse; confirmed/identified/unidentified remain global; identified + unidentified = confirmed. Movement never changes these counts, received date, actor, model/version or colours. Touch batch revision on balance movement; touch both warehouse revisions. New receipts, including commerce delivery, create the initial balance in the same transaction. Identification decrements selected location and global unidentified, increments global identified, creates VIN placement at that location with **original receipt batch ID**. Occupancy sums placements plus location balances, never both legacy batch counts and locations.

Do not create child receipts or new order lines. Commerce receipt summaries remain global and keep existing IDs/contracts; order progress continues to count each original receipt once. Shared UI discovers locations through inventory; no commerce table access from inventory or schema change in commerce. Missing legacy colours remain null until the existing explicit exact-specification identification flow resolves them; movement must not invent colours.

Quantity correction remains an audited correction of the global confirmed total. Extend its request with optional `warehouseId`, default original receiving warehouse. Apply `newConfirmed - oldConfirmed` to only that location; a negative local result is `409 exceeds_unidentified`; positive delta checks selected warehouse capacity. Still reject totals below identified count and stale batch revisions. Never silently distribute a correction across warehouses. UI sends its explicit warehouse context and labels the total as receipt-wide. This is compatibility work to preserve the existing correction operation under the new storage model.

Migration activation is a coordinated local runtime step owned by hub/DevOps, after all source work. Old writers cannot run after balance backfill: they do not maintain location counts. Down migration must refuse while any positive balance exists outside its original receiving warehouse; otherwise removing the table safely restores the old projection. Do not automatically move stock or erase history on rollback. No infrastructure files or deployment are in scope.

## Fixed proposed HTTP contract

All paths below are under `/api/v1/inventory`, with existing session/company middleware and normal error/envelope conventions; counts/revisions are decimal strings (`jsonx.Quantity`, `httpx.Revision`). No new permission.

1. `POST /warehouses/{sourceId}/stock-moves`, `If-Match: "<source warehouse revision>"`:

   ```json
   {"toWarehouseId":"uuid","toWarehouseRevision":"7","occurredAt":"2026-10-13T08:00:00Z","vehicleIds":["uuid"],"unidentified":[{"receiptBatchId":"uuid","quantity":"2","revision":"5"}]}
   ```

   Permission `inventory.vehicles.move` enforced in route and service. Validate source/destination UUIDs, distinct company-owned warehouses, nonempty selection, unique vehicle/batch IDs, positive integer quantities and revisions, existing five-minute future tolerance. Bound request to 1000 VIN IDs, 1000 batch entries, and 10000 total moved units; reject over-limit whole requests (never truncate or split into HTTP writes). Source precondition required (428); changed source/destination/batch revision is 412 `stale_revision`. Unknown/foreign batch or VIN is 404 without tenant details; wrong source placement is 409 `placement_changed`; wrong/missing source balance or insufficient quantity is 409 `exceeds_unidentified`; destination full is 409 `capacity_exceeded`; malformed fields use existing validation mapping. Both warehouses are company scoped, including when actor owns a VIN elsewhere. Preserve existing handling of reserved/consigned VINs for same-company Move: introduce no new commercial restriction.

   Resource contention returns 409 `stock_busy` without partial writes. Successful 200 `httpx.Data` body is `{sourceWarehouse: WarehouseDTO,destinationWarehouse: WarehouseDTO,movedVehicleIds: string[],movedUnidentified: [{receiptBatchId,quantity}],occurredAt}`; envelope revision/ETag is updated source revision. All changes and facts commit in one transaction. Retry of the same snapshot cannot move another quantity: source revision changed. UI never automatically refreshes revisions and resends after transport uncertainty.

2. `GET /receipt-batches/{id}` with inventory read permission returns `httpx.Data({batch: existingGlobalBatchDTO, locations:[{warehouseId,unidentifiedCount}]}, batchVersion)`; locations include positive balances, deterministic warehouse ID order, company-scoped receipt first. Original `batch.warehouseId` retains its historical meaning.

3. Warehouse inventory retains existing global batch fields; add `warehouseUnidentifiedCount` to each `unidentifiedBatches` entry, filter rows by positive local balance. `warehouseId` remains original receipt warehouse. All new UI local quantity limits use the added field; receipt-wide counts keep their existing meaning. Warehouse resource is still the explicit context.

4. Identify request adds optional `warehouseId`, defaulting to original receiving warehouse for legacy clients. Validate selected same-company balance and limit by that balance, not global count. Keep existing atomic flag, VIN validation, response and permissions. Do not auto-redirect legacy requests to a different location. Insufficient local stock is a conflict, not false completion. Correction extension is as above, existing If-Match preserved.

Existing single-VIN endpoint request/response stays compatible. Its same-company placement write must preserve receipt ID; do **not** alter shared `MovePlacement` clearing semantics used by cross-company custody. Use the locked placement plus existing `UpsertPlacement`, or a dedicated preservation method confined to same-company relocation.

## Transaction and concurrency contract

Bulk: validate shape outside transaction; inside, acquire selected VIN unit/placement locks in ascending VIN **ID** order; root batch locks in ascending batch ID order; both company-scoped warehouse locks in ascending warehouse ID order; then read/update location balances. **Every one of these acquisitions uses dedicated `FOR UPDATE NOWAIT` methods**, leaving existing blocking repository APIs unchanged for other callers. Map PostgreSQL SQLSTATE `55P03` to 409 `stock_busy` and immediately return out of the transaction; do not catch it and continue in the aborted transaction or retry internally. Revalidate visibility/placement, revisions and total capacity under locks before writing. Location mutations serialize on their root batch lock, including absent destination balances; use conflict-safe insert/upsert. Touch each warehouse/batch once, record per-VIN `vehicle.moved` and per-batch `receipt.quantity_moved` facts with original receipt ID, source/destination, quantity and occurrence date. No nested calls to separately committing single-move methods.

Identify and correction retain **batch → warehouse** ordering; they never acquire existing VIN locks. Creation locks warehouse then inserts a new independent batch/location. Align same-company single Move to unit/placement → warehouse with the same NOWAIT acquisition methods and preserve its request compatibility. Existing cross-company custody/consignment sale uses vehicle/placement/hold → warehouses. `Stock.Transfer` instead locks target warehouse before unit UPDATE, UpsertPlacement and source BumpByID; `Stock.Deliver` deletes placement before warehouse bump. A blocking bulk order would introduce wait cycles with these actual callers. NOWAIT on **all** move resource locks avoids adding a waiting edge, including mixed selections whose later unit is held elsewhere. Preserve reservation/ownership records and existing cross-company methods. Final check covers contention with target-first transfer as well as identification. No global deadlock claim about unrelated legacy flows; any remaining bulk wait/deadlock requires hub escalation, not a broad handover rewrite.

## Reading bounds, rules and approvals

Read root `AGENTS.md`, `docs/justix-auto/dev/agent-workflow.md`, relevant `internal/AGENTS.md` or `web/AGENTS.md`, assigned packet and this contract. Product references: business-logic `Vehicles, purchasing and stock`; product-workflow-improvements `R4`; task's explicit user steering. Open decisions OD-02/04/05/06/13 remain outside this same-company extension; stop if an implementation needs to resolve them. Historical Gaze references do not override ADR-14 or justify CQRS/messaging additions. Mock files are untouched design references, never policy. Large files: use the symbols named in each packet, not whole unrelated documents.

Coordinator proposal: snapshot/hash and amend `docs/justix-auto/contracts/http-domain.md` inventory subsection with the four HTTP changes, global/local count semantics, single-command preconditions and legacy defaults; record receipt-location schema invariants and local activation/rollback restrictions in the approved task or its accepted design reference. Application hub approves domain/schema; DevOps approves exact local migration/runtime activation. No production operational guarantee/dependency is added; no new ADR is needed unless either hub changes those guarantees. Slicer does not edit either canonical document.

Risks: migration/writer cutover; location/global count confusion; commerce order identification opened at original warehouse; pre-existing cross-company lock inversion; transport-uncertain retries; loaded selection above request cap; null historical colours; reservation/UI staleness. Source manifest pins inspected bytes, not proof of Git cleanliness. Base SHA is supplied by hub; source hashes must be compared before execution. Pause watchers before any edits; no incidental build/migration after BE1.

Gaps: no unresolved product rule found within movement scope. Hub must explicitly accept correction location semantics and HTTP/schema proposal before execution. Final runtime/fixture authorization and final immutable SHA are supplied by hub, not invented here. Final packet reports any unavailable scenario rather than asserting acceptance.
