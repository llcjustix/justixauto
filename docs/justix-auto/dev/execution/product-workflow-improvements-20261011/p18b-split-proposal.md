# P18b bounded split proposal

Canonical product-workflow-improvements-20261011, R3, original P18b; application hub /root. Supplied base 4fa683ba4708bb71031534260ec5cb8742ce9977, branch feature/company-commerce-workflows-20261011. Current shared dirty source is preserved; UI source writer remains exclusive. This docs-only proposal changes no original packet, accepted plan, canonical record, source or Git state.

## Two sequential implementation packets

| Packet | Coherent behavior | Context ceiling | Dependency |
|---|---|---|---|
| P18b1 | Money-aware expiry, supplier extension and refund-aware reservation cancellation | 17k | Actual P17b holder/claim and P18a accounting result |
| P18b2 | Mutual agreed conversion and ordinary order allocation/cancellation of transferred claims | 19k | Accepted P18b1 plus P17b/P18a actual interfaces |

Fresh worker contexts, one source-writer lock, no live fixture/browser/DB/runtime lock. Read root AGENTS.md, agent-workflow.md, internal/AGENTS.md, this proposal and assigned packet; targeted predecessor results supply actual DTO/API names. Worker performs required repository-root check before source edits; no Git writes. No P11–15 re-exploration or full-board reads.

This is execution decomposition inside the whole original P18b scope, including the accepted allocation clarification. It introduces no business rule or canonical task. No RESLICE_REQUIRED scope change; if a worker needs further bounded splitting, preserve acceptance and report a compact checkpoint. P18b is incomplete until both packets finish.

## Serial ownership and predecessor boundaries

Both packets explicitly own reservation_lifecycle.go, reservation_lifecycle_test.go, reservation.go, reservation_allocation.go, service/ports.go, repository/reservation.go, repository/supply.go, handler/reservation.go and handler/dto.go, all within commerce. They MUST serialize. b2 preserves completed b1 expiry/pass logic while adding conversion and converted-state interactions.

b2 additionally owns original P18b service/deal.go, model/order.go, model/offer.go, repository/deal.go and accepted seam service/fulfilment.go + reservation_allocation_test.go. The union covers every original P18b path and only the already accepted allocation-seam additions. No new production or fixture ownership is proposed.

P16/P17a create durable claim and nullable converted-order/line linkage in their still-unapplied owner schemas. P17b supplies holder reassignment/materialization/release helpers. P18a supplies row-serialized evidence/refund/credit-once primitives and money view. P18b consumes those interfaces, never duplicates their ledgers or joins another module's tables. If concrete predecessor omission is discovered at dispatch, return the exact missing path/interface for hub resolution; do not create an all-VIN prerequisite or another business restriction.

## Data and HTTP contracts

- b1 receives exact current reservation terms/version and authoritative locked money facts, including evidence-ever-submitted. It emits expiry/cancellation state and supplier-approved extension with revision/event. Extension/cancellation retain reservation If-Match and actual predecessor DTOs. No public expiry endpoint.
- b1 hands P18c an explicit bounded service pass contract including cursor/progress on protected/failed candidates and partial errors. P18c retains progress across ticks, wraps to revisit prior protected rows, owns RunOnce/timer/start/stop and accepted 60s/100/10s runtime bounds. No runtime code in these packets.
- b2 receives the mutually accepted continuation digest, exact payment/handover/route/mode agreement and P18a ledger credit seam. It emits one source-linked order, accepted term snapshot, credit link, transferred holder/claim ownership and explicit source reservation/order revisions.
- b2 keeps existing order Allocate route/input/order If-Match. It reconciles future/materialized claims in the same transaction as physical Reserve, allocation rows, order CAS/event. CancelOrder retains existing eligibility blockers and releases only eligible residual future claims.
- Actual endpoint fields, envelopes, status codes, source IDs and decimal-string revision semantics are reported at each completion for P19/P21. Proposal method names are not instructions to overwrite predecessor APIs.
- Purchase quantity-receipt claim consumption and consignment physical custody/handover remain P21a. b2 hands that packet the claim helper/provenance contract. Conversion with no/some/all VINs is already supported by b2.

