# P22 physical return adapter completion

Application hub accepts the concrete preparation finding: current commerceStock
Custody maps only consignment purpose and a commerce-order holder. P20 return
requires purpose return and an empty expected holder; the dealer is the actual
source custodian, while supplier ownership stays fixed. Original P22 must reach
this existing inventory operation through the application composition root.

Add exact P22 source ownership of:

- internal/app/app.go
- internal/app/workflow_ports_test.go

Implement a narrow typed return call or explicit supported return mode through
the already owned commerce ports/module and application adapter. Preserve the
existing consignment call/input semantics; do not silently reinterpret ordinary
calls or pass a commerce-order holder for return. Keep module-root imports only,
bound context, actual dealer source, supplier destination, agreement/request
operation reference, actor/time and one VIN to one immutable fact ID. Unknown
mode/input must fail rather than select an unintended movement.

Existing OwnsWarehouse validates supplier-owned destination on request; P20
rechecks current dealer placement and destination capacity at confirmation.
No new warehouse lookup endpoint, inventory source or schema is needed for this
adapter. Final b1/b2/b3 results and the separate source grant remain required.

Add meaningful pure TestWorkflowPort22 cases for complete return input/context,
zero holder, exact fact response and propagated failure. Run once after changes:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort22'`

If the existing consignment adapter implementation also changes, run its exact
TestWorkflowPort21 case once for that relevant change; do not repeat unaffected
retail or service families. Keep P22's narrowed service selector. Include both
added paths, actual final typed API and evidence in result-22.md. No other check,
Git/runtime/SQL action or review/QA is authorized.
