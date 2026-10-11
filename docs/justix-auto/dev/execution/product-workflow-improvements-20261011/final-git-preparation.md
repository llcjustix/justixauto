# Final Git preparation

Status: READY for a later scoped Git handoff, but this moving snapshot is not
a commit candidate.

## Verified state

- Repository root: `/Users/bakhromachilov/startups/justixauto` (apphub).
- Branch: `feature/company-commerce-workflows-20261011`.
- Expected/local milestone HEAD: `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`.
- Milestone base: `4fa683ba4708bb71031534260ec5cb8742ce9977`.
- HEAD parent is the expected base; no remote push or deployment is recorded.
- Milestone evidence says the tree was clean after commit. The current status is
  42 tracked modified paths and 258 untracked paths (including 183 under the
  execution records, 55 under `internal/`, 12 migrations, and 8 web paths).

## Lease and scope disposition

P22 (`/root/consignment_returns`) is the current sole source writer. Its lease
must be released and its result accepted before any final path selection. The
current modified and untracked source paths are therefore preserved as a moving
writer-owned set; this read-only pass does not label any of them user-owned or
unknown and does not stage them. `checkpoint.md` is also actively changing and
must not be included by inference.

The P22 handoff identifies two additional owned paths for the return adapter:

- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`

P22's original packet retains its eleven paths, and migration `000045` is
available as source scope only. The accepted P22 contract requires the shared
per-VIN authority lock, supplier-destination request/holding-dealer
confirmation, immutable movement/fact linkage, and the focused unit selector;
it does not authorize SQL execution, runtime work, or a broader source grant.

## Bounded next scope

After P22 releases the source lease, freeze the source and reconcile the final
P22 result against its exact owned paths. Then dispatch the remaining packets in
this order, each with a fresh clean-state check and a separate source lease:

1. P23v: exactly
   `internal/modules/inventory/handler/dto.go` and
   `internal/modules/inventory/handler/vehicle_custody_test.go`.
2. P23a: exactly
   `web/apps/realization/src/sale-offer-source.ts`,
   `sale-offer-source.test.ts`, `sale-form.tsx`, `sale-form.test.tsx`, and
   `sale-offer-form.test.tsx`.
3. Original P23: use its accepted packet's exact paths only after P22/P23v and
   the final backend DTO/API handoffs are frozen; do not absorb P23a files or
   unrelated existing dirty paths.

For each later commit, version control must verify root, branch, expected base
and dirty/staged paths after every writer releases; stage explicit owned paths
only; inspect the staged names; and use
`git -c core.hooksPath=/dev/null commit` with a conventional message carrying
the packet identity. No branch switch, push, hook execution, generated Swagger,
tests, diff check, runtime, SQL, or deployment belongs to this preparation.

## Blocker

Final Git preparation cannot identify a safe commit path set while P22 remains
the sole source writer and the tree contains its moving source plus execution
artifacts. Root must wait for P22 release/result acceptance, freeze source, and
grant the final explicit path list before staging or committing.
