# Company verification workflow visibility — 2026-10-13

Status: SOURCE COMPLETE; final browser validation passed. Application authority: /root.
User reports no way to submit a completed company profile in Realization or
review it in Admin. R1/R2 of product-workflow-improvements-20261011.md apply.

Existing backend has submit-verification and platform verify/return-verification
commands with permission, revision and state checks. Current frontend places a
small conditional submit button below profile facts; Admin review is hidden in
the generic company dialog footer and omits tax/bank details. The actual user's
session is not available in the attached browser, so the precise unseen state
is not established. Fix discoverability and complete the visible state flow.

Approved scope: a prominent Realization verification panel before company data,
primary submit/resubmit, pending/verified/returned explanations and return reason,
explicit refresh that updates company data and session without logging out,
permission/loading/error states. Keep company editing available. Admin seller
list gets a dedicated verification column/filter; company detail gets a separate
review panel with status and submitted-only verify/return actions, permission
guidance, and tax/bank details. Preserve those details on unrelated Admin edits.
Keep operational access separate; no auto-approval or new mandatory requisites.
No backend policy, schema, credentials, real company verification or live DB edits.

Source scope: web/apps/portal/src/features/realization/pages/settings.tsx and
web/apps/portal/src/features/admin/pages.tsx. Prefer existing shared controls and
styles; no new dependencies. Latest final-validation policy applies: finish the
whole change, then one focused browser pass using an isolated local fixture API
against the actual portal source. No unit additions/runs or automatic broader
checks/review chains. Evidence excludes live backend authorization.

Dependency: 6df3005b6410fd5aa820ed27d27bebaa97fc2bb0, with dev ancestry 4fa683b.
Task branch to be created directly at identical current source, preserving the
running app. Main checkout only. One worker, version_control owns Git. Preserve
dirty internal/modules/identity/service/company_contract_test.go and unread
test_creds.txt. No push, deployment, runtime restart or migration.
Execution directory: execution/company-verification-ui-20261013/.

Completion: both workspaces expose the verification state/action flow. Admin
shows an independent filter/column, review details and permission-gated actions;
confirmation requires the existing business type first. Client refresh updates
session then profile and reports loading/errors. Admin edits preserve user-filled
tax/bank details. Existing backend commands and revision checks reused.
[Final browser evidence](execution/company-verification-ui-20261013/final-validation.md)
covers submit -> return -> edit -> resubmit -> verify, permissions, filtering,
refresh/retry and field preservation using isolated fixtures on actual source.
No unit tests or real backend mutations; no runtime restart/push. All source and
browser leases released; unrelated dirty files preserved. Completion snapshots:
execution/company-verification-ui-20261013/entry/completion/.
