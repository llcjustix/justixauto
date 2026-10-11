# Application-hub acceptance of reservation/order allocation seam

Accepted proposal: `reservation-order-allocation-seam.md`, SHA-256
`93cfa1c5751ee7d568e587649e3b8d8ebe90ae3413a20553f7a8b292162ee7be`.
This completes interfaces already required by P17b/P18b/P21a and the accepted
future-capacity transfer; no new product restriction or review gate.

P16/P17a must include durable future/materialized claim identity and nullable
converted-order/line linkage in their new, unapplied owner schemas. Avoid a
foreign key to a not-yet-created table; a later owned migration may add that
constraint when both tables exist. Preserve one claim through conversion and
post-conversion VIN allocation; ordinary orders remain unchanged.

The proposal's exact P18b and P21a ownership additions are accepted. Existing
allocation/cancellation and quantity-receipt paths must conserve transferred
claims. A converted reservation no longer releases order-owned claims; physical
rejection/return never reopens materialized future capacity. Partial/all-future
conversion is permitted; consignment handover requires actual VINs.

Run only the actually affected unit families named by the proposal and original
adapters once after relevant changes. No runtime, migration execution or database
concurrency verification. Source writers stay serialized. Consume actual
predecessor schemas/interfaces rather than copying proposal names blindly.
