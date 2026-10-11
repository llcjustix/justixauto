# Application acceptance of the consigned retail selector seam

Accepted proposal SHA-256:
`26cc08c84bd02fe336c713e613a4c81dbc834692483278bb75823c680e529f82`.

P21b's mutation support alone does not make a consigned VIN selectable in the
existing sale form: its owned-stock inventory query deliberately excludes it.
The proposal closes this concrete original R3 journey gap without broadening
inventory owner visibility or adding a role/permission grant.

Accept the proposal's P21b4 read-projection definition and nine exact backend
paths, including the new pure handler unit. It executes after final P21b1–b3.
Accept its P23a selector definition and five existing frontend paths, before
the original P23 workspace. Each job has a 12k context ceiling (raised from the
proposal's 8k to accommodate scoped rules, final handoffs and existing form
suites without omitting acceptance). Original P21a1 persistence scope is unchanged.

Use only current-company retail.read authorization for the additional redacted
candidate projection and existing inventory.read for real detail hydration.
The normal B2B agreement DTO keeps commerce permissions. No client supplier
impersonation, finance-field disclosure, warehouse-only authorization or silent
403-to-empty fallback. Normal own-custody stock remains available for both
company types; actual sale uses the P21b transaction authority.

Final route/DTO/pagination and prerequisite source hashes are required before
frontend dispatch. No endpoint name from a proposal becomes a mock-only API.
Only the listed affected unit families run after source changes. No dormant
QA/integration definition authorizes additional verification or runtime work.
Source-writer leases are separate and sequential; this is scope acceptance,
not a current source grant or full-feature completion.
