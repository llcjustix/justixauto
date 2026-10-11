# Git milestone preparation

Status: READ-ONLY preparation; no staging, commit, branch switch, test, or
push is authorized yet.

Observed checkout:

- Root: `/Users/bakhromachilov/startups/justixauto`
- Branch: `feature/company-commerce-workflows-20261011`
- HEAD/base: `4fa683ba4708bb71031534260ec5cb8742ce9977`
- Tracked modifications: 71 paths
- Untracked paths: 187 paths before this note; 188 including this note

P12a2 remains the sole source writer at this checkpoint. Its future candidate
source scope is limited to the packet-owned paths below, subject to the
writer's release and an exact final diff review:

- `internal/modules/commerce/handler/contract.go`
- `internal/modules/commerce/handler/handler.go`
- `internal/modules/commerce/handler/contract_test.go`
- `internal/modules/commerce/module.go`
- `internal/modules/commerce/contract_wiring_test.go`
- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`

The eventual feature milestone may include other completed packet-owned source
and migration paths, but those must be enumerated from their released packet
results before staging. Do not stage by directory or wildcard.

Keep these categories out of the feature source commit unless the coordinator
explicitly supplies exact paths and intent:

- coordinator-owned `docs/justix-auto/dev/dev-state.md` and
  `docs/justix-auto/dev/task-board.md`;
- intake/state documents under `docs/justix-auto/dev/` and
  `docs/justix-auto/state/`;
- execution manifests, checkpoints, packets, and result artifacts under this
  feature's `execution/` directory, including this preparation note.

No unrelated source path was proven eligible for staging by this read-only
inspection. The working tree must be rechecked after P12a2 release because
the shared checkout is still being edited.
