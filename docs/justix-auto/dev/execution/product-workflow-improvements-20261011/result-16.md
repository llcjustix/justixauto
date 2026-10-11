# P16 result — Distributor future-supply availability

Status: DONE. Writer lease released after this report. Branch at dispatch:
`feature/company-commerce-workflows-20261011`, HEAD
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74`. `bash tools/check-git.sh` passed
before source edits. Existing dirty retail/UI/docs work was preserved.

## Implemented API and validation

`GET /api/v1/commerce/supply-lots`, `GET /api/v1/commerce/supply-lots/:id`,
`POST /api/v1/commerce/supply-lots`, and
`PUT /api/v1/commerce/supply-lots/:id` use `service.SupplyInput` and return
`handler.supplyWireDTO` in standard data/list envelopes. The DTO exposes
`totalQuantity`, `availableQuantity`, `heldQuantity`, `materializedQuantity`,
model/specification/colors, expected arrival, revision and timestamp.

Only an active verified distributor may create/update its lots; a distributor
lists its own lots and a dealer lists/gets only lots of active partners.
Create/update validates positive integral quantity, current catalog model
specification and exact exterior/interior palette. An update locks the lot,
uses the capacity projection, and rejects a total below held plus materialized
claims. This aggregate does not create inventory vehicles or alter ordinary
offer/order quantities.

## P17/P18/P21 claim handoff

`model.SupplyLot` and `model.SupplyClaim` plus `SupplyRepository` are now the
typed seam. `SupplyClaim` is unit-granular with stable ID, lot, nullable
reservation/reservation-line, nullable converted order/order-line, state
(`held`, `materialized`, `released`), and immutable materialization provenance
(`vin-allocation` or `quantity-receipt`, reference UUID, timestamp). P17 must
create a claim for each future-held unit under `Store.InTx`, after `LockLot`;
P18/P21 must retain the same claim ID while transferring order linkage and
materializing/releasing it. `ClaimsForUpdate`, `CreateClaims`, `UpdateClaim`,
and `Capacity` are the exposed repository methods. A materialized claim remains
committed even if its physical VIN is later released/rejected.

Migration `000041_commerce_future_supply` creates `commerce_supply_lots` and
`commerce_supply_claims`, indexes holder/order lookups and the unique
materialization reference, and intentionally has no foreign key to the future
reservation schema. The projection enforces the intended equation:
available + held + materialized = total.

## Evidence and exclusions

Initial complete package command
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service`
first exposed two fake-interface compile gaps, then an approved named-literal
fixture repair, and then one outsider assertion mismatch (existing
`tradingCompany` correctly returns a `companyId` validation error). After the
hub-approved exact fixture/assertion repairs, the only permitted targeted rerun
passed:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestSupplyRejectsInvalidColorQuantityAndOutsider$'`.
No migration execution, database/runtime/concurrency checks, lint/build/E2E,
Git writes, or source outside P16 owned paths were performed.
