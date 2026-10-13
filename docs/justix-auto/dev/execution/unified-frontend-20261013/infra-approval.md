# Unified frontend — infrastructure source-design acceptance

Date: 2026-10-13.
Authority: `/root/unified_frontend_infra` (DevOps infrastructure hub).
Decision: **accepted for bounded source implementation**; no correction to the
recorded contract is required before slicing or dispatch.

Accepted records read in this turn:

- `docs/justix-auto/adr-15-unified-frontend.md`.
- `docs/justix-auto/dev/unified-frontend-20261013.md`, especially U5–U6.

The shared application/runtime contract is one `@justixauto/portal` workspace
at `web/apps/portal`, one React entry and Vite process, base `/`, strict port
5191 and one output at `web/apps/portal/dist`. Existing realization, `/finance`,
`/insurance` and `/admin` browser routes remain compatible. Access selection,
remembering the last accessible section and the internal switcher remain under
application authority and existing server authorization.

`WEB_DIR` identifies the concrete built directory: local
`web/apps/portal/dist`, container `/app/web`; empty remains API-only. The API
mount/config documentation, example environment, Docker copy/default, Helm
default and tracked dev-workflow environment must change together. Document the
manual update required for private local configuration without inspecting or
modifying it.

Both Vite and Go retain document-only GET/HEAD entry fallback with API/health,
static-asset and source/module boundaries and malformed/traversal rejection.
Preserve real static responses, hashed-asset caching, non-stale entry responses,
security headers and empty HEAD bodies. These transport rules do not confer
application permissions. Update local command composition and the stop-helper's
exact process recognition together, retaining its ownership and process-identity
checks; do not invoke the stop helper.

No dependency/version, image-base, service/ingress, database-topology, cookie/CSRF
or CI-verification-policy change is accepted by this record. Verification is
limited to affected pure unit tests, once per relevant implementation change.
This is a design-authority acknowledgment, not code review, QA or evidence of
installation, build success, runtime behavior or release readiness.

No tests, builds, dependency installation, source changes, Git operations,
private-environment reads, service actions, workflows or deployments were
performed for this acknowledgment. Only this assigned execution artifact was
written. No source, test, Git or runtime lease is held; read-only checkout use is
released to the primary.
