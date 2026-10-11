# P19 stock discovery result — original R3/P19a U2 backend seam

Status: DONE for this bounded backend packet. Sole source-writer lease RELEASED after this result. Supplied branch `feature/company-commerce-workflows-20261011`; actual HEAD remained `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, with dirty predecessor changes preserved. `bash tools/check-git.sh` passed before edits. The accepted proposal/packet are recorded in `p19-stock-discovery-approval.md`. No Git write, runtime/database action, deployment, migration, browser/integration check, build, lint, typecheck, vet/race or review/QA loop ran.

## Delivered API

`GET /api/v1/commerce/reservation-stock?supplierCompanyId=<UUID>&modelId=<UUID>&limit=50&offset=0` is mounted under the existing authenticated commerce group with `auth.RequireCompany()` and `commerce.read`. `supplierCompanyId` is required; `modelId`, `limit` and `offset` are optional. Effective limit defaults to 50, accepts 1–100, and offset is nonnegative. Unknown query fields such as `companyId`, `placement` and `eligible` cannot widen the stock filter. One repository page is read; there is no all-pages scan. The list envelope is `{items:[...],nextCursor:<decimal offset string or null>,asOf:<server timestamp>}`. A full page advances by effective limit; a short or empty page ends with `nextCursor:null`, and an empty result is `items:[]`.

Each item has exactly `vehicleId`, `vin`, `supplierCompanyId`, `supplierCompanyName`, `modelId`, `modelSpecificationVersion` (historical decimal string), `exteriorColor`, and `interiorColor`. A returned `supplierCompanyId` and `vehicleId` are the IDs for the existing reservation POST. No price, terms, quantity, warehouse/custodian, owner/holder, receipt, private requisites, document, contact, or unrelated vehicle fields are serialized. Nil historical colors become empty strings.

Denied scopes: anonymous 401 `unauthenticated`; missing `commerce.read` 403 `permission_denied`; missing/stale selected company 409 `no_active_company`/`context_changed`; invalid/missing/self supplier, invalid model, malformed/negative/oversize paging 422 `validation_failed` with exact query field; unknown caller gives the existing `companyId` field error; inactive/nonseller caller 409 `company_not_trading`; wrong/missing stored caller type 403 `dealer_required`; unknown/inactive/nonseller/non-distributor supplier or inactive partnership 404 `not_found` without supplier payload. Operational directory, partnership and inventory errors propagate rather than becoming empty success. Commerce resolves the actual caller and supplier through `Directory.Company` and checks `Partnerships.ActiveBetween` before inventory is called. The caller needs neither `commerce.trade` nor `CommercialAccess`; stored verification does not gate this read.

Inventory's typed `(*Stock).ReservableStock(context.Context, inventory.ReservableStockFilter) ([]inventory.ReservableStockVehicle,error)` forces `CompanyID=supplierCompanyId`, `Eligible=true`, `Placement="warehouse"`, optional `ModelID`, and bounded limit/offset into the existing `Vehicles().List`. That repository query remains unchanged: owner equality, nonblank VIN, placement present, no `held` reservation of any holder type, stable `vin,id` ordering. Generic tenant listing remains forced to the real caller. Availability is advisory; discovery has no transaction, hold, release, claim, event or state write, and existing request/activation guards and locked hold paths are unchanged.

## Typed composition handoff

Commerce's dedicated `ReservationStockDiscovery.ReservableStock(context.Context, ReservationStockFilter) ([]ReservationStockVehicle,error)` port is separate from its mutation `Stock` and `ReservationStock` ports. `NewReservationStockService(d Deps, stock ReservationStockDiscovery)` authorizes and projects views. Root aliases expose only those typed filter/vehicle/port shapes needed by `internal/app`.

The app adapter `commerceReservationStockDiscovery` maps the inventory root filter and projection field by field, preserves the context, and uses `strconv.Itoa` for the historical specification version. Compile-time interface satisfaction is asserted. Repository-wide concrete caller search found one `commerce.New` call in `internal/app/app.go` and one commerce `handler.New` call in `internal/modules/commerce/module.go`; both were updated explicitly. The module now takes `discovery ReservationStockDiscovery` after the existing reservation-stock parameter and passes `NewReservationStockService` to the handler. The app supplies `commerceReservationStockDiscovery{inventory: inv.Stock()}`. P18c's background-free `app.New`, `NewRuntime` hook, expiry runner, and `ReservationPayments` delegation remain intact; no optional-port fallback was added.

## Affected unit evidence

Ran each selected unit command once after the final relevant edit, all exit 0:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/inventory/service -run '^(TestReservationStockProjection|TestVehicleFilter)'
ok  justixauto/internal/modules/inventory/service  0.515s
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/inventory/repository -run '^TestVehicleFilter(EligibleQueryShape|GenericQueryKeepsCustodianVisibility)$'
ok  justixauto/internal/modules/inventory/repository  0.884s
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestReservationStockDiscovery'
ok  justixauto/internal/modules/commerce/service  1.683s
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/handler -run '^TestReservationStockHTTP'
ok  justixauto/internal/modules/commerce/handler  1.304s
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/app -run '^TestReservationStockAdapter'
ok  justixauto/internal/app  1.884s
```

