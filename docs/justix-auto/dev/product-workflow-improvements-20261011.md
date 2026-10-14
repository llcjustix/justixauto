# Product workflow improvements — 2026-10-11

Status: SOURCE COMPLETE for all thirteen requested improvements; affected unit
evidence and limits are in [completion](execution/product-workflow-improvements-20261011/completion.md).
No migration execution, runtime activation, deployment or broader backlog
completion is claimed. User direction: the 13-point
request, reservation-flow approval «да давай», then «go start». Coordinator owns
this canonical record. Detailed decisions: [interview](product-workflow-interview-20261011.md).
This is one bounded feature branch with sequential packets; do not regenerate
the historical backlog or use worktrees.

## Delivery scope

R1. Company-level fixed `userType` dealer/distributor, in backend and frontend.
Keep user membership/job roles separate. No dealer/distributor mode switch.
Platform admin chooses the type; selecting an initially missing legacy value is
allowed, changing a set value is not. Bank/MFO/insurance kinds remain separate.
User explicitly confirmed retail sales are available to both dealer and
distributor. New B2B reservations use dealer buyer and distributor supplier.

R2. Company verification separate from operational access. New companies may
sign in and configure company, branches, warehouses and staff; commercial actions
(create/add vehicle, buy/order/reserve/offer/sell/ship and similar new operations)
require verification on the server and explanatory UI. Admin onboarding is name,
first admin login/password; company business type is optional during creation and
must be assigned by Admin before verification/new commercial actions. Other
requisites are optional there. Company settings hold legal form (LLC/sole proprietor/other), registration/
tax and address/contact/bank details. Platform admin verifies or returns a reason;
company can complete details and submit again. No invented licensing/compliance
guarantee. User confirmed legacy companies are classified and verified manually
in Admin; new commercial actions blocked until then, while history and servicing
previous obligations remain available.

R3. Order reservations for stock VINs and future model/specification/color
quantities. Dealer requests, distributor confirms available quantity and terms;
changed terms require buyer acceptance. Confirmed price is fixed; changes require
mutual acceptance. Free by default, optional agreed advance, per-order expiry,
supplier-approved extension. Hold quantity/VIN exactly once and allocate VIN later.
Pending/accepted money prevents automatic expiry/cancellation. Continue as purchase
or consignment with separately agreed payment/handover terms. User confirmed full
advance refund on cancellation, recorded/confirmed by accountant, and partly paid
purchase handover only when explicitly agreed in the order.
User also explicitly confirmed automatic release after the required-advance
deadline when no payment was submitted; pending/accepted evidence prevents it.

R4. User-accessible same-company warehouse relocation using existing backend Move;
explicit source/target and capacity/permission errors. Do not infer inter-company
ownership changes from warehouse moves.

R5. Distributor requests return of vehicles held by dealer on consignment; dealer
confirms. Preserve ownership separately from custody; no return of sold vehicles.
Keep a recorded request/decision, destination warehouse and actual stock movement;
block conflicting reservations instead of silently releasing them.

R6. CRM passport fields and private passport-copy uploads. Use the existing file
binding and tenant-access mechanisms; do not place copies in public URLs or logs.
Generic optional fields: document series/number, issuer, issue/expiry dates,
birth date, personal ID, registered address. No jurisdiction-specific mandated
format or new retention policy is inferred.

R7. Rename user-facing «Банк / МФО» to «Исламская линия» throughout the four apps
and shared kit. This is terminology, not changed rates or legal-product claims.

R8. Own-installment down-payment primary input is percent, with synchronized money
amount alongside. Exact integer/rational arithmetic, no floating-point money.
Preserve existing positive-less-than-price rule, currency and saved schedules.
Use explicit smallest-unit rounding in the preview and corresponding unit cases.

R9. Implement accepted contract proposal: data-filled template plus external
document upload, paper signature with scan; preserve each signed version and use
separate amendments. Coordinator assumption stated to user: cover both retail and
dealer/distributor contracts. No external e-sign integration or invented binding
legal clauses. Reuse current authenticated document storage and snapshot parties,
vehicle/order and price/terms. Templates must be editable/configurable rather than
claimed legally approved language.

R10. Company accountant preset/profile and work surface: accountant issues invoices;
sale flow displays issued invoices and waiting state. Separate invoice-management
permission from sale-management. Preserve admin authority under existing role rules.
R13. Accountant accepts/rejects installment payments; only accepted evidence reduces
debt. Reuse existing grouped allocation and payment review semantics.

R11. Insurance request/status/actions in the sale dialog; keep the insurer's own
application. R12 (amended by user 2026-10-14): clearly identified optional demo
scoring for own installments. Contracts can be signed without insurance or
scoring; name the sale tab «Страхование». No real provider integration or
inference that scoring itself creates insurance coverage. Preserve addressed
insurer decision authority and the separate vehicle-handover requirements.

## Execution and acceptance

- Applicable rules: root AGENTS.md, internal/AGENTS.md, web/AGENTS.md and
  docs/justix-auto/dev/AGENTS.md; latest unit-only verification overrides old loops.
- Start from verified project root and dev. Version-control role owns branch/commits.
  Current pre-feature HEAD: `4fa683ba4708bb71031534260ec5cb8742ce9977`.
- Only one source writer; task_slicer may write bounded execution packets. No
  automatic review/QA loops, lint, typecheck, builds, browser or integration checks.
- Run only affected unit tests once per relevant change; meaningful failures may be
  repaired and affected tests rerun. No migrations or runtime/deployment actions.
- Add new numbered migrations; never rewrite applied migrations. Preserve current
  contracts/foreign ownership boundaries; modules communicate through service ports.
- All 13 points must be tracked, including unresolved portions; never mark the whole
  feature complete because one packet is done. Final handoff states source delivered,
  affected units, unexecuted runtime activation and any still-open policy slices.
- Execution artifacts: `execution/product-workflow-improvements-20261011/`.
- Branch: `feature/company-commerce-workflows-20261011`, created from local dev
  at the recorded base; existing coordinator intake preserved. Current packet,
  historical writer assignments and unit evidence are in
  `execution/product-workflow-improvements-20261011/checkpoint.md`. No review/QA loop.
- Pre-amendment canonical snapshots/hashes:
  `execution/product-workflow-improvements-20261011/entry/implementation-start/manifest.md`.
- Final source handoffs and requirement coverage:
  `execution/product-workflow-improvements-20261011/delivery-map.md`.
  All source leases are released. Local source Git provenance is recorded in
  `execution/product-workflow-improvements-20261011/source-git-result.md`;
  final documentation is committed separately. No remote push or promotion.
- Pre-completion canonical snapshots/hashes:
  `execution/product-workflow-improvements-20261011/entry/source-completion/manifest.md`.
