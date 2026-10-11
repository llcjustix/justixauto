# Product workflow improvements — bounded execution plan

Task: `product-workflow-improvements-20261011`. Owning hub: application coordinator `/root`. Base SHA supplied by hub: `4fa683ba4708bb71031534260ec5cb8742ce9977`. Feature branch: `feature/company-commerce-workflows-20261011`. Main checkout only. Owned artifacts: this plan and `packet-*.md` in this directory. Status: decomposition complete; P01 already dispatched by hub; other packets consume predecessor results.

Authority: `docs/justix-auto/dev/product-workflow-improvements-20261011.md` and `product-workflow-interview-20261011.md`; original HTTP/domain protocol at `docs/justix-auto/contracts/http-domain.md` (common values, identity §2, inventory/commerce §3, retail/insurance §4, document §5). Latest user decisions override obsolete contract proposal details. Slicer does not edit approved contracts, board or application source. These are execution packets under one feature, not new canonical backlog tasks. RESLICE_REQUIRED is not indicated: packet splitting preserves canonical scope.

## Execution rules

- Read root AGENTS.md and dev/agent-workflow.md; backend packets also internal/AGENTS.md, frontend packets web/AGENTS.md. Latest unit-only user policy overrides scoped old lint/build/browser/review prescriptions, including React Doctor execution.
- One exclusive source-writer lock for every implementation packet. No browser, PostgreSQL fixture, live runtime, migration application or deployment locks are requested. Existing worktrees untouched.
- Worker first runs `bash tools/check-git.sh`, then edits only exact owned paths; Git writes remain version-control role. Record predecessor commit SHA at dispatch. The base above is provenance, not the expected unchanged HEAD for every later packet.
- Context budget 18k–20k tokens per worker, smaller where packet says so. Read targeted functions/interface slices; never load all packets or full source modules into a worker. Stop/split before exceeding budget; report RESLICE_REQUIRED only if canonical scope changes.
- All SQL changes are new files, no applied migration edits. Sequential migration allocation 000035–000045 below. Request exact scope extension for an unlisted production/test path; do not silently refactor shared code.
- Only affected pure unit tests, once per relevant change. Go commands disable implicit vet. Never run module-root flow tests, tools/test-go.sh, DB-backed repositories, lint/typecheck/build/browser/E2E/integration, automatic reviews or QA loops. Later relevant edits run affected units once; do not rerun already-green tests for ceremonial final acceptance.
- Unit fakes must exercise meaningful service behavior and rollback/port failure, not duplicate implementation assertions. They cannot establish actual database isolation or deployed scheduler behavior.
- Packets changing app adapters own a meaningful pure `internal/app/workflow_ports_test.go` seam case and its narrow selector; service-package units alone do not compile/test app wiring. Existing `TestOpenAPICoversRoutes` uses e2e.DB and is excluded. Never report an empty unit selector as behavioral evidence.
- Backend DTO/routes in these packets are implementation seam proposals under approved behavior. Worker reports exact implemented fields/action paths; downstream frontend consumes that report and current targeted handler DTOs. No guessed fixture schema and no contract regeneration.
- Return packet results to hub with changed paths, before/after SHA supplied by version-control, exact unit commands/results, route/DTO/port deltas and exclusions. The hub records canonical state.

## Unit command routing correction

The accepted plan digest `91174bef20b62e270574e66d7baca0c59cb6619dee954af8a042a2d5c278474d` is superseded by this command-routing correction, not by a review/QA gate or a product-scope change. Evidence: `result-02.md`. The generic workspace command silently omitted the Realization suite while reporting Kit/Admin success.

For downstream Realization units, use cwd `/Users/bakhromachilov/startups/justixauto/web` and `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/<suite>.test.tsx` (or the actual `.test.ts` filename). Preserve the `npm exec --` separator. Kit/Admin selections stay in their separate root workspace command; never mix Realization root-relative arguments into that command. Updated packets provide exact selections. Run each only once for relevant changes; a failed selection does not justify repeating another green selection. Output must explicitly list the intended suites.

P02's omitted `src/add-company.test.tsx` has already passed separately (1 file / 1 test). Its historical packet/result remain unchanged; this correction does not request a rerun.

The hub also approved P03b's exact fixture-path extension `internal/modules/commerce/service/terms_colors_test.go`, solely to supply a valid principal to existing cases. No color behavior or broader source scope changes. Original source fingerprints below remain the historical planning snapshot; concurrent source writer ownership is unchanged.

## Product decisions and narrow gaps

