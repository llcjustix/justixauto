# Insurance label and independent contract signing

Approved primary task: ../../scoring-guidance-20261014.md.
Repo /Users/bakhromachilov/startups/justixauto; branch fix/scoring-guidance-20261014;
base08c2efbc72877644f653c0c8cf8fe5af31f60ca3. Root, internal/AGENTS.md and
web/AGENTS.md apply. Latest user decision 2026-10-14 supersedes mandatory R12
scoring before signing. OD-08 and handover requirements remain unchanged.

Source ownership:
- web/apps/portal/src/features/realization/pages/retail.tsx
- web/apps/portal/src/features/realization/sale-insurance.tsx
- web/apps/portal/src/features/realization/contract-workspace.tsx
- internal/modules/retail/service/contract.go
- internal/modules/retail/service/scoring.go (stale comment only)
- Existing affected assertions in retail/service/scoring_test.go,
  retail/service/contract_test.go, realization/sale-insurance.test.tsx and
  realization/installment-servicing.test.tsx. No new test suite.

1. Remove the demoScoring condition and mandatory warning from
   DealDialogActions; retain allowedActions and unsigned-contract conditions.
2. Remove the scoring-derived canSign prop from the retail ContractWorkspace.
   Inspect callers; remove unused scoring-only canSign support or use a neutral
   fallback. Keep commerce compatible, permissions and signed-version behavior.
3. Rename tab and tabpanel accessible names to «Страхование».
4. In SaleInsurance remove mandatory-signing language; optional scoring may
   retain its present labels and explicit demonstration disclaimer. No extra
   navigation callback, score, insurer decision or automatic action.
5. Remove only currentDemoScoring/demo_scoring_required from Contract.signDraft;
   both versioned Sign and legacy RecordContract use it. Preserve every other
   validation, the transaction and installment finalization. Do not modify
   InsuranceApproved or handover, invoice or payment rules. Update stale
   mandatory-scoring service comment.
6. Adapt existing tests that enforce mandatory scoring. Demonstrate first
   own-installment signature without scoring/insurance and preserved schedule
   finalization; exercise legacy entry if available in an existing case.
   Adapt existing frontend contract-action case to an unscored deal. Primary
   runs affected existing tests once after all edits. Do not run them here.

Run bash tools/check-git.sh before code edits; use apply_patch. No Git,
canonical docs, runtime, test execution, build/lint/typecheck or browser work.
Write compact result/source hashes in this directory and release source lease.
