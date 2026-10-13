# P1 — Session/context isolation in the persistent app

Task/base/hub: plan.md. Requirements: U1 lifecycle, U3 isolation, U6; preserve R1,
R2, R10/R13. Dependency: approved base. Actor worker, 16k input ceiling, exclusive
main-checkout source/test lease; no other writer, children, Git or runtime actions.
Read root AGENTS, agent-workflow latest override, web/AGENTS, plan lifecycle
contract, ADR-15 session paragraph; targeted source below. Do not load full board,
all feature pages or the unrelated packages/api authEpoch implementation.

## Exact ownership

Modify:
- web/packages/kit/src/session.tsx
- web/packages/kit/src/http.ts
Add:
- web/packages/kit/src/session-lifecycle.test.ts
- web/packages/kit/src/http-context.test.ts
Result only: execution/unified-frontend-20261013/result-01.md.
No other source ownership. Existing kit test discovery includes .test.ts; component
units use jsdom file pragma and createElement with existing locked testing tools.

## Behavior

Keep existing API helpers and SessionGate callers compatible. Add SessionApi
scopeKey per plan. Centralize session ordering: initial load, refresh, context/
branch selection, password-required, logout and 401 cannot re-admit old views.
Changing user/company/branch scope removes previous page/dialog subtree before
new-context requests are allowed, cancels/removes old query/mutation state, and
remounts the scoped child tree. Same-context refresh may preserve ordinary form
state when still authorized; permissions/password changes must remove forbidden
content. The existing parent QueryClient remains the single lifecycle owner;
do not create a client per section or rewrite every feature query key.

Fence HTTP work at the existing send boundary. Retired requests must not return
adoptable old data, rotate live CSRF, or invoke the current unauthenticated
callback on late 401. Suppress after response-body await as well as fetch await.
Retire on context selection intent, user/logout/401 change, and final detected
identity/context change; block admission while server selection plus refreshed
session are incomplete. Preserve current CSRF/If-Match headers and network-error
semantics. Already dispatched mutations cannot be undone: do not replay them,
and never let their completion reopen a dialog or invalidate new-context data.
Refresh concurrency uses a generation check; cleanup must not leave stale global
callbacks owned by an unmounted gate. Failed/uncertain transition is a clear
error/retry/anonymous state, never stale-company content.

Keep kinds compatibility for legacy entries until P3. With kinds omitted,
CompanyGate must NOT silently auto-select its sole membership (Admin/no-company
must remain valid); P2 takes unified company selection ownership. Preserve
useCommercialAccess verification as operation-level gating. Context selection
returns only after refresh success, not merely after a swallowed error state.

## Acceptance/unit execution

Use mocked fetch promises and real QueryClient in jsdom, no network or server.
Cover A-to-B same query keys with deferred A success; deferred A 401/CSRF response
after B; logout/login different users; stale session refresh out-of-order;
context PUT pending/failure; forced-password/401 removes content; branch scope
change retires content; dialog unmount and stale mutation completion; omitted
kinds leaves a no-company platform session untouched. Check exact existing
context endpoint/If-Match and successful refresh admission, not implementation
counters alone. Preserve same-context commercial access refresh behavior.

Run once using pinned current Node/npm:
npm --prefix web run test:unit -- --project @justixauto/kit packages/kit/src/session-lifecycle.test.ts packages/kit/src/http-context.test.ts
From cwd web (direct app config avoids prior workspace-discovery omissions):
npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/accountant.test.tsx src/commercial-access.test.tsx

Both named feature files must appear in the result; pass-with-no-tests is not
acceptable. Use installed pinned binaries only; never allow npm exec to fetch.

No installs, whole suite, lint/build/typecheck or extra QA. Missing tools stop.
Report exact lifecycle/internal HTTP export names and scopeKey semantics for P2,
unit failures/exclusions and lease release. Stop on need for auth-protocol/backend
changes or ownership expansion; do not silently import a second auth system.

