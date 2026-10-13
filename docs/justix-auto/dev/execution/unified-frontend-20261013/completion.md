# Unified frontend completion

2026-10-13. **Source implementation complete.** User-authorized scope U1–U6
is implemented on `feature/unified-frontend-20261013`. Source commit:
`2b41eebb22b11c7e4ea680690d56385c94dd58d7`, parent/base
`743bd6c4accadcd50afb59534bb66dabe2f3baf8`, preserving the preceding thirteen
company/commerce changes and dev ancestry `4fa683ba4708bb71031534260ec5cb8742ce9977`.
All implementation/test writers have released. No remote push or dev/main
promotion was performed. [Source Git receipt](source-git-result.md) records
exact scope; canonical completion documents are committed separately afterward.

## Delivered behavior

- One `@justixauto/portal` workspace, React entry, router, session gate and query
  lifecycle replace four independent applications. All 94 feature source/test/
  stylesheet moves are recorded in move-manifest.md; old app entries/configs are
  retired. Earlier company, accounting, B2B, retail and insurance functionality
  is retained in ordinary feature modules.
- Current effective permissions and active company kind select usable sections.
  Dealer/distributor Realization, accountant OR permissions and no-company
  platform Admin remain supported. Verification continues to gate operations.
  Inactive memberships are labelled context choices, not assumed grants.
- A permitted direct URL takes precedence; `/` restores the last accessible
  section for the actual user/context, with a safe allowed fallback. An internal
  switcher is available in every shell variant. Existing seller, `/finance`,
  `/insurance` and `/admin` addresses, search and query/hash behavior are retained.
  Denied page elements are not mounted before context/access admission.
- Context changes retire old requests, query/mutation cache and transient pages/
  dialogs before admitting the refreshed session. Late data, CSRF and 401 replies
  cannot affect the new session; already sent mutations are never replayed.
- One Vite process uses strict port 5191 and one `portal/dist` output. Go, Docker,
  tracked environment values and local commands consume the concrete directory.
  Document-only fallback preserves API/health/static/source boundaries, headers,
  caching and HEAD behavior. The stop helper retains exact process ownership.

## Affected unit evidence

| Packet | Latest affected result | Receipt |
|---|---|---|
| P1 session/HTTP isolation | Kit: 2 files, 12 passes; Realization: 2 files, 5 passes | [result-01.md](result-01.md) |
| P2 section router/switcher | Kit: 3 files, 19 passes | [result-02.md](result-02.md) |
| P3 portal composition/policy/navigation | Portal: 3 files, 6 passes | [result-03.md](result-03.md) |
| P4 serving/local process identity | Go webui command passed; Python FakeSystem: 10 passes | [result-04.md](result-04.md) |

Receipts record exact commands, final source hashes and relevant-change reruns.
P3's shell hash supersedes the P2 hash only for removing the obsolete bootstrap.
Pure units used mocked requests, temp files and FakeSystem; no listeners or live
process control. Two bytecode cache files produced by the Python suite were
preserved outside the checkout at
`/private/tmp/justixauto-unified-test-bytecode-20261013/`; they are not source.

Per the user's unit-only development policy, no lint, typecheck, build, React
Doctor, browser/E2E/integration, automatic review/QA, installation or final-SHA
release verification was performed. These packet results are evidence for the
changed behaviors, not proof of a deployed or release-ready application.

## Configuration and delivery limits

For an existing private local `.env`, manually change `WEB_DIR` to
`web/apps/portal/dist` and portal origins to localhost/127.0.0.1 port 5191.
The image/Helm/dev workflow use `/app/web`; empty `WEB_DIR` remains API-only.
Private `.env` was neither read nor modified. No service start/stop, migration,
workflow execution, image build or deployment occurred. The latest feature and
this consolidation therefore remain source deliveries until runtime activation.

Application and infrastructure authorities accepted
[the frontend ADR](../../../adr-15-unified-frontend.md); infra-approval.md and
plan-approval.md preserve the bounded source contract. Historical task statuses,
mock bundles, original spec-team material and prior execution evidence were not
rewritten. Before-copies/hashes for canonical updates are under entry/.
