# Remaining dispatch after completed P19 UI

Working handoff, application hub /root. The current sole source lease remains
in checkpoint.md. This is not a completion or release-readiness assertion.
Latest final frontend reports: result-19a-completion.md; result-19b.md plus
result-19b-history.md; result-19c.md (nineteen workspace interactions passed).

## Execution sequence

P20 → P21a1 → P21a2 → P21a3 → P21b1 → P21b2 → P21b3 → P21b4 → P22 →
P23v → P23a → original P23 → final canonical records and scoped local Git handoff.
One source writer, main checkout, no worktrees. Latest unit-only policy applies.
Existing original packets plus accepted scope additions are authoritative.
Never stop with R3/R5 unfinished merely because earlier requirements are done.

Ready preparations at creation of this record:

- /root/custody_port_implementation: P20 complete, result-20.md, writer released.
- /root/consignment_authority: P21a1 complete, result-21a1.md, writer released.
- /root/consignment_handover: P21a2 complete, result-21a2.md, writer released.
- /root/purchase_receipt_provenance: P21a3 complete, result-21a3.md, released.
- /root/consignment_sale_authority: P21b1 complete, result-21b1.md, released.
- /root/consignment_stock_sale: P21b2 complete, result-21b2.md, released.
- /root/consignment_retail_lifecycle: P21b3 complete, result-21b3.md, released.
- /root/consignment_retail_candidates: P21b4 complete, result-21b4.md, released.
- /root/consignment_sale_selector: P23a DONE and released, result-23a.md;
  real manual/offer selection, preflight/context/error behavior and units complete.
- /root/inventory_custody_projection: P23v DONE and released, result-23v.md;
  nullable persisted DTO facts and four pure mapper cases complete.
- /root/consignment_returns: P22 SOURCE COMPLETE and released, result-22.md;
  five final routes include durable order-scoped history after a decision.
- /root/consignment_workspace: sole source writer original P23, explicit
  gpt-6-sol/high24k total; final P22/P23v/P23a handoffs ready, eight original paths.

## Exact additional contracts to carry into dispatch

P20: consignment-holder-lifecycle.md; p20-owner-custody-eligibility.md;
p20-holder-fixture-scope.md. Matching original order hold is consumed only on
successful handover, with history. Return rejects every competing hold and
uses current locked dealer placement, which may differ from initial handover.
Ordinary Reserve/eligible supply requires owner also current custodian; generic
owner-or-placement reads and same-company Move stay unchanged. Report an exact
movement/fact reference usable by P21/P22. Current assigned units are custody
family, exact holder namespace, pure eligible query; projection family only if
its implementation changes. No live database test.

P21a1: p21a1-risk-dispatch.md, p21a1-conversion-scope.md and
p21-consignment-authority-seam.md supplement original packet/split approval.
The two added paths are service/reservation.go and its existing conversion fake
in service/reservation_allocation_test.go. Agreement exists atomically at
consignment conversion even with zero VIN; order remains accepted-terms source.
Generic per-unit authority can represent prepared/handed, retail association,
pending return, sold/returned, actor/time and settlement links. Actual source
availability of migration 044 must be recorded at grant; no DB-state assertion.
Affected unit selector is TestConsignmentAgreement OR TestReservationConversion.

P21a2: final a1/P20 ports only, all three existing physical paths. Register newly
allocated selected VINs through a1's bound helper before/with handover. Purchase
debt uses already-credit-inclusive invoice Paid; optional mutually accepted
partial handover only, no invented minimum. Consignment quantity receipt fails
before minting buyer-owned stock; real allocated VIN custody preserves owner.
Future-only order conversion stays valid. App WorkflowPort21 seam unit required.

P21a3: receipt provenance and per-unit ordinal consume held future claims once.
Mixed allocated/quantity units conserve capacity. Never recycle a materialized
quantity-receipt claim as a retired VIN claim. Later Identify changes physical
identity only. Accepted allocation seam and three-way split remain in force.

P21b1–b3: p21b-split-approval.md and exact packet files. Same live per-VIN row
locks retail reservation and P22 pending return through the outer transaction.
Supplier cannot bypass live dealer consignment through ordinary Owned checks.
Actual customer delivery alone clears owner/custodian and records sold/deal/
settlement link; eligible cancel frees only that retail hold and retains debt.
Preserve ordinary owner retail for both types and old-obligation servicing.

P21b4/P23a: p23-consigned-selection-seam-proposal.md and its approval contain
exact embedded bounded definitions and paths (12k each at approval). Backend
returns redacted current-dealer candidate IDs under existing retail.read, not
the commerce finance DTO; exact final API is required before frontend work.
Frontend keeps normal owned eligible query, hydrates actual data.vehicle under
existing inventory.read, merges only approved IDs, handles all pages/failures/
context changes, and refreshes consigned selection before real sale submission.
No commerce.read grant or warehouse-only authority. These complete an existing
R3 journey missing from original P23 paths, not a new product feature.

P22: pending request is not movement/refund/debt cancellation. Supplier selects
its warehouse; holding dealer confirms. Lock the shared per-unit authority,
then actual custody/holds/capacity; preserve request/decision/movement reference
once. Migration 045 availability is a source planning check, never DB execution.
Final result-22.md supplies all five HTTP routes and exact DTO/revision/action
rules. p22-history-discovery-contract.md requires reopened-order discovery of
terminal requests without cached pending pointers or current dealer custody.

P23: actual final DTOs only, owner/custodian distinct, agreed mode/conditions
visible, real physical handover/return controls and authoritative query refresh.
Its early frontend preparation must not substitute for final backend handoff.
p23-ui-risk-dispatch.md selects fresh gpt-6-sol/high for P23a (20k) and original
P23 (24k), retaining their exact paths and unit families. These are not leases.
P23v first supplies nullable persisted ownerCompanyId/custodianCompanyId in the
existing inventory DTO; exact two-path scope and pure units are accepted in
p23-vehicle-projection-scope.md. No read authorization or stock policy change.

## Final delivery constraints

R1/R2/R4/R6–R13 already have source handoffs in delivery-map.md. R3/R5 remain
incomplete until the chain above is actually implemented. Final records must
distinguish affected unit evidence from unexecuted SQL/runtime/integration work.
No automatic QA/review, lint/typecheck/build/browser/React Doctor or deployments.
Only version_control writes Git; disable hooks per commit with
`git -c core.hooksPath=/dev/null commit ...`, explicit paths, no force/main action.
Current local milestone is ab088d97054fbabac65caae1f3f8bfe7f71fbe74; no push or
deployment has been performed. Preserve dirty predecessor/user work.
