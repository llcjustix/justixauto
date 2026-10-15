# FE1 — restore manual receipt hook binding

Task/contract: `docs/justix-auto/dev/warehouse-receipt-refresh-20261013.md` (APPROVED). Hub: `/root`. Base: `239dcfda1c59846eac8a67d24680acc93613d759`. Branch: `fix/warehouse-receipt-refresh-20261013`.

Owned source: `web/apps/portal/src/features/realization/pages/inventory.tsx` only. Add `useRefresh` to its existing named `@justixauto/kit` import. Existing `kit/src/index.ts` exports shell; shell exports this hook. No contract changes.

Acceptance coverage: R1—ManualReceipt resolves its existing hook call without ReferenceError; R2—identified/unidentified submission, duplicate guard, revision and successful refresh remain intact; R3—only this import changes, retaining all inputs, permissions and semantics.

Dependencies: VC-prepared branch; supplied base source hashes. Lock: exclusive main-checkout source writer; no concurrent branch changes. Read root/web AGENTS and task; context cap 4,000 tokens. Before source editing, run `bash tools/check-git.sh`. Preserve unrelated changes and unread `test_creds.txt`. No Git writes, runtime changes, dependencies or new tests.

Final verification: after implementation, primary runs exactly existing `receipt-colors.test.tsx` cases matching `colors manual receipt .* pins one pair` at final candidate SHA (R1/R2); inspect one-import diff (R3), record source hash/output. No automatic review/QA chain or broad checks.

Stop: source drift, missing export, wider fix, product ambiguity or ownership conflict; notify hub.
