# P02 result — Company setup and verification UI

Status: DONE

Changed paths:

- `internal/modules/identity/service/company.go`
- `internal/modules/identity/service/company_verification_test.go`
- `web/packages/kit/src/session.tsx`
- `web/packages/kit/src/org-settings.tsx`
- `web/packages/kit/src/shell.tsx`
- `web/packages/kit/src/messages.ts`
- `web/packages/kit/src/company-verification.test.tsx`
- `web/apps/admin/src/pages.tsx`
- `web/apps/admin/src/company-verification.test.tsx`
- `web/apps/realization/src/pages/settings.tsx`
- `docs/justix-auto/dev/execution/product-workflow-improvements-20261011/result-02.md`

Implemented: session, company switcher and settings display the fixed seller business type and verification state. Realization settings now edit legal form, tax ID and bank requisites; unverified and returned companies retain setup access, show the advisory and return reason, and may submit or resubmit for review with a refreshed session. Admin seller onboarding offers optional dealer/distributor selection while retaining name plus first-admin login/password as sufficient input. Admin can set an untyped seller's type once, and can verify or return submitted profiles with a reason. The narrow P01 correction allows omitted `userType` at Admin seller provisioning, retains a selected valid type, and rejects a nonempty invalid value.

P01 HTTP seam consumed: seller provisioning accepts `{ company, firstAdmin, userType? }`; company/session fields are `userType`, `verification`, `verificationReason`, `legalForm`, `taxId` and `bankRequisites`; review calls are `POST /identity/admin/companies/:id/set-user-type`, `/verify`, `/return-verification`, and self-service submission is `POST /identity/companies/:id/submit-verification`, all with the company revision through `If-Match`.

Verification run after the repair:

- `npm --prefix web run test:unit -- packages/kit/src/company-verification.test.tsx apps/admin/src/company-form.test.tsx apps/admin/src/company-verification.test.tsx apps/realization/src/add-company.test.tsx` — passed, 3 files / 8 tests.
- `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/identity/service` — passed.

The first frontend invocation caught a test fixture expectation and a new test syntax error; both were repaired and the stated command was rerun successfully. No lint, typecheck, build, browser, integration, migration, runtime, Git or deployment action was run.

Evidence clarification: the rerun's Vitest report lists exactly three passing project files: `packages/kit/src/company-verification.test.tsx`, `apps/admin/src/company-form.test.tsx`, and `apps/admin/src/company-verification.test.tsx` (8 tests total). It does **not** list `apps/realization/src/add-company.test.tsx`. The workspace uses per-project roots and that Realization project's include pattern is relative to `web/apps/realization`; the root-relative positional path did not select it. The Realization suite is therefore unverified by this packet result and requires a separately authorized, Realization-project-targeted unit command; do not treat the 8-test result as coverage of `add-company.test.tsx`.

Authorized omitted-suite completion: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/add-company.test.tsx`, executed from `web/`, passed: 1 file / 1 test. The initial form without the `npm exec --` argument separator did not load the Realization project config and failed before test setup with `ReferenceError: document is not defined`; no source repair was needed. For remaining Realization packets, use the successful command form from `web/` (substituting the `src/...test.tsx` path), or invoke the generic root workspace command with a project-aware route. The generic `npm --prefix web run test:unit -- apps/realization/src/...` passes a root-relative file argument into per-project runners; Admin/Kit happened to resolve their files, while the Realization project (whose configured root is `web/apps/realization` and whose include is `src/**/*.test.{ts,tsx}`) omitted it.

Limitations/exclusions: P03 owns server enforcement at new commercial-operation entry points. No business-type mode switch was added; provider company flows, memberships, company switching, historical data and servicing remain available. Admin creation intentionally permits an untyped seller per the coordinator's confirmed user-requirement correction; that seller remains unverified until platform classification and verification.
