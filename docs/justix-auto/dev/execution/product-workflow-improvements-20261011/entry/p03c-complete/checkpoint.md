# Feature execution checkpoint

Updated: 2026-10-11. Canonical feature: ../../product-workflow-improvements-20261011.md.
Branch: `feature/company-commerce-workflows-20261011`.
Base: `4fa683ba4708bb71031534260ec5cb8742ce9977` (local dev).

## Active assignments

- `/root/commercial_frontend_guards`: sole source writer, P03c dispatched,
  exact packet ownership, context ceiling 18k, affected unit suites only.
- `/root/commercial_backend_guards`: combined P03a/P03b DONE, writer released;
  result-03-backend.md records service and app-seam unit evidence.
- `/root/company_frontend`: P02 DONE, writer released. Omitted Realization
  add-company unit passed separately (1 file / 1 test); result-02.md.
- `/root/company_backend`: P01 DONE, writer released; result-01.md.
- `/root/workflow_slicer`: complete, idle. 29 packets accepted by coordinator;
  digest/coverage in plan-approval.md. No independent QA/review claimed.
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
- P02 source complete including optional type-onboarding correction. Identity
  service units passed after correction. Kit + two Admin unit files passed (8
  tests); omitted Realization add-company suite passed separately without
  repeating green tests. Correct routing is in plan-approval-02.md.
- P03a/P03b complete: five affected service packages passed. Narrow app unit
  TestWorkflowPort03 passed after the approved P01-origin shared reasonBody DTO
  restoration in identity/handler/admin.go; service suites were not repeated.
- P03c in progress; P04 and later pending. All 13 requirements in plan.
