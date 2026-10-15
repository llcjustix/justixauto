# Final validation

PASS. Primary ran the two selected existing manual receipt cases once after the
full one-import fix; no test changes or broader validation. Both mount the real
ManualReceipt flow, submit with VIN / quantity without VIN, enforce single
submission and If-Match, and complete the successful refresh path.

Source: inventory.tsx SHA256
db6376894d06761486262832d0467f7fffbc26496820f5a8f620cc8a7a399ea4.
Diff: only useRefresh added to the existing @justixauto/kit import. Version
control must preserve this exact source in the final local commit.

Command (cwd web, repository-pinned Node):
../docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node node_modules/vitest/vitest.mjs run --config vitest.workspace.ts --configLoader runner apps/portal/src/features/realization/receipt-colors.test.tsx -t 'colors manual receipt .* pins one pair'

Result: 1 file passed; 2 tests passed, 11 unrelated tests skipped; exit 0;
duration 1.40s (test execution 448ms). Raw output in final-test-output.txt.

No browser, live API/database mutations, runtime restart, unit additions,
lint/typecheck/build or review/QA chain. Source/check leases released. This is
focused component evidence, not full warehouse/backend integration readiness.
