# P18b1 — Reservation expiry, extension and cancellation

Proposed split of original P18b, canonical R3. Application hub /root; common authority/base/locks in p18b-split-proposal.md. Depends on actual P17b holder/claim and P18a accounting interfaces supplied at dispatch. Context ceiling 17k. One source-writer lease, serialized before P18b2. No runtime worker or application construction in this packet.

## Exact owned paths

- internal/modules/commerce/service/reservation_lifecycle.go
- internal/modules/commerce/service/reservation_lifecycle_test.go
- internal/modules/commerce/service/reservation.go
- internal/modules/commerce/service/reservation_allocation.go (existing accepted helper/release seam only)
- internal/modules/commerce/service/ports.go
- internal/modules/commerce/repository/reservation.go
- internal/modules/commerce/repository/supply.go
- internal/modules/commerce/handler/reservation.go
- internal/modules/commerce/handler/dto.go

All paths are original P18b or accepted allocation-seam ownership. No schema/payment-ledger/app/inventory edits. Use actual predecessor methods, not speculative names below.

## Behavior and transaction boundaries

1. Implement service-level expiry with injected clock and distributor-approved extension. Exact expiry and mandatory-advance deadline come only from mutually agreed reservation terms; no default date, duration or grace period. Extension is supplier-authorized, expected-revision checked, persisted with actor/time/versioned event and new explicit deadline. Preserve existing party/permission checks and original agreed economic terms.
2. Each candidate mutation owns its own Store.InTx. Acquire the SAME reservation row lock used by payment submission/acceptance, refund, extension and conversion; coordinate deterministic lock ordering with supply/physical-holder helpers. After lock, reread current state/deadlines/version and P18a money view including everSubmitted, pending, accepted, refunded and credited/linked-order facts. Candidate-selection state is only a hint. Do not read other-module tables or trust cached totals.
3. Automatic free expiry releases eligible free/no-money reservations when due. Mandatory-advance auto-release requires the agreed deadline due AND dealer never submitted any payment evidence. Rejected historical evidence is still a submission; never treat it as absent. Pending or accepted advances protect against automatic expiry; do not reinterpret a refunded paid reservation as free expiry. Cancelled/expired/converted rows are no-ops, with no repeat releases/events.
4. Manual cancellation follows P18a: pending evidence must first be decided, accepted advance requires a full refund obligation and accountant-confirmed full refund before final release. Requesting cancellation/refund is not a paid refund. Reuse the predecessor refund workflow and exact money view; no duplicate ledger, fees, bank action, partial refund or automatic evidence rejection. A final cancellation releases eligible remaining future claims plus the correct physical reservation holder in the same transaction as state/event. Release only unmaterialized future claims; materialized counters remain consumed even if VIN becomes free.
5. Converted reservations cannot release order-owned future claims/VINs. Check conversion under the same row lock before any release. Preserve history/read access; cancellation UI actions reflect refund/pending/converted states. Order cancellation belongs to b2.
6. Port failure, claim write failure, state/revision conflict or event failure rolls back all participating quantity/VIN/state changes. Bind context for inventory calls. Idempotency is persisted state/claim transition, not a process-memory check.

## Bounded pass contract for P18c

Provide a bounded service pass and document its actual signature/result. Suggested shape: ExpireDue(ctx, limit, cursor) → {nextCursor, selected, expired, protected, failed}, error. Cursor is an opaque deterministic keyset (deadline + stable ID, or equivalent) over due candidates; preserve it across P18c ticks even when some candidate transactions fail. Repository selection must be bounded; at end-of-range reset for a later pass so previously protected/failed rows are revisited. No unbounded drain or immediate error retry. A cursor is only scan progress, never release authorization.

Advance progress past protected/failed candidates, including an entire first page of them. Return progress for completed attempts on partial pass error/timeout; do not roll back earlier committed candidates. If cancellation interrupts an attempt, never report that row as successfully expired. Check context between candidates and pass it to every repository/port call. P18c supplies the accepted 100 candidate/10s pass workload bounds and 60s poll; these operational numbers never alter business deadlines.

No advisory-lock assumption in service correctness: RunOnce callback does not inherit its lock transaction and loss may overlap another pass. Row locks, money/state/deadline rechecks and idempotent holder/claim release protect each transition. Do not add an HTTP expiry endpoint, goroutine, RunOnce call, startup hook or scheduler here.

## HTTP/data seam

Keep predecessor reservation DTO vocabulary; final worker report pins actual route/method/input/revision semantics for extension/cancellation and next actions. Mutations use reservation If-Match and explicit requested date/reason as existing contract requires. Expose cancellation awaiting decision/refund versus final cancelled distinctly. Existing accounting endpoints stay owned by P18a. Route wiring remains inside owned reservation.go registration helper; if an unowned route-registration edit is concretely needed, report that exact seam to hub.

## Meaningful affected units

Actual service/pass tests with deep-copy transaction fakes, shared row-lock/money-view behavior and failure injection:
- before/at/after free expiry and mandatory deadline; no submission versus pending/accepted/rejected-ever-submitted;
- supplier-only extension, stale revision, stale selected deadline followed by extension, no invented duration;
- stale selection followed by payment submission/acceptance; protected row remains held;
- cancellation request/full-refund pending/confirmed, pending evidence, wrong or incomplete refund; final release once;
- converted/expired/cancelled no duplicate release; materialized claims never reopen;
- repeated/overlapping attempts, stock/claim/event failure rollback;
- first page wholly protected/failed still progresses to later due candidates across passes, end-of-range wraps, cancellation/timeout returns safe progress, prior successful candidate remains committed after later failure.

Run only affected units once, repo root:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^(TestReservationLifecycle|TestReservationExpiry|TestReservationExtension|TestReservationCancellation)'`
Use these names for new cases; add an existing helper-specific selector only when that helper's behavior changes. No broad service/app/DB checks or already-green accounting rerun if accounting source unchanged.

Report result-18b1.md: exact expiry pass/cursor/error contract for P18c, actual lock and money-query seam, HTTP DTO/revisions, release invariants, tests/exclusions. Stop on concrete missing predecessor seam or context ceiling; no new product defaults. No runtime readiness/exactly-once claim.

