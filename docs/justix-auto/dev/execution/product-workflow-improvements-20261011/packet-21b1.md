# P21b1 — Live consignment sale authority

Proposed under original P21b, R3/R5; R2 compatibility. Application hub `/root`.
Planning base `4fa683ba4708bb71031534260ec5cb8742ce9977`, branch
`feature/company-commerce-workflows-20261011`. Read p21b-split-proposal.md,
root AGENTS.md, dev/agent-workflow.md, internal/AGENTS.md and plan.md Execution
rules. Fresh worker context ceiling **12k**. Exclusive source-writer lease;
no browser/DB/runtime lease. Approval and all final P21a1/a2/a3 handoffs are
required; consume their actual APIs and dispatch hashes, not proposed names.
Coverage B1/B2/B4/B5/B6/B9 in the proposal.

## Exact owned paths

- `internal/modules/commerce/service/consignment.go`
- `internal/modules/commerce/service/consignment_sale_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/module.go`
- `internal/modules/commerce/repository/consignment.go`

No model, migration, store, handler, payment or inventory source ownership.
Read final consignment model/schema, accepted order terms, authority predicates
and relevant repository transaction methods in narrow slices only.

## One behavior

Provide commerce's live, tenant-scoped authority for an already handed-over
consigned VIN throughout reservation and actual retail completion. Expose narrow
module-root operations for checked eligibility and durable sale completion;
do not grant generic warehouse custody a right to sell. Bind actual dealer,
supplier owner, agreement/order, VIN, current physical-handover state and accepted
settlement reference. Require concrete VIN, active consignment and no pending
return; reject unrelated companies, future-only/unhanded, sold or returned units.

Reservation authorization must lock the **same durable per-vehicle live consignment row**
that P22 will lock before creating/checking a pending return, and retain that
lock in the outer bound transaction through inventory hold creation. A lookup
that commits its own transaction before stock reservation is insufficient.
Document precise row identity, lock acquisition order, version and bound-context
requirements in the result. Reconcile with final P21a/P20 locks; do not invent
a second independent mutex/table or use a process lock. P22 owns request/decision
operations; this packet supplies the shared authority contract, not its workflow.
The row represents one actual unit: a pending return of one VIN must not block
unrelated units in the same agreement. Consume P21a1's durable state/linkage for
the actual retail holder; record/clear any sale-reserved association in the same
transaction as reserve/cancel, without inventing a retail schema field. Eligible
cancellation restores unsold active authority and never erases its history.

Completion records the actual retail deal and delivery reference/time exactly
once against that same consignment vehicle and existing accepted settlement
obligation. It is called only inside the actual retail-delivery transaction;
listing, reservation, contract signing and accepted retail payment do not mark
sold or clear supplier ownership. Different-deal/repeated/stale completion fails
without duplicate linkage; preserve existing action replay semantics rather
than inventing request-key idempotency. Preserve all accepted debt, credit,
payment evidence, settlement terms and immutable history. No new commission,
default settlement date, payment write or debt forgiveness.

Expose only final implemented root aliases/services required by b3. Services
use ports and apperr; repositories use ambient database.Conn and owned commerce
tables. Match P21a Store.Bind/InTx behavior; never import another module or GORM
into service. No public client-supplied authority boolean.

## Meaningful affected units

In the owned sale test file, use `TestConsignmentSaleAuthority*` service cases
with stateful transaction fakes: exact accepted identity succeeds; wrong dealer,
agreement/VIN, unhanded/future, pending-return, sold/returned and stale version
fail; read failure is not treated as no agreement. Show both logical orderings
of sale authorization versus pending return against one shared authority state.
These unit schedules establish the service protocol, not live SQL serialization.
Completion links one deal/settlement once, preserves financial state and rolls
back on authority CAS, link/event failure or enclosing transaction failure.
Eligible cancellation clears only the matching retail association, restores
active unsold eligibility and retains debt/history; stale or wrong-deal cancel
must not clear another association. One VIN's pending return leaves other VINs
in the agreement unaffected.
Do not test only new helpers or copy the production predicate into the fake.

Run the affected family once after relevant change:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestConsignmentSaleAuthority'`

No tests are requested at slicing time. During implementation run
`bash tools/check-git.sh` first; no worker Git writes. Report final API/type and
error names, exact lock/transaction protocol, schema fields reused, test command
and nonempty results, changed hashes and downstream P22 expectations to hub.

## Stop conditions

Stop if final P21a schema cannot persist the required sale/settlement/holder link or
represent the shared pending-return exclusion. Original P21b has no authority
to add those columns. Do not query a future unapplied P22 table, return an
always-false pending check or synthesize policy from fixtures. Hub must resolve
the exact ownership gap before dispatch/restart. Also stop on missing final
handoff, unexplained source drift, unowned path, product ambiguity or 12k limit;
checkpoint and request a bounded split. No automatic review/QA or runtime work.
