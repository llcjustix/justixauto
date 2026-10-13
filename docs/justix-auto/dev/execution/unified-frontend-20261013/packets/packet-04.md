# P4 — Single portal runtime consumers and local process identity

Task/base/hub: plan.md. U5,U6. Dependency P3 paths/policy handoff; ADR-15 and
infra-approval.md already accepted by both authorities. Worker 16k input ceiling,
exclusive source/test lease; no children/Git/runtime. Read root/workflow,
internal/AGENTS, tools/AGENTS, deploy/AGENTS; unit-only override applies.

## Exact ownership

- internal/pkg/webui/webui.go
- internal/pkg/webui/webui_test.go
- cmd/api/main.go (webui mount composition only)
- internal/config/config.go (WebDir meaning comment only)
- Makefile (app/default, web/web-build/dev/stop-related descriptions and launch argv)
- tools/stop-apps.py
- tools/test_stop_apps.py
- deploy/Dockerfile
- .github/workflows/deploy-dev.yml (tracked WEB_DIR override only)
- deploy/values.yaml (webDir default only)
- .env.example (public example origins and WEB_DIR/comment only)
- .air.toml (single frontend-related comment only; root explicit scope amendment)

No changes to private .env, Compose/Helm templates/ingress, base images,
secret references, module APIs, migrations or CI verification policy.
Root owns README.md, deploy/README.md, root/web AGENTS and business-logic
amendments/snapshots; report concrete instructions it must include.
Result: execution/unified-frontend-20261013/result-04.md.

## Coherent runtime behavior

webui.Mount(e *echo.Echo, distDir string) mounts one concrete index/assets directory.
Remove old Apps(root)/App/prefix selection. API main passes cfg.WebDir directly,
preserving empty=API-only. Missing index preserves API-only behavior. Every old
section document URL reaches the same entry; do not introduce trailing-slash
redirects that discard query. Keep real static bytes/mime, immutable hashed assets,
non-stale entry, CSP/security headers and HEAD empty body.

Document GET/HEAD only, explicit positive text/html Accept; when present,
Sec-Fetch-Dest document/iframe/frame and Mode navigate must agree. Neither
wildcard nor q=0 is document intent. /api and descendants, /healthz and descendants,
source/module paths (/src,/node_modules,/@...), malformed/traversal paths,
missing assets including extensionless, arbitrary .html and file-shaped misses
must never receive entry HTML. Check raw and decoded path before cleaning so
traversal does not become an allowed route. /index.html itself is document-only.
Preserve ordinary static asset serving for non-HTML requests. API/health routes
registered outside webui must remain authoritative. Align these cases with P3
pure Vite helper; no browser/server process required. Do not infer user
authorization from fallback: React/server APIs enforce grants.

Docker copies only /src/web/apps/portal/dist -> /app/web; WEB_DIR=/app/web.
Helm default and tracked deploy-dev override must also use /app/web; local
.env.example uses web/apps/portal/dist and only localhost/127.0.0.1 port5191.
No edits to existing private .env; root documents manual changes.

make web runs @justixauto/portal (remove four-app selector/default ambiguity);
make dev supervises api,web with Air and one Vite process; make web-build invokes
one portal build script. Keep existing db/migrate prerequisites and API tooling;
never invoke these targets in this packet.
Stop helper recognizes exact portal cwd, supervisor names api,web, port5191, and
the actual selected npm argv. Preserve same UID/checkout ownership, PID start-time
revalidation, unknown-listener failure and graceful supervisor-before-child ordering.
Old unrelated app paths and neighboring portal-copy must not become valid targets.
Do not weaken identity safeguards or kill existing processes. Preserve safe dotenv
parsing implementation but never exercise it on actual private .env in this task.

## Pure unit execution

bash tools/go.sh test -vet=off ./internal/pkg/webui -run 'TestMount|TestPortal' -count=1
python3 -m unittest discover -s tools -p test_stop_apps.py

Only affected suites, once per relevant change; no Go vet/race/build/integration
wrappers, no tools/test-go.sh or make test. Go test disables its default vet;
compilation occurs only as part of the affected unit invocation.

Update TestMount and/or add TestPortal* using httptest/tempfiles only: old section
paths all same entry, API/health exclusions, actual static bytes/cache/mime,
entry/security headers, missing index, GET/HEAD/POST, full document-intent matrix,
missing extensionless assets, malformed/raw/encoded traversal and arbitrary HTML.
Replace historical traversal-index expectation with 404 boundary.
Python tests exclusively FakeSystem: one portal/supervisor, standalone API/Vite,
exact cwd/names/ports/argv, old/neighbor/foreign process rejection, PID reuse,
ownership, escalation, empty/idempotent and unknown listeners. Never execute
stop-apps.py main or real process discovery/signaling in a test.

Stop on mismatch with ADR/P3 policy, unexpected runtime consumer outside ownership,
or need to change deployment topology/CI policy. Record changed consumer values,
unit outcomes and manual local WEB_DIR adjustment for root documentation. No
installation, image validity, deployment or release-readiness claim.