Resolved by explicit user: company type fixed at company level and chosen by platform Admin; existing companies manually classified/verified, new operations blocked while history/old obligations remain; reservation refund is full advance confirmed by accountant; partially paid purchase handover requires explicit order agreement; mandatory advance deadline automatically releases only if dealer never submitted payment, with pending/accepted advances protected. Free reservation expiry remains money-aware. No default deadlines/percentages. Original R2 minimal creation takes company name + first-admin login/password; userType is optional at creation and may be assigned manually in Admin before verification/new commerce. P02 owns the narrow P01 backend correction and affected service units.

Contract preparation defaults accepted by “go start”: configurable data-filled template plus external document, paper signature scan, immutable signed versions and separate amendments; retail and B2B coverage is the coordinator's stated scope assumption. Existing signed metadata becomes the one known historical version; never fabricate older versions.

Only current narrow product question: whether distributor retail selling remains allowed along with dealer retail. Existing retail availability is preserved until answer; fixed userType does not replace membership roles/company switching. No broad new permission matrix is invented. This does not block setup, accounting, documents or reservation implementation.

OD-01 finance disbursement/title/partner-finance delivery remains outside this feature. OD-08 approval-to-coverage remains unresolved and outside scope; demo scoring never creates policy coverage. OD-09 preserved arithmetic excludes fees/penalties/repricing. OD-10 real provider APIs/legal forms remain excluded. OD-04/11/13 are resolved only to the extent explicit decisions above cover them.

P18c runtime shape is jointly accepted by application and DevOps hubs in `expiry-runtime-decision.md`: 60s poll / 100 candidates / 10s pass, bounded initial pass, serial loop, cancel+join before DB close, stable `commerce-reservation-expiry` lock name. RunOnce callback is outside the lock transaction; overlap after lock loss is handled by P18b row locks/payment rechecks/idempotency. No infrastructure rollout, exactly-once or hard deadline guarantee.

## Packet order and ownership

Dispatch sequentially in the following order. Dependencies in packets are logical; the single-writer rule additionally serializes them. P03a/b/c may use one bounded dispatch context only if all three fit 18k and individual changes/evidence remain identifiable; otherwise retain fresh contexts.

| Packet | Behavior | Requirements |
|---|---|---|
| P01 | Fixed company type, verification lifecycle, setup access/session capability | R1/R2 |
| P02 | Admin/settings/session company UI | R1/R2 |
| P03a | Inventory and retail new-operation guards | R2 |
| P03b | B2B/finance/insurance new-operation guards | R2 |
| P03c | Commercial UI explanation and server-error state | R2 |
| P04 | Accountant role and invoice/payment authority | R10/R13 |
| P05 | Accountant queue and invoice waiting state | R10/R13 |
| P06 | Percent-first exact installment input | R8 |
| P07 | Islamic line terminology | R7 |
| P08 | Private passport profile/binding backend | R6 |
| P09 | Passport editor/upload UI | R6 |
| P10 | Same-company warehouse Move UI | R4 |
| P11 | Retail immutable contract versions/templates | R9 |
| P12 | B2B immutable contract versions/templates | R9 |
| P13 | Contract workspaces | R9 |
| P14 | Persisted demo scoring signing prerequisite | R12 |
| P15 | Insurance and demo scoring in sale dialog | R11/R12 |
| P16 | Future supply capacity aggregate | R3 |
| P17a | Reservation request/confirmation/fixed terms and hold | R3 |
| P17b | VIN allocation and order holder transfer | R3 |
| P18a | Advance review/full refund/credit ledger | R3/R10/R13 |
| P18b | Expiry/extensions/order continuation rules | R3 |
| P18c | Automatic expiry runner | R3 |
| P19 | Reservation/payment/refund UI | R3 |
| P20 | Custody-only inventory port | R3/R5 |
| P21a | Consignment and agreed purchase handover | R3 |
| P21b | Authorized consigned retail sale and settlement link | R3/R5 |
| P22 | Recorded return request/dealer confirmation | R5 |
| P23 | Consignment/return/ownership UI | R3/R5 |

29 implementation packets. 0 automatic review/QA packets: explicit latest unit-only policy supersedes earlier automatic independent QA protocol. No independent review claimed; coordinator accepts scope.

## Shared interface acceptance at final candidate

These are acceptance mappings to affected unit cases, not instructions to run an integration suite or repeat passing units at the final SHA. Hub assembles source/results at the final candidate, flags later edits invalidating a case, and runs only then-affected units once per change. Independent runtime/CI/release checks require separate authority; no claim of release readiness.

