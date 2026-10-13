# Packet 02 — eight-character password minimum end to end

Task: `local-ui-password-fixes-20261013`; authority and acceptance: `/root`.
Base: `31b018ce1a4b49a2f56148459b4efa0893d345ce` plus accepted Packet01 checkpoint
on `fix/portal-ui-password-20261013`. Requirements: P1, P2. User explicitly chose
8 instead of 12; no product decision is pending. Context budget: 10k input tokens,
hard ceiling 14k. No children. Dispatch after Packet01 releases writer lease.

## Owned writes (exact)

- `internal/modules/identity/service/password.go`
- `internal/modules/identity/service/password_test.go`
- `web/apps/portal/src/features/admin/pages.tsx`
- `web/apps/portal/src/features/admin/company-form.test.tsx`
- `web/apps/portal/src/features/admin/users-page.test.tsx`
- `web/packages/kit/src/company-employees.tsx`
- `web/packages/kit/src/company-employees.test.ts`
- `web/packages/kit/src/session.tsx`
- `web/packages/kit/src/session-lifecycle.test.ts`
- `docs/justix-auto/dev/execution/local-ui-password-fixes-20261013/result-02.md`

Read root AGENTS.md, current override/authority in agent-workflow.md,
`internal/AGENTS.md`, `web/AGENTS.md`, assigned task, this packet and findings.md.
Read password.go/test fully; targeted frontend password definitions/test cases.
Do not load whole Admin pages/session files unless required. Read-only caller
slices: identity service user.go 134–145, 479–496, 537–575; company.go 213–233;
auth.go 390–415; cmd/bootstrap-admin/main.go 45–66. Contract is assigned task P1/P2;
HTTP response shapes and authorization are unchanged. Source-hashes.json pins
inputs. No legacy policy docs or mocks are needed.

## Implementation and policy preservation

1. Change shared `validatePassword` minimum from 12 to 8 and the matching
   `at least 12 characters` error to `at least 8 characters`.
2. Update all six current frontend help occurrences from 12 to 8: Admin pages
   company creation (`adminFields`), company-admin edit (`adminEditFields`) and
   platform staff creation; CompanyEmployees creation/reset; forced ChangePassword.
   Existing forms have no separate min12 HTML/client validator; no new client
   validation abstraction or additional restrictions are necessary. Preserve
   empty-edit behavior, confirmation and request fields.
3. Preserve rune-count semantics, max128, confirmation ordering/field names,
   current-password verification, changed-password rule, required credentials,
   login policy, Argon2 parameters/hash format, auth/session behavior and stored
   passwords. Do not change unrelated 12s, long sample passwords or hash code.
4. All production paths already call this validator: user creation, first-admin
   company onboarding, bootstrap, admin reset and own password change. The
   bootstrap CLI delegates to User.Bootstrap; it has no minimum of its own and
   needs no edit or execution. No migration/reset/account recreation is needed.

## Affected unit evidence

Add table-driven `TestValidatePassword` in password_test.go using a small error
collector or apperr.Validation. Cover seven rejected/eight accepted, Unicode
rune boundaries (seven multibyte characters still rejected, eight accepted),
128 accepted/129 rejected, matching versus mismatching confirmations and field
mapping for `password`, `newPassword`, `firstAdmin.password`. Assert minimum8
error and unchanged max/confirmation errors. Keep the existing
TestPasswordHashRoundTrip to check existing hash/verification behavior once.

Per primary's acceptance amendment in approval.md, do not add frontend tests
solely for hint changes. Prefer updating existing creation-test inputs and
payload expectations to eight characters (company first admin, platform staff,
company employee), retaining existing company-admin blank-edit and forced-gate
unit cases. Session-lifecycle.test.ts is permitted only if an existing assertion
needs adjustment; new form/reset/ChangePassword cases are not required for this
copy-only frontend change. Run the existing affected suites below. Mocked
frontend success does not prove backend enforcement; the required backend
boundary tests do. No source-text or CSS/string-replacement tests.

Run `bash tools/check-git.sh` before application edits. After the cohesive change,
run exactly these affected unit selections once each, no automatic broadening:

```sh
# project root; no database wrapper or integration package
bash tools/go.sh test ./internal/modules/identity/service -run '^(TestValidatePassword|TestPasswordHashRoundTrip)$' -count=1
# web/; installed pinned Node24.21.0 and npm11.19.0
npm run test:unit -- packages/kit/src/company-employees.test.ts packages/kit/src/session-lifecycle.test.ts apps/portal/src/features/admin/company-form.test.tsx apps/portal/src/features/admin/users-page.test.tsx
```

Record actual commands/results in result-02.md. Do not run bootstrap-admin,
tools/test-go.sh, module flow_test.go, browser/integration checks, lint/typecheck,
build, React Doctor, independent review/QA or repeat already-passing checks.

## Locks and stops

Exclusive `main-checkout-source-writer`; sequential after Packet01. No Git writes,
runtime changes, database or browser lease, remote push, migration, reset or
credential changes. Ignore unread test_creds and private env/keys. Only the hub
writes canonical docs. Stop for unexpected policy/production min12 outside owned
paths, source conflict, missing pinned toolchain or required extra ownership;
report exact evidence to hub. No policy ambiguity is presently known. If the
packet exceeds its context cap, request a split; RESLICE_REQUIRED only for an
actual canonical-scope change.
