# P13/P15 completion — DONE

Source writer release: `feature/company-commerce-workflows-20261011` at the
granted base `ab088d97054fbabac65caae1f3f8bfe7f71fbe74` (no Git operation).

## R9 contract workspace

`ContractWorkspace` now sends `bindingIds` to both owner APIs while continuing
to read owner response bindings, patches existing drafts with their contract
revision, preserves external bindings unless a replacement is uploaded, allows
company template edits with `If-Match`, renders filled immutable signed
snapshots, and uses a signed-parent selector for amendments. Commerce controls
use `commerce.trade` plus the existing visible-order party rather than invented
actions. Signed terms use only the version snapshot price.

## R11/R12 embedded sale view

`sale-insurance.tsx` contains the own-installment sale tab. It retains the
embedded kit `InsuranceCase` for server-authorized creation/status/request and
response actions, while demo scoring remains visibly separate from insurance.
The run action uses the deal revision, prevents duplicate work while busy,
shows a refresh/revision error, and invalidates current sale/insurance queries.
It requires the real `retail.deals.manage` permission. First signing remains
gated by server-projected current demo state; signed-history amendments are not
subject to a new run.

## Unit evidence

- Initial combined command (from `web`) ran both specified files once:
  `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/contract-workspace.test.tsx src/sale-insurance.test.tsx`.
  `sale-insurance.test.tsx`: PASS, 2 tests. Its interaction units prove the
  explicit no-coverage label, stale scoring failure feedback, and lack of both
  score/insurer-approval controls without the seller permission.
- The same initial run exposed two ambiguous test-library selectors in the new
  contract assertions; no product behavior failed. After correcting only those
  assertions and adding retail PATCH/preserved-binding coverage, the affected
  selection ran once:
  `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/contract-workspace.test.tsx`.
  PASS, 3 tests: commerce request `bindingIds` (never response `fileIds`),
  immutable signed snapshot price/read-only rendering, and retail PATCH with
  draft revision plus retained binding IDs.

Excluded: browser/runtime/integration checks, lint, typecheck, build, React
Doctor, review/QA, backend/database changes, Git actions, and deployments.
