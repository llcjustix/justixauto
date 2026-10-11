# P01 — Company identity, verification and setup access

Task: product-workflow-improvements-20261011; requirements R1/R2. Authority: application hub `/root`. Base `4fa683ba4708bb71031534260ec5cb8742ce9977`; branch `feature/company-commerce-workflows-20261011`. Ready; no predecessor. One source-writer lock, no runtime resources. Context ceiling: 20k tokens; read only listed files and relevant functions. Stop and split if exceeded.

Read root AGENTS.md, internal/AGENTS.md, dev/agent-workflow.md; approved feature/interview records; HTTP contract identity/session/company and errors sections. Latest user decision relayed by hub: existing sellers are manually classified and verified in Admin, new operations remain blocked meanwhile; history and servicing existing obligations remain accessible. This supersedes the interview's pending legacy question.

## Owned paths

- `internal/modules/identity/model/company.go`
- `internal/modules/identity/service/company.go`, `company_verification.go` (new), `company_verification_test.go` (new), `company_test.go`, `auth.go`, `auth_company_test.go` (new)
- `internal/modules/identity/repository/company.go`
- `internal/modules/identity/handler/company.go`, `admin.go`, `dto.go`
- `internal/modules/identity/module.go`
- `internal/pkg/auth/principal.go`, `principal_test.go` (new)
- `migrations/000035_company_verification.up.sql`, `migrations/000035_company_verification.down.sql` (new)

No other paths. New permission keys are unnecessary if existing company.edit/platform.companies.access authority is reused. If a new permission is required, request explicit scope extension to identity/model/permissions.go and migration catalog rows before editing.

## Behavior and interfaces

1. Preserve Company.Kind seller/bank/mfo/insurance. Add nullable seller `userType` dealer/distributor; users/memberships retain independent job roles. Admin seller provisioning permits omitted type: original user requirement makes company name and first-admin login/password sufficient. Admin may optionally choose type at onboarding, or assign it later before verification/new commerce. Add Admin versioned set-type action: missing value can be filled once; changing a set value fails, including company profile update bypass. Company self-creation remains available but does not let its creator assign the platform-controlled type.
2. Separate verification status (unverified/submitted/verified/returned), reason and actor/time metadata from access draft/active/suspended. New companies can authenticate/select company and configure company, branches, warehouses and staff. Do not weaken suspended/deleted user/company access. Inspect `Deps.isMember` and `Auth.SetCompany` before changing selection; if another file is needed, stop for scope addition.
3. Migration leaves every existing seller userType unset and verification unverified; never infer from current access or fixtures. Preserve access/history data. New company service paths must yield setup access, including Admin provisioning and CreateSeller. Provider kind stays separate; do not require a dealer/distributor type for providers.
4. Company profile/settings accept optional legal form (llc/sole_proprietor/other), tax ID and bank requisites alongside existing registration/contact/address. Do not invent mandatory verification documents or jurisdiction formats. Company admin submits/resubmits; platform admin verifies or returns a nonempty reason. These transitions are versioned, authorized in service and audited. Keep Admin onboarding minimal.
5. Add DTO fields to `companyDTO`, session `AccessibleCompany`, and exported `Profile`: userType + verification/reason and profile fields where appropriate. `Authenticate` loads current selected company's status each request into a minimal `auth.Principal` commercial-access capability. Introduce explicit `AllowNewCommercialOperation()` helper producing a stable 403 `company_verification_required` (missing seller type also denied). Existing `Allow(permission)` retains its current meaning, preserving reads and obligation servicing. P03 wires this explicit helper into new-operation entry points; this packet alone does not complete the R2 boundary.
6. Routes: versioned company submit-verification and Admin company set-user-type/verify/return-verification actions; use existing `POST /admin/companies/:id/:action` dispatcher without route collision, existing envelopes/If-Match and field errors. Return current company DTO. Keep access activate/suspend actions distinct. Do not modify approved contract documents; report implemented DTO/route delta to coordinator.

## Acceptance / affected unit tests

Unit fakes only: fixed type set once; invalid/provider type rejected; non-platform actor cannot set/verify; minimal onboarding gives setup-accessible unverified company; submit→return(reason)→resubmit→verify; stale version produces conflict; suspended/deleted access remains denied; session capability reflects fresh verification; verified seller with type permitted; unset/unverified seller new-operation helper denied without removing ordinary permissions. Verify profile fields survive update while registration preservation remains intact.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/identity/service ./internal/pkg/auth` (only pure unit packages; no module-root flow tests, DB, lint, vet/race, build or integration commands). Do not execute migration. Changes in existing unit fixtures may be limited to owned test files; request additional exact paths if necessary.

Stop conditions: uncertain provider verification semantics must not produce new provider restrictions; do not infer field requirements, legal approval or dealer/distributor operation exclusivity. If setup access requires broader source ownership, report exact symbol/path. Before application edits worker runs `bash tools/check-git.sh`; no Git writes by worker. Return changed paths, route/DTO delta, affected unit evidence and limitations to hub.
