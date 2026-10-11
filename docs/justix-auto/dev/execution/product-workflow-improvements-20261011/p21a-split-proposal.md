# P21a bounded split proposal

Status: proposed; root acceptance required before dispatch. Application hub `/root`;
task `product-workflow-improvements-20261011`, R3, original P21a plus accepted
reservation/order allocation seam. Planning base
`4fa683ba4708bb71031534260ec5cb8742ce9977`; hub-reported local milestone
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74` on
`feature/company-commerce-workflows-20261011`. Dispatch SHA and predecessor
source hashes remain pending final P18b/P20 handoffs.

## Mapped findings

- Original P21a combines persistent agreement, allocated ship/legacy receipt,
  custody composition and purchase quantity claim provenance in a 20k context.
- Accepted seam preserves purchase receipt batches and conversion with zero,
  some or all VINs; actual consignment handover requires VINs.
- P18b2 will supply accepted mode/payment/handover terms and converted claims;
  P20 will supply the actual inventory custody API. Do not guess either API.
- Stable fulfilment symbol mapping identifies three physical paths:
  `shipQuantity` calls Deliver, `deliverOnShipment` calls Transfer, and
  `applyReceiptDecision` calls Transfer on acceptance. All need explicit mode
  coverage. Source will change in P18b and is only a planning snapshot.
- P18a1-owned reservation/payment/ports/store source has not been read.
- Proposed split: persisted consignment authority; mode-aware physical handover
  plus custody adapter; purchase quantity provenance. No source/runtime/Git work.

## Sequential bounded packets

| Packet | Coherent behavior | Context ceiling | Dependencies |
|---|---|---|---|
| P21a1 | Durable order-linked consignment agreement and active authority lookup | 12k | Final P18b2 terms/money/source handoff; final P20 custody authorization contract |
| P21a2 | Accepted-mode physical handover through purchase or custody port | 14k | P21a1 plus final P20 exported API |
| P21a3 | Purchase quantity receipt consumes future obligations once | 14k | P21a2 plus final P18b2 claim/progress handoff |

Three packets are justified by separate persistence/API, cross-module physical
handover, and durable claim-provenance behaviors. Two would recombine custody,
money, three fulfilment entry paths and quantity materialization in one context.
This is decomposition within original P21a plus the accepted seam; no canonical
task is added and no RESLICE_REQUIRED scope change is proposed. Original P21a
stays incomplete until all three finish. Intermediate source is not a complete
handover feature; do not report P21a2 alone as quantity-claim safe.

Each packet reads root AGENTS.md, dev/agent-workflow.md, internal/AGENTS.md,
plan.md Common execution rules, this proposal and its own packet only; then
targeted final predecessor result/contract/source slices. Use incremental symbol
maps and compact findings before expanding reads. Do not load full modules,
the board or unrelated P11–19 history. One source-writer lock, main checkout;
no browser/DB/runtime resources. Before implementation run tools/check-git.sh.
No Git writes by workers. Bounded results go to the hub, which assigns report
paths and records canonical status; this slicer owns only these four documents.

All original owned paths are distributed across the packets. Shared paths
serialize: service/consignment.go and its tests, service/fulfilment.go,
service/ports.go and module.go. Accepted seam additions are only
service/reservation_allocation.go, its existing test file and repository/supply.go.
The only new files are already listed in original P21a (consignment layers,
consignment unit fixture and migration 044). No additional production/test path
is proposed. Existing repository/fulfilment.go and inventory receipt service
remain read-only. A concrete missing column, interface or fixture ownership is
returned to the hub before editing; no speculative new migration or API.

## Final predecessor handoff required before dispatch

The planning SHA is provenance, not current source. Hub must supply each dispatch
SHA, dirty predecessor path hashes if applicable, exclusive lease, and accepted
result paths. Do not bind to names suggested in older proposals where final
implementation differs.

- P18b2: exact accepted continuation mode/terms digest, payment/settlement and
  partial-handover condition fields, source-linked order/line mapping, revision
  contract, authoritative accepted debt/credit read and locking contract from
  P18a, plus all ordinary-order compatibility behavior. Credit once belongs to
  P18a/P18b; P21a must not manufacture or credit another payment.
- P18b2/P17b: final claim states/provenance constraints, order materialization
  helper, lock order, cancellation/rejection retirement and reallocation rules;
  final fulfilment progress accounting and receipt identification contract.
  Quantity receipt claims must be distinguishable from retired VIN allocations
  eligible for reassociation. Current helper inspection shows it considers
  materialized claims with no active VIN association; P21a3 must prevent that
  rule from reclaiming a quantity already delivered into a receipt batch.
- P20: exact exported custody input/result/error types and service alias;
  expected owner/source custodian/destination warehouse, order holder and
  agreement reference, supported consignment purpose and ambient transaction
  guarantees. Inventory alone locks placements/capacity/holds. Preserve its
  source-changed/duplicate/conflicting-hold errors.
- P21a1→a2: exact saved authority lookup and transaction methods, active versus
  sold/returned lifecycle predicates, accepted terms link, HTTP DTO and version
  fields. P21a2→a3: physical dispatch/money checks and actual quantity receipt
  seam; do not undo its consignment rejection to implement purchase receipts.
- Before migration 044 creation, hub confirms number availability and unapplied
  status. Predecessor schema omissions require exact hub-owned remediation;
  do not edit supply/reservation/payment schema outside packet ownership.

## Complete acceptance coverage

Local coverage labels below are traceability labels under R3, not backlog IDs.

| Label | Original/accepted acceptance | Owning packet |
|---|---|---|
| C1 | Persist agreement/order/vehicle links and exact accepted settlement/handover terms; supplier owner distinct from dealer custodian | a1 |
| C2 | Active consignment info available through module-root port for P21b; no sale/return policy added | a1, handed off by a2 |
| C3 | Allocated ship with receiving warehouse and legacy allocated receipt use accepted mode, including partial receipts | a2 |
| C4 | Reservation purchase debt blocks absent agreement; same accepted amount allowed only when exact mutually accepted conditions hold; pending evidence never paid | a2; quantity path preserved a3 |
| C5 | Consignment actual VINs only; all-future conversion allowed upstream; later VIN assignment enables custody; quantity Deliver cannot mint buyer-owned consignment stock | a2; negative preserved a3 |
| C6 | Supplier owner retained, correct dealer custody/warehouse, physical handover once; sold/returned/repeated/stale rejected | a1 predicates + a2 transition |
| C7 | Port failure rolls back commerce; bound transaction, capacity failure, order/shipment/agreement/event failure no partial transition | a2, quantity failure a3 |
| C8 | Accepted money/debt and settlement survive custody change; no pending-credit/payment duplication | a1 persisted link + a2 |
| C9 | App adapter invokes custody, preserves exact owner/custodian/holder/agreement/actor/time, not Transfer | a2 TestWorkflowPort21 |
| C10 | Purchase quantity receipt consumes the same future claims exactly once with durable shipment-line/receipt-batch provenance | a3 |
| C11 | Subsequent receipt VIN identification/retry does not consume again; Allocate cannot reuse delivered quantity/claim; materialized physical rejection/return never reopens capacity | a3; preserve P18b rejection rules in a2 |
| C12 | Ordinary purchase allocations/quantity batches retain accepted behavior; exact source/line/spec/colors, quantities, revisions and transaction failures covered | a2 + a3 |

## Verification and deferred independent checks

Latest unit-only policy prevails: zero automatic review/QA packets or loops.
No tests run by this slicer. Workers run only affected pure unit families once
per relevant change as listed in their packets. Do not rerun whole service
packages or green predecessor units ceremonially. No lint/build/vet/race,
DB/migration execution, browser/E2E/integration or deployment. New tests exercise
actual services with stateful transaction fakes, not only selector/helper logic.

If separately requested by the user, the following independent small QA packet
definitions can be dispatched against one exact candidate SHA, without code
ownership: Q21a1 (4k context, C1/C2: authority/tenant/version snapshots);
Q21a2 (6k, C3–C9: purchase money matrix and VIN custody/app adapter);
Q21a3 (6k, C10–C12: mixed VIN/quantity claims, retry and identification).
Each takes the relevant implementation result, final source hashes and named
unit family; returns reproducible evidence or a precise gap. Resource is an
immutable-checkout read lease; no source writer/DB/browser. Stop on SHA drift,
missing requested authority or a needed non-unit check. These are dormant
definitions, not authorization or a new development gate.

Final-SHA shared-interface acceptance mapping: hub records that P18b accepted
mode/debt feeds all three physical entry paths; P20 custody receives the actual
order holder/agreement in bound context; quantity batch provenance agrees with
P18b claim reassociation/progress; P21b gets active consignment authority. Tie
affected unit evidence to final hashes and flag invalidation by later edits.
Do not repeat unaffected passing units at the final SHA. Real SQL rollback,
lock-order races and cross-module receipt identification are deferred human/CI
integration checks, requiring separate explicit authority, exact final SHA and
environment/fixture lock. Unit success is not release readiness.

## Gaps, risks and stop conditions

No new product question is introduced. Actual P18b/P20 handoffs and dispatch
source hashes are required technical prerequisites. Do not invent agreement
thresholds, schedules, fees, receipt ownership modes or return eligibility.
OD-01/08/09/10 exclusions in plan.md remain; financial provider delivery and
legal terms are not added. No retail sale without a VIN.

Risk: receipt identification belongs to inventory; current app receipt summary
exposes batch counts, not a claim callback. Keep claim consumption durably bound
to the receipt batch and shipment line, so identification needs no second
consumption. If acceptance requires a new cross-module write/callback or exact
VIN association unavailable from final predecessors, report that concrete seam;
do not pretend a fake identification unit proves the real inventory route.
Similarly do not import inventory internals into app tests to force coverage.

Stop before an unowned edit, unexplained source drift, missing accepted term or
schema, unknown policy or context ceiling. Write a compact checkpoint and request
bounded reslicing; use RESLICE_REQUIRED only when canonical scope changes.

## Planning input SHA-256

Paths below are root-relative except named execution documents, which are in
this proposal's directory. Hashes of narrowly inspected source are historical
planning evidence and must be refreshed after predecessors; no evolving
P18a1 reservation_payment, ports.go, reservation.go or store.go was read.

```text
a222419ecca4935c906454884eb62d6cacace6a5b80395c90a1cee870f8b8524  AGENTS.md
b22213a76ca11a4ce4aeb04323260d1d577f28ef1230d48c15367d074e67650b  internal/AGENTS.md
b2d5153012dfec17e20a5e55a65af858d693cae27adb66ef6a5c440599149b27  docs/justix-auto/dev/AGENTS.md
4f264fb5217a9613b00df455b0661cb9b955e90793a29b17c682cbcd0010830b  docs/justix-auto/dev/agent-workflow.md
f8a0d756be25ba54b71b2b22294676f0e18adc3747f2bbc841d825e87ab88c69  plan.md
398f8ac001f63cb9510bfaeda035d2609809f9a2e4fb5a55a5f44f26e79d910b  packet-21a.md
a5576feda33362b515281f06fc9131989cf6a58a944e826593491a1796c254ae  packet-20.md
93cfa1c5751ee7d568e587649e3b8d8ebe90ae3413a20553f7a8b292162ee7be  reservation-order-allocation-seam.md
6cf415a135e9326c9019a43d3e1d462f095398b4d6510c9d0033238d1e19b588  reservation-order-allocation-approval.md
023703090b1d3bc741c9ad64d0f85bd9342cacabc98f240c40d7c5c9d55ad93c  p18b-split-proposal.md
d19e81173a1728a4c8ae711e33761fb36e96d260c8607fd4dc058376d903ed64  packet-18b2.md
31b7d08cd36056715dea05475225d49a125dea2fc4dd537f15f49ffc19d7c259  result-17b-completion.md
db377700c932f5e806cb7a96ddb469ee114d692243ffac63be696a270466c6f0  internal/modules/commerce/service/fulfilment.go
8782d273174c56e7c925d580705818b89a45b981ba5d51f6069a08c1529e028b  internal/modules/commerce/service/reservation_allocation.go
716467794e1f651d1ee0e830eac24a2658c8279a5df6952805689eba15edf8a0  internal/modules/commerce/model/supply.go
81d369b7b85a17a17647994f28fee595ccd0582eb54b310985dadcdb87fab193  internal/app/app.go
7d8dd6761e2525b3555156291455e285f0b7d9432a98eab01efc017883c75b3b  internal/modules/inventory/service/receipt.go
```

## Proposed packet SHA-256

```text
37740bbb44e6551155f7823f8532c6090402375ef6819f98a7bcc5c38777398c  packet-21a1.md
0ed643fa195a531e593878774a38346a756bfcf425a83985accb92a0099688e9  packet-21a2.md
439af375acf71408c383e13c1c44b41fc3afdbc0dce0f897babe6cc39940c01a  packet-21a3.md
```

Plan digest is reported separately to avoid self-reference. Root acceptance of
these exact artifacts is pending. No canonical/source/Git/test/runtime changes
were made by this decomposition.
