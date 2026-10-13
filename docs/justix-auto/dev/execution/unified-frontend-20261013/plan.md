# Unified frontend execution plan

Task: docs/justix-auto/dev/unified-frontend-20261013.md, U1–U6.
Hub: /root; infrastructure: /root/unified_frontend_infra.
Base: 743bd6c4accadcd50afb59534bb66dabe2f3baf8.
Dev ancestry: 4fa683ba4708bb71031534260ec5cb8742ce9977.
Branch: feature/unified-frontend-20261013; main checkout only.
Status: proposed bounded packets for root acceptance. No source/tests/Git performed.
ADR-15 and infra-approval.md record both authorities; no additional gate is needed.

## Sequence and resource ownership

| Packet | Coherent behavior | Dependency | Input ceiling |
|---|---|---|---|
| P1 packet-01.md | Isolate session/context transitions from stale requests and UI | approved task/base | 16k tokens |
| P2 packet-02.md | Select and navigate authorized sections in one router | P1 exact handoff | 18k tokens |
| P3 packet-03.md | Make the actual single portal entry/workspace and preserve feature modules | P2 exact exports | 18k tokens |
| P4 packet-04.md | Serve and launch the portal through existing runtime consumers | P3 concrete paths; ADR-15 | 16k tokens |

One worker (gpt-5.6-terra/medium unless root explicitly records escalation) at a
time, fresh scoped context, exclusive main-checkout source lease. No children.
No branch switching while any packet uses source. Version-control actor alone
handles authorized Git operations outside these packets; no remote push here.
Workers first run the mandated repository-root guard, bash tools/check-git.sh.
This is the workspace safety guard, not a development verification expansion.
No private .env reads, network/install, builds, lint, typecheck, React Doctor,
browser/listeners, real stop helper, DB/migrations, workflows or deployments.
Affected unit tests run once per relevant source change; after a meaningful fix,
rerun only affected failed/changed cases. Do not repeat passing tests for ceremony.

Each worker writes result-NN.md in this execution directory with owned paths,
actual exported signatures, behavior decisions within this plan, unit command and
raw result, exclusions, remaining failures, and released lease. Root supplies
the predecessor accepted result and immutable source checkpoint before the next
packet. If a signature must differ, report it before downstream dispatch; do not
leave downstream workers to guess. Root owns canonical documentation/backups.

## Shared implementation contract

P1 preserves SessionGate, SessionView, useSession, useCommercialAccess and HTTP
call helper signatures. Extend the returned SessionApi with scopeKey: string.
scopeKey is an opaque stable identity for the admitted user/company/branch
context; change on actual isolation-boundary changes, not every same-context
refresh. Existing consumer code need not construct it. SessionGate owns cache
retirement and unmounting; calling components never need qc.clear afterward.
selectCompany/setBranchScope resolve only after the fresh admitted session,
reject on failure; no stale view remains visible while transitioning.
No new endpoint or permission, no replacement auth protocol.

P2 exports from @justixauto/kit:
- type SectionId = 'realization' | 'financing' | 'insurance' | 'admin'.
- type PortalSection = Omit<ShellOptions, 'rootId' | 'basename'> &
  { id: SectionId; prefix: '/' | '/finance' | '/insurance' | '/admin' }.
- function PortalRoutes({ sections }: { sections: PortalSection[] }): ReactNode.
  It consumes the existing router/session; it never creates a router/provider.
- function mountPortal({ rootId, sections }:
  { rootId: string; sections: PortalSection[] }): void.
  It owns one createRoot, QueryClient provider, root BrowserRouter and SessionGate.
NavItem.to values supplied to PortalRoutes are final absolute browser URLs.
Provider links therefore carry their prefixes in P3. Keep permitted() semantics
(including OR arrays) unchanged. Search.path returns final absolute URLs.
Keep old mountApp available for intermediate old entries only; P3 removes its
unused bootstrap function/export once all four entries are retired.
No repeated per-section bootstrap, QueryClient, basename router or login gate.

P3 creates @justixauto/portal, root ID portal-root, base /, strict port5191,
dist at web/apps/portal/dist, feature modules under src/features/{old-app-name}.
sections.tsx exports sections: PortalSection[]; main.tsx only imports section
styles/sections and invokes mountPortal. React feature pages retain shared kit
imports and relative feature-local imports. Existing shared package paths stay.
Pure Vite request policy lives in src/document-policy.ts, imported by Vite config,
with request input containing only method/url/headers; tests never import Vite
config or loadEnv. P3 reports exact helper export used; P4 matches behavior, not
a guessed TypeScript API.

P4 simplifies webui.Mount to Mount(e *echo.Echo, distDir string); cmd/api supplies
cfg.WebDir directly. Remove old four-app Apps/App dispatch. WEB_DIR is concrete
dist; empty stays API-only. Docker output /app/web; local output portal/dist.

## Access policy limits

Active company kind plus existing effective permission-filtered nav determines
usable company-section routes. Permission-free dashboard/settings remain scoped
to an active matching company, preserving setup and verification behavior.
Admin admits only its currently granted nav entries, including no-company users.
Never derive grants from roles or inactive membership. No changes to OD-01/10
provider finance, OD-05/06 operational/legal verification, OD-09 refund or OD-12
recovery policy. Preserve historical approved behavior; stop if new policy is
needed. Same memberships with no matching/currently permitted company route get
explicit selection/denial, not speculative grants or repeated selection loops.

## Verification specifications and limits

No extra QA workflow artifacts or automatic QA jobs are created. Compact dormant
independent QA/final-SHA specifications appear only as evidence gaps in
acceptance-map.md, not dispatch dependencies: the latest explicit user policy
disables automatic review/QA and integration checks. They may execute only after
an explicit request for additional checks.
Earlier unit reports are not proof of a final installed build or live behavior.
Canonical U1–U6 and all thirteen predecessor preservation obligations map in
acceptance-map.md. Source hashes and exact relocation ownership are in
source-hashes.md and move-manifest.md. No backlog entries are added.

Stop on unexplained source drift, contract mismatch, missing locked tooling,
unowned path needs, broadened product grants, lifecycle uncertainty, or a context
ceiling that cannot contain the packet. Write a bounded reslice proposal; report
RESLICE_REQUIRED if canonical scope would change. No blind retries or skeleton
implementation left for another packet.

