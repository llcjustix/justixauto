# P21a3 durable receipt provenance dispatch

Application hub accepts a fresh explicit gpt-6-sol/high context with a 20k
ceiling for packet-21a3.md. This bounded risk escalation addresses existing
claim reuse, mixed allocated/quantity fulfilment and rollback of an ambient
receipt transaction. Earlier P16/P17 domain jobs required partial continuations;
the five original source paths and the accepted capacity invariant stay fixed.

Read-only preparation may inspect stable reservation_allocation.go helpers,
repository/supply.go, supply claim model and migration 041, ShipmentLine model,
and the existing inventory Identify contract. Do not read files a1/a2 are
actively writing. Final a2 physical gating/receipt entry and a1 repository/store
handoffs plus a separate sole-writer grant precede source implementation.

Preserve final a2 mode/money gates. An accepted purchase receipt consumes held
future claims once with stable batch/line and positive per-unit ordinal. A
materialized quantity-receipt claim cannot be reused as an unassociated retired
VIN claim; retirement/reassociation is only for actual previous VIN allocation.
Later Identify changes physical identity without a second commerce consumption.
No inventory source, migration, financial ledger or new product policy is added.

Write result-21a3.md with exact provenance/uniqueness, locking, quantity progress,
failure rollback and affected unit evidence. Only the packet selector and any
actually changed a2 family may run once. No SQL/runtime, broad checks, Git writes,
automatic review/QA or browser. Stop with a concrete path/field need if the
existing representation cannot preserve the original invariant.
