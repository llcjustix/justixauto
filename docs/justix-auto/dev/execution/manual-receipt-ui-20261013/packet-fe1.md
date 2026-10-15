# FE1 — Manual receipt presentation

Task/contract: `docs/justix-auto/dev/manual-receipt-ui-20261013.md`; hub `/root`.
Base: `d5622eb5470a5e7e296dd567d79b1f34bce58f2d`.
Branch: `fix/manual-receipt-ui-20261013`; dependency: version_control prepares branch.
Context ceiling: 12,000 tokens; read mapped slices in `source-hashes.json`.

Owned source paths:
- `web/apps/portal/src/features/realization/pages/inventory.tsx` (ManualReceipt/import only)
- `web/apps/portal/src/features/realization/pages/manual-receipt.css` (new)
- `web/apps/portal/src/features/realization/vin-editor.tsx` (minimal presentation prop)

Rules: root/web AGENTS; root final-only verification supersedes historical gates.
Run `bash tools/check-git.sh` before source edits. Reserve exclusive checkout writer;
primary reserves browser only after completion. No dependencies, Git, runtime,
backend, canonical-document or unrelated-file changes.

Acceptance coverage (local requirement IDs, corresponding task paragraphs):

- MR1/presentation: `.manual-receipt` bounded wrapper, grouped model/specification/colors,
  receipt method/vehicles; responsive two-column/single-column layout. Preserve shared
  Autocomplete search, labels, keyboard access, loading/errors/retries. Scope CSS;
  never override Autocomplete internals. Radios “С VIN”/“Количество, VIN позже”,
  initial VIN mode. Style labelled native `datetime-local` without changing semantics.
  In quantity mode, style quantity/date as a consistent two-column row stacking
  on mobile; both modes receive complete presentation treatment.
  Back/primary footer; suppress missing-facts summary until model/colors exist.
- MR2/VIN: opt-in empty-required-message prop defaults to current behavior.
  ManualReceipt alone shows neutral pristine guidance, reveals required error after
  VIN interaction or submit attempt. Preserve required legend, validateVins,
  invalid/duplicate/over-cap feedback, add/remove/paste/paging and drafts.
- MR3/preservation: retain exact specification/color selection, quantity, date ISO
  conversion, payload, warehouse revision, commercial guard, busy/duplicate protection,
  useRefresh, conflict refresh, success handling, warehouse context/single dialog.
  Other VinEditor consumers retain existing defaults.
- MR4/verification: after all source changes, primary performs one focused final-SHA
  browser pass with actual Vite/isolated synthetic API: pristine/interacted/submitted
  empty VIN; model/colors; both modes; invalid/duplicate/paste; desktop/mobile/date;
  request shape when date control permits; default required receipt-identification
  and optional order-fulfilment consumers. Record SHA/source hashes and limitations.
  No new/unit tests, broad checks or automatic review/QA chain. Repair/rerun only
  failed scenarios.

Stop on undecided product rules, source drift, unavailable branch/locks, or changes
outside owned paths. Report RESLICE_REQUIRED if scope/context must expand; do not
derive policy from fixtures. Hub accepts packet; no canonical backlog additions.
