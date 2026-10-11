# P18a1 dispatch snapshot

Before-copy of canonical checkpoint.md: SHA-256
`44bc2a7da230e511e9ec9fcd0a0760cdc330120620b21f91af2ad7ee3e8ac53c`.
The recoverable bytes are in this directory's checkpoint.md.

P17b final handoff: result-17b-completion.md. Its source writer is released.
Milestone HEAD remains ab088d97054fbabac65caae1f3f8bfe7f71fbe74; the source
handoff includes the preserved uncommitted P16/P17 work. Migration 043 has no
file in the checkout; no migration or database inspection/execution occurred.

Source pins at dispatch:

| Path | SHA-256 |
| --- | --- |
| internal/modules/commerce/service/ports.go | 344571dfe85450d1dbde22cbc8b712d6a2bdeac38e6357e52e4462c4c455a524 |
| internal/modules/commerce/service/reservation.go | fc4b1c21d670252f5f42e5d046b3985d45aaab4c8c77aa37e59577cddd67aee5 |
| internal/modules/commerce/service/reservation_allocation.go | 8782d273174c56e7c925d580705818b89a45b981ba5d51f6069a08c1529e028b |
| internal/modules/commerce/model/reservation.go | 8b9586c44f0035080271da3b56d7d2307d0ac36f553d2027bccbcb093f7862c8 |
| internal/modules/commerce/model/supply.go | 716467794e1f651d1ee0e830eac24a2658c8279a5df6952805689eba15edf8a0 |
| internal/modules/commerce/repository/reservation.go | fdfad07268c3a5483499b3b4e1dba8d9dd7094e168064a458a2b9e1887a9f356 |
| internal/modules/commerce/repository/supply.go | 9181c1723b69461cb5c346cffe40230ca64c70ce8116d440211aef49a840a49b |
| migrations/000041_commerce_future_supply.up.sql | 212556d6a9d811c7f49f858e5e61505da06ccc093393a496a94c3ce29aa2ebb7 |
| migrations/000042_commerce_reservations.up.sql | 49d455989c30b0c88cb329997267a1542093e96dd3ec36693baf1e3d962706ce |
