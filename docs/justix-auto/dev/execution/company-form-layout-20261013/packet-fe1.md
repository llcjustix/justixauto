# FE1 — company form layout and seller business-type radios

Status: prepared for application hub acceptance; no implementation or tests executed.
Task: `company-form-layout-20261013`; authority hub: `/root`.
Dependency/base SHA: `3aeb7b8feddb81cb49b7f0879a041d22ce760643` (supplied by hub).
Branch to be prepared by version_control: `fix/company-form-layout-20261013`.
One coherent behavior: company forms place legal form before name, with Admin seller creation choosing one fixed business type through a shared native radio control.

## Authority, dependencies and context

Approved contract: `docs/justix-auto/dev/company-form-layout-20261013.md`, F1–F4.
Read `AGENTS.md`, `web/AGENTS.md`, `docs/justix-auto/dev/AGENTS.md` and
`docs/justix-auto/dev/agent-workflow.md`; current unit-only override supersedes
historical automatic verification chains. Supporting contracts: business-logic.md
“Applications and access”; dev/product-workflow-improvements-20261011.md R1/R2;
dev/company-legal-form-20261013.md approved behavior/L1–L4. All paths above are
under `docs/justix-auto/` unless rooted explicitly.

F2 supersedes only the old Admin creation UI blank/deferred choice. API
optionality, legacy assignment, company kind and permission rules remain.
No additional product decision is required. Do not infer policy from fixtures.

Prerequisites: hub accepts this packet; version_control prepares the branch at
the supplied dependency source; hub reserves the main-checkout source writer.
Before editing application source run `bash tools/check-git.sh`; stop if the
project root differs from `/Users/bakhromachilov/startups/justixauto`.
No worker Git writes, children, worktrees, runtime restart, remote actions,
canonical-record edits or contract edits.

Context budget: target 12k tokens, maximum 16k. Read this packet, scoped rules,
contract sections and source slices below; do not load board/history, entire
large source files or mocks. If work requires broader rules/ownership or cannot
fit, stop and request a split; report RESLICE_REQUIRED only for canonical scope
changes. Source fingerprints: adjacent `source-hashes.json`.

## Exact ownership and resources

Production paths:

- `web/apps/portal/src/features/admin/pages.tsx`
- `web/apps/portal/src/features/realization/pages/settings.tsx`
- `web/packages/kit/src/ui.tsx`
- `web/packages/kit/src/design.css`

Test paths allowed to change:

- `web/apps/portal/src/features/admin/company-form.test.tsx`
- `web/apps/portal/src/features/admin/company-verification.test.tsx`
- `web/packages/kit/src/form-dialog.test.ts`

Worker result only:
`docs/justix-auto/dev/execution/company-form-layout-20261013/implementation-result.md`.
Existing Realization `add-company.test.tsx` and `company-profile.test.tsx`
are read/run-only. No package, lockfile, API, backend or schema changes.

Exclusive locks: main-checkout source writer, these seven writable source/test
paths and unit-run process. No browser or database fixture needed. Keep source
stable during its unit run. Preserve unrelated dirty
`internal/modules/identity/service/company_contract_test.go` exactly (hub hash
in manifest). Never read `test_creds.txt`, environment files or secrets.

## Implementation map

1. Admin `companyFields` (~104): put legalForm before name; remove name's
   `full: true`. Preserve optionality, initial values, groups and payload
   construction. This shared array serves seller/provider creation and edit.
   Change `sellerTypeField` (~204) to `type: 'radio'`, `full: true`,
   `initial: 'dealer'`, options only dealer/Дилер and distributor/Дистрибьютор.
   Keep its company-information group. Creation's field grouping places it
   directly after legalForm/name, before first-admin fields, despite address
   fields appearing earlier in the flat array.
2. Reuse that control in the legacy assignment action (~436). Preserve
   `c.kind === 'seller' && !c.userType`, required selection, existing endpoint
   and If-Match revision. Do not add business type to ordinary company edit,
   provider forms or company-user Realization forms.
3. Realization `CompanyEditAction` (~108): order legalForm, name, legalName,
   taxId, then existing fields. `AddCompanyAction` (~344): legalForm, name;
   remove name's full-row flag. Preserve all state/payload/session behavior.
4. Shared `FieldSpec` (~689): add a bounded radio variant with string value,
   option pairs, initial and required, reusing group/full metadata. Existing
   FormDialog initialization already handles initial strings. Extend required
   select validation (~817) to missing required radio values; submit button
   invokes submit directly, so native required alone is insufficient.
