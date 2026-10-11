# P22 — distributor consignment return result

Status: SOURCE COMPLETE. The P22 sole source-writer lease is RELEASED after this result. Work stayed in the assigned `feature/company-commerce-workflows-20261011` main checkout; `bash tools/check-git.sh` confirmed the project Git root before source edits. Migration 045 was created as source only and was not applied. No Git write, SQL/database/runtime action, deployment, build, lint, typecheck, browser/integration check, automatic review/QA loop or child agent ran.

## Public API for P23

All routes are under `/api/v1/commerce`, require an authenticated company context, and use the established error envelope. The supplier or holding dealer needs `commerce.read` for reads; the supplier needs `commerce.trade` to request; only the holding dealer with `commerce.trade` may confirm or reject. Other companies cannot read or decide another party's return.

| Route | Input and revision | Success |
|---|---|---|
| `POST /consignments/vehicles/:vehicleId/return-requests` | `If-Match` is the **unit** revision from `GET /consignments/orders/:id`; JSON `{ "destinationWarehouseId": "<supplier warehouse UUID>", "reason": "<nonempty text>" }` | `201` data envelope, new request ETag/revision `1`, plus `unitRevision` after the pending association is written. Request creates no stock movement. |
| `GET /consignments/return-requests?orderId=<order UUID>&limit=<0..100>&offset=<nonnegative>` | `orderId` is optional; omit it for the company queue. `limit=0` defaults to 50. | `200` list envelope, request date/ID descending, `nextCursor` as decimal offset when page is full. Includes pending, confirmed and rejected history for either actual party, including after the unit's pending pointer clears. |
| `GET /consignments/return-requests/:id` | Request ID | `200` data envelope with request ETag/revision; only actual parties see it. |
| `POST /consignments/return-requests/:id/confirm` | `If-Match` is the **request** revision; JSON `{ "movedAt": "<actual RFC3339 timestamp>" }` | `200` data envelope with terminal confirmed request and new request ETag. |
| `POST /consignments/return-requests/:id/reject` | `If-Match` is the **request** revision; JSON `{ "reason": "<nonempty decision reason>" }` | `200` data envelope with terminal rejected request and new request ETag. |

The request DTO has `id`, `unitId`, `agreementId`, `orderId`, `vehicleId`, `supplierCompanyId`, `dealerCompanyId`, `destinationWarehouseId`, `reason`, `status`, `requestedBy`, `requestedAt`, optional `decidedBy`, `decidedAt`, `decisionReason`, `movedAt`, `movementFactId`, `movementOperationRef`, decimal-string `revision`, `unitRevision` on create, and `allowedActions`. Only a pending request viewed by its dealer with `commerce.trade` advertises `confirm` and `reject`. `If-Match` absence/invalidity follows the existing HTTP helper; stale versions return `412`. A replay, sold/returned/unhanded VIN, current retail association, wrong party, or nonpending request never causes a second movement. The list/detail routes use commerce's retained party and order records, so dealer access does not depend on current inventory placement after return.

## Authority, movement and history

Request locks the same `commerce_consignment_units` live VIN row as retail before checking accepted agreement/order/line provenance, current retail and pending-return associations. It requires supplier ownership, `held` status and a complete P20 handover record, verifies the destination with `OwnsWarehouse(supplierID, warehouseID)`, inserts a pending return row, then writes that row's ID into `PendingReturnID` by unit version CAS in one transaction. Another VIN of the same agreement remains independent. The return table permits multiple historical requests but only one pending request per unit. No request automatically expires or releases a hold.

Dealer confirmation relocks that unit first, then the exact pending request, rechecks parties, accepted source, unit identity, version, association and lack of a retail deal. It rechecks supplier warehouse ownership and calls the P20 root custody port with purpose `return`, **zero expected holder**, supplier as unchanged legal owner and destination company, dealer as actual source custodian, the request ID as operation reference, agreement ID, actor and supplied movement time. P20 locks current physical placement and active holds, checks destination capacity, and returns one immutable fact ID for the one VIN. The fact ID and request operation reference are saved on the terminal request; the unit changes to `returned` with the same fact ID. A failing hold, custody/capacity, row CAS or event write rolls back the request decision and physical movement together, leaving it pending. No payment, credit, invoice, accepted debt or settlement terms are touched.

Dealer rejection records its reason, actor and time without a custody call; it clears only that VIN's pending association and leaves the unsold `held` unit eligible for a later request or retail decision. Request, confirmation and rejection append commerce events while the return row retains the durable request and terminal decision. Existing consignment handover adapter semantics remain unchanged; the new typed `ReturnCustodyMove` maps to inventory's return mode in `internal/app` with the bound transaction context.

## Affected unit evidence and limits

The app adapter selector passed after its final adapter/test edit:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort22'
ok  justixauto/internal/app  1.349s
```

The service selector passed after the final commerce change (the earlier pass preceded the order-scoped history addition):

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestConsignmentReturn'
ok  justixauto/internal/modules/commerce/service  0.442s
```

Stateful service fakes cover request-only intent, separate VIN independence, retail association exclusion, party-scoped and post-decision order history, exact one-time confirmation/fact linkage, rejection and re-request, replay/stale versions, and rollback on warehouse, custody, decision, unit, event and outer transaction failures. The app fake checks return purpose, empty holder, bound context, exact one-VIN input/fact and propagated failures. These pure units do not prove installed migration constraints, PostgreSQL row-lock races, deployed cross-module rollback or release readiness. The unchanged P21 consignment adapter selector was not repeated.

## Final source SHA-256

```text
fb158ab589552a67ac545a5988fc20c382835858be59a514db21cd1afe18206b  internal/modules/commerce/model/consignment_return.go
5ba23da2c3e625fd99eb4ca4819e3a7993a856b21db266961858165f295b4844  internal/modules/commerce/service/consignment_return.go
0580ae27f739a4ed290652e9ba3cda01b0ef81a2d85619aff9cb10c44e1165e2  internal/modules/commerce/service/consignment_return_test.go
7f071e26a33d18f0682a5d7e0408c0951cfaf8b3c4ef4cdc907f9fe028eeda74  internal/modules/commerce/service/ports.go
a13af1e0bdb090c82b17488f5d6c97ee601c50cdc12a7e550dba214407c5bd3f  internal/modules/commerce/repository/consignment_return.go
caa553ccc3cb1f4d018e9108e8248e49fc25e557b18a27f2d7d88d2ce3a4d81a  internal/modules/commerce/repository/store.go
56ccd31acb1165bdac2446f52c4bdb8515a55e62d4b1e400d60fe169b9af3e00  internal/modules/commerce/handler/consignment_return.go
d94ebc4a5b1ebeb1a9aacf38fea7a150cb7e589566396ec10528de2ffe31c5e0  internal/modules/commerce/handler/handler.go
1e1debe3dcd01bb97c3669eb6e1401248c9a8ed1c5e8524001e8f18f49e2cc14  internal/modules/commerce/module.go
4899e451f794b05df360a00a8c150e7158de4edec064782d8eb8fc864640ed42  migrations/000045_consignment_returns.up.sql
76252b8d5358d4a1185c6d47e7cb0b3c0e91f38381707fd4e3f91f8fc0653980  migrations/000045_consignment_returns.down.sql
4866042936eeb14577efa5988eee29cedd5927798a7f620079f5cfb39955c803  internal/app/app.go
6124f0fdbc61bad198451601618308ac251927bea330bc418122874a28e69046  internal/app/workflow_ports_test.go
```
