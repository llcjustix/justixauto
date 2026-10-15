# FE1 implementation result — 2026-10-13

Status: DONE.

Implemented the existing optional `legalForm` across the assigned company UI
surfaces. The shared kit module provides `LegalForm`, the four approved options,
and a safe Russian label fallback (`Не указана`). Admin now includes the field in
all company-kind create/edit payloads and details; Realization includes it in
creation, edits and company facts; provider organization settings show it
read-only. The existing searchable select mode is used so the blank option is
displayed as `Не указана` instead of the generic empty-select marker.

Coverage:

- L1: Admin tests cover seller, bank, MFO and insurance creation payloads,
  blank minimal creation, edit prefill and clearing while retaining `If-Match`.
- L2/L4: Realization creation covers a selected value and blank minimal creation;
  profile coverage verifies display, edit prefill, preservation during unrelated
  edits, fallback display, and existing edit gating.
- L3/L4: OrgSettings covers bank/MFO/insurance known forms plus missing and
  unknown fallback, and verifies it remains read-only.

Verification (unit-only, from `web`):

1. Initial four-file selection found focused harness issues: select controls use
   display labels, the insurance action label is `Страховая`, and the new profile
   test imported the page from the wrong directory. OrgSettings passed (5 tests).
2. After targeted repairs, Admin passed 8 tests and Realization add-company
   passed 2 tests; one profile assertion used `Сохранить` instead of the existing
   `Изменить данные` submit label.
3. The profile-only rerun passed 2 tests. Terminal total: 17 passing focused
   tests (8 Admin + 2 add-company + 2 profile + retained 5 OrgSettings). No
   lint, typecheck, build, browser, integration, React Doctor, or broader suite
   was run.

Owned file SHA-256:

```text
4643c2b3338a95207791fb15e83173b21ff78d1a823c231d1d3a1871215fea33  web/packages/kit/src/company-legal-form.ts
4df27719e1255f27dda3b3c50d9bd239bf65ff629df2dd614314092d2bf32fed  web/packages/kit/src/index.ts
4ca94059b2368ddaf2d277610f998326c5dd7178984b89af524a4ad3285044e6  web/packages/kit/src/org-settings.tsx
ed5cb9537718b8f72af0565cff458e5afe9f1e12245571d31c74c2694595f1a3  web/apps/portal/src/features/admin/pages.tsx
0cb37e6f53106cbff59f35ac60b67b4ea01f27d9794b4193ac22c351199289c2  web/apps/portal/src/features/realization/pages/settings.tsx
94634af9712ee200110ea5b536902382370d8eb424e260047bfbf26294325152  web/apps/portal/src/features/admin/company-form.test.tsx
95b61cb8adeb2b99d1d5e5fff84c6b75f573173e6eb26fd9c99caed13af434de  web/apps/portal/src/features/realization/add-company.test.tsx
cc7e4e671abd504c95fce4564b11be40d5d7a07ee64b16dad483f44587a23b9d  web/apps/portal/src/features/realization/company-profile.test.tsx
5f1f08015f633ba4865f659ef0b9549e001d44b08e22f503892ed95dae0d0136  web/packages/kit/src/org-settings.test.ts
```

Exclusions: no backend/schema/runtime/Git/canonical-document changes. The
pre-existing dirty `internal/modules/identity/service/company_contract_test.go`
and untracked `test_creds.txt` were not read or changed.
