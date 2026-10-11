# P19a original implementation completion

The initial worker released the source lease with four preserved draft paths
after two failures of its only VIN interaction test. It has not completed the
original P19a acceptance. This is unfinished implementation, not a review or
an additional QA gate. The final source must still meet packet-19a.md.

Application hub accepts a fresh explicit `gpt-6-sol/high` implementation job
with a 20k context ceiling. The diagnosed risk is an incomplete multi-state UI
plus an unresolved asynchronous discovery test, after repeated failed repair.
The same four owned source/test paths apply; no feature scope is added.

Preserve existing work and complete the actual supply create/edit, catalog
selection, partner VIN/future request, supplier confirmation and exact buyer
acceptance flows, party/permission/verification gates, exact money/revisions,
server error/stale refresh and Purchases entry. Raw UUID input is not a usable
replacement for actual catalog/partner/lot selection. Requests do not hold stock.
Do not guess the initial failure cause: diagnose the real query/list envelope
and async selection behavior. Do not settle for helper-only or export tests.

Backend sources are stable and targeted read-only DTO/service dependency reads
are authorized. GET /commerce/supply-lots already returns authorized partner
lots for a dealer (Supply.List / ListVisible), with supplierCompanyId and real
lot IDs. The final stock-discovery report defines the separate VIN read.
No new backend endpoint or supplier filter is required to select visible lots.

Run only the original named affected P19a unit family after the actual changes.
No lint/typecheck/build/React Doctor/browser/DB/integration/review/QA loop.
Write the final assigned worker report `result-19a-completion.md` with all
acceptance dimensions, actual API/props/query seams, hashes and exact unit
outcome, then explicitly release the source lease. Assigned worker result
artifacts are permitted; only canonical records remain root-owned.