Selected cases were nonzero in every package: new service/HTTP/adapter cases plus existing inventory service and repository filter cases. In-memory spies assert the exact forced inventory filter, preserved context, empty/error propagation and no inventory call for denied commerce scope. Panic store methods guard against transaction/event/claim use. Real Echo middleware/handler tests assert permission/company/context and validation errors, offset cursor, empty `[]`/null cursor, exact eight DTO keys, and ignored client scope labels. Existing repository DryRun units assert SQL predicate and ordering shape only; they do not prove live row behavior, race outcomes, or release readiness. P19a owns the downstream UI query cache and returned-ID reservation POST unit.

## Final source SHA-256

```text
d64a95dcd85e6a3178817f8b3ea9a86d43e8e86f6a8736ac70d4da6a2664ea27  internal/modules/inventory/service/reservation_stock.go
a27b0fc889a34253255898bd24e64e9b725efe0d218a08365410560ac99a4ebb  internal/modules/inventory/reservation_stock.go
d0ac435d68a62da6c166350fdd61528a2d5269edfe81d58f43a4264e00676477  internal/modules/commerce/service/reservation_stock.go
a5f6f1386771c0a93aebf1576c6e71d0bad64aa3e26cfa280466e350df09a1cf  internal/modules/commerce/reservation_stock.go
c801962246f4a7df058c26e5ab4066a367c1e09021cf69040fc292fba6756d37  internal/modules/commerce/handler/reservation_stock.go
c999445fa393db0f8b1bcc864e3d68532a7e1cc6c79277436afce090c7e9cb19  internal/modules/commerce/handler/handler.go
56cd0cb65849d57d1427792767774dc63082e7b21e626ac2b00106606ecda93b  internal/modules/commerce/module.go
2ee5dd7404d6e5f4067023011a6bf309048c600ca708fe4fac610f871db0975f  internal/app/reservation_stock.go
8df45119152eab727fe3092fa22fd07bd633bf6073b682e3f448e803b1b243ff  internal/app/app.go
c2b1a823d1777adccdfe852e429d86b599b3cd06241b88d8aed8869f53359e6  internal/modules/inventory/service/reservation_stock_test.go
4aa1da871ae4c54bb30d9dd5384e7e928961f8cc7711529b170248e40a868927  internal/modules/commerce/service/reservation_stock_test.go
a40223a0ce3a4cafa78703d81a6423d92248a504afc7c982ad7a8359a304ea1a  internal/modules/commerce/handler/reservation_stock_test.go
462d8acabed9ec2fb10af60bad920cb226cd8192a01c0b22a0a2215820bb07c7  internal/app/reservation_stock_test.go
```
