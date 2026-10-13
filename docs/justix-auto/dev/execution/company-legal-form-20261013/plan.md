# Company legal form execution plan

Task: `docs/justix-auto/dev/company-legal-form-20261013.md`.
Hub: application `/root`. Base: `5ae7a9369ff9a46f31ca6ebc37a72d54e10b2ffa`.
Branch: `fix/company-legal-form-20261013`, prepared at base by version-control
and confirmed by hub. Slicer performs no Git operations.
Owned artifact directory: `docs/justix-auto/dev/execution/company-legal-form-20261013/`.

Ready implementation: [FE1](packet-fe1.md), one worker packet, target 10k/max 14k
context. No canonical backlog additions or scope split. Source map:
[findings](findings.md); SHA-256 pins: [source hashes](source-hashes.json).
Hub approval is required for dispatch; current unit-only override removes
automatic independent plan/code-review/QA chains. No such checks ran here.

## Acceptance coverage

| Requirement | Implementation | Behavioral unit evidence planned |
|---|---|---|
| L1 Admin all kinds create/edit | FE1 shared companyFields/companyInput, Company type | company-form: four kinds, selected payload, blank minimal create, edit/reopen |
| L2 Realization create/edit/details | FE1 AddCompanyAction, CompanyEditAction, CompanyFacts | add-company + company-profile: create, facts, prefill, unrelated-edit preservation |
| L3 Finance/Insurance display | FE1 shared OrgSettings (bank/MFO/insurance consumers) | org-settings: fetched labels, fallback, no new editor |
| L4 optional/prefill/fallback/permissions | FE1 all shared-field consumers | the four selected files cover blank, selected/cleared value, missing/unknown display and existing Realization gates |
| Scope invariants | FE1 preserves business type, access, verification, selectors, signed records | retain existing form regression assertions; no change to these policy paths |

No undecided product rules or missing backend support found. No requirement is
claimed complete before implementation. Unit tests use mocked HTTP and cannot
prove live persistence or release readiness.

## Deferred independent verification packets — do not dispatch

These descriptions satisfy handoff completeness for a later explicit request.
They are excluded from active development, have no automatic dependency/gate,
and do not authorize review/QA loops or repeating passing tests. They are not
canonical backlog tasks. Current active count: 1 implementation, 0 QA.

Q1 (independent form roundtrip): target 4k/max 6k context; pinned candidate SHA
supplied by hub, read FE1 and only its Admin/Realization source and unit files.
Requirements L1/L2/L4. Depends on committed FE1 and explicit user authorization.
No source ownership; only artifact `qa-form-result.md` in this directory if
authorized. Reserve immutable-checkout read lock; no browser/DB. Independently
exercise selected creation, blank creation, loaded edit/unrelated-edit/reopen
via the existing bounded units at that exact SHA; preserve permission assertions.
Record command, SHA and observed payload evidence. Stop on moving source,
missing permission to run checks, or a scope mismatch. No standalone helper tests.

Q2 (final-SHA shared-interface integration): target 4k/max 6k context; requires
explicit user request and hub-supplied immutable final candidate SHA including
all FE1 paths. Read source hashes/coverage and shared helper/barrel, three
consumers and Financing/Insurance navigation imports only. Requirements L1–L4.
No source ownership; only artifact `qa-shared-final-result.md` if authorized.
Reserve immutable-checkout read lock; any browser/DB check needs separately
specified authorized environment and exclusive lease. At the final SHA, verify
same wire values/labels are observed by Admin, Realization and both provider
sections, optional blank roundtrips, and shared OrgSettings remains read-only.
If authorized for browser/backend integration, use a designated nonproduction
fixture to create/save/reload across these surfaces and assert persisted values;
do not infer that proof from mocked units or earlier GREEN reports. Without an
authorized environment record that integration evidence is unavailable. Stop on
source mutation, absent fixtures/environment authorization or contract drift.

## Gaps and risks

- No runtime or release evidence is produced: additional checks are prohibited
  absent explicit request. Deferred packets must remain dormant.
- Admin currently omits legalForm from PATCH and the backend applies blank;
  selected-value preservation is the main regression risk addressed by FE1.
- Missing/unknown display uses the existing `Не указана` fallback; no policy
  for persisting malformed unknown enums is introduced.
- Kit discovers `.test.ts`, not `.test.tsx`; use its existing jsdom/createElement
  convention for the new profile unit rather than changing test configuration.
- Preserve protected dirty backend test and unread credentials; leave all
  canonical documents and approved contracts to their owners.
