# Packet 02 result — password minimum eight

Status: **DONE**

Implemented P1/P2 on `fix/portal-ui-password-20261013`:

- Changed the shared identity validator to require at least eight Unicode runes
  and return `at least 8 characters`.
- Preserved the 128-rune maximum, confirmation ordering and field naming,
  Argon2 hashing/verification, and all existing caller behavior.
- Updated the six current frontend password help texts: company onboarding,
  company-admin edit, platform staff, company employee creation/reset, and the
  forced password-change message.
- Added table-driven backend boundary coverage: 7/8 ASCII and Unicode runes,
  128/129 runes, confirmation matching/mismatch, minimum-before-confirmation
  ordering, and `password`, `newPassword`, and `firstAdmin.password` fields.
- Changed existing company, platform-staff, and company-employee test payloads
  to the eight-character value `password`; no hint-only frontend tests added.

## Verification

`bash tools/check-git.sh` passed before application edits:

```text
GIT CHECK OK: /Users/bakhromachilov/startups/justixauto
```

Executed once after the cohesive edit:

```sh
bash tools/go.sh test ./internal/modules/identity/service -run '^(TestValidatePassword|TestPasswordHashRoundTrip)$' -count=1
```

Result: PASS, `ok justixauto/internal/modules/identity/service 1.250s`.
The selection contains two named test functions: ten `TestValidatePassword`
table cases and `TestPasswordHashRoundTrip`.

```sh
cd web && npm run test:unit -- packages/kit/src/company-employees.test.ts packages/kit/src/session-lifecycle.test.ts apps/portal/src/features/admin/company-form.test.tsx apps/portal/src/features/admin/users-page.test.tsx
```

Result: PASS, 4 test files and 20 tests, duration 5.54s.

## Output SHA-256

```text
e2f0a453738d138eee0a005d9d649878a6dd514006b24a8e04d8c940586a9067  internal/modules/identity/service/password.go
41a4820a76a77247118f1db87649c70955951d2fa1233cacc9df47237d21f5eb  internal/modules/identity/service/password_test.go
f6b006c08057f1e83bdc5263100fa63275f1a8f9ff012a3baf1c5ae0552aa559  web/apps/portal/src/features/admin/pages.tsx
36cbdcd0e23b270ca3136cda4622b65eee07e12e8b828f1bbc6035ad71d363c2  web/apps/portal/src/features/admin/company-form.test.tsx
425f4a3a98f55ecfedf252fc2bb1403346de9e3284da2d916172d9e01a4cd593  web/apps/portal/src/features/admin/users-page.test.tsx
ca43504ff9a63152b35d40987763b2fd56509f1e7040ea06cef2d529bd3329db  web/packages/kit/src/company-employees.tsx
4552b2fbe5134c570477d00e39b77c569608024cd7753e76be9c0212e26b667c  web/packages/kit/src/company-employees.test.ts
470317414dd67e089fd32ac42bc8eeea070b114a647e4524bcf7c5140a8579a8  web/packages/kit/src/session.tsx
```

## Exclusions

No bootstrap invocation, database/browser/runtime operation, migration,
credential access, lint/typecheck/build, integration/QA run, Git write, or
change to Packet01 UI paths. `session-lifecycle.test.ts` was run as required
but did not need a source edit.
