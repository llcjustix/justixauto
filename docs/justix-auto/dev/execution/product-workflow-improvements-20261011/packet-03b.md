# P03b — Guard new B2B commitments and applications

Requirements: R2. Dependencies: P03a. Read plan.md common execution rules and referenced source/contract slices. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 18k tokens (stop/split if exceeded).

## Exact owned paths

- `internal/modules/commerce/service/deal.go`
- `internal/modules/commerce/service/offer.go`
- `internal/modules/commerce/service/fulfilment.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/service/commercial_access_test.go`
- `internal/modules/commerce/service/order_lines_test.go`
- `internal/modules/commerce/service/terms_colors_test.go` — approved valid-principal fixture adjustment only; do not change tested color behavior.
- `internal/modules/commerce/service/offer_colors_test.go`
- `internal/modules/commerce/service/ship_delivery_test.go`
- `internal/modules/insurance/service/insurance.go`
- `internal/modules/insurance/service/commercial_access_test.go`
- `internal/modules/financing/service/application.go`
- `internal/modules/financing/service/commercial_access_test.go`
- `internal/app/app.go`
- `internal/app/workflow_ports_test.go`

## Behavior, interfaces and data flow

Gate new RFQ/order/offer creation/publication, new reservation/allocation outside accepted obligations, and seller-side new financing/insurance application with P01 helper. Do not gate insurer's authorized reviews, existing evidence, reading or fulfilling previously agreed obligations merely because legacy seller verification is unset. Add userType/verification to commerce Directory.Company only if needed to validate dealer/buyer-distributor/supplier roles in later reservation packet; adapter reads identity exported Profile, never tables. Do not invent general dealer/distributor exclusivity for existing retail/B2B actions. Distinguish new shipping commitment from execution of already accepted order; preserve old servicing. Record concrete action matrix in result.

## Acceptance and affected units

App seam unit: Company userType/verification mapping preserves provider kinds and has no implicit commercial bypass. Add/use a meaningful pure fake-port case named `TestWorkflowPort03` in the owned app test file. Use a small app-local interface around a module's exported service where required, never import its internal layer or start DB. Run this affected unit once: `bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort03($|_)'`. This compiles app wiring while running only the new pure case; existing OpenAPI test uses e2e.DB and must not run. If no meaningful pure seam can be tested without widening source, report that exact limitation instead of substituting a build/integration command or claiming no-tests as evidence.

Unverified seller cannot create new B2B/insurance/finance requests via direct service invocation. Accepted historic orders remain readable/serviceable; denied operations write nothing. Participant and existing permissions remain enforced.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/commerce/service ./internal/modules/insurance/service ./internal/modules/financing/service`. No other development verification or runtime actions.

Stop: Stop on any new product rule, ownership expansion or context ceiling; report exact missing decision/path. No unrelated refactoring.

