# React application

One app at `apps/portal`: Realization, Islamic financing, Insurance and Admin
are source modules under `src/features`, with one entry/router/session. Existing
seller routes and `/finance`, `/insurance`, `/admin` remain valid. Select sections
using effective permissions and active company kind; inactive memberships are
context choices, not grants. Remember only the last permitted section for the
actual user/context and expose an internal switcher. Context transitions retire
old requests, query state and dialogs before admitting another company's pages.
Use shared components,
approved typed contracts and the exact runnable HTML mocks in mock-map.md.
Read targeted mock symbols or inspect in a browser; never load/edit minified
bundles wholesale. Operations use approved modal patterns, not right drawers.
Preserve row details, loading/empty/error/success states, keyboard access and
table width. Demo credentials, localStorage identity and floating-point money
are not production behavior. Use locked npm/Node tooling. Development verification
is affected unit tests only, once per relevant change (user decision 2026-09-26).
No automatic lint, typecheck, build, browser/E2E/integration, React Doctor or
review/QA loops. Additional checks require an explicit user request; CI/release
checks remain separate. Changes to shared components include affected consumers.
No Git writes, deployment or canonical-document changes by implementation workers.
