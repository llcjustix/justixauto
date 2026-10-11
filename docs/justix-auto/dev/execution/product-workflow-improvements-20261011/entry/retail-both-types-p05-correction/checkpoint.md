# Feature execution checkpoint

Updated: 2026-10-11. Canonical feature: ../../product-workflow-improvements-20261011.md.
Branch: `feature/company-commerce-workflows-20261011`.
Base: `4fa683ba4708bb71031534260ec5cb8742ce9977` (local dev).

## Active assignments

- `/root/installment_terms_labels`: sole source writer, combined P06/P07 as
  p06-p07-dispatch-approval.md; exact union, context ceiling 24k.
- `/root/accountant_frontend`: P05 DONE, writer released; result-05.md.
- `/root/accountant_backend`: P04 DONE, writer released; result-04.md.
- `/root/passport_backend`: P08 read-only preparation only; must await explicit
  writer grant after P06/P07. Context ceiling 20k.
- `/root/commercial_frontend_guards`: P03c DONE, writer released;
  result-03c.md records frontend guard unit evidence.
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
- P03c complete: 2 files / 17 tests passed after the final relevant stale-dialog
  guard. New commercial actions gated; historical servicing kept.
- P04 complete: identity service passed once, retail/commerce passed after
  approved fixture changes and duplicate-constant repair. Separate invoice
  authority plus existing/future seller accountant preset implemented. R2 setup
  guards read only: no independent active-only branch/staff/warehouse blocker
  found; this source observation is not runtime verification.
- P05 complete: Kit permissions 1 file / 2 tests, Realization accountant/payment
  review 2 files / 4 tests passed. Billing supports the accountant permission set,
  invoice waiting and one parent-group review action; result-05.md.
- P06/P07 in progress; P08 preparation; later pending. All requirements in plan.
