# Acceptance coverage and evidence limits

Task unified-frontend-20261013, base743bd6c4accadcd50afb59534bb66dabe2f3baf8.
The six U requirements come from the approved task. R IDs below refer only to
preservation of the source-complete predecessor, never a new backlog task.
No verification has run during slicing. All implementation unit selectors are
in the packets; actual passing files/results must be reported by their worker.

## Unified-app acceptance

| ID | Acceptance obligation | Owner / meaningful evidence |
|---|---|---|
| U1a | Exactly one entry/router/session/query lifecycle | P1 lifecycle component units; P2 mountPortal contract; P3 retires four main files and all 20 app configs |
| U1b | One app workspace, Vite process and dist | P3 portal manifest/config/lock ownership; P4 launcher/config consumers; no install/build claim |
| U1c | Every current section page/action retained | P3 94 moves plus four composition replacements in move-manifest; R table below |
| U2a | Active company kind + effective grants, no role-name routing | P2 decision and mount-denial units; seller/provider settings remain kind-gated |
| U2b | Admin without company, dealer/distributor same seller section | P1 no implicit company choice without kinds; P2 no-company Admin and both seller types |
| U2c | Accountant OR navigation and verification operation gates | P1 existing accountant/commercial-access units; P2 pure matrix; P3 real-section navigation |
| U2d | No backend grants, no membership entitlement projection | P2 wrong-kind/absent grant and inactive membership cases; no backend module paths owned |
| U3a | Valid path/search/hash restored after login/password flow | P2 portal component units |
| U3b | Last allowed section scoped user/company; revoke/malformed storage fallback | P2 pure storage/admission decisions plus root navigation units |
| U3c | Only valid sections; memberships clearly pending context destinations | P2 switcher in three shell variants; selector failure/no/multiple membership cases |
| U3d | Company selection then session refresh before route admission | P1 deferred context PUT/GET units; P2 page render counters |
| U3e | Clear/cancel old data, no stale dialogs/requests/CSRF/401 effects | P1 deferred query/mutation and user/company/branch refresh units; P2 keyed section shell |
| U4a | Seller URLs and finance/insurance/admin families unchanged | P2 prefix/root logic; P3 actual sections + meaningful overview click units |
| U4b | SPA navigation and absolute Finance/Admin links/search fixed | P3 OverviewPage actions, actual nav/search units; no API URL replacement |
| U4c | Shell filtering/active states/settings/loading/error/password retained | P1 gate outcomes; P2 rendering/selection errors; P3 settings and real nav composition |
| U4d | Denied routes never mount or flash old content | P1 isolation, P2 explicit denied element counter; P3 real composition check |
| U5a | One Vite fallback + one Go static root | P3 pure document-policy units; P4 httptest/tempfile units |
| U5b | API/health/source/asset/file/traversal/intent boundaries | P3/P4 matched negative matrix; real static bytes and HEAD/security/cache checks in P4 |
| U5c | All tracked consumers use concrete WEB_DIR | P4 exact paths; root README/config documentation amendment |
| U5d | Launcher + safe stop-helper identity updated together | P4 FakeSystem suite only; never execute launch/stop |
| U5e | Both authorities accept shared runtime contract | ADR-15 + infra-approval.md already recorded |
| U6 | Only affected unit tests once per relevant change | Per-packet selectors/result records; no automatic QA/check loops; no installation/runtime/release inference |

## Preserve the thirteen approved predecessor behaviors

P3 mechanically moves all feature source/test files; shared kit remains in place
and backend feature modules are unowned. Named files identify relevant source
slices, not instructions to reload whole features or run every existing suite.
Paths below are destination paths relative to web/apps/portal/src/features.
Existing test files move alongside each same-name component where present.

