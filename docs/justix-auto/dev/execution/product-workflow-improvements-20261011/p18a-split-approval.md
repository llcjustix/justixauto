# Application acceptance of bounded P18a execution

Accepted proposal SHA-256
`270dfb178ef41b684e72a20b7ff0e454ac83d56f329f9c10b23d3e0caa487491`.
P18a1 packet SHA-256
`73a3b39d22100fd504a71b092e7ce5f313bf528882febc8aabd890fcca515312`;
P18a2 packet SHA-256
`1a6376f8dd79e4b68217f595ed354bfb8fc3d2e3f8b03de15a91a93ffb4d34ac`.

Both exact additive paths are accepted for P18a2:
internal/modules/commerce/service/invoice.go and
internal/modules/commerce/handler/reservation_payment_test.go.
One accepted advance affects the existing issued invoice once without creating
another payment, changing the face total/schedule, or crediting historical void
invoices. Cancellation protection also sees order credit before invoice issuance.
Use actual predecessor schemas, lock order and helpers at dispatch; no source
mutation until a separate writer grant. Source writers remain sequential.
No new product policy, DB/runtime operation or additional verification gate.
