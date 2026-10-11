# Feature execution checkpoint

Updated: 2026-10-11. Canonical feature: ../../product-workflow-improvements-20261011.md.
Branch: `feature/company-commerce-workflows-20261011`.
Base: `4fa683ba4708bb71031534260ec5cb8742ce9977` (local dev).

## Active assignments

- `/root/company_frontend`: sole source writer, P02 accepted by coordinator.
  Company Admin/settings UI plus the bounded optional-userType onboarding
  correction; see minimal-onboarding-correction.md. Result: result-02.md.
- `/root/company_backend`: P01 DONE, writer released; result-01.md.
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
- Distributor retail selling versus dealer-only retail remains a narrow question;
  preserve current availability until an answer. B2B roles are dealer buyer and
  distributor supplier for the new reservation flow.
- App + infrastructure authorities accepted expiry runner source design; see
  expiry-runtime-approval.md. DevOps delegate finished; no runtime action.

## Verification and completion

- Run affected unit tests only once per relevant change; Go commands use
  `bash tools/go.sh test -vet=off ...` to suppress implicit vet.
- No review/QA loops, builds, lint, typecheck, browser/integration, migrations,
  runtime restart, deployment or production Git operation.
- P01 source complete: isolated writable GOCACHE recovered a cache-permission
  setup failure; affected identity/service and pkg/auth unit packages passed once,
  zero failures. Its mandatory userType-onboarding assumption is corrected in P02
  with a new affected identity/service unit run; prior evidence is not reused for
  that change. Initial worker also ran git diff --check; coordinator prohibited
  further extra verification commands. No review/QA loop occurred.
- P02 in progress; later packets pending. All 13 requirements tracked in plan.
