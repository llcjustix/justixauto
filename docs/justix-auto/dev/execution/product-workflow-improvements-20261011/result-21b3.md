# P21b3 — consigned retail lifecycle result

Status: SOURCE COMPLETE for P21b3; sole source-writer lease RELEASED with this report. Work used `feature/company-commerce-workflows-20261011` at hub-supplied local HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, preserving predecessor source. `bash tools/check-git.sh` passed before edits. No Git write, SQL/database/runtime action, deployment, lint, typecheck, build, browser/integration check, review/QA loop or child agent ran. Only the eight assigned source paths changed.

## Injection and lifecycle

Retail now declares one optional `Consignment` port with a checked `ConsignmentSale` projection. `service.NewDeps` and `retail.New` accept it as a final variadic argument, preserving existing constructors and Stock fakes. The app composition passes a `retailConsignment` adapter backed by commerce's module-root Check/Reserve/ForDeal/Cancel/Complete APIs and inventory StockService's trusted Reserve/Deliver APIs. The adapter maps all unit, agreement, accepted order/order line, VIN, supplier, dealer, handover fact, revision, retail deal, actor and time facts. The inventory delivery's actual fact ID becomes commerce's durable `deliveryRef`. Retail imports neither module.

Listing Create now checks authority and stock eligibility inside its bound mutation transaction; Publish already ran there and now uses the same check. A verified dealer with the live handed-over unit and current warehouse may list it. B1's true `(nil,nil)` absence permits the ordinary owned path for both dealer and distributor. Prepared, pending-return, wrong-dealer, already-reserved, supplier-owned bypass and read failures deny without falling back to Owned. Existing list/read/update/withdraw paths keep their previous history and verification behavior.

Deal Create first reserves b1's live per-VIN retail association inside the bound transaction. A matching consigned VIN uses b2's dealer retail hold; an absent live unit uses the prior owner Stock reserve. Both capture the current immutable vehicle snapshot. Every deal/stock/authority/lead/event failure propagates through the outer retail transaction. For Deliver and Cancel, the accepted lifecycle classification first calls B1 Check: only true live absence takes the ordinary path; only the exact `consignment_retail_reserved` code proceeds to ForDeal, which verifies the same seller, VIN and deal. Other errors and mismatches fail closed. The unit lock remains held through the physical operation and retail writes.

Deliver retains the existing tenant, branch, revision, state, contract, exact accepted-payment, insurance, registration and OD-01 partner-finance gates. Consigned delivery calls b2's customer-delivery operation, then b1 Complete with its actual fact ID, then the existing deal/listing/lead/events writes in one transaction. This successful delivery alone ends supplier ownership and records the existing accepted order as the settlement obligation. Cancellation keeps the accepted-payment and pending-installment blockers; after exact association verification it releases only the deal's retail holder, clears the matching commerce association and leaves the held unit, supplier ownership, dealer custody, accepted order and historical payments intact. Delivered/cancelled history reads do not require live ownership or consignment authority.

## Affected unit evidence and limits

The final relevant retail service selector passed after the last test change:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/service -run '^(TestRetailConsignment|TestSaleColors|TestCashSalePartialPaymentLifecycleAndDelivery$|TestCashSaleDeliveryGuardsAndRollback$|TestCashSaleOtherSchemesRemainStrict$)'
ok  justixauto/internal/modules/retail/service  3.427s
```

The app adapter selector passed after its final source change and was not repeated after later service-test-only edits:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort21_Retail'
ok  justixauto/internal/app  0.814s
```

`TestRetailConsignment*` uses stateful transaction fakes for listing creation/publication, dealer reservation, ordinary dealer/distributor fallback, supplier bypass, wrong VIN/dealer/agreement, prepared/pending/competing holds, stale/repeated actions, cancellation, actual one-time delivery and rollback after stock, commerce, deal, listing or audit failure. Existing selected cash/color units retain snapshot and payment behavior. The app units check checked-authority mapping, exact physical input and delivery fact, bound context, errors and optional constructor compatibility. These units do not prove installed migrations, PostgreSQL row locks or deployed cross-module transaction behavior; CI/release and human integration remain separate.

The initial retail selector found a test-only compile typo, and its next run found a fake listing revision mistake. Both were corrected before the final passing selector. No source fix outside P21b3 ownership was needed.

## Final source SHA-256

```text
8087f979dfa278fcbc399da8761c592f17bc49d1965d55a890986ae42cf5d3a4  internal/modules/retail/service/ports.go
60917d0546547c55016ff39127664f58f41403465dde2544d5c0c559b5bb4906  internal/modules/retail/service/deps.go
8eca0ca3bfe2fc981ed5cb0532200cfffdb9c3f6479f7403387b739edb54d3ca  internal/modules/retail/service/deal.go
106b86392ca9d2cf32f3f051cbda0a40866263fb19e95e3b4b63da148efd41fe  internal/modules/retail/service/listing.go
7d77db21ae1c4fcd519efd0e5b23d614f307a1cad9662b80acbf3a6869307726  internal/modules/retail/service/consignment_test.go
7fd637ee12d8753c6281819339c8091d3ebff9a294d13375bca98633f28619e9  internal/modules/retail/module.go
ea4c70b138b9550e9eddbe0f2a1d8b527d77930f228df64886fdee3ace103682  internal/app/app.go
5f78948c448c54c7c72957073446d0c85731b0e8f859592cbbf1bd827ece8919  internal/app/workflow_ports_test.go
```

P22 can use the same commerce unit row as its return serialization point; it must lock and recheck pending return against `CurrentRetailDealID`. No P22 return adapter or physical return behavior was added here.