| Seam | Producer → consumer | Required cases / owning units |
|---|---|---|
| S1 company | identity Company/Profile/session → Principal → business service → kit Company | P01/P02/P03: fixed type, fresh verification, setup allowed, new writes denied, legacy obligation permitted; provider kind retained; no global permission deletion |
| S2 money authority | retail/commerce permission catalog → preset → issue/review services → billing/sale | P04/P05/P18a: seller cannot issue, accountant can, admin retains company authority; accepted only reduces debt; grouped review once |
| S3 reservation stock | commerce lot/reservation → inventory Holder port via app | P16/P17: request no hold; exact mutual activation; future-held→materialized VIN claim conserves total/available; no double hold; atomic holder reassignment |
| S4 accepted money | reservation payment/refund → expiry/conversion/handover | P18: pending/accepted protects; full refund confirmed before release; advance credited once; explicit partial-purchase agreement; service and runner same clock/deadline |
| S5 document/privacy | documents purpose+private storage → owner binding → CRM/contracts | P08/P09/P11/P12/P13: passport purpose/sensitive enforced server-side, tenant isolation, authenticated download; snapshots immutable, old signed version preserved |
| S6 sale insurance | retail scoring/contract → insurance application via existing ports → sale UI | P14/P15: current demo precedes new signing; seller cannot decide insurer result; insurer approval never coverage; legacy signed servicing preserved |
| S7 custody | commerce agreement → inventory custody port → retail eligibility → return | P20/P21/P22/P23: owner/custodian differ, only authorized consigned sale, sold cannot return, conflicting hold blocks, capacity/port failure atomic |

## Complete requirement acceptance coverage

| Requirement | Acceptance dimensions | Packets |
|---|---|---|
| R1 | company dealer/distributor, immutable after initial Admin assignment, separate roles/kinds, no switch, legacy missing supported | 01/02; reservation party checks 16/17 |
| R2 | minimal onboarding; login/setup; optional legal/tax/contact/bank profile; submit/return reason/resubmit/verify; server and UI gate new stock/sale/order/offer/application; old servicing retained | 01/02/03a/03b/03c |
| R3 | stock VIN and future model/spec/colors/qty; request no hold; supplier confirmation; buyer accepts changed terms; fixed price/mutual changes; expiry/advance/deadline/extension; exact hold/allocation; accepted advance credit; full refund accountant confirmation; purchase/consignment and explicit partly-paid handover; automatic runner | 16/17a/17b/18a/18b/18c/19/20/21a/21b/23 |
| R4 | explicit source/target same-company Move; permission/source/capacity errors; ownership unchanged | 10 |
| R5 | distributor request, dealer decision and actual destination movement; history; owner/custody separation; sold and reserved blocks; money preserved | 20/21b/22/23 |
| R6 | all optional passport fields; copy upload; own-tenant binding; sensitive purpose/download; no public URLs/log payload | 08/09 |
| R7 | four-app/shared finance line wording; enums/provider types/rates unchanged | 07 |
| R8 | percent primary plus synchronized exact amount; explicit smallest-unit rounding; lock and positive-less-than-price; saved schedule/currency preserved | 06 |
| R9 | retail+B2B template/editable text, data-filled preview, external upload, signed scan; snapshot/history immutable; separate amendments; legacy known record preserved | 11/12/13 |
| R10 | accountant preset/work surface; separate invoice-management permission; waiting/issued sale state; company admin authority | 04/05 |
| R11 | insurance request/status/respond actions inside sale, insurer cabinet/decision authority retained | 15 |
| R12 | clearly demo current scoring before new own-installment signing, no real provider/coverage claim | 14/15 |
| R13 | accountant accepted/rejected grouped payments; only accepted evidence reduces debt; no duplicate child review | 04/05; shared reservation accounting 18a |

## Migration allocation

000035 company verification/type/profile; 000036 accountant permissions/preset; 000037 customer passport; 000038 retail contract versions; 000039 B2B contract versions; 000040 demo scoring; 000041 future supply; 000042 reservations; 000043 advance/refund ledger; 000044 consignment agreements; 000045 returns. Up/down pairs only. P20 reuses existing owner/custodian columns. If an intervening authorized source change consumes a number, version-control/hub serially reassigns the affected unapplied new pair; no rewriting applied files.

## Findings, limits and risks

