# FE1 — Optional organization legal form across company UI

Status: READY for owning application hub `/root`; no automatic review/QA loop.
Task: `docs/justix-auto/dev/company-legal-form-20261013.md`, acceptance L1–L4.
Base: `5ae7a9369ff9a46f31ca6ebc37a72d54e10b2ffa`.
Branch prepared by version-control: `fix/company-legal-form-20261013`.
Main checkout: `/Users/bakhromachilov/startups/justixauto`.
Context target 10k tokens; hard limit 14k. One coherent behavior: expose the
existing optional legalForm consistently while preserving its value on edits.

## Authority and bounded reads

Read root `AGENTS.md`, `web/AGENTS.md`, this packet and `findings.md`. Artifact
writes also follow `docs/justix-auto/dev/AGENTS.md`. Current unit-only override
supersedes older scoped checklists: no lint, typecheck, build, browser/E2E,
integration, React Doctor or automatic review/QA. Only affected units once per
relevant change. Run `bash tools/check-git.sh` before application changes as
required by root rules. Workers perform no Git mutations.

Approved product sources: canonical task Approved behavior/L1–L4;
`business-logic.md` Applications and access;
`dev/product-workflow-improvements-20261011.md` R1/R2.
No product decision is open for this field. OD-10/OD-11 provider/compliance rules
remain outside scope: legal form grants no capability and makes no licensing claim.

Pinned existing contracts (read only if needed):
- `internal/modules/identity/model/company.go`: LegalForm/Valid, lines 61–73.
- `internal/modules/identity/service/company.go`: CompanyInput lines 33–45 and
  apply lines 105–112. Omitting legalForm on PATCH applies the empty Go value;
  therefore every company edit must submit the prefilled selection.
- `internal/modules/identity/handler/dto.go`: companyDTO/toCompany, lines 17–56.
- `internal/modules/identity/repository/company.go`: Update map around line 79.
- `migrations/000035_company_verification.up.sql`: existing enum constraint.

Targeted source reads: Admin `pages.tsx` imports/types/companyFields/companyInput
(lines 1–161), creation (211–257), CompanyDetail (388–495); Realization
`pages/settings.tsx` imports/Company/CompanyEditAction/CompanyFacts (1–185),
CompanyInfoSection edit gate (186–219), AddCompanyAction/Companies (337–428);
kit `org-settings.tsx`, barrel and selected existing tests. Do not load the full
large Admin file, board, history, mocks or unrelated sales/inventory code.

## Exact write ownership

Production files:
1. `web/packages/kit/src/company-legal-form.ts` (new).
2. `web/packages/kit/src/index.ts` (one export).
3. `web/packages/kit/src/org-settings.tsx`.
4. `web/apps/portal/src/features/admin/pages.tsx` (listed company symbols only).
5. `web/apps/portal/src/features/realization/pages/settings.tsx` (listed company symbols only).

