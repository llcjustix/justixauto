# P19 bounded UI split proposal

Canonical task `product-workflow-improvements-20261011`, original P19, requirement R3; application hub `/root`. Supplied feature base `4fa683ba4708bb71031534260ec5cb8742ce9977`, reported local milestone `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, branch `feature/company-commerce-workflows-20261011`. These are provenance, not assertions that the dirty checkout equals either SHA. Hub supplies actual predecessor SHA, current source fingerprints and final handoffs at dispatch.

Status: PROPOSED; root must accept scope and packet hashes before dispatch. Docs-only execution decomposition; original packet, canonical board, contracts and source remain unchanged. No tests, Git, runtime, review/QA or children were run. Three sequential implementation packets preserve original P19 scope; no canonical backlog additions and no RESLICE_REQUIRED scope change.

## Intermediate source map and findings

Stable UI was mapped by symbols and narrow slices, not full-page loading: PurchasesPage hosts buyer/supplier orders and existing direct/offer OrderCreate entry; trade.tsx hosts OrderDialog, allocation and InvoicePanel; BillingPage separates commerce/retail invoices using session permissions. OrderCreate has the existing commercial-access gate and captured offer terms; order-workspace.tsx derives order next actions. data.ts centralizes DTOs/hooks. Keep these hosts and shared components; no new route, navigation page, dependency or shared-package ownership is proposed.

Existing source sizes were data.ts 615 lines, order-create.tsx 308, order-workspace.tsx 31, purchases.tsx 216, trade.tsx 690, billing.tsx 312. Workers read only needed symbols. Original new reservation-workspace.tsx and reservation-payment.tsx provide concrete separate reservation and accounting behavior; no additional production files are needed. Existing OrderCreate unit tests use React Testing Library, a real QueryClient and mocked HTTP responses, providing an established pattern for behavioral tests.

P16 holds/materialized claims already consume future capacity. P17a exposes immutable requested/offered/current snapshots and exact digest acceptance; request itself does not hold. P17b allocation history distinguishes current associations from historical replacements and transfers the same claim through conversion. UI must not sum historical VIN rows as additional reserved quantity. Backend mutable source under P18a/P18b was not read.

## Sequential packets and locks

| Packet | Coherent behavior | Dependency | Ceiling |
|---|---|---|---|
| P19a | Supply entry and dealer request through exact mutual reservation activation | Final P16/P17 and final P18a2/P18b handoffs | 12k |
| P19b | Reservation advance/refund accounting in workspace and Billing | Accepted P19a plus final P18a2/P18b handoffs | 12k |
| P19c | Reservation lifecycle/VIN history through mutually agreed order continuation | Accepted P19b plus final P18b1/P18b2 handoffs | 12k |

One source-writer lease; serialize shared data.ts, reservation-workspace.tsx/test and host-page writes. No browser, DB fixture or runtime resources. A fresh worker reads root AGENTS.md, agent-workflow.md, web/AGENTS.md, plan common rules/R3, this proposal and only its packet/relevant predecessor slices. Required repository-root check precedes source edits; no Git writes by worker. Split again before context exhaustion, checkpoint exact remaining acceptance, and return RESLICE_REQUIRED only if canonical scope changes.

Union of packet ownership equals all ten original P19 paths. New files are exactly the four already authorized reservation workspace/payment files and their tests. Existing order-create.test.tsx is a read-only regression selection when order-create.tsx changes; if it requires edits, request exact ownership extension. No canonical contract edits or backend patches are authorized.

## Required handoff seams and dispatch gate

Final P18a2 and P18b results are mandatory before any P19 dispatch. Proposals alone do not define route, action or DTO names. Hub must provide exact routes/envelopes, permitted actions, role/permission mapping, decimal-string revision/If-Match behavior, Money fields, stale responses and query invalidation identities for:

- Supply create/update/list and eligible stock/future line selection (P16 final original result plus foundation continuation); retained model/specification/colors and available/held/materialized quantities.
- Reservation reads/snapshots, confirmation/acceptance, allocation/history (P17 final plus P18 amendments); concrete history/event availability. No invented history endpoint.
- Advance evidence/private attachments, reviews, full refund obligation/evidence/confirmation and accountant queue reads (P18a2). If the final API cannot populate Billing, report that exact missing seam; do not invent a queue endpoint or silently omit Billing.
- Authoritative money/credit inclusion, accepted advance assignment to orders/invoices and source identity, so invoice Paid is never incremented again in the browser (P18a2).
- Extension, cancellation, server release reason/deadline and mutually agreed purchase/consignment continuation including digest, route/payment/handover terms and partial-purchase flag (P18b1/b2). P18c runtime completion affects whole-feature readiness, not authorization to implement a browser expiry timer.

Unclear technical contracts block affected dispatch until hub provides accepted final handoff. No new product gaps identified: no default deadlines/percentages, partial purchase is explicit opt-in, refund is full and accountant-confirmed. OD-01/08/09/10 remain outside P19; unresolved distributor retail availability is untouched. Stop on any new product rule; fixtures cannot settle policy.

## Complete acceptance coverage

Local labels below map R3/P19 acceptance; they are not new canonical requirements.

| ID | Original behavior | Packet / meaningful unit observation |
|---|---|---|
| U1 | Distributor future supply model/spec/colors/quantity/arrival | a: submit selected identity, integer quantity and date; reload capacity |
| U2 | Dealer stock VIN and future no-VIN request | a: exact source IDs, quantities and omitted unavailable economics; no client hold |
| U3 | Requested/proposed/accepted snapshots, digest, price/currency/expiry/optional advance | a: both parties see exact terms; changed/incomplete offer waits for digest acceptance |
| U4 | Fixed confirmed price, later mutual changes, party controls | a: readonly accepted terms; c: explicit mutual continuation proposal, no in-place rewrite; userType AND permissions/server actions |
| U5 | Pending/accepted/rejected advance, full refund pending/confirmed, Billing queue | b: real submission/review/refund interactions and permission denial; only accepted credit affects accounting |
| U6 | Allocated VIN and historical/current associations without double counting | c: current/historical UI and allocation request; future-held/materialized conservation |
| U7 | Supplier extension, next actions, expiry/release reason and advance deadline | c: server state renders, versioned extension/cancellation; pending/accepted protection, no browser expiry |
| U8 | Purchase/consignment exact agreement, accepted advance credit, optional partial handover false | c: exact digest/mode/payment/handover payload and false/true cases; no/some/all VIN continuation |
| U9 | Existing direct order route and host page reachability | a: Purchases entry; b: Billing; c: converted order opens in existing host and direct/offer creation remains accessible |
| U10 | Errors/stale refresh, double-submit protection, usable modal states | all: HTTP failure/stale/mutation-in-flight interactions; lists/details reconcile after success |

## Verification scope and final-candidate mapping

Only affected unit tests once per relevant change. Exact Realization commands are in packets; require output naming selected suites. No typeof-export placeholders. Components and host entries are rendered, interacted with and checked against mocked actual HTTP DTOs; pure formatting assertions alone cannot cover U1–U10. Do not run lint/typecheck/build/React Doctor/browser/E2E/integration or automatic review/QA loops.

Independent QA case packets are defined here for a later explicit verification request, not scheduled development work: Q19a U1–U4/U9 request/agreement boundary; Q19b U5 money/Billing boundary; Q19c U6–U10 lifecycle/conversion boundary. Each is read-only, one exact candidate SHA, its matching unit family and no live resources. They may not approve their own implementation. Current user policy leaves their execution dormant.

At the final candidate the hub maps exact SHA/results to shared interfaces: supply→request snapshot; offer digest→acceptance; evidence/review→money-aware release; accepted advance→credit-inclusive invoice; allocation history→same claim/order; continuation digest→existing OrderDialog. Later relevant edits invalidate affected evidence only; run those affected unit selections once, not ceremonial reruns of green suites. DB isolation, deployed expiry runner, private-file transport and browser rendering remain separate CI/human integration checks requiring explicit authority. No release-readiness claim.

## Source fingerprints

Packet manifest (execution-relative; proposal digest reported separately):

```text
812b4d078ff571e42b22935b130db7667162db01dd8685397a5094dbc4b5854c packet-19a.md
c4f31912131b5b5315afba371436d3045ff23a59e6fe4ca0dd0214acbeef549b packet-19b.md
a1ee99275ea078ebb44ddcb367e174798e280f8ba92fa049cfebf83cd7cbd916 packet-19c.md
```

SHA-256 of observed stable planning inputs; final P18 source fingerprints must come from actual handoffs. Paths below are repo-relative unless marked execution-relative.

```text
a222419ecca4935c906454884eb62d6cacace6a5b80395c90a1cee870f8b8524 AGENTS.md
4f264fb5217a9613b00df455b0661cb9b955e90793a29b17c682cbcd0010830b docs/justix-auto/dev/agent-workflow.md
b8dee411a0a1043a038d72ccde611f5d12e70a6065a3d60eb64fde72600bdd02 web/AGENTS.md
5df5a66f4ad3526b36fc5f35c011eff7311eccb0e0b6b9a0e185636122542356 docs/justix-auto/dev/product-workflow-improvements-20261011.md
a2cd3b101a5d720e26aa7a96524b0252aa305b22db75895a52513b6b8777599e docs/justix-auto/dev/product-workflow-interview-20261011.md
e1a8f6f2114126e19b0b706c7e308df66ea8e4e1f1530f3ae6eed09cf6214361 web/apps/realization/vitest.config.ts
f14f059c17a5982d67bf6fbbea6c8ee7270cdb015f245ec93348887aac929d81 web/apps/realization/src/data.ts
cf77da93968659671f9454274991a8b306e6ef1b8125b927abea7ae6490f753d web/apps/realization/src/order-create.tsx
3ca61eec350031a407c2518a8292dd9275336b28a8ae06b989f5350eda87a5bf web/apps/realization/src/order-create.test.tsx
aa32dfaa08e74c9557a81d0e95bd1df87c4ce2cec721fd74cd1688c44a80bc44 web/apps/realization/src/order-workspace.tsx
ee81016a83deeb423e5e5c17f97b8b9b067d15285b915ab00762e540907a541d web/apps/realization/src/pages/purchases.tsx
56de5ce6f45860e535138d3e536146c971bdea56a052929ca9bbe29f420a4c8c web/apps/realization/src/pages/trade.tsx
a4ef86a97946b9ded4ff1e3c80f227935159a79596c9cd02a124299989e5000e web/apps/realization/src/pages/billing.tsx
```

Execution-relative inputs:

```text
f8a0d756be25ba54b71b2b22294676f0e18adc3747f2bbc841d825e87ab88c69 plan.md
0f438cc5dab682800d6c870315b2653766407c859bc9ef8ff1164d6457d268b6 packet-19.md
fb6879df1fe88a3d41d1e426dee092d7e5ce0d06bcc72cb202de0dc9a34c9446 result-16-foundation.md
3c6db0ff93d026a9c27d9dd6e81241e49be640c7e7768ba482289065408984e5 result-17a-completion.md
31b7d08cd36056715dea05475225d49a125dea2fc4dd537f15f49ffc19d7c259 result-17b-completion.md
270dfb178ef41b684e72a20b7ff0e454ac83d56f329f9c10b23d3e0caa487491 p18a-split-proposal.md
023703090b1d3bc741c9ad64d0f85bd9342cacabc98f240c40d7c5c9d55ad93c p18b-split-proposal.md
```
