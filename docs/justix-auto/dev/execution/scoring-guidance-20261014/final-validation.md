# Focused final validation

Primary `/root`, 2026-10-14. Branch fix/scoring-guidance-20261014;
base08c2efbc72877644f653c0c8cf8fe5af31f60ca3. Source hashes in result.md.

The latest user decision supersedes the former mandatory demo-scoring rule.
The sale tab/tabpanel is «Страхование». Legacy contract recording and versioned
draft signing no longer check scoring in the UI or shared server signDraft.
Permission, signed scan, tenant, revision, immutable-version, current-terms,
transaction and schedule-finalization behavior remains. Handover insurance
approval is not changed. Optional scoring remains a demonstration, not coverage.

After implementation, seven affected Go service tests passed once (0.586s),
including eight rollback/guard subcases. The first sandbox invocation did not
execute tests because the existing Go cache was not writable; the same bounded
command succeeded with cache access:

```sh
bash tools/go.sh test ./internal/modules/retail/service -run '^(TestContractSign.*|TestContractAmendmentAndReadGuards|TestDemoScoringDoesNotBlockFirstOwnInstallmentSignature|TestDemoScoringExcludesCashAndAmendments)$' -count=1 -v
```

From web, using the repository-pinned Node:

```sh
../docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node node_modules/vitest/vitest.mjs run --config vitest.workspace.ts --configLoader runner apps/portal/src/features/realization/contract-workspace.test.tsx apps/portal/src/features/realization/sale-insurance.test.tsx apps/portal/src/features/realization/installment-servicing.test.tsx -t 'contract workspace|sale insurance|shows draft terms in contract action'
```

Five frontend cases passed immediately. The sixth reached and submitted the
contract action without scoring, then failed on invoice issuance because its
old session fixture supplied no invoice-management permission. The fixture now
accepts optional permissions, defaulting to the unchanged empty list; only this
invoice-capable case receives retail.invoices.manage. Application gates were
not changed. Reran only that affected case with the same command restricted to
installment-servicing.test.tsx and `-t 'shows draft terms in contract action'`:
one passed, 72 unrelated skipped, 1.47s. Six selected frontend cases now pass.

No new tests, broad suites, lint/typecheck, test builds, browser/E2E, integration
or review/QA loops. The server unit tests cover shared signing; the legacy
recording entry point was source-traced to the same service, not HTTP-tested.
Local API reload is separately covered by standing local-run authorization and
runtime-result.md. No sale/payment mutation, new migration, remote push or
production operation. Source/check leases released. Unrelated identity test
SHA256 a4d4b3f67107b5362810dcc96f68f1de8e59cae466dacafb7e100e29a59fe99f
is preserved; test_creds.txt remains unread and untouched.

The user's follow-up asks where submitted installment payments are confirmed.
Current source path: /billing → «От клиентов» → «Очередь бухгалтера» →
«Проверить весь чек» → «Деньги поступили» → «Принять весь чек».
The company accountant role includes retail.payments.accept. This is guidance
from existing source, not a new payment action or confirmation on the user's behalf.
