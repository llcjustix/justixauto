# P21b bounded split proposal

Status: proposed; hub acceptance and predecessor handoffs required. Application hub `/root`, feature
`product-workflow-improvements-20261011`, R3/R5 with R2 compatibility.
Planning base `4fa683ba4708bb71031534260ec5cb8742ce9977`, branch
`feature/company-commerce-workflows-20261011`. This is a docs-only decomposition
within original P21b ownership, not a canonical task or policy change.

## Mapped findings

- Original P21b spans commerce authority, inventory trusted reservation,
  retail listing/reservation/delivery/cancellation and application adapters.
- Final P21a1/a2/a3 results are prerequisites; no consignment implementation
  exists in the inspected source yet. Do not prescribe final APIs from proposals.
- Accepted holder lifecycle consumes the supply hold at actual custody handover.
  Retail authorization and P22 pending returns must lock the same live agreement
  authority within the bound transaction. Mere placement is not permission.
- Supplier ownership ends only at actual customer delivery; cancellation frees
  the retail hold and leaves active consignment/settlement obligations intact.
- Both company types retain normal owner retail; R2 historical servicing stays
  unchanged. The older plan's distributor-retail question is superseded by the
  hub's explicit assignment.
- Current stock Reserve is owner-only; its LockOwned repository operation locks
  by vehicle ID and exposes the actual owner/custodian record, so the existing
  inventory ports can support a separate trusted consignment path without a
  repository edit. The current root StockService alias exposes service methods.
- Retail Listing.Create checks eligibility before its transaction; publication
  checks inside Bind. Deal.Create checks Owned and then reserves inside Bind.
  Deliver and Cancel already enclose retail/stock mutations in retail InTx.
- Stock.Deliver clears owner/custodian only at actual customer delivery but
  currently ignores DeletePlacement errors. Consigned delivery must propagate
  participating failures. Preserve normal Deliver unless a shared correction
  is actually necessary; no unrelated stock rewrite.
- App retailStock currently uses concrete inventory service wiring. Small
  app-local interfaces enable pure tests of actual adapters with exported types.
- No source, tests, Git, runtime, canonical record or contract was changed.

## Sequential bounded packets

| Packet | Coherent behavior | Ceiling | Dependencies | Original owned paths |
|---|---|---|---|---|
| P21b1 | Live per-unit consignment sale authority and completion linkage | 12k | All final P21a1/a2/a3 results; P20 lock/holder contract | Five commerce paths |
| P21b2 | Trusted consigned physical hold and actual customer delivery | 12k | b1; final P20/P21a custody facts | stock.go and inventory consignment_sale_test.go |
| P21b3 | Retail lifecycle and actual application adapters | 14k | b1 + b2; all final P21a results | Six retail paths plus app.go/workflow_ports_test.go |

These are three execution packets inside original P21b, not additional backlog
items. The union is exactly the original 15 paths; no production/test/model/
migration path is added. b1/b2 expose trusted capabilities; b3 enables retail
use only once both exist. Original P21b remains incomplete until all acceptance
is covered. A two-packet split would combine physical stock lock/failure rules
with retail transaction orchestration and application tests in one context.

Read each fresh packet and proposal, scoped rules and only final predecessor
result/API slices needed for it. Do not read the whole board, all source modules
or unrelated feature history. Persist compact mapped findings before further
large reads. Source-writer is exclusive across all packets and hubs; no fixture,
browser or deployment resources. Root supplies the actual dispatch SHA and
dirty predecessor hashes, accepted result paths and lease. Planning base is
provenance, not a claim that current files equal that commit. Workers check the
project root with tools/check-git.sh before editing, and perform no Git writes.
Worker reports go to hub-assigned paths; this slicer owns only these four files.

## Required final handoff and concrete ownership gaps

P21a1 has not executed. Hub explicitly confirmed that the following requirements
belong in its already-owned model/migration 044 and generic repository contract;
they do not authorize this slicer or P21b to edit those paths. Root can clarify
a1 before dispatch within existing total scope. No names or exact API/schema
representation are prescribed here.

1. Durable **per vehicle/unit** live authority, linked to actual accepted
   agreement/order, supplier owner, dealer custodian, physical-handover fact,
   current retail holder/deal and any pending-return request. Required lifecycle
   distinctions are unhanded, held in dealer custody, sale-reserved,
   return-pending, sold and returned; final implementation may represent these
   with state plus links. A return for one VIN cannot block unrelated VINs.