| ID | Preserved behavior | Relevant source preserved; affected evidence |
|---|---|---|
| R1 | Fixed dealer/distributor company type; both can sell | realization/navigation.tsx, pages/settings.tsx; admin/pages.tsx; shared session Company unchanged; P2 kind matrix |
| R2 | Setup/history allowed before verification; new commercial operations gated | realization/commercial-access.test.tsx, order-actions.tsx, sale-form.tsx, pages/settings.tsx; admin/pages.tsx and company-verification.test.tsx; kit session/org-settings unchanged except lifecycle |
| R3 | Reservation terms/payment/expiry/allocation/purchase/consignment | realization/reservation-workspace.tsx, reservation-payment.tsx, order-create.tsx, order-workspace.tsx, order-fulfilment.tsx; byte-preserved move |
| R4 | Same-company vehicle move | realization/vehicle-move.tsx and pages/inventory.tsx; byte-preserved move |
| R5 | Consignment return request/confirmation and custody | realization/consignment-workspace.tsx and pages/trade.tsx; byte-preserved move |
| R6 | Private customer passport fields/uploads | realization/customer-passport.tsx and pages/crm.tsx; byte-preserved move; fileUrl API remains authenticated path |
| R7 | Исламская линия terminology | sections.tsx transfers finance brand; financing/pages.tsx and realization/pages/partners-finance.tsx; no API/path terminology rewrite |
| R8 | Exact percent/minor-unit installment arithmetic | realization/installment-percent.ts, installment-calculator.ts, installment-terms-form.tsx; byte-preserved move |
| R9 | Filled/uploaded/signed versioned retail and wholesale contracts | realization/contract-workspace.tsx, pages/retail.tsx, pages/trade.tsx; byte-preserved move |
| R10 | Accountant invoices and waiting states separate from sale grants | realization/pages/billing.tsx, navigation.tsx, accountant.test.tsx; P1 affected suite, P2/P3 nav admission |
| R11 | Seller insurance flow retains addressed insurer authority | realization/sale-insurance.tsx, pages/retail.tsx; insurance/pages.tsx, navigation.tsx, queues.ts; P3 prefix nav unit |
| R12 | Demo scoring before own-installment signature, no real claim | realization/pages/retail.tsx, installment-servicing.tsx; byte-preserved move |
| R13 | Accountant accepts/rejects grouped installment evidence | realization/installment-payment-review.tsx, installment-payment-form.tsx, pages/billing.tsx; P1 existing accountant behavior suite |

Changed paths do not authorize reworking financial/legal rules. OD-01/05/06/09/10/12
remain outside scope unless the already approved R/U task explicitly resolved a
narrow point. Never derive policy from mocks or test fixture identities.

## Independent QA specifications — dormant evidence gaps only

These small specifications satisfy decomposition coverage. They are not jobs,
gates, scheduled reviews or authority to run extra tests. Latest U6 prohibits
automatic QA; an explicit user request must activate any additional verification.
No separate QA workflow artifacts are created.

QA-A, session/selection: independent qa actor gpt-5.6-sol/high, 10k context,
exact requested candidate SHA supplied by root; read-only checkout/test lease,
no source edits, browser, DB or network. Requirements U2/U3. Owned source paths
none; report only a root-assigned evidence path. Dependency accepted P1/P2.
If explicitly activated, use their named lifecycle/selection unit selectors and
report actual files plus late-response/no-company/inactive-membership evidence,
not implementation self-report. Stop on SHA drift, missing selectors or any runtime
dependency. This would repeat unit evidence, so it is currently NOT authorized.

QA-B, actual navigation: independent same role, 10k, read-only exact candidate,
P3 dependency; U1/U4/R1/R2/R7/R10. Own no source, report only assigned artifact.
If explicitly activated, direct portal-config invocation of routes.test.tsx and
features/realization/navigation.test.tsx at the supplied SHA. Capture clicked
Finance/Admin destinations, denied mounts and company setup/accountant access.
No build/browser. Stop on stale SHA, unavailable workspace or missing cases.
Currently NOT authorized.

QA-C, serving/process boundaries: independent same role, 10k, read-only exact
candidate P4 dependency; U5. Own no source, report only assigned artifact.
If explicitly activated, P3 pure policy selector plus P4 webui/FakeSystem commands;
match positive/negative document intent and process identity cases. No listener,
private env or actual signaling. Stop immediately if fixture isolation is absent.
Currently NOT authorized.

Final-SHA shared-interface integration specification: independent qa actor,
12k, one immutable root-supplied final candidate and its dev ancestry; exclusive
read-only checkout/test lease. Covers shared session -> portal routes -> real
section exports and Vite/Go request-policy agreement, plus manifest/runtime
consumer identity. Explicit additional authorization must first define execution
mode; the current task permits no integration run. Proposed smallest check uses
real composed portal routes with mocked API and deferred company transition,
then the matched pure Vite/httptest matrix at that same SHA; no browser/DB/server.
Earlier passing packet units are historical evidence, not cumulative final-SHA
integration. Stop on change under test, unresolved contract mismatch or any need
for install/build/live service. No production/release claim.

## Remaining limits

No unresolved product decision was needed for this decomposition. Source-only
risk remains around persistent-client ordering, combined CSS, manual lock identity
reconciliation, and drift between the two document policies. Unit coverage addresses
behavior within mocks/fixtures; installation, full feature visual parity, actual
Vite build/proxy, deployed Go/image/workflow and cross-tab/server concurrency
are not established by this plan or its unit-only execution.

