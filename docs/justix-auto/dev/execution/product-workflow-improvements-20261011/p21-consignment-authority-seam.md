# Consignment authority persistence required by P21/P22

Application clarification of original P21a1/B2B consignment ownership, before
its source implementation. This adds no product decision or owned path.
Consume alongside consignment-holder-lifecycle.md and the final P18/P20 APIs.

P21a1 owns the durable per-vehicle authority used by subsequent retail and
return workflows. Model, unapplied migration 044 and generic repository
operations must represent the actual accepted agreement/order, supplier owner,
dealer custodian and physical handover, together with the current retail deal
association, pending-return association and final sold/returned transition.
Unhanded future units must be distinguishable from actual dealer-held units.
Representations and exported names are left to the implementation.

Record retail delivery deal/reference/time/actor and linkage to the existing
order settlement obligation durably. Cancellation of an eligible retail hold
or resolution of a return request may restore unsold availability; immutable
history and accepted financial terms remain. No commission, new invoice,
default payment date or debt cancellation is introduced.

Each VIN has its own live row identity and transaction-bound lock/update API.
P21b retail authorization and P22 pending-return creation must lock and recheck
that same row before physical stock locks, retaining the lock through the outer
transaction. One VIN's return must not block other units of the agreement.
No separate process lock or hardcoded absence of a pending return is sufficient.

P22 owns new return model/service/repository files and shared Store/ports, not
the prior consignment model/repository. P21a1 therefore supplies generic typed
per-unit update operations and durable links sufficient for P22's later service
to record and resolve pending returns. Do not read a nonexistent future return
table or implement return policy prematurely in a1. Report exact final fields,
row keys, revisions and bound transaction semantics for downstream packets.

P21a2 activates custody only after actual successful P20 handover. Preparation
alone is not retail eligibility. P21b completes sale only at actual customer
delivery; P22 completes return only at dealer confirmation and physical movement.
Existing accepted order debt and payment evidence survive either custody change.
