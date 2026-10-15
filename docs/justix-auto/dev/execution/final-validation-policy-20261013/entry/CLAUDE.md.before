# JustixAuto — Claude Code

User decisions, 2026-09-26: main agent handles routine work; delegate only when
useful. During development, ONLY UNIT TESTS are allowed as verification.
These decisions override the reading lists, role handoffs, verification rules
and Git-role exclusivity in AGENTS.md, scoped rules and historical task packets.

## Work directly

- Answer questions, inspect code, implement routine fixes and perform ordinary
  Git operations in the main session. Do not create a seven-role workflow.
- Read only relevant source, callers and scoped technology rules. Consult
  business-logic.md and open-decisions.md for affected product rules; do not
  invent unresolved behavior. Load the selected task/dev-state only when needed.
  Do not load the full backlog or Gaze audits for a small fix.
- Read-only explanations require no tests. For changes, run affected UNIT TESTS
  once. Do not add tests for simple reversible wording/configuration edits.
  Do not rerun passing tests unless the relevant code changes.
- No automatic lint, formatting checks, typecheck, builds, vet, race runs,
  dead-code scans, OpenAPI generation/checks, doctor, config validation commands,
  browser/E2E/integration tests, Docker fixtures, or review/QA/aggregation loops.
  Do not invoke make check, make test-go, devtool test or broad go test ./....
  Existing task/skill checklists do not override this rule.
- End-to-end tests only on explicit user request (user decision 2026-09-26):
  do not write, extend or run them otherwise — HTTP flow tests with
  internal/e2e or a database/S3, browser checks. This overrides
  internal/AGENTS.md, which asks for end-to-end behaviour tests.
- For Go unit tests, use bash tools/go.sh test with the affected package path;
  unset TEST_DATABASE_URL and TEST_S3_ENDPOINT so integration tests stay skipped.
  For React, target the affected Vitest unit files. Documentation-only work needs
  no checks. Additional verification requires a new explicit user request.
- Local Git hooks do not repeat checks. CI/release checks remain separate; do not
  claim their results or integration readiness from local unit tests.
- Use Edit/Write, preserve unrelated changes, and report the result briefly.

## Delegate only when useful

Use a project role for substantial independent implementation or specialist
advice, not for routine edits, Git, or additional verification. The roles in
.claude/agents are optional capabilities. Reviewer/QA/aggregation roles are used
only if the user explicitly requests that work. Do not use global crew-* agents.
Keep the configured model/effort; do not mechanically mirror Codex configuration.

Pass a small brief with owned paths, source identity, relevant rules and expected
result. Delegates omit project-wide instructions, so include the unit-test-only
policy and all applicable constraints. Read the partial result before deciding
whether to resume a delegate that reaches its turn limit.

At most three delegates total, one source writer, and one holder per exclusive
resource. No worktrees or simultaneous source edits. Claude supports nested
agents; this project deliberately disables nesting so the main session owns
dispatch. DevOps retains infrastructure decision authority.

## Project boundaries

- Backend: Go modular monolith, Echo + GORM, handler → service → repository,
  one PostgreSQL, single public schema with module-prefixed tables, SQL
  migrations (ADR-14). Modules call each other's services, not their tables.
  Money uses integer minor units and currency. Frontend: four React apps; HTML mocks are the design reference.
  Do not edit generated mocks or copy Gaze code/credentials.
- Verify repository identity with git rev-parse --show-toplevel before edits:
  it must be this project, not the enclosing startups repository. For Claude,
  this direct read replaces the compiling tools/check-git.sh wrapper.
- One task branch from dev in the main checkout. Never switch branches while
  source is dirty or in use. Stage/commit explicit paths. No force pushes,
  automatic conflict resolution, history rewriting or destructive cleanup.
- Ordinary task/, feature/, fix/ and infra/ pushes to the same-named origin ref
  retain standing authorization. User decision 2026-09-27: agents may push to
  dev (fast-forward only, never force); a push to dev deploys to the dev
  server. main/production promotion is human-only; agents never write main.
- No secrets, external messages or deployment without explicit authority.
  The permission hook is an additional check, not a sandbox or proof of approval.
- The main session owns canonical records. Save a recoverable snapshot and
  SHA-256 before changing one. Never mark unfinished or unintegrated work done.
