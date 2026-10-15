# Source findings — local-ui-password-fixes-20261013

Authority: application hub `/root`. Assigned dependency:
`31b018ce1a4b49a2f56148459b4efa0893d345ce` (provided by hub; no Git operations).
Planning only; no application, runtime, credential, canonical-record or contract writes.

## Incremental source map

- `web/packages/kit/src/portal.tsx:117`: admitted pages always construct the
  `Разделы` nav. Its destinations already represent permitted section homes;
  separate buttons represent matching memberships for unavailable sections.
  Reuse those exact lists for visibility, preserving company selection semantics.
- `web/packages/kit/src/shell.tsx:158,184,227`: the same React node is inserted
  into Insurance, Admin and Workspace sidebars. No shell change is necessary.
- `web/packages/kit/src/ui.tsx:24`: Admin/Insurance sidebar anchors become blocks.
  `web/packages/kit/src/design.css:432` fixes `.btn-sm` at 32px while
  `:4345` and Admin `design.css:38` add 12px vertical/16px horizontal padding.
  Long labels also inherit button nowrap. Use one scoped switcher item style,
  with enough specificity to win against these existing sidebar selectors,
  wrapping and natural height. Do not globally change sidebar/button styling.
- `internal/modules/identity/service/password.go:68`: shared validator counts
  Unicode runes; minimum 12, maximum 128, then confirmation equality. Change only
  minimum and error text. Argon2 hash/verify parameters and existing hashes stay.
- Callers: `user.go:143` (creation), `:490` (bootstrap), `:564` (set/reset),
  `company.go:227` (first company admin), `auth.go:403` (own password change).
  `cmd/bootstrap-admin/main.go:64` delegates to Bootstrap; no separate minimum.
- Six frontend 12-character help texts: Admin `pages.tsx:171,359,532`, shared
  `company-employees.tsx:70,138`, shared `session.tsx:406`. No separate production
  password minLength=12 or client-side length check found in inspected web source.
- Existing focused tests: `portal.test.ts` already covers route memory, direct
  URLs, context sequencing, inactive-company choice and every shell variant.
  `password_test.go` covers hash compatibility; `user_test.go` covers required
  creation credentials. Company/staff and session unit suites cover relevant forms.

Production searches were limited to internal/, cmd/, web/ and tools/;
legacy mocks/docs, `.env`, keys and untracked `test_creds` were not read.
Search matches in unrelated tests (numeric amounts or long sample passwords)
are not policy and do not require changing.

The current unit-only user override suppresses independent review/QA and
integration execution. No checks were run during slicing. Layout correctness
will retain a manual verification limitation because jsdom is not a layout engine.
