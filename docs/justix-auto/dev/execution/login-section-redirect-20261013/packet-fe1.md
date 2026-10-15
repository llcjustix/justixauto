# FE1 — Recover unavailable section URLs through common entry

Task: `login-section-redirect-20261013`; authority: application hub `/root`.
Approved contract: `docs/justix-auto/dev/login-section-redirect-20261013.md`.
Dependency/base: `d078b4b0e200942735d581ea6c4be761d84e9add` (hub supplied).
Branch: `fix/login-section-redirect-20261013`, prepared by version_control.
Context limit: 8,000 tokens; read only these rules, contract, and targeted source.
Source fingerprints: adjacent `source-hashes.json`.

## Ownership and rules

Only application write: `web/packages/kit/src/portal.tsx`.
Worker result: `docs/justix-auto/dev/execution/login-section-redirect-20261013/implementation-result.md`.
Read `AGENTS.md`, `web/AGENTS.md`, `docs/justix-auto/dev/agent-workflow.md`,
and `docs/justix-auto/dev/final-validation-policy-20261013.md`.
Contract references: ADR-15 section access/context paragraphs;
`business-logic.md`, “Applications and access.” No unresolved product rule.
Use apply_patch; before application edits run `bash tools/check-git.sh`.
Reserve the sole checkout writer; no concurrent source readers or branch changes.
No Git writes, children, canonical edits, new dependencies, backend/session API
changes, credentials, database resets, restarts or remote actions.
Preserve unrelated `internal/modules/identity/service/company_contract_test.go`
(hub hash in manifest) and leave `test_creds.txt` unread.

## One behavior

In `PortalRoutes`' unavailable-section branch, compute requested-section matching
companies once. Show its existing `CompanySelection` only for a non-Admin
section with at least one candidate and an active company outside that set.
Retain `portalCompanyId` preference there. Otherwise return `<Navigate to="/"
replace />`. Common entry already validates remembered/default permitted
sections and offers all matching company memberships. Do not change its logic.

Current failure: stale Admin with no current context sees empty destinations
and false denial; a provider URL without matching memberships sees an empty
company selector. Both should reach common entry instead.

## Acceptance coverage

Local IDs below map to the approved contract, not new backlog items.

- R1: Admin logout → seller login at `/admin/companies`, no active company:
  common entry selects the single seller company and reaches allowed content.
- R2: Several eligible companies require explicit choice; requested non-Admin
  candidates and `portalCompanyId` preference remain effective.
- R3: Stale `/finance` or `/insurance` without matching memberships falls back
  to another accessible section/company rather than an empty selector.
- R4: An already permitted deep URL retains pathname, query and hash;
  common entry validates user/context-scoped remembered section.
- R5: Context PUT and session refresh finish before company page admission;
  memberships never create grants. No eligible context/grant still denies access.

## Final verification and stop conditions

Finish the full source change first. No new/running units, lint, typecheck,
builds, React Doctor or automatic reviewer/QA loops. Primary owns one focused,
independent browser scenario with isolated mocked API sessions covering R1–R5
and shared router/session/company-selection interaction at the final candidate
SHA. Reserve the browser and freeze source during that check; record SHA and
evidence in the result. It establishes frontend behavior, not live authorization.
Repair only failures and rerun affected scenarios. No separate dormant QA packet.

Stop for source/base drift, ownership conflict, required edits outside portal.tsx,
or undecided access rules. Report RESLICE_REQUIRED if canonical scope must change.
