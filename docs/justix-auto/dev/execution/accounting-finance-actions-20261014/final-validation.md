# Final affected-unit verification

Date: 2026-10-14. Coordinator: /root.
Branch: fix/accounting-finance-actions-20261014.
Base: f958c783568b0f08c1854a9c7b6b4dc453b898f0.

Result: 22 backend top-level cases (plus 33 subcases) and 39 distinct frontend
cases passed. Repeated repair runs are not counted twice. No application-source
changes occurred after these checks started; repairs affected test fixtures only.
This is bounded unit verification, not release or live-browser acceptance.

## Backend

Run once from repository root, after both source packets were complete:

```sh
set -o pipefail
GOCACHE=/private/tmp/justixauto-accounting-go-cache \
bash tools/go.sh test \
./internal/modules/retail/service ./internal/modules/retail/handler \
-vet=off \
-run '^(TestInstallmentInvoices.*|TestInstallmentPlan.*|TestInstallmentTerms.*|TestContractSign.*|TestDealActionProjection.*)$' \
-count=1 -v 2>&1 | tee /private/tmp/justix-accounting-units-backend.log
```

PASS: service 17 cases (0.662s), handler 5 cases (1.065s). Covers saved monthly
amounts/dates, explicit issuance, removal of automatic issuance at all three
callers, financial permissions/scope, expected revision, duplicate and incomplete
prerequisites, invalid draft, atomic rollback, signed-term locking, legacy manual
plan authority and actor-dependent action projection.

## Frontend

Every run below used working directory `web`, `set -o pipefail`, and this prefix:

```sh
DEBUG_PRINT_LIMIT=1800 \
../docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node \
node_modules/vitest/vitest.mjs run --config vitest.workspace.ts --configLoader runner
```

The first affected-file pass selected:

```text
apps/portal/src/features/realization/accountant.test.tsx
apps/portal/src/features/realization/cash-sale.test.tsx
apps/portal/src/features/realization/installment-payment-review.test.tsx
apps/portal/src/features/realization/reservation-payment.test.tsx
apps/portal/src/routes.test.tsx
```

Result: 22 passed, 3 failed, 4.81s. Log:
`/private/tmp/justix-accounting-units-frontend.log`.

The second pass selected these files and exact case filter:

```sh
apps/portal/src/features/realization/installment-servicing.test.tsx \
apps/portal/src/features/realization/order-workspace.test.tsx \
-t 'shows draft terms in contract action|prefills a saved draft|locks the issued down payment|sends finance|retains finance|reviews a complete parent|retries missing complete parent|retains a parent decision|opens the same monthly invoice|keeps first-installment billing|posts 10.25|distinguishes invoice errors'
```

Result: 9 passed, 4 failed, 70 skipped, 7.23s. Log:
`/private/tmp/justix-accounting-units-flows.log`.

The seven failures were fixture issues: missing server action for an eligible
extra invoice, incomplete B2B order shape/action-form submission, outdated seller
guidance, two ambiguous background/modal review selectors, seller read-only action
label, and a missing session/router wrapper. Repairs retained the financial
assertions. One new meaningful monthly-command test covers both the Accounting
action with confirmation/revision and its absence in Realization.

Only those failures plus the new case were run again:

```sh
apps/portal/src/features/realization/accountant.test.tsx \
apps/portal/src/features/realization/cash-sale.test.tsx \
apps/portal/src/features/realization/installment-servicing.test.tsx \
apps/portal/src/features/realization/order-workspace.test.tsx \
-t 'gives the accountant surface|keeps B2B invoice issuance|keeps invoice issuance out|reviews a complete parent|retries missing complete parent|retains a parent decision|distinguishes invoice errors|issues a saved installment draft'
```

Result: 7 passed, 1 failed, 87 skipped, 1.89s. Log:
`/private/tmp/justix-accounting-units-repair.log`. The remaining B2B case observed
the command before queue invalidation completed. Its final disappearance assertion
was changed to await the existing asynchronous refresh; no source change.

Final one-case rerun:

```sh
apps/portal/src/features/realization/accountant.test.tsx \
-t 'keeps B2B invoice issuance'
```

PASS: 1 passed, 4 skipped, 1.58s. Log:
`/private/tmp/justix-accounting-units-b2b-final.log`.

## Boundaries and preservation

No lint, typecheck, build, vet/race, browser/E2E/integration, review/QA loop,
schema migration, runtime restart, live payment/user mutation or remote push.
The backend cache is isolated under /private/tmp. Existing role/cabinet behavior
is inherited from the dependency; this change completes financial-action scope.
All source writers and check leases are released for the local explicit-path
commit. The commit must use LEFTHOOK=0 to avoid unsolicited checks/generation.

Protected unrelated identity test SHA-256 remained
`a4d4b3f67107b5362810dcc96f68f1de8e59cae466dacafb7e100e29a59fe99f`.
`test_creds.txt` was not read, changed or staged. Final affected source and test
hashes are in [source-manifest.md](source-manifest.md).