## Acceptance coverage

| Original P18b / accepted seam behavior | Packet |
|---|---|
| Free/no-money expiry, required-advance deadline only if never submitted; pending/accepted protected; rejected history not absent | b1 |
| Supplier extension, expected revision, versioned event, no default duration | b1 |
| Manual cancellation awaits evidence decision and full accountant-confirmed refund; exact idempotent hold release | b1 consuming P18a |
| Row locks and money/deadline/state recheck after stale selection, converted-state protection | b1, preserved by b2 |
| Bounded progress past protected/failed candidates, partial commit, timeout/cancellation and retry safety | b1 → P18c |
| Exact mutual continuation, changed price/terms require renewed acceptance, no standard schedule | b2 |
| Purchase/consignment mode, absent partial-handover=false, explicit agreed conditions retained | b2; physical enforcement P21a |
| Unique reservation→order source link, credit accepted advance once, refund/conversion race | b2 consuming P18a |
| Conversion with zero/some/all VINs, atomic holder transfer and no duplicate claim | b2 consuming P17b |
| Later ordinary order Allocate reconciles same claims; ordinary orders unchanged | b2 |
| Eligible order cancellation releases future residual once; materialized physical release never reopens lot availability | b2 |
| Bound transactions and failure rollback; no extra inventory units/ownership transfer on conversion | both |

Latest user answers resolve product policy. No new deadline, retention/fee, refund ratio, financial schedule, required VIN-at-conversion or role restriction is authorized.

## Verification and limitations

User policy overrides historical review/QA gates: affected pure unit families once per relevant change, pinned Go with -vet=off and approved writable GOCACHE. Exact selectors are in packets; do not run entire service/app suite by habit or repeat green b1 scan tests when b2 only adds independent transition behavior. Shared transition/helper changes justify their affected selectors once.

Tests invoke actual lifecycle/conversion/Fulfilment.Allocate/Deal.CancelOrder paths using transaction fakes with claims, holders, money credit and events. They include no/some/all VIN, stale/overlap and rollback cases; render/helper-only tests are insufficient. Existing adapter/accounting units stay applicable if their code is unchanged. No extra review/QA, lint/typecheck/build/vet/race/browser/E2E/integration, migration application, scheduler execution or deployment.

Slicer ran no tests or source checks. Real database lock races, rollback across deployed inventory/documents, scan throughput and shutdown remain unverified under unit-only policy. RunOnce callback has no ambient lock transaction and may overlap after lock loss; no exactly-once/hard-deadline claim. No new product gaps; technical predecessors and source-writer availability remain dispatch dependencies.

## SHA-256 manifest

Packets:

```text
acac4a7ce0abb81a6bc9069e84b67758c6c59249580b2a9449ba9a7ddf010e3c  packet-18b1.md
d19e81173a1728a4c8ae711e33761fb36e96d260c8607fd4dc058376d903ed64  packet-18b2.md
```

Targeted planned-interface/authority inputs (application source hashes are supplied by actual predecessor results at dispatch; no currently changing source was inspected for this split):

```text
398a8ba2bbf4e21613ce07cd30b0f004d23ae9146d37d3d1f337ceeee7ce1a8d  packet-18b.md
d7b1d7befb5055397de2a9d7b77e019535f8e8512eb3e5253bf289e7663b082b  packet-18a.md
ab748fe94d69d4ab5ec70540565bdaf50572da44a41bfdc19a6304c93c303b19  packet-17b.md
93cfa1c5751ee7d568e587649e3b8d8ebe90ae3413a20553f7a8b292162ee7be  reservation-order-allocation-seam.md
6cf415a135e9326c9019a43d3e1d462f095398b4d6510c9d0033238d1e19b588  reservation-order-allocation-approval.md
9cf0569bb380f77d13ba7f23a168f388cc7582110ad0248e82a89d8e0b3c6f5f  expiry-runtime-decision.md
```

