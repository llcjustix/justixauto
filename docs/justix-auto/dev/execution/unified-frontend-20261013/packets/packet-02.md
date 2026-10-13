# P2 — One section router and permission-aware shell

Task/base/hub: plan.md. Requirements U1–U4,U6; preserve R1,R2,R7,R10/R13.
Dependency: accepted P1 source/result-01.md with exact SessionApi/HTTP handoff.
Worker, 18k input ceiling, exclusive source/test lease, no children/Git/runtime.
Read root/workflow/web rules and plan contracts; only relevant session/shell
symbols, existing nav/main definitions and accepted mappings.

## Exact ownership

Modify web/packages/kit/src/shell.tsx and web/packages/kit/src/index.ts.
Add web/packages/kit/src/portal.tsx,
web/packages/kit/src/section-selection.ts,
web/packages/kit/src/section-selection.test.ts,
web/packages/kit/src/portal.test.ts.
May modify web/packages/kit/src/shell.test.ts for behavior, not snapshots.
Result: execution/unified-frontend-20261013/result-02.md.
Session/HTTP changes belong to P1: report a dependency defect before touching them.

## Behavior and public handoff

Implement exact PortalSection, SectionId, PortalRoutes and mountPortal signatures
from plan.md. Shared existing Shell rendering keeps all variants, sidebar groups,
bottom items, active states, branch/company controls, security dialog, global
search and banners. Convert cabinet reload anchors to internal selection controls.
Provide the section/context switcher in workspace, Insurance and Admin shells.
Company controls must call P1 selection without a second cache clear; branch
controls rely on the same lifecycle. Key shell/page subtree by admitted scopeKey
and section so transient dialogs cannot survive company/section changes.
Keep legacy mountApp usable until P3 but do not introduce a second portal router.

Selection uses longest owned prefix (/finance, /insurance, /admin), then seller
paths; root / is common landing, /dashboard is seller home. Valid direct path,
search and hash survive login/forced-password and take precedence over remembered
section. A denied page is never mounted, including before context resolution.
Use existing permitted nav policy; kind-check unguarded dashboard/settings.
Admin matching platform grants work without a company and coexist with active
company sections. No role-name string comparisons or verification-wide denial.

Persist only advisory section ID under an actual user + company-or-platform key
(for example justixauto:last-section:v1:<user>:<context>), validating on every
entry against current grants/kind. Persist only admitted section, never pending
company destination. Storage malformed/unknown/unavailable is harmless. Choose
remembered usable home at /; revoked/unavailable history falls back deterministically
to first usable section in supplied sections order, then explicit selection/no
access. Prefix-only entries select their first usable nav home; nested denied
routes fall back safely without mounting their elements.

An inactive membership is only a context-selection option labelled as such.
Do not advertise it as an already authorized section. For company-scoped direct
links lacking a matching active kind, show matching memberships; one can select
once automatically consistently with legacy behavior, multiple require a choice,
zero shows explicit no access with valid escape destinations. PUT context and
fresh GET session must finish before section/page admission. Newly selected but
insufficiently granted company must not cause selection loops. Preserve requested
allowed deep link across successful selection. Failed selections retain a clear
retry/choice state. Choosing Admin never forces selecting a company.

## Units (once per relevant change)

npm --prefix web run test:unit -- --project @justixauto/kit packages/kit/src/section-selection.test.ts packages/kit/src/portal.test.ts packages/kit/src/shell.test.ts

Use .test.ts, jsdom pragma/createElement for components, real MemoryRouter and
QueryClient with mocked API. Pure decision table covers seller dealer/distributor,
unverified settings, accountant OR nav, bank/mfo, insurer, platform no company,
wrong-kind grants, no permissions, multiple memberships, absent membership,
revoked remembered section and malformed/unavailable storage.
Component cases exercise direct URL restoration, root last allowed, internal
switch (no document reload), forbidden element render counter remains zero,
company PUT+refresh before route admission, selection failure, query/hash
preservation, correct absolute paths/search and each shell's switcher.
P1 deferred isolation tests are not rerun without a relevant source change.

Stop if existing grants cannot determine a route; report ambiguity rather than
inventing a permission. Preserve billing/commerce operation gates. Record exact
exports and final shell nav/search path expectations in result-02.md for P3.

