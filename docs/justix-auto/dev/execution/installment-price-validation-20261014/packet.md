# Bounded frontend correction

Primary-approved task: ../../installment-price-validation-20261014.md.
Base20cbd41a9961a005b887e646e2d6e8a27d555fcb;
branch fix/installment-price-validation-20261014 in the main checkout only.
Read root AGENTS.md and web/AGENTS.md, product R8 in
../../product-workflow-improvements-20261011.md; OD-09 unchanged.

Own source: web/apps/portal/src/features/realization/installment-terms-form.tsx.
Read adjacent installment-percent.ts, installment-calculator.ts, sale-form.tsx
and installment-servicing.tsx only as needed. Do not change financial helpers,
saved contracts, dependency files, tests, backend or unrelated source. If scope
needs expansion, report the specific reason before writing.

1. previewTerms must identify invalid/missing price before attempting percent
   conversion, using a plain Russian price instruction. An empty price should
   give contextual guidance, not falsely mark valid30% invalid.
2. Derive the displayed down-payment amount from the current percentage and
   current price in percent-origin mode, using existing BigInt helpers. A
   percentage can be entered before price, retained through price clearing,
   and recalculates when price changes. Do not use effects or floating-point
   arithmetic to round-trip the user input.
3. Money-origin input must remain exact; editing amount switches origin as
   before. Invoice-fixed down payments remain locked and authoritative.
4. Replace precision/rounding implementation prose with a short explanation
   that percentage and amount are synchronized. Keep field labels and approved
   validation/calculator boundaries compatible with existing consumers.

Run bash tools/check-git.sh before source edits. Use apply_patch. No tests,
build, lint, typecheck or browser until the finished source is handed back to
primary. No Git/runtime/canonical document writes. Write a compact result and
source SHA256 in this execution directory; release source lease afterward.

Primary final coverage:30% before price gives price guidance;10000→3000,
20000→6000; clearing and re-entering price retains30%; exact money-origin
amount; real invalid percentage remains blocked; invoice-fixed fields still
read-only. Prefer one small browser draft flow; no real sale submission.
