# P19c — Reservation lifecycle, VIN history and order continuation

Canonical task `product-workflow-improvements-20261011`, parent P19, R3/U4/U6–U10 and U5 credit seam. Application hub `/root`; feature base `4fa683ba4708bb71031534260ec5cb8742ce9977`, branch `feature/company-commerce-workflows-20261011`. Root accepts hashes/scope and supplies actual predecessor SHA/source hashes. Dependencies: accepted P19b plus final P18b1/P18b2, P18a2 and P17b handoffs. Context ceiling 12k; fresh worker.

Read root AGENTS.md, agent-workflow.md, web/AGENTS.md, plan common rules/R3, proposal and only relevant predecessor interfaces. Contract reference: http-domain.md common values, identity §2 and commerce/inventory §3. Map trade.tsx OrderDialog and order-create/order-workspace symbols before reading targeted bodies; never load the entire host set at once. P18 final APIs determine lifecycle/continuation routes, statuses/actions, agreed term snapshots, digest and If-Match fields.

## Exact owned paths

- `web/apps/realization/src/reservation-workspace.tsx`
- `web/apps/realization/src/reservation-workspace.test.tsx`
- `web/apps/realization/src/order-create.tsx`
- `web/apps/realization/src/order-workspace.tsx`
- `web/apps/realization/src/data.ts`
- `web/apps/realization/src/pages/purchases.tsx`
- `web/apps/realization/src/pages/trade.tsx`

Exclusive source-writer; no browser/DB fixture/runtime resources. Required `bash tools/check-git.sh` before source edits; no Git writes. Preserve P19a/b behavior and shared modal/Money/query/session conventions. Existing order-create.test.tsx is read-only; request explicit extension if a necessary fixture repair is discovered.

## One behavior and boundaries

Complete the authoritative active reservation→continued/released state workflow. Show available history and current VIN associations separately; supplier allocates by accepted source line and eligible vehicle using actual P17b action, never by arbitrary claim selector. Held future quantity, materialized quantity and historical replacement rows must not be added as new independent capacity. Refresh server counts/allocations after mutation; show immutable source identity and read-only claim provenance when useful.

Expose supplier-approved extension and permitted cancellation using final P18b contract/revision. Show current next actions, agreed expiry, mandatory-advance deadline and server release reason. No client timer may mark expired or release quantity. Pending/accepted advance protects reservation; rejected history is not equivalent to never submitted. Refund confirmation/undecided evidence follows P19b money view and server actions, without bypassing settlement. History remains readable for completed/released/converted reservations.

Continue to purchase or consignment with exact mutually agreed price, currency, payment, route and handover terms from P18b2. Changed terms require renewed counterparty acceptance of exact digest; no standard schedule or implicit acceptance. Partial-purchase handover is an explicit opt-in field, default unchecked/false, persisted through proposal/acceptance and conversion according to final DTOs. No/some/all allocated VINs are valid continuation states when allowed by server; do not introduce an all-VIN prerequisite. Conversion sends exact accepted digest and revision, displays linked source credit without adding it again, and opens the resulting order in existing OrderDialog. An existing continuation link reopens that order rather than resubmitting conversion.

Keep direct/offer OrderCreate flows accessible and preserve their existing terms/validation. Prefer a distinct reservation continuation state inside the authorized workspace; touch order-create only for a concrete shared-form need. Physical handover/custody and returns remain P21/P23, not client-side ownership mutation here.

## Acceptance and affected units

Add named `P19c lifecycle and continuation` cases in reservation-workspace.test.tsx. Render real workspace/host seams with actual HTTP fixtures: current vs historical VIN association and conserved counts; allocation line/vehicle payload; server next actions; supplier extension; released reason/deadline; pending/accepted protection and rejected-history handling; refund-before-release state; purchase and consignment exact digest/terms; false-by-default vs explicitly accepted true handover; no/some/all VIN conversion; changed offer/stale revision; duplicate click/link reopening; open resulting order and preserved direct/offer entry. Assert interaction payload and UI state, never typeof exports. Failures refresh/reconcile rather than silently retrying a stale digest.

From `/Users/bakhromachilov/startups/justixauto/web`, once after relevant changes:

`npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/reservation-workspace.test.tsx`

This full workspace selection covers a/c because c changes their shared detail/host state. If order-create.tsx changes, also run its existing behavioral regression suite once:

`npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/order-create.test.tsx`

If c alters accounting behavior/interface, run the affected P19b named selection once; otherwise retain its earlier evidence. Require intended suite names/nonzero cases; failed selections never justify rerunning unrelated green suites. No lint/typecheck/build/React Doctor/browser/integration/runtime or automatic review/QA.

Report final source/order/credit identities, query refresh seams, exact test results and unmet acceptance. Stop on missing final history/continuation API, new product rule, unowned path or 12k ceiling; checkpoint and resplit without dropping acceptance. Hub maps final SHA to proposal U1–U10 and shared interfaces, with only newly affected units rerun. Separate human/CI integration remains unproven.
