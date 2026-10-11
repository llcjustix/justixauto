# P02 — Company setup and verification UI

Requirements: R1/R2. Dependencies: P01. Read plan.md common execution rules and referenced source/contract slices. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 18k tokens (stop/split if exceeded).

## Exact owned paths

- `web/packages/kit/src/session.tsx`
- `web/packages/kit/src/org-settings.tsx`
- `web/packages/kit/src/shell.tsx`
- `web/packages/kit/src/messages.ts`
- `web/packages/kit/src/company-verification.test.tsx`
- `web/apps/admin/src/pages.tsx`
- `web/apps/admin/src/company-form.test.tsx`
- `web/apps/admin/src/company-verification.test.tsx`
- `web/apps/realization/src/pages/settings.tsx`
- `web/apps/realization/src/add-company.test.tsx`
- `internal/modules/identity/service/company.go`
- `internal/modules/identity/service/company_test.go`
- `internal/modules/identity/service/company_verification_test.go`

## Behavior, interfaces and data flow

Consume P01's actual reported DTO/action names, never assumed enum spellings. Extend shared Company/session types; show fixed seller type and verification status/reason. Admin seller onboarding asks name, business type and first-admin credentials; optional requisites belong in settings. Admin may fill missing legacy type once and verify/return submitted details. Self-created company shows awaiting platform type assignment. No mode switch. Settings edit legal form, registration/tax, bank/contact/address and submit/resubmit. Keep branches/warehouses/staff setup and current company selection usable pre-verification; do not hide old history. Refresh session/company queries after actions. Explanatory banner is advisory; P03 provides server enforcement. Preserve provider kinds and role assignment screens.

## Acceptance and affected units

Admin set-once control disappears after classification; settings cannot change type. Returned reason and resubmit visible; stale/errors/loading shown. Minimal onboarding roundtrip matches P01. Company switch reloads verification/type; unverified workspace still exposes setup.

Run once after relevant change: `npm --prefix web run test:unit -- packages/kit/src/company-verification.test.tsx apps/admin/src/company-form.test.tsx apps/admin/src/company-verification.test.tsx apps/realization/src/add-company.test.tsx`. No other development verification or runtime actions.

Stop: Stop on any new product rule, ownership expansion or context ceiling; report exact missing decision/path. No unrelated refactoring.

Coordinator correction from original R2: name + first-admin login/password suffice. Admin userType selector is optional at creation; missing type blocks verification/new commerce until manual platform assignment. P02 owns the three backend paths above solely to correct any P01 mandatory-type validation and relevant tests; invalid nonempty/provider misuse still rejected and assigned type fixed. Run affected identity/service units once for this relevant correction in addition to listed frontend units: `bash tools/go.sh test -vet=off ./internal/modules/identity/service`. Do not require business type in minimal form despite earlier wording in this packet.

