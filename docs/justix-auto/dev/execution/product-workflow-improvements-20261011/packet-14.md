# P14 — Demo scoring before own-installment signing

Requirements: R12. Dependencies: P11. Read plan.md common rules, exact named source and relevant contract sections. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 20k tokens; split before exceeding.

## Exact owned paths

- `internal/modules/retail/model/scoring.go`
- `internal/modules/retail/service/scoring.go`
- `internal/modules/retail/service/scoring_test.go`
- `internal/modules/retail/service/contract.go`
- `internal/modules/retail/service/ports.go`
- `internal/modules/retail/repository/scoring.go`
- `internal/modules/retail/repository/store.go`
- `internal/modules/retail/handler/scoring.go`
- `internal/modules/retail/handler/handler.go`
- `internal/modules/retail/handler/dto.go`
- `internal/modules/retail/module.go`
- `migrations/000040_retail_demo_scoring.up.sql`
- `migrations/000040_retail_demo_scoring.down.sql`

## Behavior, interfaces and data flow

Persist an explicitly demo scoring run for own-installment sale before first new signing: ID, deal revision/terms digest, mode demo, demonstrative result, actor/time. Expose POST deal/demo-scoring and GET current run via deal DTO; authorized sale staff, tenant and If-Match. Deterministic technical demonstration, no real credit assessment or fabricated provider result; no arbitrary score thresholds/decline policy. Signing requires current completed demo run, not insurer approval or policy coverage. If terms/subject changes invalidate referenced digest, rerun required. Legacy already-signed contracts remain serviceable and are not retroactively blocked. Does not create insurance application or approval.

## Acceptance and affected units

Unsigned own-installment cannot sign before current demo completion; current completion permits contract step only; changed terms invalidates; cash/legacy signed not blocked; outsider/stale rejected; demonstration result does not change Insurance.Approved.

Run once after relevant change: `bash tools/go.sh test -vet=off ./internal/modules/retail/service`. No other development verification/runtime actions.

Stop: Only demo completion is the approved sequencing gate. Do not introduce business decline/override/score-threshold policy; if requested result variants need such policy, return that narrow gap.