Behavioral unit files:
6. `web/apps/portal/src/features/admin/company-form.test.tsx`.
7. `web/apps/portal/src/features/realization/add-company.test.tsx`.
8. `web/apps/portal/src/features/realization/company-profile.test.tsx` (new).
9. `web/packages/kit/src/org-settings.test.ts` (new; jsdom pragma + createElement,
   matching kit's existing `.test.ts` discovery and DOM test convention).

Result artifact only:
`docs/justix-auto/dev/execution/company-legal-form-20261013/implementation-result.md`.
No other paths are owned. Preserve dirty
`internal/modules/identity/service/company_contract_test.go`; never read
`test_creds.txt`, `.env` or credentials. No backend, migration, dependency/config,
navigation, compact-selector, historical contract or canonical-document changes.

## Implementation

1. Add a small shared kit module exporting `LegalForm` as
   `'' | 'llc' | 'sole_proprietor' | 'other'`, typed select options compatible with
   `FieldSpec`'s mutable `[string, string][]`, and a safe `legalFormLabel` lookup.
   Options: empty → `Не указана`, llc → `ООО`, sole_proprietor → `ИП`, other →
   `Другое`. Lookup tolerates missing/empty/unknown values with `Не указана`.
   No whole-company model abstraction, new enum semantics or helper-only tests.
2. Admin: add legalForm to local Company type using the shared type; add optional
   select labelled `Организационная форма` to companyFields' company group,
   initialized from the fetched value or empty. Serialize `legalForm` explicitly
   in companyInput, with blank remaining `''`. Add labelled detail value using
   shared lookup. Existing common paths cover all four kinds. Preserve legalName,
   registration, If-Match, administrator writes and refresh behavior.
3. Realization: reuse shared type/options in existing CompanyEditAction; preserve
   its optional behavior and current payload. Add optional select and explicit
   company.legalForm payload to AddCompanyAction; show shared label in
   CompanyFacts. Preserve `company.create`/`company.edit` gates and session refresh.
4. OrgSettings: extend local Org type and render one organization fact with the
   shared label. Both bank/MFO Financing and Insurance routes already consume it;
   no navigation edits or new edit action. Preserve loading and read-only notice.
5. Cover payload/prefill/preservation behavior in the bounded unit selection
   below. Use mocked API responses, real form interactions and normal rendering;
   avoid exports added only for testing and snapshots of option constants.

## Acceptance and affected unit selection

- L1: Admin creation cases for seller, bank, mfo and insurance assert optional
  select, correct API endpoint/kind, and submitted enum. Distribute known values
  across cases so all three labels/value mappings are exercised. Existing minimal
  creation test asserts blank legalForm; do not add required inputs.
- L1/L4: Admin detail/edit loads stored legalForm, displays its label, prefills
  select and sends the same value through an unrelated name/contact edit with
  existing If-Match. Parameterize the common path across kinds without duplicating
  test harnesses. Include intentionally clearing a selected value to `''` and
  reopening from the mocked saved GET result. This proves the frontend roundtrip,
  not live database persistence.
- L2/L4: extend Realization add-company unit to assert selected legalForm payload
  plus existing refresh; keep minimal blank creation. New profile unit renders
  SettingsPage with bounded session/API mocks, observes facts and edit prefill,
  saves an unrelated edit without losing selection and reopens refreshed data.
  Check no create/edit buttons without their existing permission. Avoid changes
  to branch/warehouse implementations; stub unrelated child data where needed.
- L3/L4: OrgSettings unit renders fetched bank/MFO/insurance profiles and verifies
  stored labels; missing/empty/unknown values show `Не указана`. Assert no
  organization edit action/select is introduced. Use mocked session and unrelated
  employee/role components to keep the unit local.
- L4 fallback rendering also has a small case on each of Admin and Realization
  profile surfaces; do not create a full Cartesian product of kinds × enum ×
  permissions. Unknown display must not become a new selectable enum. Blank and
  known persisted values are the edit-roundtrip contract; malformed unknown data
  is a display-resilience case only.

After relevant edits, run once from `web` with the pinned Node/npm environment:

```sh
npm run test:unit -- apps/portal/src/features/admin/company-form.test.tsx apps/portal/src/features/realization/add-company.test.tsx apps/portal/src/features/realization/company-profile.test.tsx packages/kit/src/org-settings.test.ts
```

These four files cover the shared helper's consumers directly. No helper mapping
unit, unrelated whole-suite run, frontend build or backend unit run is needed.
If a selected test fails, fix only the relevant issue and rerun only affected
units once for that new change. A passing run is not release/integration proof.

## Dependencies, resources and stop conditions

Depends on prepared branch/base and existing legalForm backend contract only;
no packet dependencies. Hub reserves exclusive main-checkout source-writer lock
for these nine files. No browser/database/server resource is requested. No Git
mutation, runtime restart/reset or remote promotion. Findings/source manifest
are read-only worker inputs.

Stop and report to `/root` if source hashes differ unexpectedly, another source
writer appears, repository-root check fails, API contract is absent/incompatible,
or implementation requires an unowned path/product/permission change. Never
reinterpret demo fixtures as policy. For a context overflow, request a smaller
packet before proceeding; report RESLICE_REQUIRED only if canonical scope must
change. Do not broaden the backlog. Unknown enum write policy, a new edit workflow
or backend redesign requires a new decision; no such work is part of FE1.

Report changed paths, L1–L4 coverage, exact unit command/results and limits to the
owned result file. No independent QA or integration execution is authorized.
