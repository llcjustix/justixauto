# Warehouse bulk movement — 2026-10-13

Status: COMPLETE LOCALLY, 2026-10-14. User reports warehouse VIN rows cannot open
movement, states moving one at a time is inconvenient, and explicitly includes
quantities without VIN. Authority /root; continuation of product-workflow-improvements R4.

Deliver a visible movement action within the warehouse inventory dialog. Select
several/all loaded VIN vehicles and quantities from one or more unidentified
batches, choose a different warehouse of the same company and an occurrence
date, then submit one atomic command. Any invalid/stale/capacity/permission case
rejects the entire command. No loops of independent single-VIN HTTP writes.
Keep existing single-VIN movement compatible and warehouse-to-vehicle navigation
available. Preserve ownership, custody, specification, colours, receipt provenance
and order association; movement does not create purchases or change payments.

Latest user steering asks automatic VIN selection versus manual. Primary proposal
in commentary: both are available. Manual selection plus explicit 'select N'
chooses the first N VINs in ascending order; show the resulting list before
confirmation and allow edits. Never silently substitute IDs at submission.
Unidentified quantities are entered separately per source batch.

Pending VIN quantities may be partially moved. Source/destination occupancy and
remaining identification quantities must stay consistent. Destination VIN entry
must remain possible and tied to original receipt/order. Do not reinterpret
movement as quantity correction or overwrite an original received total.
Use existing inventory.vehicles.move permission and same-company boundary;
preserve error messages, drafts and safe retry. Selection UI must explain missing
destination/access and have keyboard/mobile support with one active modal flow.

Implementation design/contract is delegated to bounded slicer packets, accepted
by primary before writing. Backend then frontend serialized in main checkout;
no worktrees. Branch fix/warehouse-bulk-move-20261013 starts at current dependency
4b92239779c5f3be1f7ea563123d8a829aee2fbe to retain user's local changes.
Version_control owns Git; preserve unrelated company_contract_test.go and unread
test_creds.txt. Local commit only, no push/deploy.

Root/internal/web AGENTS apply with final-only verification. Finish full feature
before tests. No unit-test additions/runs per packet, broad checks or automatic
review/QA chain. Final focused HTTP/browser E2E on disposable data should cover
mixed/partial movement, rollback, capacity/stale/tenant/permission boundaries,
later VIN identification and UI selection/success. Any schema/runtime activation
must use a scoped local plan; never reset or mutate user data for testing.

Execution records: execution/warehouse-bulk-move-20261013/.

Accepted implementation contract: execution/warehouse-bulk-move-20261013/design.md
and approval.md. Keep global receipt counts; add per-warehouse unidentified
balances via migration 46. Explicit source/destination/batch revisions protect
one mixed atomic stock-moves command. Dedicated NOWAIT move acquisitions prevent
waiting cycles with existing custody paths. Correction delta is confined to an
explicit warehouse, default original. BE1 → BE2 → FE, then primary final pass.
Air was paused during implementation; the old API drained before local balance
backfill. Local activation is complete: private backup validated, clean schema46,
Air resumed once, new API8090 ready with the exact HTTP-verified binary, and
Vite5191/PostgreSQL preserved. The initial process-exit polling interruption and
bounded continuation are retained in the runtime evidence.

Final acceptance: execution/warehouse-bulk-move-20261013/final-validation.md.
Seven focused HTTP cases and primary CUA manual/all/first-N selection, mixed
movement, stale draft preservation, mobile layout, destination VIN identification
and original-order location choice passed. One replay precedence defect was
repaired with only affected continuation rerun. No broad suites or production
claim. Runtime/cleanup receipt: execution/warehouse-bulk-move-20261013/runtime-result.md.
Application hashes: execution/warehouse-bulk-move-20261013/final-source.json.
Canonical before-copies: execution/warehouse-bulk-move-20261013/entry/completion-manifest.json.
Local commit only; preserve unrelated changes and do not push.
