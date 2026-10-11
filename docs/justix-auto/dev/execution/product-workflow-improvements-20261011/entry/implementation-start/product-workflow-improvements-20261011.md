# Product workflow improvements — 2026-10-11

Status: user-authorized implementation preparation. User direction: the 13-point
request, reservation-flow approval «да давай», then «go start». Coordinator owns
this canonical record. Detailed decisions: [interview](product-workflow-interview-20261011.md).
This is one bounded feature branch with sequential packets; do not regenerate
the historical backlog or use worktrees.

## Delivery scope

R1. Company-level fixed `userType` dealer/distributor, in backend and frontend.
Keep user membership/job roles separate. No dealer/distributor mode switch.
Platform admin chooses the type; selecting an initially missing legacy value is
allowed, changing a set value is not. Bank/MFO/insurance kinds remain separate.

R2. Company verification separate from operational access. New companies may
sign in and configure company, branches, warehouses and staff; commercial actions
(create/add vehicle, buy/order/reserve/offer/sell/ship and similar new operations)
require verification on the server and explanatory UI. Admin onboarding is name,
first admin login/password and company business type; other requisites are optional
there. Company settings hold legal form (LLC/sole proprietor/other), registration/
tax and address/contact/bank details. Platform admin verifies or returns a reason;
company can complete details and submit again. No invented licensing/compliance
guarantee. Legacy migration policy awaits the concise question sent this turn.

R3. Order reservations for stock VINs and future model/specification/color
quantities. Dealer requests, distributor confirms available quantity and terms;
changed terms require buyer acceptance. Confirmed price is fixed; changes require
mutual acceptance. Free by default, optional agreed advance, per-order expiry,
supplier-approved extension. Hold quantity/VIN exactly once and allocate VIN later.
Pending/accepted money prevents automatic expiry/cancellation. Continue as purchase
or consignment with separately agreed payment/handover terms. Reservation advance
refund and partially paid purchase handover remain gated pending the current answer.

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
application. R12. Clearly identified demo scoring before signing for own installments;
no real provider integration or inference that scoring itself creates insurance
coverage. Preserve addressed insurer decision authority.

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
