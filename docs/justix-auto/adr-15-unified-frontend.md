# ADR-15 — One frontend application with access-scoped sections

Date: 2026-10-13. User direction: combine the four frontends into one app; select
the section by access, remembering the last permitted section with an internal
switcher. Application hub accepts this source design. Infrastructure authority
acceptance is recorded in dev/execution/unified-frontend-20261013/infra-approval.md
before its source packet is dispatched. Deployment is not authorized here.

Use one npm workspace `@justixauto/portal` at `web/apps/portal`, Vite base `/`,
strict development port 5191 and output `web/apps/portal/dist`. Keep feature
areas as ordinary source modules. One React entry, session provider, root router
and query-client lifecycle own authentication, context and internal navigation.
No dependencies or library versions need to change.

Retain existing browser URLs: Realization routes plus `/finance`, `/insurance`
and `/admin` and their nested links. Section routing uses current server grants
and company kind; backend authorization remains unchanged. A membership in an
inactive company is only a possible context destination, not a permissions
projection. Company switching must finish context selection and session refresh
before admitting routes and must clear prior tenant query/dialog state. Platform
Admin works without company context. Last-section storage is advisory, scoped to
the authenticated user/context and validated against current access.

`WEB_DIR` names the concrete directory containing `index.html` and `assets/`:
local `web/apps/portal/dist`, container `/app/web`; empty remains API-only.
Update the API mount, config comment, example env, Docker, Helm values and tracked
dev workflow together. Existing private local values require a documented manual
configuration update; this task does not inspect or mutate them.

Vite and Go serve the single entry for permitted document GET/HEAD deep links.
API and health paths, missing assets, source/module paths, file-shaped requests
and malformed/traversal requests never become entry HTML. Preserve static bytes,
hashed asset caching, non-stale entry, CSP/security headers and empty HEAD bodies.
No listener or live environment is needed for the corresponding pure unit tests.

`make dev` runs API plus one web process, retaining the existing API toolchain;
`make web` targets portal. Update the stop-helper's exact process recognition
without weakening ownership checks or invoking it. Existing service/ingress and
database topology, cookie/CSRF contracts, image bases and CI verification policy
remain unchanged. No rollout, production promotion or operational readiness is
inferred from these source changes.
