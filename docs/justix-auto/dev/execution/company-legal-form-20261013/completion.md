# Company legal form — source completion

Application authority /root accepts FE1 for L1–L4. Source writer released.
Branch: fix/company-legal-form-20261013. Dependency:
5ae7a9369ff9a46f31ca6ebc37a72d54e10b2ffa, with dev ancestry 4fa683b.

Delivered source behavior:

- Shared optional values: Не указана / ООО / ИП / Другое, with stable existing
  wire values and a safe missing/unknown display fallback.
- Admin seller, bank, MFO and insurance creation/edit forms now include legal
  form; detail cards display it. Editing prefills and sends the selected value,
  preventing an unrelated edit from omitting and resetting it.
- Realization company creation, existing company editor and profile facts use
  the same field/options. Company creation remains possible without this value.
- Shared provider organization profiles show the value for Financing/Insurance.
  Existing read-only access and all permission/business-type rules are preserved.
- Searchable select mode displays the explicit empty option consistently with
  the existing form controls. Compact company selectors and historical signed
  documents were not changed.

## Unit evidence

[Implementation result](implementation-result.md) records actual commands and
final source hashes. Terminal focused passes: Admin company forms 8,
Realization add-company 2, Realization profile 2, shared OrgSettings 5 — total 17.
Initial new-test harness/assertion errors were repaired. Only affected test files
were rerun; the provider file and its consumed source were unchanged after its
initial pass, so that passing selection was not repeated.

These are component units with mocked HTTP, not a live database roundtrip or
visual/release certification. No lint, typecheck, build, browser/E2E/integration,
React Doctor or independent review/QA checks ran. No backend or migration change,
new API contract, runtime restart, database/credential operation or remote action.
Existing local watchers remain available for the user's manual testing.

## Preservation and Git handoff

Unrelated dirty internal/modules/identity/service/company_contract_test.go is
unchanged at SHA-256 a4d4b3f67107b5362810dcc96f68f1de8e59cae466dacafb7e100e29a59fe99f.
Untracked test_creds.txt was never read or modified. Both remain outside staging.
Canonical before-copies/hashes: entry/ and entry/completion/.
Version-control may make one explicit-path local task commit after the worker's
result is finalized. No push, dev/main promotion or deployment is part of this task.
