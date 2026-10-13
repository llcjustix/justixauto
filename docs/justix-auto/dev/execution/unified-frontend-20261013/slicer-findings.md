# Slicer checkpoint

Task: unified-frontend-20261013. Hub: /root. Base:
743bd6c4accadcd50afb59534bb66dabe2f3baf8. Actor:
/root/unified_frontend_slicer. Read-only source discovery; no Git or tests run.

Accepted frontend/infra mappings plus targeted composition reads establish:

- Split shared session isolation from router/section composition. Existing HTTP
  calls can update CSRF and fire unauthenticated callbacks after a context/user
  transition; fencing only QueryClient results is insufficient. SessionGate also
  needs stale refresh suppression and synchronous removal of previous content.
- Keep SessionGate/useSession existing callers working. The router packet consumes
  the lifecycle packet's explicit handoff, then exposes mountPortal/PortalSection
  for the single-entry relocation packet. No new auth transport or backend API.
- Seller/provider settings and seller dashboard are permission-free nav entries
  within their active company kind. Preserve this existing access behavior;
  unverified sellers still reach setup and verification. Platform nav entries all
  carry explicit grants. Inactive memberships are context choices only.
- Seller URLs already use /dashboard, so / can be the common last-section landing.
  Finance pages navigate to /applications and /programs; Admin overview card paths
  and /audit need prefix adaptation. API paths must remain untouched.
- Source relocation is mechanical except composition/navigation and these exact
  in-page links; preserve every source/test/CSS file through an explicit manifest.
  Finance CSS includes globally named interaction/status selectors, so load order
  and any shared selectors need targeted examination during composition, not a
  blind concatenation or broad redesign.
- Kit Vitest currently includes only .test.ts with node default; new component
  units can remain .test.ts with per-file jsdom pragma and React.createElement,
  avoiding an unrelated test-config change. Existing seller suites already use
  React Testing Library through locked dependencies.
- Four sequential implementation packets are intended: session lifecycle,
  section router/shell, actual workspace/source consolidation, runtime consumers.
  Additional dormant QA/final-SHA check specifications must never trigger an
  automatic QA loop under the current unit-only user decision.
