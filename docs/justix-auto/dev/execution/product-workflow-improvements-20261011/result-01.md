# P01 result — Company identity, verification and setup access

Status: NEEDS_CONTEXT (implementation complete; required unit command could not initialize in the sandbox).

Changed paths:

- `internal/modules/identity/model/company.go`
- `internal/modules/identity/service/company.go`
- `internal/modules/identity/service/company_verification.go`
- `internal/modules/identity/service/company_test.go`
- `internal/modules/identity/service/company_verification_test.go`
- `internal/modules/identity/service/auth.go`
- `internal/modules/identity/repository/company.go`
- `internal/modules/identity/handler/company.go`
- `internal/modules/identity/handler/admin.go`
- `internal/modules/identity/handler/dto.go`
- `internal/pkg/auth/principal.go`
- `internal/pkg/auth/principal_test.go`
- `migrations/000035_company_verification.up.sql`
- `migrations/000035_company_verification.down.sql`
- `docs/justix-auto/dev/execution/product-workflow-improvements-20261011/result-01.md`

Implemented: nullable seller `userType` (`dealer`/`distributor`) that platform access holders may set once; unverified/submitted/verified/returned company verification with reason, reviewer actor/time, optimistic revisions, authorization and audit events; optional legal form, tax ID and bank requisites. New Admin seller provisioning requires `userType`; self-created sellers remain untyped/unverified and setup-accessible. Selected suspended companies are denied while draft companies remain selectable; `auth.Principal.AllowNewCommercialOperation()` returns forbidden `company_verification_required` until a selected seller is typed and verified, without changing ordinary permissions.

HTTP delta: `POST /identity/companies/:id/submit-verification` with `If-Match`; existing `POST /identity/admin/companies/:id/:action` additionally accepts `set-user-type` (`{userType}`), `verify` and `return-verification` (`{reason}`), all returning the current company DTO. Company DTO adds `userType`, verification/reason/reviewer metadata, `legalForm`, `taxId`, and `bankRequisites`; session accessible company and exported profile add type plus verification/reason.

Verification command run once: `bash tools/go.sh test -vet=off ./internal/modules/identity/service ./internal/pkg/auth`.

Result: blocked before compilation for both packages because Go could not read `/Users/bakhromachilov/Library/Caches/go-build/b7/b77518bad182c829b18a3bb1259c37867ab660515aee841a15ab1832563cc02d-d` (`operation not permitted`). No migration was executed and no test was rerun. `git diff --check` passed after formatting.

Limitations/exclusions: P03 must invoke the new commercial-operation helper at each new-operation entry point. No provider verification restriction was introduced, no existing data was inferred or changed by the migration, and no contract/canonical document was edited.

## Follow-up unit execution

The cache-isolated command `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/identity/service ./internal/pkg/auth` passed once: `ok justixauto/internal/modules/identity/service 0.990s`; `ok justixauto/internal/pkg/auth 1.062s` (2 packages, 0 failures). No repair was required. Final status: DONE.
