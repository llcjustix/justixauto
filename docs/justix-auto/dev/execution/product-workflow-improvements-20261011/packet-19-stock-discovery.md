# P19 stock discovery — bounded implementation packet

Status: PROPOSED; `/root` acceptance required before dispatch. Canonical task
`product-workflow-improvements-20261011`, original P19a U2/R3; supporting R1/R2.
Branch `feature/company-commerce-workflows-20261011`; supplied milestone
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74`. Hub records actual predecessor SHA,
dirty-source fingerprints, accepted proposal/packet digest and writer lease.
One coherent behavior: dealer discovery of active partner distributor physical
stock for the already implemented reservation request. Context ceiling **16k**,
fresh worker, target 12k; checkpoint/split before exceeding the ceiling.

## Read and dependencies

Read root `AGENTS.md`, `internal/AGENTS.md`,
`docs/justix-auto/dev/agent-workflow.md`, sibling
`p19-stock-discovery-proposal.md`, `packet-19a.md` and relevant result slices:
`result-16.md` API; `result-17a-completion.md` API/ports;
`result-18b2.md` saved source/claim handoff. Authority:
`docs/justix-auto/dev/product-workflow-improvements-20261011.md` R1–R3,
`product-workflow-interview-20261011.md` accepted reservation scheme,
`docs/justix-auto/contracts/http-domain.md` common transport §1 and relevant
inventory/commerce §3 only. Latest approved behavior overrides historical HTTP
proposal details. Do not load the board, mocks, entire upstream plans or modules.

Source reading budget: inventory `vehicle.go` List/listQuery, `stock.go`
VehicleInfo/Vehicle, `service/ports.go` VehicleRepository; commerce `deps.go`
tradingCompany, `ports.go` Company/Directory/Stock, `supply.go` List,
`reservation.go` dealer/Request/activate only, handler constructor/routes and
supply list; `httpx` list/query helper and `database.Page`; existing inventory
filter units and app workflow-port unit patterns. Inventory/commerce root aliases
and constructor/app wiring may be read only after final P18c handoff.

Hard dispatch dependency: P18c releases source writer and supplies its final
result, changed-path hashes and exact commerce constructor/app Stock accessor
seam. The slicer did not inspect evolving `module.go`, `internal/app/app.go` or
`cmd/api/main.go`; do not guess their final signatures. Follow-on P19a consumes
this packet's implemented route/DTO handoff, not this proposal as API evidence.

## Exact owned paths

Production:

- `internal/modules/inventory/service/reservation_stock.go` (new): typed narrow
  filter/projection and method on existing `Stock` service.
- `internal/modules/inventory/reservation_stock.go` (new): root aliases for those
  public cross-module filter/projection types only; no runtime registration.
- `internal/modules/commerce/service/reservation_stock.go` (new): commerce-owned
  filter/port/view, constructor and discovery service; no existing stock port
  expansion or Deps constructor change.
- `internal/modules/commerce/reservation_stock.go` (new): commerce root aliases
  for port/filter/projection needed by app.
- `internal/modules/commerce/handler/reservation_stock.go` (new): GET parsing
  and DTO mapping only.
- `internal/modules/commerce/handler/handler.go`: one service constructor
  parameter/field and one company-scoped read route only.
- `internal/modules/commerce/module.go`: after P18c release, add one explicit
  typed discovery dependency to module construction and pass the constructed
  discovery service to handler. Preserve all predecessor runtime behavior.
- `internal/app/reservation_stock.go` (new): typed inventory-to-commerce adapter.
- `internal/app/app.go`: after P18c release, construct/pass that adapter to
  commerce.New using inventory's existing exported Stock accessor; no runtime
  loop/clock/lock/shutdown changes.

Tests:

- `internal/modules/inventory/service/reservation_stock_test.go` (new).
- `internal/modules/commerce/service/reservation_stock_test.go` (new).
- `internal/modules/commerce/handler/reservation_stock_test.go` (new).
- `internal/app/reservation_stock_test.go` (new).

Worker report: sibling `result-19-stock-discovery.md`, created by the dispatched
worker only. No frontend, migration, inventory query/handler, existing mutation
port, approved contract, permission catalog or command entry-point edits.
Any additional constructor consumer or fixture repair requires exact-path hub
extension before editing; no arbitrary source tree glob ownership.

Exclusive resource: sole main-checkout source writer, including commerce module
and app composition paths, after P18c release. No browser, runtime, live fixture
or DB lease. Run `bash tools/check-git.sh` before source edits; preserve all dirty
predecessor work, use apply_patch, no worker Git writes or children.

## Typed interface and HTTP contract proposal

Inventory service defines `ReservableStockFilter { SupplierCompanyID, ModelID
string; Limit, Offset int }` and `ReservableStockVehicle { ID, VIN, ModelID
string; ModelSpecificationVersion int; ExteriorColor, InteriorColor string }`.
Implement `(*Stock).ReservableStock(context.Context, ReservableStockFilter)
([]ReservableStockVehicle, error)`; root exports aliases from the new root file.
It is an internal trusted-module read, not an HTTP method and not an
actor-impersonation API. It never constructs or switches `auth.Principal`.

Validate required supplier UUID, optional model UUID, limit 0–100 and nonnegative
offset. Omitted/zero limit uses established pagination default 50. Then invoke
`store.Vehicles().List` with exactly supplier CompanyID, optional ModelID,
`Eligible:true`, `Placement:"warehouse"`, requested limit/offset; no search,
warehouse or general visibility parameter. Existing inventory query continues
to own owner equality, nonempty VIN, placement and absence of any held inventory
reservation, stable `vin,id` order and database pagination. Map to the narrow
projection; nil historical colors become empty strings, matching `Stock.Vehicle`,
never guessed catalog defaults. Do not call actor-facing Vehicle.List with a
fabricated supplier principal, read commerce/identity tables or add row locks.

Commerce service declares a separate `ReservationStockDiscovery` interface with
`ReservableStock(ctx, ReservationStockFilter) ([]ReservationStockVehicle,error)`.
Its filter has supplier/model/limit/offset; vehicle fields match the narrow
inventory projection except specification version is a decimal **string**.
`ReservationStockService` embeds existing `Deps`, stores the explicit port, and
is constructed by `NewReservationStockService(d Deps, stock ReservationStockDiscovery)`.
Its `List(ctx, principal, filter)` returns views carrying the safe vehicle fields
plus validated supplier ID/name. No optional port, dynamic assertion, fallback
to ordinary inventory listing or cross-module layer import.

App adapter `commerceReservationStockDiscovery` holds a small typed interface
whose method uses `inventory.ReservableStockFilter/ReservableStockVehicle` root
aliases. Map fields explicitly and use `strconv.Itoa` for specification version;
pass the original context and validated supplier/filter unchanged. Commerce and
inventory modules never import one another. Root aliases allow app to import
only module roots. Final P18c handoff determines the existing Stock accessor,
not this packet's public method/type names.

`GET /api/v1/commerce/reservation-stock?supplierCompanyId=<UUID>&modelId=<UUID>&limit=50&offset=0`

- `supplierCompanyId` required; optional `modelId`; limit/offset as above.
- Existing identity authentication, `auth.RequireCompany()` and
  `auth.Require(model.PermRead)` (`commerce.read`); service repeats permission
  and empty active-company validation for direct callers.
- `httpx.List` envelope: `{items:[...],nextCursor:null,asOf:<server timestamp>}`.
  Pagination is explicitly offset-based like existing supply/stock endpoints;
  advance offset by the requested effective limit. A short page ends the list;
  a full final page may be followed by one empty page. No unbounded fetch, total
  count, cursor invention or 101-row lookahead against the repository's 100 cap.
- Exact item fields: `vehicleId`, `vin`, `supplierCompanyId`,
  `supplierCompanyName`, `modelId`, `modelSpecificationVersion` (string),
  `exteriorColor`, `interiorColor`. Each row denotes one currently eligible VIN;
  no fabricated quantity, price, revision, terms or persistent availability flag.
- Never serialize inventory model/VehicleRow/VehicleInfo directly. No warehouse,
  custodian, receipt, holder/order/buyer IDs, private company requisites, history,
  documents, contact details or unrelated VINs. Empty `items` is `[]`.
- Extra query actor/company/placement/eligible/warehouse fields do not grant
  authority or widen the forced filter. There is no sibling detail endpoint.

## Authorization and errors

Service validates permission and active context before data access. Resolve the
caller with existing `tradingCompany(ctx,p.CompanyID,"companyId")`; enforce
stored `UserType == dealer`. Enforce required supplier UUID different from the
caller and valid model/paging fields. Resolve supplier via the same stored
Directory port; verify active seller distributor, and verify
`store.Partnerships().ActiveBetween(ctx,caller,supplier)` before ANY stock call.
Do not accept types/active flags/partner flags from input or infer them from
principal permissions. Caller/supplier identity lookup and partnership failures
must not silently return another company's stock or fall back to caller stock.

Observable errors:

| Situation | Expected response |
|---|---|
| Anonymous | Existing middleware 401 |
| Missing commerce.read | 403 `permission_denied`, zero Directory/stock reads |
| No selected company / stale context header | Existing 409 `no_active_company` / `context_changed` |
| Bad/missing supplier UUID, self supplier, bad model UUID, malformed/negative/oversize paging | Existing field validation 422 on exact field; integer parse via httpx.IntQuery |
| Caller unknown / inactive or nonseller | Existing tradingCompany field error / 409 `company_not_trading` |
| Stored caller type missing, distributor or another type | 403 `dealer_required` |
| Supplier unknown/inactive/nonseller/non-distributor, or no active partnership (requested/ended/declined/withdrawn included) | 404 `not_found`; no stock read and no supplier payload |
| Authorized partner has no eligible stock | 200 empty items |
| Directory/partnership/stock dependency operational error | Propagate existing apperr/error handler; never mask as empty success |

Use apperr.ErrNotFound for hidden supplier scope, matching P16 detail hiding;
do not return supplier names inside failed authorization errors. Directory
ErrNotFound is the only lookup error normalized to 404; genuine failures pass
through. Read permission alone does not authorize requests: do not call the
reservation dealer helper because it adds trade/CommercialAccess checks. Existing
P16 read behavior remains: active dealer can read with `CommercialAccess=false`;
stored verification does not become a new discovery prerequisite. All original
reservation command guards remain untouched.

Availability is advisory. GET has zero holds, claims, events or state writes.
Request creates no hold; existing supplier confirmation/buyer acceptance still
rechecks source and obtains holds under existing locks. Sold means ownership is
gone/transferred and therefore fails supplier ownership; do not invent a new
sale-status join. Warehouse means existing placement, not assumed custody.

## Complete acceptance and affected unit cases

| ID | Requirement mapping | Required observable cases |
|---|---|---|
| SD1 | U2/R3 discovery | Real GET route yields supplier VIN IDs and exact historical model/spec/colors; use distinct dealer-owned, partner-owned and outsider-owned identities in fakes; validated supplier is passed to stock |
| SD2 | R1/R2 authorization | Read permission vs trade-only; nil/missing context; stored caller type mismatch; client labels cannot override; active seller dealer + distributor + active partner success in either relationship direction; every denied scope has zero stock calls; CommercialAccess false read success |
| SD3 | U2/R3 isolation | Inventory fixed owner/eligible/warehouse filter; existing repository SQL excludes no-owner/sold, other owner in supplier warehouse, blank VIN, no placement and every held holder namespace; ordinary tenant list remains forced to caller |
| SD4 | U2/R3 bounded read | Required/invalid/self supplier, invalid optional model, malformed integer, limit 0/1/100/101, negative limit/offset; default50; offset/model forwarding, empty page, stable existing VIN/ID ordering; never all-pages scan |
| SD5 | R3 no hold | Spies panic/count on transaction, hold/release, event/claim creation: discovery calls none; dependency failure propagation; existing request/activation paths unchanged |
| SD6 | U2 safe HTTP | Actual handler/middleware tests with Echo httptest and in-memory ports, exact route/fields/envelope, historical string spec version, empty []/nil cursor, omitted sensitive fields, permissions/company/context errors and field errors |
| SD7 | S3 typed app seam | Fake inventory root projection through real adapter retains context marker, supplier/model/paging, exact VIN/model/colors and converts spec integer to string; error/empty mapping; compile-time satisfaction of dedicated commerce port |
| SD8 | P19a U2 consumer | Final backend result supplies route/DTO/errors/paging; downstream P19a component unit sends returned supplierCompanyId + vehicleId in real reservation POST shape, no fabricated terms or read hold; rejects stale availability using existing server error flow |

Tests must exercise real services/handler/adapter with in-memory collaborators;
unit fakes can embed unused repository interfaces and explicitly spy read calls.
Do not make a fake return every owner's vehicles and claim SQL isolation is
proved. Inventory service spy establishes forced filter; existing repository
DryRun cases establish SQL predicate/ordering shape only. They use disabled
automatic ping and no database/network execution. Real row behavior/races remain
outside unit-only evidence. SD8 belongs to P19a and is not implemented here.

From repo root, run each affected selection once after its relevant final edit,
using `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011` prefix:

```sh
bash tools/go.sh test -vet=off ./internal/modules/inventory/service -run '^(TestReservationStockProjection|TestVehicleFilter)'
bash tools/go.sh test -vet=off ./internal/modules/inventory/repository -run '^TestVehicleFilter(EligibleQueryShape|GenericQueryKeepsCustodianVisibility)$'
bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestReservationStockDiscovery'
bash tools/go.sh test -vet=off ./internal/modules/commerce/handler -run '^TestReservationStockHTTP'
bash tools/go.sh test -vet=off ./internal/app -run '^TestReservationStockAdapter'
```

Name new tests with those prefixes and report nonzero selected cases. No
module-root DB-backed tests, OpenAPI checks, tools/test-go.sh, integration,
browser/E2E, lint/typecheck/build/vet/race, automatic review/QA or green rerun.
Fix failing affected units and rerun only that selection. Do not broaden a test
run to diagnose unrelated dirty-predecessor failures; report exact blocker.

## Independent QA-sized packets and final-SHA mapping (not dispatched)

The unit-only user decision disables automatic QA/review loops. These are small
coverage definitions for an explicit future verification request, not scheduled
jobs or permission to repeat development units. Independent evaluator uses exact
hub-supplied immutable SHA and a fresh context; no code ownership.

- Q-SD-A, authorization/DTO: SD1/SD2/SD4/SD6; at most 6k context, commerce
  service/handler units and result only; affected unit selector above; no runtime.
- Q-SD-B, inventory/app boundary: SD3/SD5/SD7; at most 6k context, projection,
  existing listQuery and adapter units only; affected selectors above; no DB.
- Final candidate shared-interface check F-SD, SD7→SD8: hub maps producer result
  and P19a unit evidence to the final source SHA; verifies exact supplier/vehicle
  IDs, decimal-string specification, pagination and stale-stock errors survived
  later edits. Final-SHA live integration (partner read→request→competing hold→
  confirmation rejection) is a separate human/explicit-request gate needing
  immutable source plus dedicated DB fixture and identity membership setup;
  it is not authorized here. No previous GREEN list is claimed as integration
  evidence. A later source edit triggers only affected unit tests once.

These definitions do not add canonical backlog tasks. Default dispatch count is
one source packet, zero independent QA jobs, zero integration jobs.

## Stop and handoff

Stop before writing on missing hub acceptance/P18c final handoff, live writer,
unexpected changed source seam, unowned constructor consumer, new dependency or
schema, new product exposure/verification/custody rule, need to read another
module's tables, uncertain supplier authorization, or context >16k. Checkpoint
remaining coverage and request exact-path reslice; RESLICE_REQUIRED only if the
canonical task scope would change. Never resolve policy from fixtures.

Report final changed paths/hashes, actual predecessor/final SHA when supplied,
exact HTTP and typed port signatures, constructor handoff, selected units and
output, denied-scope/no-write evidence and exclusions. Explicitly release writer.
P19a remains responsible for the UI hook/query cache scoped to the real dealer
context + selected supplier + model/paging and the returned-source POST flow.
Do not declare original P19a, whole R3 or release readiness complete.