2. Durable actual retail deal, sold time/actor and existing settlement obligation
   linkage, with revision/transition behavior and immutable history. Pending
   return resolution and eligible sale cancellation must restore active unsold
   eligibility without destroying recorded requests, holds or financial links.
3. Exact row identity and generic transaction-bound repository operations that
   let P21b and P22 lock/recheck/update that same per-unit authority. P22 owns
   consignment_return.go and shared Store/ports, **not** the earlier consignment
   model/repository. P21a1's existing generic repository contract therefore must
   support pending-return linkage and resolution without future unowned edits.
4. P21a2/P20 actual custody input/results, owner/custodian/warehouse/holder facts,
   active supply-hold consumption at handover, lock order and ambient transaction
   semantics. P21a3 final handoff confirms accepted order mode/quantity accounting
   and does not undo active authority predicates. All three accepted results
   remain dispatch prerequisites, even if a1's schema alone looks sufficient.
5. P18b2 authoritative accepted continuation/order provenance and existing
   settlement terms are retained. Its result is evidence of implemented source,
   not proof P21a handover or P21b sale is complete. Do not invent payment APIs,
   another invoice/credit, fees, schedules or an acceptance ceremony.

Absent any required durable representation or generic operation, b1 is blocked
at dispatch and returns exact missing paths/fields to root. No querying a future
unapplied P22 table, hardcoded no-pending-return answer, commerce-table access
from inventory, separate authority mutex or new unauthorized migration. The
ownership gap is concrete: P21b cannot repair missing consignment schema and
P22 cannot repair the original consignment repository under their current lists.
It is presently an unfulfilled technical prerequisite, not an undecided product
rule. A final API requiring an inventory module-root alias or incompatible
retail constructor/fake edits is a separate exact-path stop, not implicit scope.

## Complete acceptance coverage

B labels are local traceability under R3/R5 (and preserved R1/R2), not new tasks.

| Label | Original acceptance / approved invariant | Packet evidence |
|---|---|---|
| B1 | Verified dealer can list/publish/reserve a concrete active consigned VIN only under the accepted agreement and actual handover | b1 authority; b3 listing/Create |
| B2 | Wrong company/owner/custodian/agreement/VIN, mere placement, future-only or unhanded state cannot grant sale | b1 identity/state; b2 physical facts; b3 no Owned bypass |
| B3 | Checked trusted reservation retains supplier owner, binds exact retail holder, blocks every competing physical hold | b2; b3 bound orchestration |
| B4 | Pending return and retail reservation serialize on the same live per-unit row; sold/returned/stale denied; unrelated VIN remains independent | b1 protocol/state; b3 consumer; P22 final shared check |
| B5 | Supplier ownership ends only actual customer delivery; sold transition is once, exact retail sale/actor/time and existing settlement linked | b1 durable completion; b2 physical exit; b3 atomic call chain |
| B6 | Eligible cancellation frees only retail hold and restores/retains active unsold consignment; B2B debt, accepted money and settlement unchanged | b1 authority cancellation; b2 release; b3 existing blockers |
| B7 | Stock/commerce/deal/listing/lead/event failures roll back all participating mutations; no partial completion or swallowed placement failure | b1/b2 local failure units; b3 enclosing transaction fakes |
| B8 | App adapters forward actual dealer/owner/agreement/VIN/holder/actor/time/settlement and bound context; error remains error | b3 TestWorkflowPort21_Retail family |
| B9 | Both dealer and distributor retain ordinary owner flow; history and R2 old-obligation servicing preserved; exact snapshot/payment/checklist/version rules unchanged | b2 owner compatibility; b3 retail compatibility |
| B10 | No automatic debt forgiveness, commission or new default; existing settlement terms and partner-finance/insurance exclusions preserved | b1 financial invariants; b3 checklist/legacy matrix |

Accepted-lock protocol is a contract, not proof of SQL behavior. b1 provides
authority, b2 inventory locks, b3 an outer transaction that retains both. P22
must subsequently use the identical per-unit row before persisting pending
return and check physical hold conflicts under the compatible lock order. It
must not create a competing request path that sidesteps b1. P22 stays an existing
separate packet with its original physical-return and money-preservation scope.

