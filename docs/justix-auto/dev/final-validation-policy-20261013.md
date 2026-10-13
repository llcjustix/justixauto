# Final validation policy — 2026-10-13

Status: APPLIED. Explicit user decision: remove time-consuming unit-test work
from each implementation step, test at the end, and use E2E when needed.

Complete the whole requested change before validation. Unit-test writing and
runs are not mandatory per edit, subtask, packet or agent handoff. The primary
chooses the smallest useful check of the finished behavior; focused browser/E2E
is authorized when necessary, and existing units may be selected when useful.
Repair failures and rerun only the affected final scenario. Documentation-only
changes need no tests. Existing suites and CI remain in place.

This supersedes the 2026-09-26 unit-only policy and older packet checklists for
all agents. It does not introduce automatic lint/typecheck/build/vet/race,
React Doctor or review/QA loops. Environment scope, source ownership, Git,
production and human integration rules remain in force.

Applied to active root/scoped AGENTS and CLAUDE guidance, Claude role briefs,
agent-workflow.md and workflow.md. Historical receipts remain historical.
No tests or runtime actions were needed for this documentation-only change.
Recoverable before-copies and SHA-256 manifest:
execution/final-validation-policy-20261013/entry/manifest.txt.
