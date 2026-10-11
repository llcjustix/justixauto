# P16 foundation continuation for reservation consumers

The hub consumed result-16.md and its declared schema interface before P17.
The report remains historical, but these concrete foundation gaps need repair:

1. Migration 041 forbids materialized claims without order_id. Accepted P17b
   explicitly assigns VINs before conversion, so materialized reservation-owned
   claims must be valid without inventing an order. Preserve valid owner linkage
   and the held/materialized/released capacity invariant.
2. One claim is one unit, but (materialization_kind, materialization_ref) is
   unique. A quantity receipt can materialize multiple units. Add a durable
   per-reference unit ordinal/identity, supporting the same receipt reference
   for distinct claims and rejecting duplicate application of the same unit.
   Preserve immutable provenance; VIN allocation can use ordinal 1. Use a typed
   model field, repository mapping and explicit uniqueness in the still-unapplied
   new migration. Current/replacement allocation history remains P17b/P18b work.
3. SQL total_quantity is integer (signed 32-bit), whereas jsonx.Quantity and Go
   int on this platform allow larger values. Service validation must reject
   values above the actual storage range before persistence. No arbitrary
   business quantity limit is introduced.
4. An already committed lot cannot silently redefine model/specification/colors
   beneath accepted claims. Reject such identity changes while held or permanently
   materialized quantity is nonzero; quantity increases, uncommitted reductions
   and expected-arrival updates retain current semantics. This preserves accepted
   source identity, not a new commercial restriction.

Exact owned paths for this continuation: commerce/model/supply.go,
commerce/service/supply.go, commerce/service/supply_test.go,
commerce/repository/supply.go (all below internal/modules/), and
migrations/000041_commerce_future_supply.{up,down}.sql. No other source or
canonical edits, no Git/runtime action. Fresh bounded worker context (12k),
escalated to gpt-6-sol/high under the workflow's explicit risk-escalation rule:
the original packet had repeated fixture failures and a consumer-blocking
schema mismatch. This is implementation completion, not an automatic QA gate.

Add meaningful service units for storage overflow and committed-identity edits;
run affected TestSupply cases once after changes with pinned Go -vet=off and
approved GOCACHE. No SQL execution, repository integration, build/review/QA or
test that merely string-matches DDL. Record static schema changes and unit-only
limits in result-16-foundation.md, then final writer release. P17a waits.
