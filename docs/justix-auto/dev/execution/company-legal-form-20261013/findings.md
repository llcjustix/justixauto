# Source findings — company-legal-form-20261013

Authority: application hub `/root`. Assigned source base:
`5ae7a9369ff9a46f31ca6ebc37a72d54e10b2ffa` (task-supplied; no Git inspection).

## Incremental map

- Admin `pages.tsx`: `companyFields` feeds seller/provider creation and the common
  `CompanyDetail` edit action; `companyInput` is the common POST/PATCH serializer.
  All four company kinds use this path. Local `Company` lacks `legalForm` and
  detail facts omit it. Minimal fix touches these shared points once.
- Realization `pages/settings.tsx`: `CompanyEditAction` already exposes the correct
  optional enum and sends it in PATCH; `AddCompanyAction` and `CompanyFacts` omit
  it. Existing edit options can become shared without changing permissions.
- Kit `org-settings.tsx`: GET-backed organization profile omits `legalForm`; both
  Financing and Insurance navigation use this component. Keep it read-only.
- Repository-wide company-endpoint/legalName search found no additional company
  create/edit/profile surfaces. Membership/company selectors, branch forms and
  signed-contract party rendering are outside the approved scope.
- Additional targeted surface audit: Realization `pages/partners.tsx` lines
  102–145 render name/country and relationship status/actions only; there is no
  row-opening action, company-profile modal or company GET (despite introductory
  copy mentioning a profile). Financing `pages.tsx` PartnersPage lines 226–253
  shows seller names and application counts without a detail action. Realization
  `pages/partners-finance.tsx` and Insurance `pages.tsx` use company selectors and
  transaction party labels/application details, not organization profiles.
  `pages/crm.tsx` has no company profile/form symbols. These require no edits;
  introducing new profile UIs or adding legal form to relationship summary bands
  would exceed this existing-form/profile task.
- Backend model, company input validation, DTO, repository and migration 000035
  already support `'' | llc | sole_proprietor | other`; no backend changes needed.
  Invalid values fail backend validation; unknown-value fallback concerns display,
  not a new enum or permission rule.
- Existing behavioral units: Admin `company-form.test.tsx` exercises real form
  submission and editing; Realization `add-company.test.tsx` exercises minimal
  creation and session refresh. Neither yet proves legal-form serialization or
  preservation. No organization-profile test currently exists.

## Guardrails

Use one shared kit module for the optional frontend type, select options and
safe Russian label lookup. Do not extract an entire company model or redesign
forms. Preserve business type, verification/access, revisions and refresh flows.
Use `Не указана` for missing/empty/unknown display (already the existing select's
empty label); never display an unknown backend code as an approved choice.

Protected dirty backend test and `test_creds.txt` were not read or changed.
No Git, tests, browser, runtime or source modifications performed by this slicer.
Unit-only verification overrides older automatic review/QA/integration gates.
Deferred independent QA descriptions in the plan are not dispatch authorization.
