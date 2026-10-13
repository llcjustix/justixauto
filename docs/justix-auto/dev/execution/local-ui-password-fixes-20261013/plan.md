# Local UI and password corrections — execution plan

Task: `docs/justix-auto/dev/local-ui-password-fixes-20261013.md`.
Owning hub: application `/root`; scope is the user-approved task, not a new
backlog entry. Status: Packet01 accepted and dispatched by primary; Packet02
includes primary's test-scope amendment in approval.md. No implementation or
verification performed by slicer. Branch prepared by version_control:
`fix/portal-ui-password-20261013`, identical source dependency
`31b018ce1a4b49a2f56148459b4efa0893d345ce`. Normal dev ancestry is
`4fa683ba4708bb71031534260ec5cb8742ce9977` as supplied in the task.

## Artifacts and dispatch

1. [Packet01](packet-01-switcher.md): hide redundant switcher and apply one
   scoped, wrapping shared style. Three source/test owned paths. Portal unit
   suite once. Result: result-01.md.
2. [Packet02](packet-02-password.md): authoritative minimum8 and corresponding
   frontend guidance. Nine source/test owned paths. One selected Go unit run
   and one affected frontend unit run. Result: result-02.md.

Each packet gives exact owned paths, targeted source slices, contract references,
scope rules, requirements, tests, locks, dependencies and stop conditions.
Both fit a 10k target / 14k hard input context. The packets are functionally
independent, with disjoint write ownership; execution is serialized because
the repository permits only one checkout source writer. Order: hub accepts
Packet01 → worker completes/releases source lease → hub accepts/dispatches
Packet02. Version control exclusively owns any Git mutations. No push is
authorized for this task. The slicer did not execute Git commands.

[findings.md](findings.md) is the incremental source map and diagnosis.
[source-hashes.json](source-hashes.json) pins SHA-256 of the inspected inputs,
including source, rules and approved contract paths. Base SHA is supplied by the
hub; filesystem hashes are independently computed. Hashes do not assert that
new canonical docs existed at that code commit.

## Complete acceptance coverage

| Requirement | Implementation/evidence | Limit |
|---|---|---|
| U1: sole Admin, no memberships, no redundant block | Packet01: visibility from exact destination/choice lists; focused portal unit scenario | No layout measurement |
| U2: several admitted destinations retained | Packet01: multi-section case, existing four-variant internal-navigation cases | Shared style source applies to all variants |
| U2: inactive company choices retained | Packet01: one admitted destination still exposes existing choices; existing no-reload, PUT/session-refresh tests | Membership remains a context choice, never a grant |
| U2: fix height/padding/alignment/overflow | Packet01: dedicated item class; natural height, constrained width, wrapping, scoped precedence, section-derived active class | Browser/manual observation remains separate; no CSS-value test claims |
| U2: routes, access, context and memory unchanged | Packet01: existing denial, deep URL search/hash, remembered/revoked section and context sequencing unit cases | No new grants or route policy |
| P1: 8 accepted, 7 rejected | Packet02: authoritative table-driven rune-boundary unit tests | Pure validator evidence, not a live account mutation |
| P1: creation/reset/own change/bootstrap consistent | Packet02: shared validator caller map; all six help texts updated; existing form tests use eight-character payloads | Bootstrap CLI is read-only; no live execution; no new hint-only tests |
| P1: remaining policy preserved | Packet02: max128/129, Unicode and confirmation boundary/error mapping tests; unrelated checks unchanged | No new password policy |
| P2: existing credentials/accounts unchanged | Packet02: unchanged hash functions/parameters, existing hash roundtrip unit test; no schema/data operations | No release-readiness claim |

## Verification authority and shared interfaces

Exactly two implementation packets; zero independent QA/review packets and
zero automated final-SHA integration checks. The current user unit-only override
in root AGENTS.md, agent-workflow.md and the assigned task supersedes historical
automatic review/QA/integration workflows. The owning hub accepts the plan;
there is no independent review loop to schedule. Workers run only listed affected
unit tests once per relevant change; passing tests are not rerun merely because
Packet02 finishes. Record each result with its concrete source checkpoint and
affected paths so the final assembled changes are attributable.

Shared interfaces preserved: PortalRoutes → Shell ReactNode insertion;
section-selection/session permission and context contracts; identity validator
→ all password mutation callers; frontend password payload/error contract.
No interface shape changes connect the two packets. Their unit suites cover
the affected behavior. Final-SHA browser or integration checks require a separate
explicit user request; earlier unit reports are not cumulative integration proof.

## Gaps, locks and risks

No unresolved product rules or acceptance requirements omitted. No canonical
scope change, so RESLICE_REQUIRED is not triggered. Root/scoped rules and the
user's explicit minimum8 decision are sufficient; demo fixtures are not policy.

Reserve the main-checkout-source-writer lock for one implementation worker at
a time. No browser/database/runtime lease, restart, reset, credentials, private
env read, worktree, deployment or remote push is part of this plan. Keep the
running watchers and account/database intact; never read/change user test_creds.

Known limits: CSS layout cannot be established by jsdom; the user may observe
the live watcher refresh manually. Bootstrap behavior is covered through the
shared validator/call path rather than executing its state-changing command.
Missing pinned tools or unexpected policy/source drift must be reported, not
worked around with broader checks, downloads or expanded implementation scope.
