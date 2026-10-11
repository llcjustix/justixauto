# P18b1 result — reservation lifecycle

Status: SOURCE COMPLETE. Writer lease RELEASED after this report. Work used the existing `feature/company-commerce-workflows-20261011` checkout at supplied milestone HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, preserving predecessor dirty source. `bash tools/check-git.sh` passed before source edits. No Git write or runtime/database action was made.

## Lifecycle and accounting contract

- Active reservations can end as durable `expired` or `cancelled`, with server `releasedAt` and `releaseReason`. Supplier `Extend` persists a later explicit effective `expiresAt` and, only when expressly requested, a later effective `advanceDueAt`; it records `extendedBy`/`extendedAt`, expected-revision CAS and a versioned event. The original accepted `CurrentTerms`, digest, revision, price and advance remain unchanged, so existing evidence provenance remains valid. An expiry-only extension does not advance an advance deadline.
- Automatic expiry uses the existing reservation `FOR UPDATE` row lock, then `ReservationPayment.MoneyLocked` in the same commerce transaction. Free reservations expire at their effective expiry. Mandatory-advance reservations expire when either their effective agreed advance deadline or final expiry is due **only if no evidence was ever submitted**. Pending, accepted, rejected-history, refunded, credited and converted sources are protected. Selection is a hint; row status, order link, effective deadlines and money are rechecked after the lock.
- `Cancel` requires a buyer or supplier with `commerce.trade`, reservation If-Match and reason. It calls the P18a locked money view and `ReleaseEligibleLocked`; pending evidence must be decided and accepted advance must have a full accountant-confirmed refund before release. A refund request alone does not settle money. The existing `/refund-requests` and `/refund-confirmations` routes own that evidence flow. No money movement is implied.
- Final expiry/cancellation locks sorted supply lots and claims after the reservation, marks only its unmaterialized held claims `released`, retires current reservation-holder physical associations without changing materialized claim provenance/counters, and releases initial plus allocated VIN holds through transaction-bound `ReleaseReservation`. State, claim, association, holder and event writes commit or roll back together. Converted reservations cannot release order-owned claims or VINs. Automatic events use the reserved zero UUID system actor and explicit `automatic` event detail; manual events use the caller.

## P18c bounded pass handoff

`Reservation.ExpireDue(ctx context.Context, limit int, cursor string) (ReservationExpiryPass, error)` accepts 1–100 candidates. `ReservationExpiryPass` has `NextCursor`, `Selected`, `Expired`, `Protected`, `Failed`. The opaque base64url JSON cursor encodes the candidate's UTC `(deadline, reservation ID)` keyset; repository `DueCandidates` selects active unconverted due rows ordered by that pair with a bounded `LIMIT`. A later pass feeds `NextCursor`; an empty cursor starts at the beginning. When the selected page has fewer than `limit` rows, `NextCursor` is empty, so a later tick wraps and revisits protected/failed rows. An exactly full last page returns its cursor; the next pass sees the end and resets it.

Each candidate runs in its own `Store.InTx`. Protected and failed attempts advance the cursor; earlier successes remain committed if a later attempt errors. The returned error is the first attempt error (joined with context cancellation when relevant), with completed attempt counts and cursor preserved. Selection errors return without attempts. Context is checked between attempts and after the row lock/event; an interrupted transaction is never counted expired. P18c owns the 100-candidate/10-second/60-second runtime loop and cursor retention. The pass makes no advisory-lock or exactly-once assumption.

## HTTP and read contract

Two `commerce.trade` mutations were added under `/api/v1/commerce`, both returning `httpx.Data` 200 with decimal revision and ETag:

- `POST /reservations/:id/extensions`: If-Match plus `{expiresAt, advanceDueAt?, reason}`. Supplier only; explicit later effective dates, with advance due no later than expiry.
- `POST /reservations/:id/cancellations`: If-Match plus `{reason}`. Buyer or supplier; payment/refund guard above.

Reservation GET/list and mutation DTOs retain accepted snapshot vocabulary and additionally expose effective `expiresAt`, `advanceDueAt`, `extendedBy`, `extendedAt`, `releasedAt`, `releaseReason` and `cancellationState`. Under a locked money read, active state is `ready`, `awaitingEvidenceDecision` or `awaitingRefund`; terminal/converted states are distinct. Allowed actions expose supplier `extend`, eligible `cancel` and unsettled `requestRefund` as appropriate. No public expiry/release endpoint was added.

## Affected unit evidence and limits

The final relevant lifecycle change passed:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^(TestReservationLifecycle|TestReservationExpiry|TestReservationExtension|TestReservationCancellation)'`

Result: `ok justixauto/internal/modules/commerce/service 0.665s`. The same selector passed after the preceding relevant implementation edit. Existing affected reservation service/handler selectors passed once after the read and route changes:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service ./internal/modules/commerce/handler -run '^(TestReservationTransitions|TestReservationHTTP)'`

Results: service `0.339s`, handler `0.565s`. The new fake-backed units exercise exact due boundaries, rejected and newly submitted history, stale selection after deadline change, accepted/refund/pending protection, supplier extension and stale revision, claim/holder/association rollback, cursor advance past protected/failed rows and cancellation during an attempt. These are units; SQL migration, live DB row-lock races, real cross-module rollback and scheduler runtime remain outside the unit-only development evidence.

## Final source SHA-256

```text
b91ea655c01c023a34c86ba09528b78c4db2aecb929eacf024f9b394adf40216  internal/modules/commerce/model/reservation.go
f04ec31de6556c7d7376f2c58da64b3c9918d44d40bf820e3b9bb46b0311315e  migrations/000042_commerce_reservations.up.sql
20b411ac3248e3a215e24572a431d863be8dfeab176c20d2cda32ea0322c7b09  internal/modules/commerce/repository/reservation.go
7158dad8f265cb83fe8708a1c74920f29bfa12b75e813727ed4beb4ec08823c6  internal/modules/commerce/service/ports.go
e03e78e08eb049ffb1786d895895a030ed9fb63c965419e87af8b75d422aca3d  internal/modules/commerce/service/reservation.go
a2ace88b592d395f40f1eb448dcb25975a9e7b466ac48041ecb12f583475bbd5  internal/modules/commerce/service/reservation_lifecycle.go
0b1de44d230f3154db0350438ea10f6f4c325f16d7a3f6b5263cff70e8da1ccb  internal/modules/commerce/service/reservation_lifecycle_test.go
64782726a07bf4eee7b7343b0dbeeaf0cdcb49297ed20e9498b82413b9997572  internal/modules/commerce/handler/reservation.go
98da498c461de897dc424868803e7df11a7148470d2d90ed4a9cb64889d1f9ae  internal/modules/commerce/handler/handler.go
```
