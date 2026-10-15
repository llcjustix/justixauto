# Final affected-unit verification

Date: 2026-10-14. Root coordinator.
Branch: fix/accounting-queue-ui-20261014.
Base: a99f72c7c72463943a96f15cae049d5d47630f18.

Result: 20 affected existing cases passed once, 2 files, 1.62s. No test failures
or reruns. Source was frozen after the queue presentation and switcher packets;
no source/test edits after this run. Only two copy-sensitive assertions in the
existing accounting case were adapted before the run; no new cosmetic tests.

Command, working directory `web`:

```sh
set -o pipefail
DEBUG_PRINT_LIMIT=1800 \
../docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node \
node_modules/vitest/vitest.mjs run \
--config vitest.workspace.ts --configLoader runner \
apps/portal/src/features/realization/accountant.test.tsx \
packages/kit/src/portal.test.ts \
2>&1 | tee /private/tmp/justix-accounting-queue-units.log
```

Accounting cases retain grouped-payment deduplication and accept-command coverage,
B2B issuance with revision and queue refresh, outside-Accounting read-only
controls, and retail/commerce query boundaries. Portal cases cover permitted
entry, remembered section, direct links, company choices and existing switchers.
These are affected existing behavioral cases, not new tests of CSS or a new
snapshot assertion for reciprocal switcher filtering.

Source inspection: queue uses local prefixed CSS, semantic lists/headings,
min-width/wrapping and a narrow-screen stack. Monetary fields use existing money
formatting and accurate labels. Optional vehicle snapshot is guarded, and no
new queries were added. Sidebar display filtering leaves available sections,
admission/default/remembered routes and company-choice data unchanged.

Limits: no browser/E2E or rendered visual validation, lint/typecheck/build,
React Doctor, independent review/QA loop, runtime/schema/live-data action or
remote push. No backend/source model/financial-policy changes. Source/check
leases are released; commit uses explicit paths and LEFTHOOK=0.

Final source digests: [source-manifest.md](source-manifest.md). The protected
identity test remains modified/unstaged with unchanged SHA-256
`a4d4b3f67107b5362810dcc96f68f1de8e59cae466dacafb7e100e29a59fe99f`.
Untracked test_creds.txt remains unread and untouched.