- Source inspection found CreateSeller auto-activation but Admin draft access, no company verification/userType. Principal previously held permissions only. Never use permission removal as verification enforcement.
- Both retail invoice issuance and commerce issue/void currently use broad sale/trade authority; both need distinct invoice permissions. Existing installment grouped review already supplies accepted-only accounting.
- Existing Move is same-company. Existing Transfer changes both owner and custody. Future supply is a new capacity aggregate; ordinary offer quantity is not an ordinary-order cap.
- Inventory already has one active VIN reservation and separate ownership/custody columns. Future allocation must not free the same unit for another reservation. Cross-module calls use declared ports and Store.Bind/ambient transactions; no cross-module repository/table access.
- Retail currently requires owned stock, so custody-only handover requires explicit active-consignment sale eligibility and completion linkage. Returning a vehicle is not a settlement/refund.
- Current RecordContract overwrites metadata. Conversion preserves only known current historical signed facts, not reconstructed history. Templates are configurable company content, not legal advice/approved clauses.
- Existing documents support private PDF/JPEG/PNG and sensitive downloads. Passport binding must check sensitivity/purpose on the server; the upload UI alone cannot enforce privacy.
- Large shared files (app.go, retail handler, Realization data/retail/trade and Admin pages) are intentionally serialized. Exact path ownership overlaps only between sequential packets.
- Unit fakes cannot prove SQL migration execution, actual concurrency/isolation, multi-replica scheduler exclusions, file-store deployment or browser behavior. Those remain explicit unexecuted limits, not failed development gates.
- No app source/Git/runtime operations were performed by slicer. No canonical backlog inflation. Any actual interface/context expansion gets a bounded corrective packet within this feature, not an invented product task.

## Packet content digests

SHA-256 of exact packet bytes at plan finalization; plan digest is reported separately to avoid self-reference.

```text
42fac42ee895bd67ea99b61ea43cae02a5843e653e9d796d9101e4083c0239ae  packet-01.md
26c33654a0b224936f4056fdf1b6dfc277eeb86f900da0a869cf64c872863071  packet-02.md
c9ecf2a2d30d2e722bf037b3d4623f3d30b75897fa8ba254f8bd96bc5f4c06b1  packet-03a.md
29c7648af12fcc54a9f482834bf11ce711e2ea9f639167f0cc091436d0b2eace  packet-03b.md
0e858bbab3d2258c8902e749fb5223751189e7ea7614afb964b863544532dee0  packet-03c.md
c0a6dde0728969680f4f811d4d14c05ca4b5358709333a858da23804cb01f3a8  packet-04.md
938ceede881690ca93be70a24815531eb9137cc40f95c3e7a5f7655fc0373e25  packet-05.md
08612e98011d00e23f1ef4a8911b34544045b06684aebb24675524d2ef93001d  packet-06.md
37260661f06d2e3fa768cf1fb5c92534a8ac91b6937eadcd02e637bc7bc641be  packet-07.md
9585c038036fced3bee04428c0f03607b841286dffa466f1dc0cc7043e947277  packet-08.md
2602cc7b4fa2c309ea5c48a2167367904746aa773a54ffa15ee651ce2d4941ba  packet-09.md
80dc0582392797d22f6e077fb57f676d150aa6a9b7f0c0b283fbc6c3be304c30  packet-10.md
4a89df838133081caaf06c333ffe1b113a214f0ef388e41bb4d7a414abc050ca  packet-11.md
abf19187364f584fc0be82122bf64657d3b20613ed15db47b0cb1fa08f8f3a7b  packet-12.md
75ff3faa56f6fcadbca4c277c757d4526a32e8334bce833670fdcedfdf54113b  packet-13.md
c5ca6c266997775c9f5e00998cb9157325412c645cf0bb71b42ee066b8a65dbe  packet-14.md
677c5a874a132d4a6e993c26cbf0962ff5e2f6b08370c1d187136734fc8b4100  packet-15.md
0ba9524ac38999a191a9400b8c14c1ad90fa602d8ca0975a80a118cb27f4374c  packet-16.md
0d362be3f84dd9ca421ffc4334f0c28349cd30194204ce17ba2d1da3556f5472  packet-17a.md
ab748fe94d69d4ab5ec70540565bdaf50572da44a41bfdc19a6304c93c303b19  packet-17b.md
d7b1d7befb5055397de2a9d7b77e019535f8e8512eb3e5253bf289e7663b082b  packet-18a.md
398a8ba2bbf4e21613ce07cd30b0f004d23ae9146d37d3d1f337ceeee7ce1a8d  packet-18b.md
1c848d4c2d5cfd3bf726d8277c9c7c73b1bdec07691cf42fe23f9f759dbe8727  packet-18c.md
0f438cc5dab682800d6c870315b2653766407c859bc9ef8ff1164d6457d268b6  packet-19.md
a5576feda33362b515281f06fc9131989cf6a58a944e826593491a1796c254ae  packet-20.md
398f8ac001f63cb9510bfaeda035d2609809f9a2e4fb5a55a5f44f26e79d910b  packet-21a.md
6f6fa395022ced0eacf8c69cad77900e471628b7d017fcbed601d645e2a5c1d6  packet-21b.md
4570f0656759d54b7b9f6c625f777cea0299e361dd986345a432301979dc2e03  packet-22.md
fd780f08372b2ffca228aa5eb9cfa70db6d7bf8ba4153ad2a4f59b45852d31c1  packet-23.md
```

