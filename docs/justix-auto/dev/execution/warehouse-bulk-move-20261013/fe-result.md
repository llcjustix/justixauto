# FE — Warehouse mixed stock movement result

Status: DONE (implementation handoff; no verification command was run).

Implemented the single-modal warehouse stock movement flow. It exposes the
permission-gated entry, destination/capacity choice, occurrence date, explicit
VIN selection (manual, all loaded, or first N by VIN then ID), visible editable
selection, and per-location quantities for receipt batches. One POST uses the
accepted mixed-command contract and captured revisions. Pending commands freeze
controls; stale/capacity/placement/busy facts require explicit reconfirmation
without replacing the selected IDs. Validation retains the draft, unknown
transport results are never retried automatically, and a committed result closes
the move view even if refresh cannot complete.

Receipt identification now reads the global receipt locations, explicitly lets
the user choose a positive local balance, sends `warehouseId`, caps VIN entry by
that local balance, and displays receipt-wide totals separately. Warehouse stock
rows show local and global pending counts; correction carries the explicit
current `warehouseId` and explains its receipt-wide meaning. Existing receipt
color/specification drafting remains in the shared editor.

Integration corrections preserve every selected VIN and entered batch quantity
through refresh until the user explicitly removes it or confirms refreshed
data. Invalid quantities are rejected rather than omitted, first-N is bounded,
the current local balance is enforced, and the refreshed destination revision
comes from the actual reload result. The parent modal cannot close during a
pending move; VIN buttons replace the warehouse dialog with the existing vehicle
dialog. The local time default, plain-language copy, correct refresh keys, and
an explicit empty receipt-location choice are also covered.

Changed paths and SHA-256:

- `web/apps/portal/src/features/realization/pages/inventory.tsx` — `afcaae518c89e63db641f2794a9b29f9b306492d8c09e5a95af233a347e6b3df`
- `web/apps/portal/src/features/realization/warehouse-move.tsx` — `921508ceb3eef15ba10c729c972a676164263e93b36bce341e40715bf451e1f6`
- `web/apps/portal/src/features/realization/warehouse-move.css` — `11c95779cd8a077dd64020d5876a8ebc5dad82846c4b62b2781f901e4772ff22`
- `web/apps/portal/src/features/realization/receipt-identification.tsx` — `932001b58b5baf5f861f8b8da0673227d6eb543d2c435410d9e5fd20e8aebc5a`

Excluded: tests, build, lint, typecheck, browser/runtime, Git writes, backend,
canonical documents, shared kit/global CSS, and credentials. `bash tools/check-git.sh`
passed before source edits. Source-writer lease: released.
