# Automatic reservation expiry — joint authority

Date: 2026-10-11. Application authority: `/root`. Infrastructure authority:
`/root/expiry_runtime_authority` (devops_orchestrator).

Application hub accepts the bounded implementation shape recorded in
[infrastructure decision](expiry-runtime-decision.md) for P18b/P18c.
This authorizes source implementation only; no new infrastructure dependency,
environment mutation, running migration or deployment is authorized here.

- Existing API process, explicit start/stop, one synchronous loop.
- Defaults: initial bounded pass, then 60-second cadence; at most 100 candidates
  per pass and a 10-second pass timeout.
- Existing `database.RunOnce`: lock contention skips; other errors logged once
  for that pass and retried at a later pass. Shutdown cancels/joins before DB close.
- Its callback does not inherit the advisory-lock transaction; lock loss can
  overlap passes. Domain writes therefore use row serialization, fresh money/
  deadline checks and idempotent capacity/VIN release. Do not claim exactly-once
  execution, a strict expiry deadline or rollback of earlier committed candidates.
- Submitted/pending or accepted payment prevents automatic expiry. User explicitly
  approved automatic release after a required-advance deadline with no submitted
  payment. Refund completion remains accountant-confirmed external money movement.
- Unit fakes for cadence, contention, cancellation, transient failure and domain
  races; no live runtime/DB/cluster checks. Deployed activation remains unverified.
