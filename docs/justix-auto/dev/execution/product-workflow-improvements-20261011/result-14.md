# P14 result — DONE

Implemented persisted, explicitly demonstrative scoring for new own-installment
signing. `POST /retail/deals/{id}/demo-scoring` requires `retail.deals.manage`
and the deal `If-Match`; it records a new completed demo run and returns the
fresh full deal envelope. `GET /retail/deals/{id}` and the POST response expose
`demoScoring` as `{id,dealRevision,termsDigest,mode,result,completedAt}`.

The fingerprint is SHA-256 over the covered signing subject/terms only:
customer ID, saved vehicle snapshot, price minor units/currency, payment scheme,
and the generated installment draft. It deliberately excludes invoice,
document, event, and other incidental activity. A run stores the revision at
completion for audit, while currentness is determined by its completed demo mode,
demonstrative completion result, and matching fingerprint; an actual covered
change therefore requires a rerun.

Wire clarification for P13/P15: the run request has no body (an empty request
body is accepted) and requires `retail.deals.manage` plus the deal `If-Match`.
The fixed strings are `mode:"demo"` and `result:"completed"`. Both POST and
GET deal detail return the latest persisted run in `demoScoring`; POST returns
the newly created run after its deal revision increment. Detail does **not**
compare that run's digest to current covered terms and therefore returns an old
run unchanged when it is stale. The server performs that comparison only at the
shared first-sign gate. There is no DTO `current`/`stale` indicator, and the UI
must not calculate an undocumented digest from client state. This is the exact
missing UI freshness seam; P13/P15 should rely on the signing response until a
bounded server DTO/status addition is authorized.

`Contract.signDraft` is the shared first-sign path for draft signing and legacy
`contract-records`. It requires a current demo run only for the first
own-installment signature. Cash is bypassed; amendments and already-signed
historical servicing bypass it as well. No insurer port, insurance application,
coverage, provider result, score value, decline threshold, or override policy
was added.

Added migration `000040_retail_demo_scoring` and typed `Store.Scoring()` with
the production adapter. Approved shared fixtures now preserve transactional
rollback and seed only the existing first-sign finalization case with a current
run. New service units cover missing/current/stale/outsider scoring, fingerprint
invalidation, cash/amendment exclusion, and rollback.

Evidence: `GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash
tools/go.sh test -vet=off ./internal/modules/retail/service` passed. The first
run exposed a cash-sign test assumption in pre-existing finalization behavior;
the repaired test checks the P14 cash bypass directly, then the same affected
suite passed. No database/runtime, build, lint, typecheck, browser, review, QA,
Git, or deployment action was performed.
