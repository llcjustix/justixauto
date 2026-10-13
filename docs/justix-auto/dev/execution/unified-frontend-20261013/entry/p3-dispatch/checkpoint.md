# Unified frontend checkpoint

2026-10-13. Canonical task ../../unified-frontend-20261013.md; ADR-15.
Branch feature/unified-frontend-20261013, implementation base743bd6c4accadcd50afb59534bb66dabe2f3baf8,
dev ancestry4fa683ba4708bb71031534260ec5cb8742ce9977. Version-control preparation
completed cleanly; prior branch/dev/main preserved. Git lease released.

User approved one app and last permitted section + internal switcher. Shared
session returns active-company grants only; preserve company selection boundary.
FE and DevOps mappings complete and released. Infra authority accepted ADR-15
in infra-approval.md. P1 is implemented, accepted and released; result-01.md
records four exact source hashes, 12 Kit and 5 Realization latest unit passes.
Slicer has completed and released; final bundle is accepted in plan-approval.md.
P2 /root/unified_portal_router is the sole source/test writer under p2-dispatch.md
and p2-risk-routing.md, after completing read-only preparation. P3
/root/unified_portal_workspace has a read-only prep grant for stable old feature/
config files only and holds no source lease; no one reads P2's changing files.
Next: P2 section router → P3 portal source/workspace → P4 serving/
launcher, then canonical completion and scoped local Git. No automatic
review/QA/ReactDoctor/build/runtime or remote actions. All prior source features
must survive the relocation. The new request is not implemented yet.
