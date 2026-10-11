# P14 UI freshness seam completion

Application hub: original R12/P14 plus P15 requires the signing UI to use the
latest current demo run. The completed service rejects stale fingerprints, but
result-14.md confirms GET/POST expose the last run without currentness. The UI
must not derive an undocumented digest. This is unfinished cross-packet
acceptance, not a new score/decline policy or a review/QA loop.

Assign a fresh 10k-context worker, sole source writer, to add mandatory boolean
`current` to the `demoScoring` DTO when a run exists. Retain the latest run for
history even when false; absent run remains absent/null under current API shape.
Both GET deal detail and POST demo-scoring must expose the same server-computed
state. Use the same fingerprint/mode/result predicate as first-sign validation.
Keep revision, actor/time, mode/demo result and covered fingerprint fields
unchanged; no client digest, fake provider approval or new business threshold.

Exact ownership: original packet-14.md paths, its fixture approval if needed,
plus one optional pure new handler unit
`internal/modules/retail/handler/scoring_dto_test.go`. Read narrowly; no need to
edit migration/store schema or unrelated fixtures for a derived response flag.
Report a concrete additional source seam before expanding ownership.

Add meaningful service freshness projection cases for absent/current/stale
covered terms and incidental invoice/document activity; preserve shared first
signature enforcement. Run only affected scoring units once per relevant
change (actual prefix from scoring_test.go), plus new pure handler selector
only if added. No repeat of the whole green retail package, no app/DB/browser/
build/typecheck/lint/review checks, Git or runtime activity.

Write result-14-freshness.md with final DTO semantics and exact unit evidence.
P13/P15 waits for this seam. Release source only after all edits and the final
report; any later change requires a new grant.
