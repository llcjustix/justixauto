status: DONE

exact_change: `configureServer` entry fallback now uses an async loader and awaits `readFile` before passing the HTML string to `server.transformIndexHtml`.
source_sha256: ccd55c738d47144c743e7ae68d7867466a0de61e0e73bcb9336cb3e19ac85fe4 (`web/apps/portal/vite.config.ts`)
unit_tests: not run — this one-line configuration correction has no affected unit suite; the packet explicitly excludes unrelated policy/navigation units and a mirroring test.
lease: RELEASED (`web/apps/portal/vite.config.ts`)

