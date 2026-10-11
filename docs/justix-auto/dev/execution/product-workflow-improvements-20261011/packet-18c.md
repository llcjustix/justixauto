# P18c — Automatic reservation expiry runner

Requirements R3. Dependencies P18b. Application hub /root; base/branch in plan. Exclusive source-writer lock; no runtime resources. Read root/internal rules, P18b result, cmd/api/main.go, internal/app/app.go and internal/pkg/database/lock.go. Context ceiling 12k tokens.

## Exact owned paths

- `internal/app/app.go`
- `internal/app/reservation_jobs.go`
- `internal/app/reservation_jobs_test.go`
- `internal/modules/commerce/module.go`
- `cmd/api/main.go`

## Behavior and seam

P18b expiry must be invoked automatically. Expose minimal commerce expiry service from module root; retain app.New's existing compatibility for tests, add explicit opt-in runtime start/stop hook at cmd/api composition. Timer starts with app lifetime and exits on cancellation; no hidden goroutine in unit app construction. Use existing database.RunOnce with a stable reservation-expiry job name for at-most-one concurrent replica scan. Service scans bounded batches, rechecks state and money while locking; another replica's lock means skip this tick. Existing accepted/pending advances never expire; mandatory-advance deadline expires only when payment was never submitted. No new external scheduler/dependency, environment variable, manifest or deployment. Engineering tick interval controls polling latency, never extends or replaces the agreed business deadline; document expected maximum idle polling delay in result.

Read `expiry-runtime-decision.md`: application and DevOps hubs jointly accepted the shape. Defaults: 60-second poll, at most 100 candidates/pass, 10-second pass context; one bounded initial pass. Same commerce instance as HTTP, no duplicated module construction or public expiry endpoint. Stable RunOnce name `commerce-reservation-expiry`; callback does not inherit advisory-lock transaction and lock loss can overlap service work, so P18b row locks/rechecks/idempotency remain mandatory. One serial loop, current clock at pass execution, ErrLocked skips; other errors produce one sanitized error and retry next tick. Cancel on SIGINT/SIGTERM immediately (not after HTTP drain delay) and all server-return paths, stop ticker and bounded join before DB close, report timeout. No exactly-once/hard expiry-time promise.

Expose injected tick/clock/run-once seams to unit-test without sleep or database. Add fake cases for initial bounded pass, per-pass timeout, no overlapping local passes, lock skip, next-tick error retry, current-clock use and cancel/join on server error. No live activation, deployment or migration execution.

## Acceptance / units

Fake tick invokes expiry; lock held elsewhere skips; error does not busy-loop; cancellation stops; default test app construction does not start timer; fake expiry receives current time and bounded batch. P18b units own money/deadline state truth. Run once: `bash tools/go.sh test -vet=off ./internal/app -run TestReservationJob` (only this new pure unit selector, not app OpenAPI/integration suite).

Stop if scheduler requires new infrastructure, dependency or operational guarantee. Source delivery does not establish production timer activation or multi-replica concurrency; those remain unexecuted runtime evidence.
