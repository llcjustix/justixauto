# Feature execution checkpoint

Updated: 2026-10-11. Canonical feature: ../../product-workflow-improvements-20261011.md.
Branch: `feature/company-commerce-workflows-20261011`.
Base: `4fa683ba4708bb71031534260ec5cb8742ce9977` (local dev).
Local milestone: `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`; no remote push.

## Active assignments

- `/root/installment_terms_labels`: P06/P07 DONE, writer released;
  result-06-07.md has exact conversion/terminology and unit evidence.
- `/root/accountant_frontend`: P05 plus additive access correction DONE,
  writer released; result-05.md and result-05-correction.md retained.
- `/root/accountant_backend`: P04 DONE, writer released; result-04.md.
- `/root/passport_backend`: P08 DONE, writer released; result-08.md corrected.
- `/root/passport_warehouse_frontend`: P09/P10 DONE, writer released;
  result-09-10.md records both features and affected unit evidence.
- `/root/retail_contract_backend`: P11 PARTIAL, writer released at context
  boundary. result-11.md is not acceptance; known lifecycle/compatibility work
  remains.
- `/root/retail_contract_lifecycle`: P11c1 DONE, writer released; result-11c1.md
  records guarded same-row signing, production wiring and rollback unit evidence.
- `/root/retail_contract_drafts`: P11c2 DONE, writer released; result-11c2.md.
- `/root/retail_contract_compatibility`: P11c3 DONE; result-11c3.md pins final
  compatibility/finalization/DTO handoff and unit evidence. Writer released.
- `/root/b2b_contract_backend`: original P12 prep only, no writer lease.
- `/root/b2b_contract_domain`: P12a1 PARTIAL, writer released; result-12a1.md.
- `/root/b2b_contract_completion`: P12a1 continuation DONE, writer released;
  result-12a1c.md records complete service/Preview and rollback unit evidence.
- `/root/b2b_contract_api`: P12a2 DONE, final result-12a2.md; writer released.
- `/root/demo_scoring_backend`: P14 service gate complete, result-14.md; released.
- `/root/demo_scoring_freshness`: P14 freshness projection DONE, released;
  result-14-freshness.md records server-derived currentness and focused units.
- `/root/commercial_frontend_guards`: P03c DONE, writer released;
  result-03c.md records frontend guard unit evidence.
- `/root/commercial_backend_guards`: combined P03a/P03b DONE, writer released;
  result-03-backend.md records service and app-seam unit evidence.
- `/root/company_frontend`: P02 DONE, writer released. Omitted Realization
  add-company unit passed separately (1 file / 1 test); result-02.md.
- `/root/company_backend`: P01 DONE, writer released; result-01.md.
- `/root/contracts_sale_frontend`: initial P13/P15 implementation released;
  result-13-15.md is partial despite its DONE label. Handoff consumption found
  request/response file-field mismatch and unfinished original UI behavior.
- `/root/contract_sale_ui_completion`: P13/P15 continuation DONE, writer released;
  result-13-15-completion.md records corrected adapters, draft/template editing,
  immutable snapshots and affected interaction units.
- `/root/future_supply_backend`: historical P16 read-only preparation, no writer lease.
- `/root/future_supply_implementation`: initial P16 released; result-16.md.
  Supply API delivered; claim foundation needs the named continuation below.
- `/root/supply_foundation_completion`: sole source writer, bounded 12k context;
  explicit gpt-6-sol/high escalation under agent-workflow risk rule. Assignment
  p16-foundation-completion.md fixes concrete P17/P21 consumer schema gaps,
  storage overflow validation and committed source identity.
- `/root/reservation_request_backend`: P17a read-only READY; awaits actual P16
  foundation result and separate writer grant. Initial request terms clarification
  is p17a-request-terms-clarification.md; no default price/expiry/advance.
- `/root/consignment_custody_backend`: P20 read-only READY, no writer lease;
  current warehouse move uses owner-or-placement visibility, not owner-only.
- `/root/workflow_slicer`: reservation-to-order allocation handoff accepted in
  reservation-order-allocation-approval.md. Docs-only P18b bounded split accepted
  in p18b-split-approval.md; no source work or independent QA/review.
- `/root/workflow_git`: milestone committed, lease released. Result in
  milestone-01-git-result.md; unexpected generated OpenAPI scope adopted in
  milestone-01-openapi-note.md. Future commits use per-command disabled hooks.
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
- User explicitly confirmed retail selling for BOTH dealer and distributor;
  retail-both-types-decision.md resolves the plan's pending question. B2B roles
  are dealer buyer and distributor supplier for the new reservation flow.
- App + infrastructure authorities accepted expiry runner source design; see
  expiry-runtime-approval.md. DevOps delegate finished; no runtime action.
