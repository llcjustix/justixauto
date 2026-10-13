# FE1 implementation result — company-form-layout-20261013

Status: DONE

Implemented F1–F4 on `fix/company-form-layout-20261013` from assigned source
`3aeb7b8feddb81cb49b7f0879a041d22ce760643`. Legal form now precedes company
name in the Admin and Realization create/edit field definitions. Admin seller
creation and legacy missing-type assignment use the shared, full-row native
radio field with dealer initial value; provider forms and ordinary edits do not
receive that field. The shared dialog validates required radios and associates
field errors with the radio group and controls.

## Verification

Pinned toolchain: Node `v24.21.0`; npm `11.19.0` from
`docs/justix-auto/dev/local/toolchains/node-24.21.0/bin`.

Unit command count: 2.

1. From `web/`:
   `PATH=/Users/bakhromachilov/startups/justixauto/docs/justix-auto/dev/local/toolchains/node-24.21.0/bin:$PATH npm run test:unit -- apps/portal/src/features/admin/company-form.test.tsx apps/portal/src/features/admin/company-verification.test.tsx apps/portal/src/features/realization/add-company.test.tsx apps/portal/src/features/realization/company-profile.test.tsx packages/kit/src/form-dialog.test.ts`
   Result: 25 tests; 21 passed and 4 failed. Failures were test-harness-only:
   unavailable `toBeChecked`/`toHaveAttribute` matchers, a missing `within`
   import, and the legacy dialog submit button label is `Назначить бизнес-тип`.
2. Affected-failure rerun only, from `web/`:
   `PATH=/Users/bakhromachilov/startups/justixauto/docs/justix-auto/dev/local/toolchains/node-24.21.0/bin:$PATH npm run test:unit -- apps/portal/src/features/admin/company-form.test.tsx apps/portal/src/features/admin/company-verification.test.tsx packages/kit/src/form-dialog.test.ts`
   Result: 3 files passed; 21 tests passed; 0 failures.

After the rerun, a narrow CSS selector correction changed the radio input rule
to target the fieldset that owns both classes, scoped its legend, and permits
wrapping. No layout unit exists in the authorized list; no extra check ran.

## Changed owned paths and SHA-256

- `web/apps/portal/src/features/admin/pages.tsx` — `3baef00d37781485cccdf1139d30f80dd253c76362f5f5aeae51bd388c5f1927`
- `web/apps/portal/src/features/realization/pages/settings.tsx` — `65c725ba3d149e9a4caa71f3f1d3cd32cdb651b03dff57b622174217090c8a18`
- `web/packages/kit/src/ui.tsx` — `dc56d8e9b45745b5f3636af7157cee1d37ad703f495a73909895030c5da986e9`
- `web/packages/kit/src/design.css` — `6aa8b257b1be51eeeb85f5e1725b2be579e06fbee40f11716d55202b0dce87be`
- `web/apps/portal/src/features/admin/company-form.test.tsx` — `93110b3c0919fb896aa9af011dde70bc4abb15737996e101672500268d7ed009`
- `web/apps/portal/src/features/admin/company-verification.test.tsx` — `344390f6bc681f37f9f8c752b39df76662dded4a77568ec0a2334b60ccdc767d`
- `web/packages/kit/src/form-dialog.test.ts` — `f8ab303bf16300a9ea8df808568f8d3bd831ed4b601a63bb993c27d381614b2f`

Protected unrelated file preserved: `internal/modules/identity/service/company_contract_test.go`
remains `a4d4b3f67107b5362810dcc96f68f1de8e59cae466dacafb7e100e29a59fe99f`.

Excluded: Git writes, canonical document edits, package/API/backend changes,
runtime/remote actions, lint, typecheck, builds, browser/E2E/integration,
React Doctor, review/QA loops, and visual layout validation.
