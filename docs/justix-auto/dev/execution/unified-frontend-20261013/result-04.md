# P4 portal runtime result

Status: **DONE**. Actor: `/root/unified_portal_runtime`. Branch/base:
`feature/unified-frontend-20261013` / `743bd6c4accadcd50afb59534bb66dabe2f3baf8`.
`bash tools/check-git.sh` passed before edits. P3's accepted result and pure
document policy were consumed. No Git operation, private `.env` read, process
discovery/signalling, runtime/server action, build, installation, network,
lint, typecheck, vet, race, deployment, review or QA was run.

## Completed consumers

- `webui.Mount` now accepts one concrete portal directory. Missing `index.html`
  remains API-only. Static files retain their bytes/MIME and hashed-asset cache;
  entry responses are non-stale with the existing security headers and empty
  HEAD body. Entry fallback requires GET/HEAD, positive `text/html` intent and
  compatible Fetch Metadata. Raw/decoded traversal, malformed paths, API,
  health (`/healthz`, `/readyz`), source/module, missing assets and file-shaped
  requests do not receive entry HTML.
- API composition passes `cfg.WebDir` directly. Local/configuration values are
  `web/apps/portal/dist`; image, Helm and tracked dev deployment values are
  `/app/web`; local origins are only localhost/127.0.0.1 port 5191.
- Make runs one portal Vite process. The stop helper recognizes only the exact
  portal cwd, port 5191, `api,web` supervisor and selected npm argv while
  preserving UID/cwd/start-time and listener safeguards. Tests use FakeSystem.

Root documentation already contains the required manual private-local
`WEB_DIR=web/apps/portal/dist` adjustment; P4 has no differing instruction.

## Focused unit evidence

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/pkg/webui -run 'TestMount|TestPortal' -count=1
ok   justixauto/internal/pkg/webui  0.633s

python3 -m unittest discover -s tools -p test_stop_apps.py
Ran 10 tests in 0.007s
OK
```

No runtime, deployment or release-readiness claim follows from these unit tests.

## SHA-256

| Path | SHA-256 |
| --- | --- |
| `internal/pkg/webui/webui.go` | `4946a589ad0b6a6a5ce2bf7272c75a01feb5360d60e885dbbb27537469862585` |
| `internal/pkg/webui/webui_test.go` | `16212f33eb6789a8ab39896523def2acd9db6148de0cfc408bcd6463e13311e3` |
| `cmd/api/main.go` | `085580ac60a491b82c8edaea611d3260efe50c0b5da68f448825ad298d8294cc` |
| `internal/config/config.go` | `60c08fdefaeffe587a01bbfb9ffbfec335767f18c53b1fa07022914b53d2483e` |
| `Makefile` | `8c4ab6f712a1d07d6e11a0810c438e0337108a0aaf5d869003cf572c9596b974` |
| `tools/stop-apps.py` | `2385aea3eb04c123db34a5a33404b66b1d7469b41f96bae3dd9b84c788d33ba0` |
| `tools/test_stop_apps.py` | `44bd584575efb0af68d3b9f177e9dedfadddb8c6fccff34d120bffea48d1d8de` |
| `deploy/Dockerfile` | `580b19b4f5466834d9a3a546c1c09754853cfb2d8bab850c1773024f6fa49015` |
| `.github/workflows/deploy-dev.yml` | `8fd605344b901332852066dfd88c8b9ede7bafcaa52f8101beec2eb28574bde1` |
| `deploy/values.yaml` | `b4f3da5352408f5d628e3cf3aa7e595703afa0cdb53b10d2fe25f88bfb21ddfa` |
| `.env.example` | `9c2f5153134826ce0c1993d037bef7f8a13e0e4d32c581f6adbc198102236c33` |
| `.air.toml` | `3d20ad6d49be7abc33f184b67762290e968d47ec05dddf44b62beab20d14d7b1` |

The P4 source/test writer lease is released. Exclusions: canonical docs,
private `.env`, portal source, deployment topology, CI policy and all live
operations.
