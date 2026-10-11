# Feature execution checkpoint

Updated: 2026-10-11. Canonical feature: ../../product-workflow-improvements-20261011.md.
Branch: `feature/company-commerce-workflows-20261011`.
Base: `4fa683ba4708bb71031534260ec5cb8742ce9977` (local dev).

## Active assignments

- `/root/company_backend`: sole source writer, P01 accepted by coordinator.
  Identity/backend type, verification, setup access, migration 000035; result-01.md.
- `/root/workflow_slicer`: execution plan/packet documents only; no source edits.
- `/root/workflow_git`: branch created, idle, no commit/push. No checkout switching
  or staging source until the implementation writer releases the checkout.
- `/root`: canonical documentation and product decisions only.

## Decisions and pending input

- Company legacy type/verification manual; new commercial operations blocked,
  history and servicing old obligations preserved (explicit user answer).
- Full advance refunds confirmed by accountant; partly paid purchase handover
  only by explicit order agreement (explicit user answer).
- User accepted proposed contract implementation with «go start»; both retail/B2B
  scope is the stated coordinator assumption.
- Required-advance deadline missed without any payment submission: user explicitly
  chose automatic release. Pending/accepted evidence always prevents expiry.

## Verification and completion

- Run affected unit tests only once per relevant change; Go commands use
  `bash tools/go.sh test -vet=off ...` to suppress implicit vet.
- No review/QA loops, builds, lint, typecheck, browser/integration, migrations,
  runtime restart, deployment or production Git operation.
- No source packet is complete yet. All 13 requirements remain tracked in plan.
