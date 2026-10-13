# Source completion and local Git packet

2026-10-13, /root. P4 result-04.md is accepted with its 12 source hashes, passing
affected Go webui command and 10 Python FakeSystem cases. All P1–P4 source/test
writers explicitly released; no agent uses source or test resources. Accepted
results implement U1–U6 within unit-only evidence limits. Root owns remaining
canonical completion records. No build, runtime activation or release is claimed.

Grant the version_control actor exclusive Git use for one local source commit.
Branch must remain feature/unified-frontend-20261013; expected pre-commit HEAD
743bd6c4accadcd50afb59534bb66dabe2f3baf8. Repository root must be this project.
No branch switch, worktree, fetch, push, dev/main change, rewrite, conflict
resolution, cleanup, runtime, tests or hook execution is authorized.

Use source-commit-paths.txt as the exact allowlist (247 explicit paths, including
deleted old source/config paths, relocated destinations and optional unchanged
shell.test.ts). Stage only changed source paths from that list. Root's dirty
canonical docs are expected and must stay unstaged: AGENTS.md, web/AGENTS.md,
README.md, deploy/README.md, docs/justix-auto/business-logic.md,
docs/justix-auto/adr-15-unified-frontend.md, docs/justix-auto/dev/dev-state.md,
docs/justix-auto/dev/task-board.md, docs/justix-auto/dev/unified-frontend-20261013.md,
and this execution directory. Report any unexplained source path rather than
including it. Before staging, check that the index has no unexpected staged work.

Disable hooks per mutation command: `git -c core.hooksPath=/dev/null ...`.
Use literal explicit paths/pathspec-from-file; no `git add .` or wildcard staging.
Inspect staged paths and diff summary for scope, commit with message
`feat(web): unify frontend sections in one portal`, then record the actual commit
scope and SHA in source-git-result.md. This is Git provenance, not code review or
new verification. Leave canonical docs untouched and release the Git lease.
