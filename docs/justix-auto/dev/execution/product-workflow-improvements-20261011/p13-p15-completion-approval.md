# P13/P15 implementation continuation

The initial result-13-15.md is retained as historical implementation evidence,
not completion acceptance. While consuming its API handoff, the coordinator
found an explicit mismatch: commerce request bodies use bindingIds (P12a2),
whereas the new UI sends the response property fileIds. Targeted context showed
draft edits use POST although both owners expose PATCH, no existing-template
editing, and absent contract-query invalidation. These are unfinished original
P13 requirements, not a new review/QA gate or product change.

The two current tests assert only that exported components are functions. They
do not establish the behavioral acceptance named in packets 13/15. Replace them
with affected interaction units as part of completing the implementation; do
not run unrelated or already-green backend selections.

Continue within exactly the original P13/P15 union, using actual handler/DTO
contracts and shared form behavior as context. Signed snapshots must never fall
back to a mutable live price. Existing drafts need initial values and intentional
preserve/clear behavior for files/text/template. Show filled preview, version
mismatch, privately loaded documents, and selectable signed-parent amendments.
Request uploads use bindingIds for both owners; response reading remains owner
specific. Revisions and source query invalidation must follow actual API responses.

Fix the observed owner permission wiring: commerce does not currently return
the invented contract/manage-contract actions. Use existing commerce.trade and
actual party visibility. Retail amendment controls must stay available for
existing signed contracts without requiring a new demo run; first base signing
still uses the mandatory server current flag and service authority. Handle
scoring failure/stale-revision reload visibly and gate its action by the existing
permission. Complete original embedded-insurance behavior and meaningful units.

No new product/legal policy, dependencies, canonical docs or backend changes.
Use separate fresh bounded implementation contexts if needed; a partial report
must name remaining acceptance instead of claiming DONE. Only the actually
affected Realization unit files run once after relevant changes, with explicit
Realization config/runner. No review/QA, lint/typecheck/build/browser/runtime.
