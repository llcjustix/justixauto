# P2 — Hide reciprocal cabinet switching

Approved by root for the explicit screenshot follow-up recorded in
../../accounting-queue-ui-20261014.md. Dependency: P1 source writer released.
Branch fix/accounting-queue-ui-20261014, base a99f72c7c72463943a96f15cae049d5d47630f18.

One source owner: web/packages/kit/src/portal.tsx. Read web/AGENTS.md and
switcher-findings.md; use apply_patch after bash tools/check-git.sh. Derive a
separate displayed-destinations list in PortalRoutes to omit Accounting from the
Realization switcher and Realization from the Accounting switcher. Use it both
for the existence check and mapping, preserving company-choice behavior. Other
sections keep their entries. When no meaningful section/company choice remains,
render no switcher. Do not change section admission, grants, default/remembered
entry, direct routes, company context or Shell.

No dependencies, backend, financial changes, new tests or other source edits.
No Git/canonical changes, runtime actions or worker verification. Root performs
one final existing-unit pass for accountant.test.tsx and portal.test.ts after
both P1/P2 are done; no browser/lint/typecheck/build/React Doctor/review loops.
Protected identity test and unread test_creds.txt must remain untouched.
Worker records result and source hash in switcher-result.md, then releases.
