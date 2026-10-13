# Accepted implementation plan

2026-10-13, application authority /root. Accept the four sequential source
packets and U1–U6 coverage in the slicer's final bundle:
`c0d5cc1d3cdb40df69344ac212bc6d32b8492e808e37fe560eeaed3b676af2db`.
Plan SHA-256: `15dbee1352477f3581d7120201fcf4385280d23fe8c8c5f8d5799c2527a4ebbe`.
Exact component hashes are retained in result-slicer.md. This records owning-hub
acceptance, not an independent review or executable QA result. The latest user
unit-only rule supersedes the historical automatic review/QA sequence. Dormant
specifications in acceptance-map.md are evidence gaps, never dispatch gates.

P1 was accepted earlier in p1-approval.md. Its final packet hash
`057d60d43cfca37f39ed03ccb958e20489e38d5c70adfc8ffc0762337e08ffc9`
incorporates the already authorized direct-config Realization unit command;
source ownership and lifecycle contract remain unchanged. P2's fresh explicit
gpt-6-sol/high context ceiling is 24k including prep, as in p2-risk-routing.md.
Its read-only preparation is complete and has released the checkout; source
execution waits for P1's complete result and explicit lease transfer.

Execution clarifications within existing ownership:

- P3 must give the portal Vitest project an explicit root, following the existing
  Realization configuration. Run only the three affected portal files directly
  from `web`: `npm exec -- vitest run --config apps/portal/vitest.config.ts
  --configLoader runner src/routes.test.tsx src/document-policy.test.ts
  src/features/realization/navigation.test.tsx`. This replaces the generic
  workspace command to avoid silently omitting app tests. All intended suites
  must appear; no installs or pass-with-no-tests fallback.
- P3 and P4 preserve `/readyz` as well as `/healthz` and API boundaries. This
  follows the accepted infrastructure mapping; neither health URL becomes SPA
  HTML. Keep the pure policy cases aligned without running a server.
- P4's Go unit command uses the existing writable cache:
  `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh
  test -vet=off ./internal/pkg/webui -run 'TestMount|TestPortal' -count=1`.
- P3 removes the legacy mountApp only after P2's final export handoff; any
  necessary explicit barrel export removal must be separately recorded by root
  if the final barrel does not use export-star.

Both authorities accepted ADR-15 (infra-approval.md). No new product decision,
permission grant, dependency, runtime action, remote push or deployment is
authorized. One source writer; root alone updates canonical documentation.