- Future order conversion preserves durable lot claims with zero, some or all
  assigned VINs; ordinary post-conversion allocation/cancellation and quantity
  receipt must conserve those claims. Accepted technical ownership additions:
  reservation-order-allocation-approval.md; no new product restriction.

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
- P05 initial evidence: Kit permissions 1 file / 2 tests, Realization accountant/payment
  review 2 files / 4 tests passed. Billing supports the accountant permission set,
  invoice waiting and one parent-group review action; result-05.md. Follow-up
  found commerce-only navigation and retail-only default-tab gaps, now fixed
  by result-05-correction.md: Kit 2 files / 2 tests, accountant 1 file / 3 tests.
  Kit pure tests renamed to .test.ts to resolve actual runner discovery; both
  filenames explicitly appeared in successful output. No config change.
- P06/P07 complete: percent/calculator/terms units passed; sale-form passed
  12 tests after verified-company fixture and fixed-down event-guard repair.
  Admin label/company-form 2 files / 7 tests passed. No green unrelated suites
  repeated; result-06-07.md.
- P08 complete: documents/service, retail/service and pure TestWorkflowPort08
  passed. Approved owned-file lookup and existing payment fake adaptations
  recorded; documents green selection not repeated after retail fixture repair.
- P09/P10 complete: two Realization files / 6 tests passed. Passport details
  stay inside customer modal; move uses actual permission/source/destination
  route with refresh and visible server errors; result-09-10.md.
- P11 partial implementation: private identity/app projection units and retail
  units passed for existing cases, but legacy contract-records still mutates
  fields. Targeted continuation-context read also found reusable source drafts,
  missing full installment snapshot, missing persisted draft edit/external-file
  support and service-level contract-read permission. These are unfinished P11
  acceptance, not new product scope or a review/QA gate. Slicer prepares bounded
  completion accepted as P11c1 → P11c2 → P11c3; all three now done. Production adapter
  wiring and structured DTO/finalization requirements also belong to completion.
  Previous reports do not establish lifecycle completion.
- P11c1 passed both authorized contract-service and pure production-wiring unit
  selectors. P11c2/P11c3 supply the remaining draft/snapshot/compatibility/
  finalization/DTO acceptance; no database or runtime claim.
- P11c2 affected TestContract selector passed after draft editing, exact snapshots,
  private source files, plain-text replacement and stale-economics protection.
  P11c3 completed legacy compatibility, finalization and structured DTO handoff.
- P11c3 affected contract/finalization and handler selectors passed. A newly
  added direct Sign-to-plan test first failed on unsupported fake policy and
  missing issued-invoice status. Root authorized the diagnosed fixture repair;
  only that test reran and passed with original amount/date/row assertions.
  P11 backend complete; result-11c3 is the final API handoff, not partial result-11.
- P12a1 initially delivered typed storage, raw terms/party/VIN snapshots and basic
  units, with Preview, edit file grants, rollback and explicit acceptance pending.
  Its fresh continuation completed those gaps. No extra
  priorSignedRevision API is invented: accepted draft/order revisions and parent
  identity remain authoritative. Earlier green selectors do not finish P12a1.
- P12a1 continuation complete: authorized no-write Preview, amendment draft/order
  CAS, draft file author grants, deep file-grant rollback and explicit original
  behavior cases passed the affected service selector once. Unchanged Store
  wiring was not repeated. The late TemplateDetail addition and release timing
  are recorded in p12-source-release-note.md; final service selector passed.
- P12a2 complete: actual-order private identity adapter, typed composition and
  structured authenticated HTTP/DTOs passed the three assigned app/handler/
  module unit selectors. Service units were not repeated. P12 backend complete;
  result-12a2.md is the final UI handoff. Runtime/DB behavior remains unverified.
- P14 persisted demo run/shared first-sign gate complete; affected retail/service
  units passed after a cash-test fixture adjustment. Fingerprint covers customer
  ID, saved vehicle, exact price/currency, scheme and generated installment draft.
  Latest persisted run remains visible when stale. Freshness continuation added
  mandatory server-derived current boolean to both response paths; null means
  no run. Targeted scoring and pure DTO units passed, no whole-package rerun.
- Local milestone committed with accepted generated API reference; no push,
  migration or deployment. Combined P13/P15 continuation after completed
  P11/P12/P14 preserves logical dependencies. Initial two typeof-export tests
  are smoke evidence only, not original interaction acceptance. Completion
  repairs known API mismatch, PATCH/refresh/permission gaps and scoring errors;
  affected behavioral units accompany the actual source change.
- P13/P15 continuation complete: two insurance interactions passed in initial
  combined run; three contract interactions passed after ambiguous test selector
  repair and added PATCH preservation coverage. Only the affected contract file
  reran. File-field mapping, draft PATCH, editable templates, snapshot-only signed
  price, actual permission gates and visible scoring errors are implemented.
  No backend units or additional checks ran. P16 is now dispatched.
- P16 initial service units executed after fixture compiler repairs. One new
  outsider assertion expected the wrong existing error class; diagnosed repair
  passed its targeted selector. Other passing service cases were not repeated.
  Claim handoff consumption found pre-order materialization forbidden by DDL
  and an insufficient per-receipt uniqueness key. The fresh bounded continuation
  also aligns quantity range with SQL integer and protects committed source
  identity. This is implementation completion, with no automatic QA or DB checks.
  All requirements
  remain tracked in the accepted plan; no whole-feature completion claimed.
