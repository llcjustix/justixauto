# P21a1 — Durable consignment authority

Proposed under original P21a, R3; application hub `/root`. Read
p21a-split-proposal.md for accepted inputs, planning base/branch, common rules,
hashes and stop conditions. Root acceptance and final P18b2/P20 handoffs required.
Fresh context ceiling 12k. Exclusive source-writer, no DB/browser/runtime lock.
Coverage C1/C2 and persistence portion of C6/C8. No automatic QA/review.

## Exact owned paths

- internal/modules/commerce/model/consignment.go
- internal/modules/commerce/service/consignment.go
- internal/modules/commerce/service/consignment_test.go
- internal/modules/commerce/service/ports.go
- internal/modules/commerce/repository/consignment.go
- internal/modules/commerce/repository/store.go
- internal/modules/commerce/handler/consignment.go
- internal/modules/commerce/handler/handler.go
- internal/modules/commerce/handler/dto.go
- internal/modules/commerce/module.go
- migrations/000044_commerce_consignments.up.sql
- migrations/000044_commerce_consignments.down.sql

These are original P21a paths. Read final order/accepted-terms/payment source
only as targeted dependencies; do not edit them. Migration pair requires hub
confirmation of available number/unapplied status. Internal AGENTS.md applies:
service owns rules, handler only parsing/DTO, ambient repositories, optimistic
revision, module-root exports, integer minor units and existing tenant policy.

## Behavior and interface

Consume P18b2's actual mutually accepted continuation snapshot, order/source
link and version. Persist the order-linked consignment agreement and concrete
vehicle records, preserving accepted settlement/handover terms and distinct
supplier owner/dealer custodian. Reuse the accepted order snapshot where it is
already durable; avoid a second independently mutable payment agreement. No
extra user agreement ceremony, endpoint name or default terms are prescribed.
If a missing acceptance rule is found, stop rather than deriving it from mocks.

Provide transaction-bound service/repository operations for a2 to record physical
handover and to reject already sold/returned/handed-over vehicle state. Expose
active consignment information through commerce's root API for P21b, scoped by
actual company/order/vehicle/agreement; lookup must distinguish intended future
consignment from a vehicle actually handed into dealer custody. Preparing an
agreement is not physical handover or retail availability. Preserve durable
order debt/settlement linkage through all states. Retail completion/return
mutations remain P21b/P22; do not implement their product policies here.

Wire service construction, repository store and handler/DTO only as required by
the actual accepted consignment operation/read view. Use existing permissions,
httpx envelopes/ETag and decimal-string revisions. Report actual endpoints/DTOs
and module-root types in the result; downstream packets must not guess them.
No public inventory custody primitive and no cross-module table reads.

## Acceptance and affected units

Use service/consignment_test.go for meaningful `TestConsignmentAgreement*`
cases: accepted snapshot is retained; changed/stale/unaccepted/cross-party input
cannot replace it; owner/custodian differ; order/vehicle source is scoped;
duplicate agreement/vehicle transition fails without duplicate state; active
lookup excludes unhanded, sold and returned records; persisted obligation
survives status change. Inject agreement/vehicle write and revision/event
failures using deep-copy transaction fakes. Exercise public service operations,
not only model predicates. Actual handover/stock failures belong to a2.

Run once after relevant changes from repo root:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestConsignmentAgreement'`

Do not run SQL or package-root DB tests. Report exact selector/count/results;
no-tests is a gap. Handoff exact persisted fields/keys, transaction and lookup
contracts, tenant/revision semantics, unapplied migration status, changed paths
and source hashes to a2/P21b. No app adapter or fulfilment edits in this packet.
Stop on missing final accepted terms/custody authorization semantics, unowned
source need or context ceiling; no product policy invention.
