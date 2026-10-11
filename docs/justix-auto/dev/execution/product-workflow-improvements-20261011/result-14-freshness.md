# P14 freshness seam — DONE

`GET /api/v1/retail/deals/{id}` and `POST /api/v1/retail/deals/{id}/demo-scoring`
both use the shared deal response and now expose a present `demoScoring` as
`{id,dealRevision,termsDigest,mode,result,current,completedAt}`. `current` is
mandatory whenever a run exists. A missing run remains `demoScoring: null`; the
latest persisted run remains visible with `current:false` when its covered
signing subject or terms have changed.

The scoring service derives `current` with the same completed-demo mode/result
and fingerprint predicate used by the shared first-own-installment-sign gate.
The fingerprint remains server-derived and continues to exclude deal revision,
invoices, and contract/document activity. No client digest, provider approval,
score policy, threshold, or persistence/schema behavior changed.

Owned changes: `internal/modules/retail/service/scoring.go`,
`internal/modules/retail/service/scoring_test.go`,
`internal/modules/retail/handler/dto.go`, and
`internal/modules/retail/handler/handler.go`; optional authorized pure mapper
coverage was added at `internal/modules/retail/handler/scoring_dto_test.go`.
No additional source seam was required.

Unit evidence (2026-10-11):

- `GOCACHE=/private/tmp/justixauto-p14-freshness-go-cache bash tools/go.sh test -vet=off ./internal/modules/retail/service -run '^TestDemoScoring'` — PASS.
- `GOCACHE=/private/tmp/justixauto-p14-freshness-go-cache bash tools/go.sh test -vet=off ./internal/modules/retail/handler -run '^TestDemoScoringDTOProjectsMandatoryCurrentness$'` — PASS.

Excluded: whole-package reruns, database/runtime/browser checks, build,
typecheck, lint, review/QA, migrations, Git operations, and external actions.
