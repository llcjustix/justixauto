# Conditional test-server admin bootstrap

Status: APPROVED FOR IMPLEMENTATION, 2026-10-15. User asks for admin credential
seeding during Docker CI/CD when no admin exists. Execution ID:
`ADMIN-BOOTSTRAP-20261015`; this does not close historical MFA/bootstrap tasks.

Application authority `/root` accepts the existing active-platform-admin
predicate and unchanged creation guard. Infrastructure authority accepts the
existing dev Compose route and migration → bootstrap → API sequence in
[ADR-16](../adr-16-dev-admin-bootstrap.md) and its
[decision](execution/admin-bootstrap-20261015/devops-decision.md).
OD-12 production recovery/release configuration remains outside scope.

Implement the [single bounded packet](execution/admin-bootstrap-20261015/packet.md):
package the existing CLI, add `-if-missing`, isolate a one-shot Compose service,
pass explicit GitHub secrets through the bootstrap step, and document setup.
No default login/password, account resets, public route, schema or new dependency.
Existing admin skips without input; invalid initial input, collisions and DB
failures block deployment. Concurrent valid seeds create at most one admin.

Dependency: current completed source at
`f91daa8e624da44f242a53c6677f127dcf0c87d8`, with dev ancestry
`4fa683ba4708bb71031534260ec5cb8742ce9977`. Version control creates
`infra/admin-bootstrap-20261015` from dev and fast-forwards to that dependency,
preserving the completed work. Untracked `test_creds.txt` stays unread and
unmodified. No worktree, remote push, server operation or production promotion.

Root owns canonical records; worker owns the packet's six source/runbook paths;
version_control owns branch/commit. No automatic review/QA chain per the latest
2026-10-13 policy. Finish all implementation before one focused final pass:
disposable PostgreSQL CLI behavior plus quiet Compose/workflow wiring checks.
No broad build, lint, typecheck or test suite. Record any unexecuted coverage.

Before-copies and SHA-256 for the existing canonical records are in
[entry/manifest.md](execution/admin-bootstrap-20261015/entry/manifest.md).
No final verification or live setup has yet been completed.
