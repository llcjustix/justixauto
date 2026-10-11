# P17b persistent allocation ownership

P16 final handoff is result-16-foundation.md. It supplies stable per-unit claims
and immutable first materialization provenance; current and replacement VIN
allocation history remains P17b/P18b, as already accepted in the reservation
allocation seam. One immutable initial reference alone is not full association
history after a physical rejection/reassignment.

At P17b dispatch, consume actual P17a schema and add a typed, persisted
claim-to-allocation association/history if missing. Reuse the same claim after
legitimate reassignment; keep prior physical association facts and never consume
future quantity again. A durable bridge owned by commerce is acceptable; no
browser-provided source-claim identity or inferred same-model matching.

Exact additive P17b ownership, only if needed for this existing requirement:
internal/modules/commerce/model/reservation.go,
internal/modules/commerce/model/supply.go,
migrations/000041_commerce_future_supply.up.sql,
migrations/000041_commerce_future_supply.down.sql,
migrations/000042_commerce_reservations.up.sql,
migrations/000042_commerce_reservations.down.sql.
These are new unapplied migrations. Existing packet already owns both commerce
repositories and service ports/allocation code. No other schema/table/source
ownership, product changes or additional verification. P18b consumes the actual
resulting helper; quantity receipt remains P21a with stable reference ordinals.
