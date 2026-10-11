# P18c reservation expiry runtime contract

Date: 2026-10-11. Infrastructure authority: `/root/expiry_runtime_authority`
(`devops_orchestrator`). Assigned consultation only; application authority is
`/root`. This execution artifact records infrastructure acceptance of the shape
below; application-hub agreement is required before dispatch. It is not a
canonical ADR, rollout authorization or readiness evidence.

## Allowed operational shape

Use the existing API process, commerce service and PostgreSQL `database.RunOnce`.
No additional service, dependency, environment variable, manifest, migration
execution, cluster operation or deployment is part of this packet. Preserve
`app.New` compatibility and its absence of background work. Expose an explicit
runtime hook from application composition, using the same commerce module
instance as HTTP routes, and start it only from `cmd/api` after construction
succeeds. Do not duplicate module construction or add a public expiry endpoint.

Proposed named code defaults for joint acceptance: **60 seconds between polls,
at most 100 candidates per pass, 10-second context timeout per pass**. One initial
bounded pass on explicit runtime start is allowed and recommended, followed by
periodic passes. No unbounded drain loop or immediate retry. Obtain current time
from the injected clock when a pass executes, rather than using a queued tick's
timestamp. Polling settings never supply or modify a reservation deadline.

For an idle healthy runner, the scheduling component of expiry delay is up to
60 seconds. Processing, backlog, contention, failures and process downtime can
add delay; this is not a maximum business release time or an availability SLA.
Persisted due reservations are reconsidered after restart, rather than relying
on remembered timer events. The application hub owns deadline/payment rules.

## Concurrency and transaction boundary

Run one synchronous loop per process, never one goroutine per tick. Wrap the
entire bounded service pass in `database.RunOnce` with the stable shared name
`commerce-reservation-expiry`; all replicas and future compatible versions must
use that same name. Treat `errors.Is(err, database.ErrLocked)` as an ordinary
skip, retrying only at the next scheduled opportunity.

Existing `internal/pkg/database/lock.go` holds its advisory lock in a transaction
but calls `fn(ctx)` without `database.WithTx`. Consequently the service pass is
**not** part of that lock transaction. The lock holds one pool connection while
service transactions use another; the existing pool limit is 20, unchanged here.
Do not assume that returning an error from `RunOnce` rolls back completed service
transactions, or alter the shared database helper within this packet.

The advisory lock prevents concurrent passes while its holding transaction
remains alive. A lost/cancelled lock connection can release it before separate
service work has fully stopped. It is not an exactly-once guarantee. P18b must
keep transactional state/payment rechecks, row serialization against payment
submission/acceptance and extension, and idempotent release of quantity/VIN.
Use existing service ports/ambient transaction binding for cross-module stock
changes. The timer neither inspects payment tables nor decides release policy.
Protected or failed candidates must not permanently starve other due candidates;
bounded candidate selection/progress belongs to the application service contract.

## Lifecycle and errors

Derive runner and per-pass contexts from API lifetime; all service/database work
must receive them. On SIGINT/SIGTERM, cancel immediately without waiting for the
existing HTTP drain sleep. Also cancel on every server-return path, including
listener failure. Stop the ticker, wait for the loop before normal database
closure, and bound the join with the existing shutdown timeout/budget. Report a
join timeout; do not hang process cleanup or claim a completed scan. Preserve
existing HTTP drain behavior and database/telemetry cleanup ordering.

Cancellation due to shutdown is normal. Other scan failures, including pass
timeouts, produce one structured, sanitized job-error log and retry on a later
tick. No hot loop, silent permanent stop, API crash or readiness redefinition.
Log the job name and safe outcome/error classification; do not log payment
documents, customer data, secrets or database credentials. Any already committed
candidate work is safe to encounter again under P18b's idempotency contract.

## Scope of evidence

Read-only inspection: root/internal/deploy/dev rules, current R3, P18b/P18c,
`cmd/api/main.go`, `internal/app/app.go`, commerce composition/store ports and
database lock/pool/ambient-transaction helpers. No source edits, Git writes,
tests, runtime actions or deployment checks were performed for this consultation.
P18c development verification remains its affected pure unit selector once;
there is no added review/QA loop. Real replica exclusion, shutdown under database
failure, production activation and throughput remain unexecuted runtime evidence.
