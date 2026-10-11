# P13/P15 result — DONE

Preflight passed on `feature/company-commerce-workflows-20261011`; active P14
backend/doc changes were preserved. This packet changed only its approved
Realization/kit union and this result artifact. The source writer is released.

## R9 — contracts

`contract-workspace.tsx` supplies owner-specific retail and commerce adapters:
retail sends `body`/`bindingIds` with the deal revision; commerce sends
`content`/`fileIds` with the order revision. It supports company plain-text
templates, draft preview/history, private authenticated file links, same-ID
draft signing, and a separate amendment flow against the selected signed parent.
Signed versions are labelled read-only and render structured parties, vehicle,
price and payment facts alongside the filled text rather than raw snapshot JSON.
The workspace is a tab in both `DealDialog` and `OrderDialog`; existing payment,
invoice and navigation tabs remain available.

## R11 — insurance

`InsuranceCase` is now the embedded kit body/action view; `InsuranceDialog`
retains its previous wrapper API for the existing cabinet. The own-installment
sale tab creates/selects an insurer application and renders its status/history
and server-allowed seller/provider actions without navigating away or nesting
the full insurance dialog. Server `allowedActions` remains the role gate, so a
seller cannot approve its own request.

## R12 — demo scoring

The own-installment sale tab shows the persisted `demoScoring` state, calls the
no-body scoring endpoint with the current deal `If-Match`, and labels it as
demonstrative only: no coverage or provider approval is asserted. Missing and
stale (`current:false`) runs remain visible; signing controls and the retained
legacy record-contract CTA require a current run. Cash, existing history and
amendment paths are not retroactively blocked.

## Unit evidence

From `web`, once after the final relevant changes:

```text
npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/contract-workspace.test.tsx src/sale-insurance.test.tsx
```

PASS — 2 files, 2 tests. No browser, build, typecheck, lint, React Doctor,
review/QA, Git operation, migration/runtime action, or deployment was run.
