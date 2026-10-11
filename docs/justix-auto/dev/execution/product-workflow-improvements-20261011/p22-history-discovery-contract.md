# P22 durable return history discovery

Application hub's final P22 handoff clarification: original R5/P23 must show
request, reason and decision when an order is reopened, including after its
unit's PendingReturnID clears and the dealer no longer has inventory custody.
Cached request IDs or current vehicle access cannot be the only discovery path.
This closes the original return-history interface within P22's existing paths;
it adds no product policy, source ownership or review/QA step.

Final result-22.md supplies GET
`/api/v1/commerce/consignments/return-requests?orderId=&limit=&offset=`.
It is actual-party-scoped under commerce.read, retains pending/confirmed/rejected
history and supports old-obligation access. P23 must consume all relevant pages
with the existing HTTP list semantics and use exact per-request revisions and
allowedActions. No reliance on a cleared unit pointer or local mutation cache.
P22 included post-decision discovery and tenant cases in its final affected
service selector; its unchanged app adapter units were not repeated.
