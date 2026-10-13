# Infrastructure map — read-only authority handoff

Actor /root/unified_frontend_infra, DevOps authority Astra/xhigh; base743bd6c.
No source/Git/env/process changes, tests or runtime checks. Its proposed one-app
contract is accepted by the application hub in ADR-15, pending recorded exact
infrastructure acknowledgment before source dispatch.

Exact producers/consumers: web/package.json and package-lock.json; four old app
package.json/index.html/vite.config.ts/vitest.config.ts/tsconfig.json; Makefile;
tools/stop-apps.py; internal/pkg/webui/webui.go and webui_test.go;
cmd/api/main.go; internal/config/config.go; deploy/Dockerfile;
.github/workflows/deploy-dev.yml; deploy/values.yaml; .env.example.
Root handles current README/deploy README/AGENTS documentation and .air.toml comment.

One workspace @justixauto/portal, base /, strict port5191, portal/dist; one Docker
copy to /app/web. WEB_DIR means concrete dist, empty remains API-only. Deploy-dev
currently overrides image env with web/apps, so tracked workflow must change too.
Helm config already forwards the value. Compose/ingress/service topology needs
no change. CI already uses Make/npm; preserve its verification policy.

Lockfile includes old canonical workspace/link records and old extraneous
web/apps/* records; reconcile identities without dependency upgrades. Vitest
workspace discovery reads each workspace manifest/config; portal includes all
relocated src/**/*.test.{ts,tsx}. Existing apps/* lint/coverage patterns fit it.

Make dev currently runs four Vite processes. Change supervisor to api,web and
update stop-apps.py exact cwd/name/ports recognition together; preserve process
ownership/identity safeguards. Only FakeSystem tests, never run stop helper.

Vite already has custom document-only GET/HEAD HTML fallback and Fetch Metadata
checks; remove only the sibling section-prefix exclusion. Extract a pure request
policy helper if needed for units without starting Vite/reading env. Go current
Apps(root) chooses four dist dirs and insufficiently guards bare /api, missing
extensionless assets or arbitrary .html. Replace with one dist root, strict API/
health/asset/source/file/traversal exclusions and document intent, preserving
static bytes/cache/security headers/HEAD. Its current traversal fixture expecting
entry HTML must be corrected to the accepted boundary. Pure httptest/temp-file
units suffice; no DB/listener/build is authorized.

Affected infrastructure units: internal/pkg/webui; tools/test_stop_apps.py;
pure Vite request policy cases in the unified project. No .env/private config,
network, images, deployment or runtime activation in this task.
