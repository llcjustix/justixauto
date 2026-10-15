# Accountant queue presentation packet

Task: accounting-queue-ui-20261014. Authority/owning hub: primary `/root`.
Approved contract: `docs/justix-auto/dev/accounting-queue-ui-20261014.md` (entire bounded task).
Assigned branch: `fix/accounting-queue-ui-20261014`.
Assigned base: `a99f72c7c72463943a96f15cae049d5d47630f18` (provided by hub; no Git operations by slicer).
Artifact owner: `docs/justix-auto/dev/execution/accounting-queue-ui-20261014/packet.md` only.
Scope: one implementation packet P1 and one small final verification packet V1; no new canonical backlog entries. Ready for hub acceptance; not an implementation completion report.

## P1 — Present actionable billing queue rows

One behavior: make existing retail accounting/history and pending B2B issuance rows readable, distinguishable and responsive without changing their decisions or data flow.

Exact source ownership:

- `web/apps/portal/src/features/realization/pages/billing.tsx`
- `web/apps/portal/src/features/realization/pages/billing-queue.css` (new, locally imported)
- `web/apps/portal/src/features/realization/accountant.test.tsx` (only minimal existing assertions sensitive to changed copy, if needed; retain behavioral coverage)

Dependencies: approved task above, unchanged Deal/Order/InstallmentPaymentGroup contracts in `web/apps/portal/src/features/realization/data.ts`, existing kit Badge/Button/Panel and portal tokens. No backend, infrastructure, new package, request or shared component change is required. Read root `AGENTS.md`, `docs/justix-auto/dev/agent-workflow.md`, `web/AGENTS.md`, approved task and targeted source slices only. Context budget: 12,000 input tokens; if that is insufficient, return to hub before expanding scope.

Implementation and acceptance coverage (local requirement IDs map to every approved-task clause):

| ID | Required finished behavior | Evidence/check |
| --- | --- | --- |
| AQ1 | Replace retail bare paragraphs with a padded list of divided rows using existing typography/tokens/components. Client name is the clear heading; compact summary gives `waiting.length + submitted.size` and brief queue/history help. | Inspect final JSX/CSS; existing render case in V1. |
| AQ2 | Distinguish repeated clients using a visible deal ID/reference; show `deal.vehicleSnapshot?.vin` when present. Keep meaningful identification when snapshot is null/missing; do not invent make/model/VIN or introduce lookups. | Inspect use of current typed fields and optional access. |
| AQ3 | Waiting rows display existing `deal.price` as «Стоимость автомобиля», never as invoice total. Submitted rows display `payment.claimedAmount` as «Сумма чека» plus `payment.externalReference`; use existing `money` formatter, no arithmetic. Gracefully omit unavailable optional data; existing reduced test fixtures must still render. | Inspect field/label mapping; V1 grouped-review render. |
| AQ4 | Use clear status Badge and separate action region aligned right on wide screens. Preserve existing button names/callbacks; stack at narrow widths, allow long client names, references, VIN, money and buttons to wrap without clipping. Use local prefixed selectors, `min-width: 0` and wrapping/grid or flex rules; preserve keyboard focus. | Inspect scoped CSS and unchanged Button semantics; V1 action reachability. |
| AQ5 | Pending B2B issuance uses the same local row layout: buyer heading, order reference, existing `lines(order.terms.lines)` description, issuance status and existing «Выставить счёт» action. If showing `order.total`, label it as order value. Compact count/help uses eligible orders only. | Inspect shared local classes; V1 B2B issuance. |
| AQ6 | Preserve `waiting`, `eligibleOrders`, `submitted` Map deduplication, authority predicates, query keys/fetchers, financial scopes and all dialogs/callbacks/refetch behavior. Outside Accounting retain history title, read-only wording/actions and existing restrictions. Existing invoice tables, summaries and reserve/advance panels are untouched. | Final source inspection plus V1 grouped check, B2B, read-only and query-scope cases. |
| AQ7 | No catalog requests, new permissions, financial/backend/API/schema/policy changes, floating-point money calculations or fabricated snapshot content. No cosmetic tests or unrelated edits. | Final owned-path diff inspection by hub; V1 existing request-scope checks. |

