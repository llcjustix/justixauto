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
follows the root final-only policy (user decision 2026-10-13): finish the full
requested change before testing, without automatic unit-test writing or runs
per edit/packet. At the end use focused existing tests or browser/E2E when useful;
include affected consumers of shared components. No automatic lint, typecheck,
build, React Doctor or review/QA loops. CI/release checks remain separate.
No Git writes, deployment or canonical-document changes by implementation workers.
