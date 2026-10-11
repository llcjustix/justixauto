# P21b2 — Trusted consigned stock hold and customer delivery

Proposed under original P21b, R3/R5; application hub `/root`. Planning base
`4fa683ba4708bb71031534260ec5cb8742ce9977`, branch
`feature/company-commerce-workflows-20261011`. Read p21b-split-proposal.md,
root AGENTS.md, dev/agent-workflow.md, internal/AGENTS.md and plan.md Execution
rules. Fresh worker context ceiling **12k**. Dependencies: accepted final b1,
all final P21a results and P20's actual custody/lock contract. Exclusive
source-writer, no browser/DB/runtime lease. Coverage B3/B4/B5/B7/B9.

## Exact owned paths

- `internal/modules/inventory/service/stock.go`
- `internal/modules/inventory/service/consignment_sale_test.go`

Read only targeted final inventory ports/model, module-root StockService alias,
owned vehicle/placement/reservation repository methods and P20 implementation.
No inventory module.go, ports.go, repository, model or migration edits. No app
or retail enablement in this packet. Existing root StockService alias exposes
service methods; do not import inventory/service from application code merely
to use a new input type. Reuse exported root types/scalar inputs if sufficient;
otherwise report the concrete missing alias path rather than widening scope.

## One behavior

Supply trusted inventory operations that consume b1's checked consignment
authority, reserve for the actual retail-deal holder and complete that exact
holder's customer delivery. Final signature follows predecessor handoffs; this
packet does not prescribe an API before those exist. Inventory cannot decide
commerce agreement/return policy or read commerce tables. The trusted caller
holds b1's authority lock in the same ambient transaction and carries the
checked agreement and exact owner/custodian/VIN/retail holder facts.

Under inventory's established vehicle/placement/hold locks, recheck supplier
ownership, dealer custody and actual dealer placement against those facts;
reject changed source, absent placement, wrong holder/company, empty authority
and active competing hold. The supply hold was consumed by P20 at physical
handover; never silently release a surviving order or another retail hold.
Hold creation retains supplier ownership/custody facts and durable history.
Same-holder retries keep existing reservation semantics, never create a second
hold. Preserve normal owned-stock Reserve for both company types and its current
authorization boundary; an arbitrary custodian must not enter that path.

Actual delivery rechecks the checked identities and exact retail holder,
then atomically clears company ownership/custody, removes placement, updates
warehouse state, finalizes the hold and appends the customer-delivery fact.
Ownership ends here only. Propagate participating write/lookup failures;
the inspected generic Deliver ignores DeletePlacement errors, so the consigned
delivery path must not claim success after a placement failure. Keep repair
limited to this required path/shared helper if needed, without unrelated cleanup.
Preserve ordinary Deliver behavior unless an actual shared correction is required;
do not broaden this finding into an unrelated generic stock rewrite.
Inventory's transaction joins the outer retail/commerce transaction. If any
later completion/deal/event step fails, inventory changes must roll back too.

Existing holder-scoped Release remains the cancellation primitive: it releases
only that retail hold, never ownership/custody or the consignment agreement.
Commerce settlement/payment records are outside this packet and are untouched.

## Meaningful affected units

`TestConsignmentStock*` cases in the owned file exercise actual Stock operations
with deep-copy transaction fakes: valid checked dealer hold; owner and ordinary
dealer/distributor compatibility; false custody/wrong owner/warehouse/holder/
agreement failure; competing hold rejection without release; same-holder retry;
release leaves ownership/custody; actual delivery clears only once; wrong holder
and repeat rejected. Inject hold creation, placement delete, ownership clear,
warehouse update, hold finalization and fact failures and demonstrate unchanged
state. Carry a bound transaction marker through every participating call.

Run once after relevant change:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/inventory/service -run '^TestConsignmentStock'`

Do not run repository/DB tests or claim these fakes prove SQL locks. During
implementation first run tools/check-git.sh; no worker Git writes. Handoff
actual module-root reachable signatures, checked facts required, lock order,
error behavior, release semantics, hashes and nonempty unit evidence to b3.

## Stop conditions

Missing predecessor API/lock contract, inability to express checked facts using
owned paths, schema/repository needs, source drift, undecided product rule or
12k ceiling stops this packet. Do not broaden generic custody permissions or
add an inventory HTTP endpoint. No automatic QA/review or runtime work.
