# FE1 implementation result

Status: DONE

- Changed `web/packages/kit/src/portal.tsx`: the unavailable-section branch now computes matching companies once, presents `CompanySelection` only for non-Admin sections with candidates when the active company is outside that set, and otherwise returns to `/` through a replacing navigation.
- Preserved the existing `portalCompanyId` preference, admitted deep-link handling, and all session/context logic.
- `bash tools/check-git.sh`: passed before the application edit.
- Result SHA-256: `web/packages/kit/src/portal.tsx` — `89bbae47ae491902f1335e08a7c20806bf23605504eb60705b5c947b7e150fca`.
- Exclusions: no tests, runtime actions, lint/typecheck/build, or review/QA were run, per packet; `internal/modules/identity/service/company_contract_test.go` was not read or changed; `test_creds.txt` remained unread.

Writer lease released for the primary's focused browser regression.
