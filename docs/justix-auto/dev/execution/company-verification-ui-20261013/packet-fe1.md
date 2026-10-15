# FE1 — Company verification workflow visibility

Task: `company-verification-ui-20261013`; authority: `/root` (application hub).
Dependency/base supplied by hub: `6df3005b6410fd5aa820ed27d27bebaa97fc2bb0`;
dev ancestry: `4fa683b`. No Git verification performed by slicer.
Contract: `docs/justix-auto/dev/company-verification-ui-20261013.md`, approved
scope implementing R1/R2 of `product-workflow-improvements-20261011.md`.
Local acceptance IDs below refine this scope; they add no backlog tasks.

## Ownership and reading

Implementation owns exactly:
- `web/apps/portal/src/features/realization/pages/settings.tsx`
- `web/apps/portal/src/features/admin/pages.tsx`

Read root `AGENTS.md`, `web/AGENTS.md`, `docs/justix-auto/dev/agent-workflow.md`,
the approved scope, and relevant source slices. Hashes: `source-hashes.json`.
Context budget: 12,000 input tokens; target settings company helpers/Companies
and Admin Company/companyFields/companyInput/CompaniesPage/CompanyDialog.
If insufficient, stop for reslicing; do not expand canonical scope.

## Behavior and acceptance coverage

- **FE1-R1:** Put a prominent verification panel above Realization profile data.
  Explain unverified/submitted/verified/returned states; show return reason.
  Primary submit/resubmit appears only for unverified/returned with explicit
  `canEdit = session.can('company.edit')`; otherwise explain permission needs.
  Keep profile editing available according to existing permission and optional
  requisites optional; introduce no completeness gate.
- **FE1-R2:** Explicit refresh reloads company and session without logout.
  Await session refresh first, then company refetch: hub confirms session refresh
  retires in-flight HTTP requests; do not use `Promise.all`. Submission follows
  the same order. Use existing controls for pending,
  loading, fetch/mutation/refresh failures and retry; prevent duplicate actions
  and stale/missing-revision writes. A successful mutation followed by refresh
  failure must not encourage duplicate submission.
- **FE1-R3:** Seller Admin list has a dedicated verification column and all/four
  status filter, independent of operational access; reset clears both filters.
  Preserve non-seller lists, existing search and table states.
- **FE1-R4:** Seller detail contains a separate status/reason review panel.
  Verify/return are submitted-only and explicitly gated by
  `session.can('platform.companies.access')`; show permission guidance otherwise.
  Return requires reason. Explain missing `userType` and expose the existing
  permission-gated assignment action; do not invent assignment policy.
- **FE1-R5:** Show tax ID/bank requisites; preserve their exact existing values
  on unrelated Admin edits. Keep legal-name/registration preservation intact.
- **FE1-R6:** Verification remains separate from operational access. Never claim
  verified alone opens commerce. Preserve routes, status vocabulary, revision
  checks, existing backend policy, modal patterns and keyboard access. No new
  dependencies, shared-component changes or mandatory fields.

## Execution and final verification

One implementation writer/main-checkout lock; run `bash tools/check-git.sh`
before code edits. Version-control role owns Git. No push/restart/deployment.
After the complete change, primary alone performs one isolated-fixture browser
pass at the final candidate SHA: submit → Admin return/reason → resubmit →
verify → same-session refresh; cover R1–R6, permission-denied/missing-userType,
loading/failures, filter reset and unrelated-edit preservation. Reserve browser
and fixture exclusively. This checks the shared company/session/list interface;
live backend authorization is excluded. No unit writing/runs, lint, build,
typecheck or automatic review/QA chain.

Stop on contract/hash drift, undecided product rules, ownership expansion or
resource conflict; report `RESLICE_REQUIRED` for canonical-scope changes. Preserve
protected `internal/modules/identity/service/company_contract_test.go`; never
read `test_creds.txt`. No other records, credentials or live DB edits.
