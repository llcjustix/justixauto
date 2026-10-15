# Warehouse manual receipt refresh hook — 2026-10-13

Status: SOURCE COMPLETE; two selected final receipt cases passed. Source/check
leases released; final-validation.md records the exact source hash and evidence.
Authority: /root from user report at
http://127.0.0.1:5191/warehouses. Screenshot shows ReferenceError: useRefresh is
not defined in ManualReceipt (inventory.tsx:208). Inspection confirms the hook
is called but missing from the @justixauto/kit import; kit exports it via shell.

Scope: add the existing useRefresh import in
web/apps/portal/src/features/realization/pages/inventory.tsx only. Preserve all
receipt inputs, financial/inventory semantics, validation, revisions, duplicate
guards, refresh and permissions. No new tests, dependencies or runtime changes.
Existing product rules and mock-map warehouse receive flow remain authoritative;
this restores intended behavior and introduces no product decision.

Branch fix/warehouse-receipt-refresh-20261013 at immediate current UI dependency
239dcfda1c59846eac8a67d24680acc93613d759, same main checkout. This retains the
user's current local UI changes. One source writer; version_control owns Git.
Preserve unrelated backend test SHA a4d4b3f67107b5362810dcc96f68f1de8e59cae466dacafb7e100e29a59fe99f
and unread test_creds.txt. Local commit only, no push or deployment.

Final-only validation: primary runs the existing two receipt-colors.test.tsx
cases matching 'colors manual receipt .* pins one pair' after the fix. These
mount the actual dialog/form, exercise identified/unidentified submission,
duplicate guard, revision and successful refresh. No extra suites, new tests,
typecheck/build/lint/browser/review/QA chain. Record final source hash and output
in execution/warehouse-receipt-refresh-20261013/. Root/web AGENTS apply.