## Source fingerprints

Snapshot time: 2026-10-11T07:11:39.922686+00:00. These are **observed planning-time working-file hashes**, not a claim that every byte still equals the supplied base commit: P01 source writing was authorized while read-only slicing continued. They cover existing targeted inputs and owned-source seams; absent new files are specified in packets. Before dispatch consume predecessor exact commit/result and re-read changed interfaces. Requirement sources pin recorded decisions at snapshot; latest hub-relayed correction on optional type at creation is explicitly recorded above. No Git operations were used to collect these hashes.

```text
a222419ecca4935c906454884eb62d6cacace6a5b80395c90a1cee870f8b8524  AGENTS.md
c4430395e810edaaffdb24b43e485b38efb57369701a62d8113ebd25d87e0735  cmd/api/main.go
5bd0439c6aca0b5a824c340c4c15692734c70296b0125d82c63a739a21ba65ba  docs/justix-auto/contracts/http-domain.md
b2d5153012dfec17e20a5e55a65af858d693cae27adb66ef6a5c440599149b27  docs/justix-auto/dev/AGENTS.md
4f264fb5217a9613b00df455b0661cb9b955e90793a29b17c682cbcd0010830b  docs/justix-auto/dev/agent-workflow.md
9cf0569bb380f77d13ba7f23a168f388cc7582110ad0248e82a89d8e0b3c6f5f  docs/justix-auto/dev/execution/product-workflow-improvements-20261011/expiry-runtime-decision.md
7773e0e290127ece7ee6a7d7278a9acef64048334f695911a44a44685e2af4f3  docs/justix-auto/dev/product-workflow-improvements-20261011.md
b9ded73a2fbb4051cc459a35f5ccce3cf1fb35ddf529a9e24899d651beeb835a  docs/justix-auto/dev/product-workflow-interview-20261011.md
98d573b8a2a8a515dca0df89a5558ebad0cd4a9666e3f2bcd4fe5892210cc959  docs/justix-auto/open-decisions.md
b22213a76ca11a4ce4aeb04323260d1d577f28ef1230d48c15367d074e67650b  internal/AGENTS.md
84877c627bded9cedcfe0a71bec7c75492e0b7acf26878fd8423b55146df4aef  internal/app/app.go
72fc23b16fa36a34da8dcccfc78f09a58b99ccd2f34f13dfed33b96e0e005f62  internal/app/openapi_test.go
e5d46d544e8021a5dbdda10b3b93faeb373cbe8a4e5872a98bcedec95d323ff8  internal/modules/commerce/handler/dto.go
3a28f7f6222b4c6cfb6fa11d15337715a9f7488dcfd299c284e5fdaac99edc7e  internal/modules/commerce/handler/handler.go
5782f8d9e1a058c1379cc1d43acddbbacc0303f3a6196e83409b04f47e2da382  internal/modules/commerce/handler/invoice.go
8bcff56a60a4583f9d0009d49e359130e826a43be430bda87d61a9b053a60544  internal/modules/commerce/model/invoice.go
5dd4bbc7afd9030915a583449d16ec73a2ac83fe07af94f8ad163fc17872c2a2  internal/modules/commerce/model/offer.go
a71e3dfd2bc378d2638c1ae33f8686f0c3026b24c3d4e0d5247075b51072c1f2  internal/modules/commerce/model/order.go
b069cfa4e1117461a4411c53c199dc04a50bdcfb079bf057216e4d5e8308626f  internal/modules/commerce/model/permissions.go
93506fe77c4c15003cf60545af823154709a586681ac8f22c086dcb728c8b22f  internal/modules/commerce/module.go
e5f8103891c55eea811d8b9b022de87a397a1bbb96822d093cb9f9dd666da65d  internal/modules/commerce/repository/deal.go
f558adf2f4bbd43bf40fccece7f8754dc04d047ad72724e997d3d6d19181156b  internal/modules/commerce/repository/store.go
13d534804d41c856e3fc1a3472022ffa6131134b29a2dafb630c6733eef73a45  internal/modules/commerce/service/deal.go
b0b81742a2478f0edce53f9db85b2eaa0a4853c92bcfcd080826ae5e56c242b6  internal/modules/commerce/service/deps.go
db377700c932f5e806cb7a96ddb469ee114d692243ffac63be696a270466c6f0  internal/modules/commerce/service/fulfilment.go
16538fa5a8082056a40cad295fafc2a947217c04d7351491beaba771ac5fad83  internal/modules/commerce/service/invoice.go
89b2f132602f906ba44d9893e2e9e6adfb60634d0df3f7a9983aedb195956d9b  internal/modules/commerce/service/offer.go
4d53ad1ddc4c36854efb32b085860e1617bdbf9e80e6053f78e2111b8c9891a9  internal/modules/commerce/service/offer_colors_test.go
02f89434a65e70c9d43ba80ef039dd0172e96031e28dd9a82e8bbb14ea17dc1c  internal/modules/commerce/service/order_lines_test.go
696c0a0d0a4f65312cbeaf058ade3f9ec7581eaf777e6d0e9672cfb0ccc9dedd  internal/modules/commerce/service/ports.go
7107c0991b73bfc045b61cdc295890413f7bae5defcb5a7f11e9f2ed68b3be30  internal/modules/commerce/service/ship_delivery_test.go
1d2187924e07aa01d9168d935d315316bd9373336d80fc4d7aa8d3d8a03ba31e  internal/modules/documents/module.go
5f2b69f92ee049a2aca582e4426aa0488df9e06a1fe656c0fda0262ea199005a  internal/modules/documents/service/documents.go
8ab38a8a29d72a3b520e41d6a835159b1d2691814df2a61568e1fa524351b10b  internal/modules/financing/service/application.go
d011532b2d7aee67d95097b9526987a4a2d17685973a6f70d36b2b7916bf9938  internal/modules/identity/handler/admin.go
3f3eda7cbc690741f9de3e9046d6eb34fdc0b154b44912f933b98593e819d44b  internal/modules/identity/handler/company.go
f970624a393b6a5c725fd7c9f4a089a4c8f9e77775ff06f9a42f9ae897b34b72  internal/modules/identity/handler/dto.go
241a020c1187c2f2cc4016c540e6f0be28d09399f448d2e102533ddeae5d1211  internal/modules/identity/handler/middleware.go
bd4422940152409a228bd11ac614b393d69a06e8837d1902e19bd319a5807ead  internal/modules/identity/model/company.go
ff726f6678565667dfe0832c53f559646d6bbe21c32a52f69e849c74afb70a13  internal/modules/identity/module.go
4a930a92e8479f81d6b09abafac5be0c948c8c464f7fa930837891258a46fc8d  internal/modules/identity/repository/company.go
165abee17397952a605f17de7ff97fc7049c9c9fcb3ab1bcf08816885df2991a  internal/modules/identity/service/auth.go
81bd500cf45bbb718f298d87ee1dae2652e3a68595b4b4dd629fb48fb257c2fd  internal/modules/identity/service/company.go
2ec387845b85d213c9144034a2acf25c61e28a4a71c8f7ad1a699aeb4eb133cc  internal/modules/identity/service/company_test.go
e9751eaca4e2758cae91a72612e1c8097fa49a54221e9313a8f63c73e80a5569  internal/modules/identity/service/company_verification.go
7711470c0ac1b7fb19324dae0441f774f247f698b14d9e5041881d810e5317f9  internal/modules/identity/service/company_verification_test.go
f50f960b3e94daed20ad0511481e415cc12071e7de5d4233bdd0bb7169e901c6  internal/modules/identity/service/deps.go
20649228a7b7cacce5555a75efb16c35fef7770d4e5eb93c480726328dc378f0  internal/modules/insurance/service/insurance.go
4f1187db7d323fbe83f0e2711b02104e2f6db1e2f708fba79aebbad474505a58  internal/modules/inventory/module.go
f618ab0d9edd695578f67327dce3c6df067c734e2cf2d9a21a34a616f2d60d9b  internal/modules/inventory/repository/reservation.go
5d91771af595f3f2add6218a964c757f275ebe7a89653315fce504a2ba589c19  internal/modules/inventory/repository/vehicle.go
ac6439e9c8224bb1e47b0c04a2dec8a77b4cc045320740f3fc6f475a443a2436  internal/modules/inventory/service/ports.go
102fa608d574c316f7c9fe52730a31747073895370296a7bb66733f50995cce8  internal/modules/inventory/service/receipt.go
6eec5de7445a17badf4d9fe2c28b36ffefd369a3eb08b0ac56d1bc88c9c5fb75  internal/modules/inventory/service/receipt_colors_test.go
69c23eba951a9cbf0f0bd68e51da3ec0132e1eb49ef582c902c0b4ceb9e154b7  internal/modules/inventory/service/stock.go
b125c0495d6397d547155a782b88667c4b9beee085579917fe71b318f5e417f2  internal/modules/retail/handler/dto.go
0587421739a845c8f1a8225c6b58f50a8cc68b1286cfa985ebaace47efaf9515  internal/modules/retail/handler/handler.go
b88f7bf9bc98df63d7c70ba8232edc60845f3eeff7698cfb54cdaf98a514bf10  internal/modules/retail/model/customer.go
3e2630a0effaa21ff5b038add8a187658bbe5cb8c60bb2e766485882f12a62db  internal/modules/retail/model/permissions.go
29dcc04134b6e90af8a505f943ed7007d242ecdcd71018fe431838a8c67cfcac  internal/modules/retail/module.go
8f5fdda7944fdf2728b438733f2ba539a2535189af5964966806eae4d7a8a7fd  internal/modules/retail/repository/crm.go
4e6560abea06f8dece295314ec318453e9ab7b795e5470976b9d6ff308365a5d  internal/modules/retail/repository/store.go
6e57f468c248ebe9247998c3082367ca9d2d8527bd02e8dc0981789a804dba74  internal/modules/retail/service/cash_sale_test.go
bffebd0de278d27d375fc79b243135391c856bd704cfca31e5f080ce3e7733b1  internal/modules/retail/service/crm.go
55d3201ccb1fb7799bd4f1723517a71512a27cb73a07692bdb49aa8f7f63a4e5  internal/modules/retail/service/deal.go
123411ca54ba918ef4a9941e7ca2901560c00928806d6fd96693c754e05deaec  internal/modules/retail/service/deal_colors_test.go
80bf08193a800f0643767aacacc25949b7e76ade9255730bbbea7925aa2a78e9  internal/modules/retail/service/deps.go
7b50fc1310d09b7b7fcb944693a171bb6470d4d845ca254a4434f6e2df6a6c09  internal/modules/retail/service/installment_payment_test.go
c01b6e481e5cf696fa97c8b2964c24026d2e2771404325c698d414df45cd0447  internal/modules/retail/service/installment_test_helpers_test.go
57f0c1f04e61d99b1bb7fa17ee1896ed23d53a82595ac3341302f0a416738d71  internal/modules/retail/service/listing.go
6ec63fa26934d3d7cf310489731eec8545ffff6809938b6eeb4b4b85e72d272f  internal/modules/retail/service/ports.go
099c48b63042c401c34207e8130fe72f9e027e93973de73c045e2f4c48ac99b6  internal/pkg/auth/principal.go
3918d6c6a7ed3a7c69432e00d4bc212a9d56d2808f05e7e6c5e672a638380ce2  internal/pkg/auth/principal_test.go
4e0df61f4497e86e412f10a57e232366d4c15864ecba5c6259c2a2d586785e9a  internal/pkg/database/lock.go
d91776a00195f9d615d1f0c73d4f8410b3cc53ae8ad284021e7146535c286ab4  migrations/000035_company_verification.down.sql
bb8cd6ba9b409b8810fd7c3039c486ec26fe24e7b6328b4c5f1b043010f88919  migrations/000035_company_verification.up.sql
b8dee411a0a1043a038d72ccde611f5d12e70a6065a3d60eb64fde72600bdd02  web/AGENTS.md
ae089cb8982807de41739d179adb5420f578ca7f180243188873bee30cfac163  web/apps/admin/src/company-form.test.tsx
4c725ed0b2b28e7d79ebafdca74d0e3fb1edd6eb5b82e66d834bacbdc769307f  web/apps/admin/src/labels.test.tsx
8fdceb70d814b2934a1dde944741e9518122ba6820f6b42576996b57ebb3b75c  web/apps/admin/src/labels.ts
0d3cdac7fc204f19ac1378403eab9deb46cd0b28dbf9d683689658405e24c318  web/apps/admin/src/main.tsx
56a66c4aaf660189fd9ca402cff09a84749e96e09da317d87732202d5ea9a1f5  web/apps/admin/src/pages.tsx
15e47aeb1f71321998ece5b16b37e3494aac87ae795f64bfbf92b55fee59a919  web/apps/financing/src/main.tsx
f68631ec7463829e356311eba0772ab80f76116c4013e1d0e8457b65fe78092a  web/apps/realization/src/add-company.test.tsx
fa8615d87549b71c4c96df12459d4daf3ce0cb8c1b37c6bb0a5ee671b5dfaa38  web/apps/realization/src/data.ts
6825f75a29e5f35245994c2d55721204cef3428395fcd99f1ec2e20c69618f97  web/apps/realization/src/installment-calculator.test.ts
a7c1bea2718cd8b14bf80d3a3c3f88240a3b217ba92ef6b7fee3db31d27cb6a1  web/apps/realization/src/installment-payment-review.test.tsx
f6a7bf07f956d3bba7d4e1bac0de7fdf2e187506b6d00fbaac0047fb1c7cb43e  web/apps/realization/src/installment-payment-review.tsx
8aec9e16069a319826acd73aca7d649e02c1f387ee41fb2bc98907398a8c6186  web/apps/realization/src/installment-terms-form.tsx
c86ca92f6014c83cb7123ac2265936c6d93e73f62f063dd96739dddc3e57141e  web/apps/realization/src/navigation.tsx
61532b42ab99347c4f126df52d6599b04735dfa742a125a3fc0cbdf8c5eb3698  web/apps/realization/src/order-actions.tsx
d808dc72c51466d77b0e34e5c56cc97b426174e70ec9b466b5d733e803426b1e  web/apps/realization/src/order-create.tsx
ec8d62b8d1a6bc5a9fb85d2d43d2d85c2d7320feb6a45122d7de41c06d483c9c  web/apps/realization/src/order-fulfilment.tsx
aa32dfaa08e74c9557a81d0e95bd1df87c4ce2cec721fd74cd1688c44a80bc44  web/apps/realization/src/order-workspace.tsx
f7c48efd287614fe891b2f50abd0cfc4acbeb30043b9455a3754278e26a9a3ca  web/apps/realization/src/pages/billing.tsx
ab727a197c39d3ad0e73f38502af53c9a4c9620179c8e73bdcbc953c0e0dfe4a  web/apps/realization/src/pages/crm.tsx
e147823928ad333fc780e65a41b0823f605dfc011f8884d2005fa7d15b9b6328  web/apps/realization/src/pages/inventory.tsx
231b206189d8566a4fe6949ddad111324f011f390a2c5c1f9dbf31fcf4038e66  web/apps/realization/src/pages/offers.tsx
5197f9f33021bc0f634ed30a285f878de6fc7ed3932c7162b1af42d3cb5c17fc  web/apps/realization/src/pages/partners-finance.tsx
ee81016a83deeb423e5e5c17f97b8b9b067d15285b915ab00762e540907a541d  web/apps/realization/src/pages/purchases.tsx
2fb85cad172cd4698ca5b7914ceb1fd8c1b5e24647c194367a4009a0e319cc3f  web/apps/realization/src/pages/retail.tsx
f7fd71f9a96ece4585ca4c62586bf78a1a66f110b01d81d58a97e189531a164f  web/apps/realization/src/pages/sales.tsx
7b664c79a1496aee970a65cabc4c84798192a147078254d48597bef6d4a1344f  web/apps/realization/src/pages/settings.tsx
21f8204a1ee363ac91a39305e3bd32b9e17f56fae8b2f7de3e857dd53cc81aa9  web/apps/realization/src/pages/stock.tsx
3d0a0ae21537271bc2efcec73245c0eb64d44423ed456dbda2bf131fe9492663  web/apps/realization/src/pages/trade.tsx
ff385d4d33baf40c86b6a3d7c71c0712c2b553ec6694ba3aa6ef613d2f07e604  web/apps/realization/src/retail-actions.tsx
40f5b3527dd6f1651bbdc567174495f2ff4a5392423cf8c64a068a4291a64d4b  web/apps/realization/src/sale-form.test.tsx
b36cf32fff9bfe80babf4652cae2e287bfda84ea4e38ee781cf3978e6fd5ac33  web/apps/realization/src/sale-form.tsx
792f2b5da2487ece2bb0e895cc108bc61f0b2b324612d06531a6d56a11589a97  web/apps/realization/src/sale-offer-form.test.tsx
753a082c2a69928c501f9c348c9714c6aed88117283c5913d8f00565e26573d3  web/apps/realization/src/terms-dialog.test.tsx
081a98f5f8e7730c6a11a51d313641f80923ff840ba59638b64110979d0bc51d  web/package.json
e1032226c63a5703b142373ca16d38b5c2f8312bb7486705ae5caf3a7a080d75  web/packages/kit/src/applications.tsx
5060b596ecf49329ed7c82a3d217514e4f33625f36cd117813f617e6e50f8776  web/packages/kit/src/messages.ts
1b3be33d18f17aef6f5b6e5a27c1bf24283f9830b7e05836f1315c24e76b052d  web/packages/kit/src/org-settings.tsx
7674d51bace30bc11f530d2a677e8fed99a5140e96b34b3356a7f8727d9ef7fc  web/packages/kit/src/permissions.test.ts
7990e28af28805c91c6a66753bd8153f70fc006a5412b95470189ac1c2891314  web/packages/kit/src/permissions.ts
d693c96917c4da96bd256b1fde966b61c20abb07145b79a40773eb84a97275c7  web/packages/kit/src/session.tsx
c94c21f6dfa2ff9b49c863ae9538f79e5f6f56abb1d847b27fb7e3a2c5b88a10  web/packages/kit/src/shell.tsx
```
