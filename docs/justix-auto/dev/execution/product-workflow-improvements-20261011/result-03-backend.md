# P03a/P03b backend commercial-operation guards

Status: DONE

The combined P03a/P03b source implementation is complete. The required narrow
app seam test passed after the coordinator-approved P01-origin compile repair:
the shared `reasonBody` request DTO was restored in the identity admin handler,
resolving its existing users, membership and company-user consumers.

## Operation matrix

| Operation | Unverified selected seller | Verified selected seller | Legacy/service path |
| --- | --- | --- | --- |
| Direct inventory receipt | denied before validation or writes | allowed | VIN identification and quantity correction unchanged |
| Retail deal or listing draft/publication | denied before dependency access or writes | allowed | listing withdrawal; deal contract, registration, invoice/evidence, payment, delivery and cancellation unchanged |
| B2B RFQ creation/send, quotation, quote acceptance/order creation | denied before dependency access or writes | allowed | reads, cancellation, addenda and accepted-order servicing unchanged |
| B2B offer creation/publication | denied before dependency access or writes | allowed | withdraw and existing-offer version work unchanged |
| Allocation, shipment and receiving on accepted B2B orders | unchanged | unchanged | accepted-order fulfilment remains available |
| Seller insurance/financing application draft creation | denied before repository/sale access or writes | allowed | legacy application edit, submit, evidence/document exchange, insurer/provider review and decisions unchanged |

The retail deal guard retains its existing `retail.deals` permission check
before the commercial check, so a caller without that permission still receives
the independent permission denial. No dealer/distributor-exclusive rule was
introduced.

## Changed paths

- `internal/modules/inventory/service/receipt.go`
- `internal/modules/inventory/service/receipt_colors_test.go`
- `internal/modules/inventory/service/commercial_access_test.go`
- `internal/modules/retail/service/deal.go`
- `internal/modules/retail/service/listing.go`
- `internal/modules/retail/service/deal_colors_test.go`
- `internal/modules/retail/service/installment_test_helpers_test.go`
- `internal/modules/retail/service/commercial_access_test.go`
- `internal/modules/commerce/service/deal.go`
- `internal/modules/commerce/service/offer.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/service/terms_colors_test.go` (hub-approved fixture-only extension)
- `internal/modules/commerce/service/commercial_access_test.go`
- `internal/modules/insurance/service/insurance.go`
- `internal/modules/insurance/service/commercial_access_test.go`
- `internal/modules/financing/service/application.go`
- `internal/modules/financing/service/commercial_access_test.go`
- `internal/modules/identity/module.go` (hub-approved root `Profile` alias only)
- `internal/modules/identity/handler/admin.go` (hub-approved P01-origin shared `reasonBody` DTO repair)
- `internal/app/app.go`
- `internal/app/workflow_ports_03_test.go` (hub-approved app seam file)
- `docs/justix-auto/dev/execution/product-workflow-improvements-20261011/result-03-backend.md`

## Ports and tests

Commerce `Directory.Company` now carries optional seller `UserType` and
`Verification` copied from identity's exported public profile. The app-local
`companyProfiles` port adapts only `CompanyProfile`, and does not map or create
any commercial-access capability. `TestWorkflowPort03` uses a pure fake to
assert seller type/verification and bank-provider-kind mapping.

Passed once after the relevant service changes:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/inventory/service ./internal/modules/retail/service ./internal/modules/commerce/service ./internal/modules/insurance/service ./internal/modules/financing/service`

Result: all five service packages passed.

Passed once after the coordinator-approved P01-origin seam repair:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort03$'`

Result: `ok justixauto/internal/app`.

## Exclusions

No P03c frontend work, handler/DTO/route change, migration, contract/canonical
document change, runtime action, Git write, or broad resource-load guard was
made. Existing direct supplier delivery enters inventory through the accepted
commerce-order fulfilment path and remains ungated.
