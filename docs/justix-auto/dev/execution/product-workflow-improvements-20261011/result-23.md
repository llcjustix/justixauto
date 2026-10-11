# P23 — consignment, return and ownership UI result

Status: SOURCE COMPLETE. The P23 sole source-writer lease is RELEASED. Work used `feature/company-commerce-workflows-20261011` in the main project checkout at supplied HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, preserving all dirty predecessor source. `bash tools/check-git.sh` confirmed the project Git root before edits. No Git write, SQL/runtime action, deployment, browser/integration check, build, lint, typecheck, React Doctor, automatic review or QA ran. P23a's five source paths and the owned-but-unneeded `order-actions.tsx` were untouched.

## Behavior and exact interfaces

- `data.ts` types the persisted `Terms.reservation` acceptance, consignment agreement/unit and return request DTOs, and the inventory's nullable actual `ownerCompanyId`/`custodianCompanyId`. Order and vehicle details, the warehouse inventory table and vehicle list display current owner and custodian separately. Order unit facts come from current inventory detail; inaccessible current facts are stated as unavailable, never inferred from historical agreement parties. Order and shipment detail query keys include company and context revision.
- `AcceptedHandover` displays the saved purchase/consignment mode, agreed handover text/conditions, explicit partial-purchase permission and exact optional minimum, plus the order's settlement schedule before existing shipment and receipt actions. Consignment quantity shipment is hidden and also rejected by the panel guard; a reservation order missing accepted provenance cannot trigger physical UI submission. Existing mode-aware endpoints and order revisions remain authoritative.
- `ConsignmentWorkspace` reads the agreement at `GET /commerce/consignments/orders/:id` and complete return history at `GET /commerce/consignments/return-requests?orderId=&limit=100&offset=`. It consumes `nextCursor` directly, rejects missing/duplicated/skipped pages, and publishes no partial history when a later page fails. Pending, rejected and confirmed requests remain visible after reopening the order and after unit pending pointers clear, even if dealer vehicle detail access has ended.
- Supplier requests a return only for an unsold, handed, held unit with no current retail association or pending request. The form chooses a company-scoped destination warehouse and requires a reason; POST uses that **unit's** revision in `If-Match`. Request success explains that no physical movement occurred. Dealer confirmation requires actual movement time; rejection requires a reason. Both use the **request's** revision and only the server-advertised `allowedActions`. No frontend owner/custodian mutation occurs.
- The order panel retains order total and issued invoice `paid`/`outstanding` as supplied by existing authority; it does not add reservation advance to the credit-inclusive Paid value or treat a return as debt cancellation. Company verification/type is not a UI gate on old return obligations; current `commerce.trade` permission and final server decisions apply.
- Successful writes reread current company session, order, invoices, agreement, every return page and warehouses, then invalidate order/consignment/return/vehicle/stock/warehouse queries. A failed authoritative reread leaves actions locked with a refresh-only retry and no POST replay. Stale/conflict/capacity errors remain visible. Existing fulfilment and receipt writes gained corresponding committed-write refresh handling.

## Affected unit evidence

From `web/`, the assigned Realization suite passed after the final relevant source edit:

```text
npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/consignment-workspace.test.tsx
Test Files 1 passed (1); Tests 9 passed (9)
```

The real rendered interactions cover accepted partial purchase and invoice Paid retention; null current owner/custodian; supplier request payload and unit revision; old unverified dealer confirmation with actual time and request revision; reasoned rejection; sold/retail/permission exclusions; complete multipage post-return history independent of vehicle access; failure of a later history page without a partial display; server capacity error and committed-write refresh retry without replay; and the existing OrderDialog's mode-aware physical action. The first run had one test expecting raw text for a known server code that kit maps to a user message; the fixture expectation was corrected. The final affected suite passed after subsequent source edits. Pure UI/HTTP fixtures do not prove deployed route wiring, physical movement, database races, CI or release readiness.

## Final source SHA-256

```text
79f0fff93a5d764f006c5292ca7f779acf88bdffad6e4589e33b9650b6f95413  web/apps/realization/src/consignment-workspace.tsx
09ffb8f52f6f7112f1f68186bc74fc116cf9629c5f52da7a14701eeeaa003900  web/apps/realization/src/consignment-workspace.test.tsx
d7e5e3c58c26c8d17ddeb9de411267a0a73186036a7dfe7fa7aa04a11bfb0e39  web/apps/realization/src/data.ts
94c384cb633aba94f38d6035bfc2bdaa22a1dc1f729e554c7e112b4ae8996b63  web/apps/realization/src/order-fulfilment.tsx
d129488996057f3b6f7b65d0358ac470bfc75bced927b004daf205a9b7163ba7  web/apps/realization/src/pages/trade.tsx
167c0ad5d452b2a16d6cd6137d5ac81adeba0921c2f456ced4030aebb9051471  web/apps/realization/src/pages/inventory.tsx
157e3e0aed7c91f23ad6bdf608495fa1612bf2c2dcceeab19a4d93f83fb595c6  web/apps/realization/src/pages/stock.tsx
```