## Unit-only execution and dormant independent definitions

Latest unit-only policy applies. Workers run only their affected pure unit
families once per relevant change, with vet disabled. No slicer tests; no lint,
typecheck, build, DB/migration execution, browser/E2E/integration, automatic
review/QA chain, runtime or deployment. No-tests is a gap, not evidence. App
uses distinct TestWorkflowPort21_Retail names to preserve P21a units without
ceremonial reruns. Actual service behavior is covered with transaction fakes;
deep-copy rollback tests do not establish live transaction isolation.

The following small independent QA packet definitions are **dormant**, executable
only if separately requested by the user. They add no current verification gate,
task or QA dispatch. Each consumes an exact SHA, relevant packet result and final
source hashes, reads only named slices, owns no source, returns evidence to a
hub-assigned artifact, and uses an immutable-checkout read lease. Stop on SHA
drift, missing authority, absent meaningful cases or need for non-unit execution.

| QA definition | Budget | Dependency / acceptance | Narrow evidence |
|---|---|---|---|
| Q21b1 | 4k | b1; B1/B2/B4/B5/B6/B10 | ConsignmentSaleAuthority family: identity/lifecycle, independent VINs, completion/cancel linkage and preserved debt |
| Q21b2 | 4k | b2; B3/B5/B7/B9 | ConsignmentStock family: trusted facts, holds, physical exit/failure atomicity, owner compatibility |
| Q21b3 | 6k | b3; B1–B10 retail portions | RetailConsignment and WorkflowPort21_Retail families: bound consumers, cancellation/delivery rollback, R2 and company types |

## Final-SHA shared-interface checks

These define acceptance/evidence destinations and future human/CI integration
work; **no integration execution or repeat passing units is authorized now**.
Hub associates affected unit evidence with final file hashes, marks evidence
invalidated by later relevant edits, and runs only then-affected units once.
Do not treat three old GREEN reports as cumulative final verification.

- **F21b authority/stock/app, B1–B9:** at final candidate SHA, map P21a's actual
  handed-over per-unit authority and consumed supply hold → b1 locked authority
  → b2 retail hold → b3 exact deal. Verify final adapter signatures and state
  provenance match the accepted result files. Unit evidence covers these seams;
  actual cross-module SQL rollback requires an explicitly authorized later
  integration packet (6k, immutable SHA plus exclusive PostgreSQL fixture lease).
- **F21b/P22 serialization, B3/B4/B6/B7:** after P22 at the same final SHA, future
  small integration packet (6k, exclusive PostgreSQL fixture) exercises two
  competing transactions, reservation-first and pending-return-first; exactly
  one conflicting action succeeds, no foreign hold is released, another VIN
  under the same agreement remains eligible, and transaction failure leaves no
  request/hold gap. Also check P22 completion rejects sold or retail-held units.
- **F21b settlement/exit, B5/B6/B7/B10:** future small integration packet (6k,
  same immutable SHA/fixture lease) injects completion failure after inventory
  mutation and checks inventory/retail/commerce rollback; actual delivery clears
  supplier ownership and links sale once, cancellation retains consignment and
  debt. No real money/provider operations. Current fake-unit evidence cannot
  replace these later live checks.

All future independent/integration definitions stop without explicit user
authorization and exact final SHA/environment/resource assignment. No release
readiness is asserted. No infrastructure or shared operational guarantee changes
are proposed; any such need goes to both owning hubs.

## Risks and stop conditions

No new product rule is resolved here. Actual P21a schema/API handoffs are missing,
so implementation is not ready to dispatch. P21b3 is bounded by targeted symbol
reads and stateful fakes; if constructor compatibility or cross-module test
setup exceeds 14k, checkpoint and split again within the same original scope.
No unowned fake edits. Stop before a product ambiguity, missing source/lock
contract, unexplained drift or any unowned mutation. RESLICE_REQUIRED is reserved
for a changed canonical scope; this three-way decomposition does not change it.
OD-01/08/09/10 exclusions from plan remain. Neither fixtures nor historical HTTP
proposal rows override latest approved semantics/current handoff APIs.

## Planning input SHA-256

Working-file snapshots, collected without Git; refresh relevant paths after
predecessors. Execution basenames below are relative to this proposal's directory.
Consignment model/service/repository do not exist yet in the inspected snapshot;
their final hashes are a dispatch prerequisite, not invented provenance.

