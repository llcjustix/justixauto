# Unified frontend checkpoint

2026-10-13. Canonical task ../../unified-frontend-20261013.md; ADR-15.
Branch feature/unified-frontend-20261013, implementation base743bd6c4accadcd50afb59534bb66dabe2f3baf8,
dev ancestry4fa683ba4708bb71031534260ec5cb8742ce9977. Version-control preparation
completed cleanly; prior branch/dev/main preserved. Git lease released.

User approved one app and last permitted section + internal switcher. Shared
session returns active-company grants only; preserve company selection boundary.
FE and DevOps mappings complete and released. Infra authority accepted ADR-15
in infra-approval.md. P1 is accepted in p1-approval.md; /root/unified_session is
sole source writer, fresh explicit gpt-6-sol/high16k. Exactly four Kit lifecycle/
HTTP/unit paths; final result-01.md required before P2. Slicer has completed and
released its artifacts; final bundle is accepted in plan-approval.md. P2
/root/unified_portal_router completed read-only preparation and holds no lease.
It awaits P1's accepted result and explicit source grant; p2-risk-routing.md
records its fresh model/context routing. No agent reads P1's changing source.
Next: P1 completion → P2 section router → P3 portal source/workspace → P4 serving/
launcher, then canonical completion and scoped local Git. No automatic
review/QA/ReactDoctor/build/runtime or remote actions. All prior source features
must survive the relocation. The new request is not implemented yet.
