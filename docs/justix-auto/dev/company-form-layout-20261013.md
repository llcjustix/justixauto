# Company form layout and business-type choice — 2026-10-13

Status: SOURCE COMPLETE; source writer released. Application authority: /root.
User requested organization legal form before company name in one row, then
business type as radio buttons below that row, defaulting to dealer.

## Approved scope

F1. In Admin common company create/edit and Realization create/edit forms,
show organization legal form first and company name second in the same grid
row. Preserve optional legalForm, prefill, payloads and existing responsive rules.

F2. Admin seller onboarding displays a full-row business-type radio group
immediately below that row. Options are Дилер / Дистрибьютор; dealer is initially
selected and submitted unless the operator selects distributor. This supersedes
the earlier Admin creation UI's blank/deferred type choice. API optionality and
legacy records remain unchanged. Provider forms do not show seller business type.

F3. Reuse the radio control for Admin's existing one-time legacy type assignment.
Already-assigned types remain immutable; do not introduce a type editor or
company-user self-assignment. Realization creation still uses its existing API.

F4. Use native accessible radios with shared form state and errors. Cover default
submission, alternate selection/exclusivity and unchanged assignment permissions
through affected units. No layout/CSS snapshot tests or broad redesign.

## Execution

Dependency HEAD 3aeb7b8feddb81cb49b7f0879a041d22ce760643 on
fix/company-legal-form-20261013 (dev ancestry 4fa683b). Version-control prepares
fix/company-form-layout-20261013 at identical dependency source without checking
out older files under active local watchers. Main checkout only, one source writer.
Preserve unrelated dirty internal/modules/identity/service/company_contract_test.go
and unread test_creds.txt. No backend, schema, permission, runtime or remote action.

Root and web/AGENTS.md unit-only policy applies: affected units once per relevant
change, no automatic review/QA, lint, typecheck, builds, browser/E2E/integration
or React Doctor. Task_slicer prepares one bounded frontend packet; primary
accepts it and maintains canonical records; worker edits only assigned paths;
version-control owns branch and explicit-path local commit. Artifacts live in
execution/company-form-layout-20261013/. Pre-edit board/state snapshots are in
that directory's entry/manifest.txt.

Product references: business-logic.md Applications and access;
product-workflow-improvements-20261011.md R1/R2; company-legal-form-20261013.md.
No open domain decision is changed beyond the user's explicit creation default.

## Completion

FE1 implements F1–F4 in four production files and three affected test files.
Legal form precedes name in Admin and Realization create/edit. Admin seller
creation offers full-row native dealer/distributor radios with dealer selected;
the same field supports existing one-time assignment. Fixed types stay immutable.
Shared radio validation, state and accessible errors are included.

Twenty-five distinct affected units passed: the initial five-file run had four
harness assertion failures; only those three affected files were rerun after
repair (21 passes), retaining four Realization passes. A later narrow CSS-only
selector/legend/wrapping correction has no browser/layout evidence. No extra
checks ran. See execution/company-form-layout-20261013/implementation-result.md
and completion.md for exact commands, hashes and limits. Unrelated dirty files
remain preserved. No runtime, database, credentials or remote action.
