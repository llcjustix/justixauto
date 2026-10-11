# P18b1 bounded lifecycle persistence scope

The read-only worker found that current ReservationStatus and migration 042
allow only requested/offered/active. Original P18b1 acceptance requires durable
expired/cancelled states and a server release reason; this is missing predecessor
persistence, not a new product rule.

The application hub authorizes these exact additional P18b1 paths after its
separate source-writer grant:

- internal/modules/commerce/model/reservation.go
- migrations/000042_commerce_reservations.up.sql

Scope is the terminal status CHECK/constants and minimal lifecycle fields
needed for explicit effective deadlines, release reason/time and their DTOs.
The new migration has not been applied by this work; no runtime/schema operation
is authorized. The down migration drops the table and needs no change merely
for additive columns. Existing repository/service/DTO paths are already owned.

Preserve immutable requested/offered/accepted agreement snapshots and their
digests. A supplier extension must not rewrite the price, advance amount,
accepted payment provenance or original accepted digest. Record the explicit
effective expiry (and explicit effective advance deadline if extended), actor,
time and revision through durable fields/events. Do not invent a duration,
grace period or silently extend an advance deadline from an expiry change.
Money-aware release reads the effective saved deadlines and the locked ledger.

Paid cancellation intent/undecided/refund states may come from the existing
authoritative ledger; no duplicate payment/refund storage is authorized.
P18b2 retains conversion/order ownership. Unit-only evidence must include
preservation of accepted terms/payment association across deadline extension.
