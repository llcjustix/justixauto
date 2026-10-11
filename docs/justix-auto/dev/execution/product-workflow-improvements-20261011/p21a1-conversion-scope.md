# P21a1 atomic consignment authority creation

Application hub accepts the concrete preparation finding: the durable
order-linked consignment agreement must exist after a successful consignment
conversion, including orders with no assigned VIN. Do not defer that agreement
until handover, create it during GET, or add another user acceptance ceremony.

Add exact P21a1 ownership of:

- internal/modules/commerce/service/reservation.go
- internal/modules/commerce/service/reservation_allocation_test.go

The production change is one conditional invocation in Reservation.Convert's
existing Store.InTx after accepted order/claim/holder/credit work and before the
source reservation CAS/events. The bound helper lives in owned consignment.go
and uses the same Store. Persist one unique agreement for the accepted order,
with immutable reservation/order/party/digest/accepted-revision references.
The already accepted order terms remain the settlement and handover authority;
do not introduce independently mutable payment terms or another money ledger.
Any agreement write failure rolls back the complete conversion. Ordinary
purchase conversion and the no/some/all-VIN invariant remain unchanged.

The added test file owns only the existing conversionStore agreement/unit map
and transaction-copy support plus affected conversion assertions/failure cases.
It currently embeds Store and would otherwise panic on the new repository
method. Preserve other allocation/conversion behavior and source helpers.

P21a1 exposes a bound prepared-unit operation. It may prepare existing allocated
VINs at conversion. P21a2 ensures each selected VIN allocated after conversion
is registered before/with actual handover, in that same transaction. A1 does
not need fulfilment.go ownership. No live sale authority exists without actual
successful handover; a prepared row or an allocated VIN alone is insufficient.
No requirement to allocate every order unit before a valid partial handover.

Replace the original a1 unit selector with this affected family selection:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^(TestConsignmentAgreement|TestReservationConversion)'`

Run once after the source change. No SQL/runtime, broad suite, review/QA or
other checks. Final P20 contract and separate writer grant still precede edits.
