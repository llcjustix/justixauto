# P19a — Supply and exact reservation agreement

Canonical task `product-workflow-improvements-20261011`, parent P19, R3/U1–U4/U9–U10 in p19-split-proposal.md. Application hub `/root`; feature base `4fa683ba4708bb71031534260ec5cb8742ce9977`, branch `feature/company-commerce-workflows-20261011`. Hub records actual predecessor SHA/source hashes and accepts proposal/packet digest before dispatch. Dependencies: final P16/P17 results AND final P18a2/P18b handoffs; proposals are insufficient API authority. Context ceiling 12k; fresh worker.

Read root AGENTS.md, dev/agent-workflow.md, web/AGENTS.md, plan common rules/R3 and proposal. Contract reference: docs/justix-auto/contracts/http-domain.md common values, identity §2 and commerce/inventory §3 only where relevant; actual final handoffs resolve new APIs. Read narrow supply/catalog/session/query helpers and PurchasesPage symbols. No full board, entire trade page, mutable backend source, fixtures-as-policy or unrelated P11–15 exploration.

## Exact owned paths

- `web/apps/realization/src/reservation-workspace.tsx`
- `web/apps/realization/src/reservation-workspace.test.tsx`
- `web/apps/realization/src/data.ts`
- `web/apps/realization/src/pages/purchases.tsx`

Resource: exclusive source-writer; no browser, live fixture/DB or runtime. Required `bash tools/check-git.sh` before source edits; no Git writes. Preserve dirty predecessor files. Use shared modal/table/query components and existing company commercial-access, userType and permission checks; legacy obligations/history remain readable while new operations are blocked by verification rules.

## One behavior and boundaries

Implement supply→request→exact mutually agreed activation. Distributor enters/edits available future model, historical specification, exterior/interior colors, quantity and expected arrival using actual P16 rules. Dealer browses available physical stock and future lines, requests selected VINs or future units without requiring fabricated VINs/prices. Request does not optimistically reserve capacity. Refresh authoritative capacity after activation and preserve current selected identity on errors.

Purchases hosts discoverable reservation/supply entry without losing existing sent/received order lists or direct/offer creation. Workspace modal/list exposes party names, immutable requested/offered/accepted snapshots, their digests, exact quantities, price/currency, expiry, optional advance and agreed deadline. Free is the default; no percentage/deadline schedule is invented. Supplier confirmation and buyer acceptance use actual server actions and current If-Match revision. Changed or initially incomplete terms require exact offered digest acceptance; accepted price is readonly. Do not implement a local accepted-price edit or guess P18 proposal action names.

Show server next actions and current reservation state; P19b owns money interactions, P19c lifecycle/allocation/conversion controls. Establish a small reusable detail entry with selected reservation ID and refresh seam so later packets extend it without duplicating state/queries. No speculative API handlers or inactive fake buttons for unfinished operations. Money uses exact integer minor units and currency; revisions remain exact strings, never Number coercion.

## Acceptance and affected units

Create named `P19a supply and agreement` cases in reservation-workspace.test.tsx. Render real components and Purchases host with mocked actual HTTP/session: supply identity/date payload and capacity refresh; stock VIN request and future no-VIN request; omitted unknown desired terms; no hold on request; exact match vs changed/incomplete offer; accepted readonly terms; buyer/supplier/userType/permission/verification gates; loading/empty/failure/stale refresh; duplicate submission blocked. Assert actual method/path/body/revision and visible transition, not export existence. Include modal keyboard/close behavior where affected.

From `/Users/bakhromachilov/startups/justixauto/web`, run once after relevant changes:

`npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/reservation-workspace.test.tsx -t 'P19a supply and agreement'`

Output must name the intended suite and nonzero selected cases. Only affected unit tests; no lint/typecheck/build/React Doctor/browser/integration/review/QA loop. If tests fail, fix and rerun only affected selection.

Report changed paths, digest/revision/query/host seams, exact test output and exclusions. Stop on missing final API/queue/history/terms agreement contract, new policy or unowned path. If 12k cannot fit, checkpoint and split with hub; no acceptance omission. Hub source/result record feeds final shared-interface mapping U1–U4/U9–U10; no automatic final rerun.