```text
a222419ecca4935c906454884eb62d6cacace6a5b80395c90a1cee870f8b8524  AGENTS.md
b22213a76ca11a4ce4aeb04323260d1d577f28ef1230d48c15367d074e67650b  internal/AGENTS.md
b2d5153012dfec17e20a5e55a65af858d693cae27adb66ef6a5c440599149b27  docs/justix-auto/dev/AGENTS.md
4f264fb5217a9613b00df455b0661cb9b955e90793a29b17c682cbcd0010830b  docs/justix-auto/dev/agent-workflow.md
5df5a66f4ad3526b36fc5f35c011eff7311eccb0e0b6b9a0e185636122542356  docs/justix-auto/dev/product-workflow-improvements-20261011.md
5bd0439c6aca0b5a824c340c4c15692734c70296b0125d82c63a739a21ba65ba  docs/justix-auto/contracts/http-domain.md
f8a0d756be25ba54b71b2b22294676f0e18adc3747f2bbc841d825e87ab88c69  plan.md
6f6fa395022ced0eacf8c69cad77900e471628b7d017fcbed601d645e2a5c1d6  packet-21b.md
4570f0656759d54b7b9f6c625f777cea0299e361dd986345a432301979dc2e03  packet-22.md
57e6cb169133171c8af9b762a2d3c38b5ff90bdf615a3f09676a84800bb7c108  p21a-split-proposal.md
38e672e91dcf8c537fb8fc2f0f63427e0fa270a34dc9281e950ddf4fdbc410ef  p21a-split-approval.md
37740bbb44e6551155f7823f8532c6090402375ef6819f98a7bcc5c38777398c  packet-21a1.md
a15e2045edba08bb4ac315aa8eaffacb617e4a5f0926f1738c8844e6e8ad697f  consignment-holder-lifecycle.md
5a186de1f9e90ef07693462974fd4aa1307c3b98a45eca2564fa12a6f72c287d  result-18b2.md
56cd0cb65849d57d1427792767774dc63082e7b21e626ac2b00106606ecda93b  internal/modules/commerce/module.go
e8a1ecf5db66163bdb5fd989bbd6afff40c8abdde011d13f265089e0a2957901  internal/modules/retail/service/deal.go
c1412eb0212165f558737f73ce38e109576d9f8d7f7638026d6f6afd212f2d17  internal/modules/retail/service/listing.go
13ebdad9c79655186759e845b9c4510c0639f6cbdbefacf439bd83ceb620b6a1  internal/modules/retail/service/ports.go
80bf08193a800f0643767aacacc25949b7e76ade9255730bbbea7925aa2a78e9  internal/modules/retail/service/deps.go
f479efee17ddc48e5642cb89a47278cf75718a7aefcbf2e9595171a811f450d1  internal/modules/retail/module.go
87916c574e59385b981cacf6f3d4cc5b7f10bc6a2370c721e3387957fd2f4e31  internal/modules/inventory/service/stock.go
07f978d4355ae20d8217bcb6c1375f939ff9e5f811059664fa649f9cfd452888  internal/modules/inventory/service/ports.go
5d91771af595f3f2add6218a964c757f275ebe7a89653315fce504a2ba589c19  internal/modules/inventory/repository/vehicle.go
ac44463866c8e86f85074b699c1ac06dc7742a79475f13c8d6d5d11f599cc677  internal/modules/inventory/module.go
8df45119152eab727fe3092fa22fd07bd633bf6073b682e3f448e803b1b243ff  internal/app/app.go
de8758b9d7ce709447576118ebf0d0eec96c86f5522870416007ab0c243df50c  internal/app/workflow_ports_test.go
```

## Proposed packet SHA-256

```text
1687a3afcfe767561bb800ade7fd22f8993d67cec02963b60eb39c2692426182  packet-21b1.md
f9ebacf47ef99a403e58cf8e6264f0ca7b6a657eeec485b9263c04fbacfc6049  packet-21b2.md
fb8910af636dc7b994c2629a5eb045df85fb374642b4b8566d5b86f8736e32a0  packet-21b3.md
```

Proposal digest is reported separately to avoid self-reference. Approval of
these exact artifacts belongs to the owning hub; no automatic review/QA loop
or independent review claim is introduced.
