# Optional insurance and independent contract signing

Status: SOURCE COMPLETE, 2026-10-14. Primary `/root`. Latest explicit user decision:
«Переименуй просто на страхование и договор может быть подписан без страхования».
This supersedes the initial scoring-guidance proposal and the mandatory scoring
portion of product-workflow-improvements-20261011 R12.

Name the sale tab and accessible panel «Страхование». Contract signing and
recording must work without insurance or a current demo-scoring run. Remove
the scoring prerequisite in both frontend entry points and the shared backend
signing service. Keep optional scoring honest about its test nature; no copy
may claim it is required for signing. Preserve signed-scan, permission, tenant,
revision, immutable-version and current-terms validation. OD-08 insurer decision
authority and the separate vehicle-handover checklist remain unchanged.

One serialized full-stack packet at execution/scoring-guidance-20261014/packet.md.
Use main checkout branch fix/scoring-guidance-20261014 from the current local
dependency08c2efbc72877644f653c0c8cf8fe5af31f60ca3, preserving recent local fixes.
No worktrees, deployment or push. Preserve unrelated identity test and unread
test_creds.txt. Update existing assertions that encode the superseded rule;
run affected unit/component tests once at the end. No browser/integration,
broad check/review loops or real sale mutation. Existing local-run authorization
covers a bounded reload of the local API after the source change; no migration
or environment configuration changes are needed.

Implemented in the assigned packet; [final affected units](execution/scoring-guidance-20261014/final-validation.md)
passed: seven backend cases and six frontend cases. One pre-existing invoice
fixture lacked its required permission; repaired the fixture and reran only
that case. No new test suites or product permission changes. Source/check leases
released. Local activation is recorded separately in the runtime result; no push.
