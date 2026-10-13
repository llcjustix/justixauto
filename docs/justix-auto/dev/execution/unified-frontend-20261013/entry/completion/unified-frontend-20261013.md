# Unified frontend — 2026-10-13

Status: user-authorized implementation in progress; source design and four
bounded packets accepted in execution/unified-frontend-20261013/plan-approval.md.
User requests one app instead of four, opening the right section
from the user's role/access. Explicit follow-up: open the last accessible section
and retain an internal switcher when several sections are available.

## Scope and acceptance

U1. One React entry, BrowserRouter, session/login flow and query-client lifecycle;
one frontend npm workspace, Vite dev process and production output. Realization,
Islamic financing, Insurance and Admin remain permission-controlled sections,
not independently bootstrapped applications. Preserve the implemented 13-point
company/commerce changes and all existing section pages/actions.

U2. Route selection uses the server's effective permissions plus active company
kind, not role-name strings or company membership as a grant. Platform Admin is
available with matching platform permissions without requiring company context.
Dealer and distributor share Realization; accountant navigation retains its
existing permission rules. Verification remains an operation gate, not a new
whole-section login gate. No role grants or backend data permissions are widened.

U3. Keep permitted direct URLs and restore them after login. At the common entry
choose the last accessible section for the actual user/company; fall back to an
allowed home if history is no longer allowed. The switcher only offers valid
destinations. /identity/session supplies permissions for the active company only:
other memberships may offer a company-selection destination but cannot be treated
as already permitted sections. Select company and refresh session before rendering
its routes; handle multiple/no matching companies explicitly. Cancel/clear stale
query state and unmount prior company content across context/user/logout changes.
Do not leak previous-company data through late requests or retained dialogs.

U4. Keep existing Realization routes and /finance, /insurance, /admin deep links.
Internal navigation remains inside the SPA; repair Finance/Admin absolute paths
that used their old basename. Preserve search, sidebar filtering, active states,
settings/onboarding, errors, loading and forced-password flows. A denied route
must not mount unauthorized page content or flash an old company section.

U5. One portal build is served by Vite/Go/Docker/config. Preserve API/health and
static asset boundaries, document-only SPA fallback and security/cache behavior.
WEB_DIR now names the concrete dist directory. Update tracked configuration and
local commands together; do not read/edit private .env, start/stop services,
execute workflows, migrate data or deploy. Record the app/infrastructure contract
in ADR-15 with both authority acknowledgments.

U6. Run only affected unit tests once per relevant change. No automatic review/QA,
lint/typecheck/build/vet/race/browser/E2E/integration checks or React Doctor. Source
verification does not establish installation, deployment or release readiness.

## Execution

Application authority: /root. Infrastructure authority: /root/unified_frontend_infra.
Branch: feature/unified-frontend-20261013 in the main checkout; no worktrees.
Normal dev ancestry: 4fa683ba4708bb71031534260ec5cb8742ce9977. Adopted approved
predecessor dependency and task implementation base:
743bd6c4accadcd50afb59534bb66dabe2f3baf8. Version control created the task branch
from dev then fast-forwarded that branch only to the dependency; dev/main and
the earlier feature branch were not changed. Clean state confirmed at intake.

One source writer. Fresh scoped packets and exact path ownership are required;
primary owns canonical records and snapshots, workers only assigned results.
Version control alone writes Git; disable hooks per command and stage explicit
paths. No remote push/dev/main promotion or runtime action in this task.
The direct user implementation request and answered access decision authorize
this scope; do not add a second generic permission gate from the feature skill.
Product ambiguities, if discovered, block only their affected part.

Execution artifacts: execution/unified-frontend-20261013/. Historical mocks,
original spec-team documents and prior feature evidence remain preserved.