Keep layout implementation local and small. Preserve semantic list/heading structure and native buttons rather than turning whole rows into new actions. `Deal.contractReference` is a contract reference, not a new deal-number contract; use the existing ID for deal identification and do not manufacture a reference field. Reuse the existing `lines` helper without adding another hook/request.

Rules/resources: one main-checkout source writer reserved by `/root` before execution; lock the three owned source paths until handoff. No browser, PostgreSQL fixture, live data, runtime service or remote resource needed. Before application edits, worker follows `bash tools/check-git.sh` root gate. Use apply_patch; preserve user changes. Worker performs no Git writes or canonical-document edits. Protected identity test and unread `test_creds.txt` must remain untouched and unread. Version control, if dispatched by hub, uses explicit paths and the approved task's `LEFTHOOK=0`; this packet does not grant deployments or production actions.

Stop conditions: changed approved contract/source hashes affecting this packet, another writer using owned files, unknown money/reference semantics, required new API/catalog field, changes needed outside owned paths, or unresolved product rule. Return the concrete gap to hub; never infer policy from fixtures. If resolving it changes canonical task scope, report `RESLICE_REQUIRED`. No product-rule gaps identified in current bounded scope.

## V1 — Final affected-unit verification by root

Dependency: all P1 work finished and source frozen. This is a small verification handoff for root, not an automatic reviewer/QA-agent loop. No checks run by slicer or per edit/packet. Use existing tests; only copy-sensitive assertion updates are permitted. On the complete candidate, with pinned Node 24.21.0/npm 11.19.0, run once from `web/`:

```sh
npm run test:unit -- --project @justixauto/portal apps/portal/src/features/realization/accountant.test.tsx
```

The existing file covers one deduplicated grouped-check acceptance, B2B issuance with revision header, read-only commerce decisions outside Accounting, and retail/commerce query boundaries. Keep those assertions; adjust only the two old sentence-based waiting-row selectors if layout changes their text. Do not add cosmetic tests. Existing waiting fixtures omit price/snapshot; do not derive product policy from them or change unrelated fixture behavior.

Root records command/result and exact final candidate SHA (or working-tree digest followed by matching committed SHA). Shared interfaces remain unchanged; the same final-candidate run checks their existing BillingPage → DealDialog/OrderDialog/InstallmentPaymentReview consumers. No separate integration suite or repeated earlier-packet results. After a failure, repair and rerun only affected scenarios. No browser/E2E, lint, typecheck, build, React Doctor or review/QA loops, per the approved task's latest user policy. Visual responsive acceptance rests on JSX/CSS inspection; the unit suite does not prove rendered appearance.

V1 owns no source, locks the frozen candidate during its run, and stops on unexpected unrelated failures or need for broader checks; report exclusions to hub. Implementation and acceptance authority remain with `/root`.

## Inspected source SHA-256

Hashes describe the inspected working files, not a Git-history verification. New local CSS is absent from the inspected source and is assigned for creation.

| Path | SHA-256 |
| --- | --- |
| `AGENTS.md` | `9535ac1f07c3c61574633449375b85feba89baaaca33474c5ed6d875c4953469` |
| `docs/justix-auto/dev/agent-workflow.md` | `f2714233d1619a2cb2d8c1b10b2622b8cebf2c2b60108b226efaff0f15999fb6` |
| `docs/justix-auto/dev/accounting-queue-ui-20261014.md` | `e10c021c62f57a512ac57a14b0d53f5d6377f12ad341feb05ca30402eeab0285` |
| `web/AGENTS.md` | `45c3250f9aa660310646a20c9bb9c2fd382ed8f82881124811cf5682f474e304` |
| `web/apps/portal/src/features/realization/pages/billing.tsx` | `91dbe4e8964b31e62203b656cf8afc57151213a2ff00e74e9e97286d40ed5352` |
| `web/apps/portal/src/features/realization/data.ts` | `d7e5e3c58c26c8d17ddeb9de411267a0a73186036a7dfe7fa7aa04a11bfb0e39` |
| `web/apps/portal/src/features/realization/accountant.test.tsx` | `50a86004de930698cd0fd17fb1f4b9d008db5e573f9eb668329eed00d562343d` |
| `web/packages/tokens/src/tokens.css` | `3699748dfedb69deaf2dc97529cde4c293c95f1b119d43052bc659229c841d1e` |
| `web/package.json` | `165b9ffae9bb1936595d112ee2654262cce67d006143e66a0708d7e41110f125` |
