# Focused final validation

Primary `/root`, 2026-10-14. Source accepted for this bounded frontend correction.
Base20cbd41a9961a005b887e646e2d6e8a27d555fcb;
branch fix/installment-price-validation-20261014.
Only application source: web/apps/portal/src/features/realization/installment-terms-form.tsx,
SHA256 ec04dacd9f977ae623493abe6fd2b191e92d6d5a00ad0fc5433199603329aef3.

The diff validates positive price before percent conversion, derives displayed
amount from current price/current percentage using existing exact helpers and
simplifies synchronization help. No financial helper, precision, backend,
currency, invoice or saved schedule contract changed.

After the full change, primary ran three existing affected component cases once:
live price/down/term/date updates; exact percent/money-origin input; both inputs
locked by an invoice-fixed amount. Result: one file passed,3 passed,10 unrelated
cases skipped, exit0,1.82s (tests485ms). No new unit tests or test edits, broad
suite, lint/typecheck/build/React Doctor or review chain.

Command from web with repository-pinned Node:

```sh
../docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node node_modules/vitest/vitest.mjs run --config vitest.workspace.ts --configLoader runner apps/portal/src/features/realization/sale-form.test.tsx -t 'updates price/down/count/date live|uses a percent-first exact down payment|locks both synchronized down-payment fields'
```

CUA first opening attempt hit an automatic approval-review timeout, not a stated
safety rejection. Reconnection succeeded; the fresh owned Chrome tab at
http://127.0.0.1:5191/sales showed login. No credentials were read or entered;
the tab was closed. Therefore empty-price30% and percent-origin price changes
were not separately exercised in a browser in this pass. These fixes are bound
by source inspection; the three existing tests cover related shared behavior,
not a new regression case for the exact screenshot. No visual/E2E or release
readiness claim is made.

Vite remains the user's existing local service; no restart or data mutation was
needed. Source/check/browser leases released. Local scoped commit only; retain
unrelated identity test and unread test_creds.txt.
