# P21a3 receipt provenance preparation accepted

Application hub consumes /root/purchase_receipt_provenance read-only preparation.
The existing representation is sufficient: MaterializationKind quantity-receipt,
MaterializationRef = a preassigned persisted ShipmentLine.ID, and positive unit
ordinal 1..quantity; ShipmentLine.ReceiptBatchID links the physical inventory
receipt. Migration 041 has the unique (kind, ref, ordinal) key. There is no SQL
foreign key for the generic reference: the single service transaction must
persist both sides and validate the accepted order/source/lot relationship.

The existing retired-claim search in materializeOrderClaims needs a predicate
restricting reuse to vin-allocation materializations. A quantity receipt remains
consumed even though it has no current VIN allocation association. This is the
original accepted capacity invariant, not a new restriction on future booking.

Inventory Identify already changes batch identification counts and creates VIN
units without a commerce callback. P21a3 preserves the durable batch link and
does not consume again. No new direct per-VIN-to-claim-ordinal association or
Identify callback is required. Evidence must not claim real Identify integration
was executed: only affected service units are authorized, no database/runtime.

Preparation has made no source edits. Final a1/a2 handoffs and separate writer
grant remain required. Original exact five-path ownership is unchanged.
