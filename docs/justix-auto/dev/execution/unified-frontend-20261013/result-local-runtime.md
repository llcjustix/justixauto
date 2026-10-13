# Local runtime started

2026-10-13, application hub receipt of /root/local_portal_start observations.
User requested local startup after main publication. Current source branch:
fix/portal-local-start-20261013, base feeed7292e5b0f607f6bc29b80dbe47ff4b55040,
with the one-line Vite correction in result-local-start-fix.md. No remote fix
publication or production action is part of this startup request.

Persistent local services are running:

- Portal: http://127.0.0.1:5191/, make supervisor PID26849, Vite PID26877,
  one listener. Final document GET returned HTTP200 text/html, 526 bytes with
  portal-root, /src/main.tsx and the Vite development client.
- API: http://127.0.0.1:8090/, Air PID26222, dev-api PID26309.
  GET /readyz returned HTTP204 after startup.
- Existing justixauto-postgres-1 container and volume, loopback port55433;
  migration version45, dirty=false. No reset or replacement of user data.

Before applying additive migrations35–45, DevOps preserved a verified private
pg_dump archive (182026 bytes) at
/private/tmp/justixauto-local-start-20261013-7460dzvi/before-migration-34.dump,
SHA-256 d985e50395d120db6b583ff438d61cae6efdd574380de29c0f1b6a7af92f711f.
The private directory also contains runtime-receipt.md, api-air.log and portal.log.
Source configuration secrets are not copied into this tracked record.

Private .env was used for authorized startup but not changed. The obsolete
WEB_DIR value was overridden in the API process environment to portal/dist;
the development UI is served by Vite. The original owned tool-session portal
was replaced with the same detached command and pinned Node24.21.0, without
duplicate listeners. Docker Desktop was started for the local database.

Verification was limited to the readiness needed for the requested startup:
API readiness and one successful persistent portal document response, following
the initial reproduced HTTP500 and its async HTML-loader correction. No browser,
E2E/integration suite, lint, typecheck, production build or additional unit suite
was run. Go compilation and forward migrations were startup steps, not release
verification. The user will check application behavior manually.

The processes are intentionally left running. `make stop` stops this checkout's
API and portal and keeps PostgreSQL data. PIDs are an observation at this time,
not a standing authorization to signal later reused process IDs.
