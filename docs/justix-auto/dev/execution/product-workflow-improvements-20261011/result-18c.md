# P18c result — automatic reservation expiry runner

Status: DONE. Sole source-writer lease RELEASED after this report. The work used
the supplied `feature/company-commerce-workflows-20261011` checkout and
preserved existing dirty predecessor changes. `bash tools/check-git.sh` passed
before edits. No Git write, runtime/database action, deployment, migration,
browser, integration, build, lint, typecheck, vet/race or review/QA loop ran.

## Delivered runtime shape

- `app.New` remains compatible and starts no goroutine. `app.NewRuntime` retains
  the same commerce module instance used by HTTP routes and exposes an explicit
  `Runtime.StartReservationJobs` hook used only by `cmd/api`.
- The job calls `commerce.Module.ExpireDue(ctx, limit, cursor)`, which exposes
  P18b's existing bounded service API without a public endpoint. The module
  adapter now also delegates `ReservationPayments`, the predecessor interface
  member required for commerce to compile.
- Each process runs one serial loop: an initial pass then a 60-second ticker.
  Every pass has a 10-second context and limit 100, uses stable
  `commerce-reservation-expiry` `database.RunOnce`, skips `ErrLocked`, logs one
  sanitized error classification for other failures, and waits for a future
  tick. The cursor is updated from P18b's pass result before its returned error,
  retaining partial progress. The injected clock is sampled at actual pass
  execution; `ExpireDue` independently obtains that same app clock while it
  selects and rechecks due records, with no stale as-of value introduced.
- `cmd/api` starts the runner after successful composition. Signal cancellation
  reaches the runner immediately, before the existing HTTP drain. Every server
  return path cancels and joins it with `ShutdownTimeout` before the earlier DB
  close defer; a join timeout is reported. This source change does not activate
  a deployed runtime or establish replica/runtime evidence.

For an otherwise idle healthy process, polling contributes up to 60 seconds of
delay after the persisted business deadline. Backlog, lock contention, failures
and downtime can add delay; no strict expiry-time or exactly-once guarantee is
claimed.

## Unit evidence

Ran once after the final relevant source change:

```text
bash tools/go.sh test -vet=off ./internal/app -run TestReservationJob
ok   justixauto/internal/app  1.091s
```

Fake-backed units cover the initial bounded pass/current execution-clock sample,
advisory-lock skip, transient-error next-tick retry with cursor retention,
per-pass timeout without busy-looping, serial local passes, cancellation/join
and ticker stop. They do not execute PostgreSQL advisory locks, signal delivery,
HTTP server failure, or deployed multi-replica behavior.

## Final source SHA-256

```text
5688630d24da5bc459b5513d860dc193633f6f426913709c91e796332cca3ecd  internal/app/app.go
80a7ccdacfd79fe1f12d51f529c9c5f1dd9bf273fb36d83b8400131cc4426992  internal/app/reservation_jobs.go
8d181b05dc17da62a76b7bca7815accf2e0ad2b5b9cdae1828c92cbb83531eac  internal/app/reservation_jobs_test.go
85c99910ab9d5731b11b898bde8d5a436e86e37a7124b1f1dc46f21e17cf04b8  internal/modules/commerce/module.go
2c880a67608e5af949b6d5a8d6f8f92fd079516382215a42317021e491e79945  cmd/api/main.go
```
