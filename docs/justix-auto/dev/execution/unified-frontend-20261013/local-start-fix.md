# Local startup correction

2026-10-13. User explicitly requested local startup after publishing main at
feeed7292e5b0f607f6bc29b80dbe47ff4b55040. This authorizes necessary local process
startup, compilation and non-destructive forward migration, not production
operations or new remote publication. DevOps /root/local_portal_start owns the
local runtime; no application source write is delegated to that actor.

Observed P3 startup defect: Vite starts on 127.0.0.1:5191, but document GET /
with Accept:text/html returns HTTP500, Application entry unavailable. The Vite
configureServer callback passes the Promise returned by readFile to
transformIndexHtml instead of the loaded HTML string. This is within the already
accepted P3 entry/config ownership and changes no product or access contract.

Correction branch fix/portal-local-start-20261013 has normal dev ancestry4fa683b
and the published dependency HEAD feeed7292e5b0f607f6bc29b80dbe47ff4b55040. Version
control prepared it without changing checkout bytes under the running portal.

Grant one worker the sole source lease for web/apps/portal/vite.config.ts only:
make the entry loader callback async and await readFile before transformIndexHtml.
Preserve all policy, routing, proxy, caching, HEAD and preview behavior. Run the
repository guard before editing; apply_patch only. No other source or Git edits.
No unit suite changes behavior here; do not rerun unrelated policy/navigation
units or add a test mirroring this one-line async correction. User-requested
startup readiness is one document HTTP recheck by the runtime owner, not a
browser/E2E/integration suite or a new automatic review/QA loop.

Return the exact source hash and released lease in result-local-start-fix.md.
Root will then have DevOps recheck the running portal and version control commit
only the correction and these execution records. No remote push in this packet.
