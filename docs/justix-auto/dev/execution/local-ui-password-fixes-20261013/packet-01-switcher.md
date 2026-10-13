# Packet 01 — useful, consistently styled section switcher

Task: `local-ui-password-fixes-20261013`; authority and acceptance: `/root`.
Base: `31b018ce1a4b49a2f56148459b4efa0893d345ce`, hub-provided; execution branch
`fix/portal-ui-password-20261013` prepared by version_control at identical source.
Requirements: task U1, U2; preserve unified-frontend U2–U4, ADR-15.
Dependency: hub accepts this packet; exclusive checkout writer lease.
Context budget: 10k input tokens, hard ceiling 14k. No child agents.

## Owned writes (exact)

- `web/packages/kit/src/portal.tsx`
- `web/packages/kit/src/ui.tsx`
- `web/packages/kit/src/portal.test.ts`
- `docs/justix-auto/dev/execution/local-ui-password-fixes-20261013/result-01.md`

Read root `AGENTS.md`, current override/authority in
`docs/justix-auto/dev/agent-workflow.md`, `web/AGENTS.md`, assigned task,
this packet and `findings.md`. Source reads: owned files (ui.tsx only styles),
`section-selection.ts`, shell.tsx sidebar insertions (128–230),
kit design.css button/Insurance sidebar rules (390–440, 4332–4367),
Admin design.css sidebar rules (28–60). Contract slices:
`docs/justix-auto/dev/unified-frontend-20261013.md` U2–U4/U6,
`docs/justix-auto/adr-15-unified-frontend.md` access/context paragraphs,
`docs/justix-auto/business-logic.md` Applications and access.
See source-hashes.json when available; packet source was inspected at supplied base.

## Diagnosis and implementation

PortalRoutes always supplies a chooser even if it contains only the current
destination. The chooser uses `btn btn-secondary btn-sm`; fixed 32px height
conflicts with Admin/Insurance sidebar 12px vertical padding, block anchor
display and button nowrap. Shell already shares the node across all variants.

1. Extract the existing unavailable-section/matching-company mapping into one
   local choices list. Preserve its exact filters and ordering. Show the chooser
   only when `destinations.length > 1 || companyChoices.length > 0`; otherwise
   pass null/undefined to Shell. Do not count unrelated memberships or infer
   another company's permissions. Existing same-section company controls remain.
2. Give chooser links and company buttons one dedicated switcher item class,
   replacing the compact button classes and inline one-off layout. Define the
   chooser and item rules in ui.tsx's existing shared styles. Scope specificity
   to beat `.admin-sidebar nav a/button`, `.ins-sidebar nav a` and `.ins-sidebar
   button` without changing those global selectors. For example, sidebar-scoped
   chooser rules plus `.portal-switcher .portal-switcher-item` and its active,
   hover and focus-visible states. Use existing tokens. Items need natural height,
   consistent padding/alignment, min-width:0, max-width/width constrained to their
   container and wrapping/overflow-wrap for long labels. Preserve accessible nav
   name, native buttons/links, active indication and visible keyboard focus.
   If defining active styles, derive the selected destination class from
   `destination.id === section.id`, so a nested/sibling page still marks its
   section rather than relying on the destination home URL alone. Include a
   behavior assertion for section indication on an admitted deep page.
3. Leave permission helpers, destinations, navigation paths/state, session
   sequencing, last-section persistence and Shell insertion points unchanged.
   No design.css, feature CSS or shell.tsx change is needed.

## Affected unit evidence

Extend `portal.test.ts` with a sole-platform-admin session having no memberships:
Admin page renders and navigation named `Разделы` is absent. Also cover a sole
active company section with no alternative choices. Keep/add a positive case
with multiple permitted sections but no inactive company choices, and retain
the existing one-destination-plus-inactive-membership case. Existing tests cover
all four section variants, internal link navigation, direct URL search/hash,
remembered/revoked sections, denial and company PUT/session-refresh ordering.
Use role/name and behavior assertions; no CSS-value/source-string snapshots or
claims that jsdom proves visual layout.

Before application edits run `bash tools/check-git.sh` as required by root rules.
After the cohesive change, from `web/`, using installed pinned Node24.21.0 and
npm11.19.0, run once:

```sh
npm run test:unit -- --project @justixauto/kit packages/kit/src/portal.test.ts
```

Record actual command/result and changed paths in result-01.md; do not claim
visual verification. No independent QA/review, browser, integration, lint,
typecheck, build, React Doctor or repeated passing checks are authorized.

## Locks and stops

Exclusive `main-checkout-source-writer`; no Git mutation by worker, no worktrees,
no runtime restart/reset, credentials, private env, database or remote operation.
Preserve user files including unread `test_creds`. Hub owns canonical records.
Stop for conflicting source edits, missing pinned toolchain, required writes
outside ownership, or a needed access/context rule change. Report the specific
gap; do not invent policy. If coherent work exceeds 14k context, request a split;
mark RESLICE_REQUIRED only if canonical task scope must change.