5. `FieldInput` (~1081): native radio inputs with one unique group name per
   field instance (existing useId is available), shared controlled checked
   state, matching labels and fieldset/legend. Keep separate dialogs/fields
   from affecting each other's selection. Associate field errors with group
   and/or inputs using stable IDs/aria-describedby and invalid state. Reuse
   existing nested server-error mapping; preserve optional radio empty values.
   Do not refactor other field controls.
6. CSS: bounded radio fieldset/options/input rules using existing tokens and
   check-line conventions (~1630). Neutralize inherited full-width/text-input
   dimensions for radios. Avoid broad legend/input overrides. Existing
   form-grid/full-row and responsive rules (~2389) supply layout; do not
   redesign the grid.

## Acceptance and affected units

| Requirement | Implementation/evidence |
|---|---|
| F1 Admin/Realization create/edit legalForm then name in one desktop row | Field order and absence of name full-row flag in the two owned page files; existing responsive rules retained. Existing Admin legal-form cases cover seller/bank/MFO/insurance optional values, edit prefill and payload; existing Realization units cover optional create, prefill and unrelated edit preservation. No new layout/CSS snapshot test. |
| F2 seller radio follows row; dealer default; distributor alternative; providers excluded | Admin company-form units assert two labeled native radios, dealer checked initially and submitted without interacting with type; choose distributor and assert dealer unchecked plus distributor payload. Assert provider cases have no seller radios/type field and preserve provider payload. Retain minimal onboarding. |
| F3 legacy assignment and immutability | Admin company-verification units open missing-type seller assignment, select/submit via existing endpoint with correct If-Match. Parameterize already-fixed dealer/distributor records: assignment absent and ordinary edit offers no type mutation. Existing Realization edit gate remains covered by company-profile unit. |
| F4 shared state/accessibility/errors | kit form-dialog units cover named native radio group/labels, controlled exclusive choice and flat submitted value; missing required radio blocks submit, then choosing a value allows it; nested server field error associates with the radio field. Check independent instances use distinct group names. Retain existing select/multiselect behavior tests. |

Update Admin's existing section-group assertion to inspect top-level section
fieldsets or exclude nested groups structurally: preserve exact company/address/
contacts section order and login containment. Do not weaken it to mere presence
because a radio fieldset adds a nested group.

Run once after the relevant changes with locked Node 24.21.0/npm 11.19.0, from
`web/`:

```sh
npm run test:unit -- apps/portal/src/features/admin/company-form.test.tsx apps/portal/src/features/admin/company-verification.test.tsx apps/portal/src/features/realization/add-company.test.tsx apps/portal/src/features/realization/company-profile.test.tsx packages/kit/src/form-dialog.test.ts
```

Do not run lint, typecheck, build, React Doctor, browser/E2E/integration or full
suite. A relevant repair authorizes one rerun of its affected units; do not
repeat passing unaffected tests. Record command, toolchain, counts/failures,
source identity, exclusions and changed paths in implementation-result.md.
Units do not establish visual correctness or release readiness.

## Dormant independent verification definitions

These definitions satisfy handoff coverage only; they are NOT scheduled or
authorized development checks. No automatic reviewer/QA loop or additional
artifact is required. Activation requires explicit user authorization and
hub-assigned evidence paths; reviewers/QA own no source paths.

- QA-U1: at an exact candidate SHA, independently exercise only shared radio
  required/error/isolation units (F4). Budget 4k tokens; dependency FE1; lock
  immutable checkout and unit-run process; stop on source movement or unmet
  toolchain. Never duplicate the worker run under current policy.
- QA-U2: at that exact SHA, independently exercise Admin onboarding/legacy
  assignment and existing Realization unit files (F1–F3). Budget 6k tokens;
  same locks/dependency/stop rules; no browser/database.
- INT-FINAL: separately authorized final-SHA shared-interface check: Admin
  FormDialog radio → userType creation/legacy endpoint, provider exclusion,
  fixed-type guard and Realization legalForm preservation at one final source
  SHA. Budget 6k tokens; depends on all final changes; reserve immutable
  checkout and any explicitly authorized environment first. Scope/commands
  must be approved before execution. Prior unit results are not cumulative
  integration evidence.

## Stop conditions and known limits

Stop for changed contracts/base sources, competing writer, product ambiguity,
new permission/API/backend requirement, broader ownership, missing locked
toolchain, or verification requiring unauthorized commands. Report the narrow
blocker to /root; do not solve it through fixtures or scope expansion.
Visual row alignment/responsive appearance and native browser keyboard behavior
remain unverified under the unit-only policy. CSS bleed and nested accessible
group assertions are the principal implementation risks. No known acceptance
mapping gap; implementation and visual/release validation are still outstanding.

