# P12a1 proposal — B2B contract lifecycle, persistence and exact snapshots

Canonical task product-workflow-improvements-20261011, R9, original P12; application hub /root. Read p12-execution-plan.md common rules and original P12 only as needed. Dependency: accepted P11c1–c3 result interfaces; no source dispatch while P11 writer holds lease. One coherent behavior: persist an editable private contract draft and consume it into an immutable signed order document. Context ceiling 17k; stop with a compact checkpoint before exceeding it.

## Exact owned paths

- internal/modules/commerce/model/contract.go (new)
- internal/modules/commerce/service/contract.go (new)
- internal/modules/commerce/service/contract_test.go (new)
- internal/modules/commerce/service/contract_test_helpers_test.go (new unit-only fixture addition)
- internal/modules/commerce/service/ports.go
- internal/modules/commerce/repository/contract.go (new)
- internal/modules/commerce/repository/store.go
- internal/modules/commerce/module.go (only Contracts store adapter/conformance here)
- internal/modules/commerce/contract_wiring_test.go (new pure unit addition)
- migrations/000039_commerce_contract_versions.up.sql (new)
- migrations/000039_commerce_contract_versions.down.sql (new)

No HTTP/app wiring in this packet. Existing colorsStore embeds Store and derived fixtures inherit it, so no existing fixture source extension is presently necessary. Do not edit other tests speculatively. New helper fake must implement the used methods, not rely on nil embedded methods.

## Domain and repository contract

1. Add commerce-owned company templates and order contract rows. Capture ID, order ID, recording company, template ID/version, priorSignedVersionId, kind/state, plain-text content, external/signed file IDs, reference/signed date, complete party/terms/vehicle snapshots, accepted-terms digest/source order revision, author/timestamps and contract revision. Add schema constraints/indexes for tenant/order history and permitted draft/signed transitions. Migration 000039 is new; do not edit 000038 or execute migrations.
2. Declare Contracts() on actual service.Store, implement repository.Store and production storeAdapter with compile-time conformance. No unchecked optional repository assertion. Repository methods explicitly scope template owner, order/document identity and revision. Use actual contract revision column in CAS, not a helper assuming version.
3. Own drafts are editable; saved signed documents are immutable. Sign changes the SAME draft row draft→signed with expected revision. Signed ID equals source draft ID; never create a signed clone leaving reusable source. A documentary amendment consumes a NEW draft and links explicit priorSignedVersionId from a signed document of this order. Preserve every prior signed snapshot/file association. Serialize initial document/amendment acceptance with order optimistic revision; reject repeated initial signing once the order already has its base signed document. Do not add a digital second-party signature workflow.
4. Each public service read checks commerce.read; each mutation checks commerce.trade plus actual order-party authorization. Company-owned templates/drafts remain private to their authoring company; actual buyer and supplier can read signed history/detail, including records authored by the other party. Both are eligible recording sides. Unauthorized callers cannot enumerate private snapshots. Existing-obligation documentation/history must remain serviceable under earlier verification decisions; do not add a blanket new-commerce eligibility gate here.
5. Use existing Deps plus a separate narrow ContractParties port in ports.go, injected into NewContract in this new service file. Do not extend public Directory.Company or edit deps.go. Define typed private requisite snapshot fields: ID/name/legal name/registration number/tax ID/bank/address/phone/email. Request by authenticated viewer company + actual order ID; no client-selected company pair. P12a2 supplies authorized adapter. Also define a small OrderPartiesFor(ctx,viewerCompanyID,orderID) lookup using the scoped Deals().Order and rechecking Party; export only through module-root aliases in a2. It reads actual stored party IDs, not draft snapshot IDs.
6. Snapshot exact accepted Order.Terms JSON, including unknown/future keys and nested fields. Do not round-trip only through model.Terms, which drops future reservation fields. Validate JSON; preserve numbers without float64 conversion; store a deep copy and stable digest of accepted JSON representation. Use structured JSON in later DTOs. Known fields include lines/model/specification version/colors/quantity/unit Money, route/delivery/payment schedule/warranty/service. Snapshot currently counting allocations' IDs/line IDs/VINs/status from commerce Fulfilment().Allocations; do not query inventory ownership or fabricate VINs for future quantities. Capture source order/offerVersion/quotation references. No pricing, quantity-cap, schedule, invoice, order-addendum or inventory changes.
7. Signing compares saved accepted-term snapshot against current authoritative order terms inside the transaction. Changed economics require an explicit draft refresh, never silent regeneration at sign. Draft edit exposes refreshSnapshot explicitly; refresh rereads actual party/terms/allocation data while body remains explicitly editable. Snapshot creation/refresh retains exact template provenance. Later template/company/order changes never rewrite signed rows.
8. Supply service template create/edit/list/detail, authorized read-only Preview, and contract draft/create/edit/list/detail/sign/amend operations needed by a2. Preview uses actual party/order snapshot and creates no rows, file grants or order revision. Templates and filled previews are plain text. Literal supported placeholder replacement only; preserve A&B and angle brackets exactly. No HTML escaping in stored text, executable expressions or legal prose. Fill available buyer/supplier requisites and known accepted line/payment/delivery fields; hand off exact placeholder names. An explicit edited content value overrides generated template body without losing template ID/version. Allow external-document-only drafts; reject final state with neither content/template-derived content nor files. Draft edit distinguishes omitted fields (preserve) from explicit empty (clear if still valid).
9. Each contract command uses expected order revision; edit/sign/amend additionally require expected draftRevision. Template edit has its own revision. Create/edit/sign each bump order revision exactly once by existing Deals().UpdateOrder with unchanged economics/status, and draft revision as appropriate. No stale-pointer overwrite: perform snapshot/read/CAS/order update in one Store.InTx using bound context. An order revision conflict after draft CAS must roll everything back.
10. Draft external-file ownership/binding uses Files.Share(boundCtx, author, fileID, author, resourceType, draftID): no early counterparty grant. Signing requires explicitly supplied signed-scan binding(s), retains draft external IDs, deduplicates final IDs, verifies author ownership and grants only the actual other order party, bound to SAME signed ID. All file grants, contract CAS, order bump and event write are one transaction. Injected failures must rollback. Never expose public URLs/storage keys or attach unrelated company/customer documents. Files.Share joins through st.Bind(ctx); pass that context to cross-module requisites lookup too.

## Meaningful pure units

TestContract cases must invoke actual create/edit/sign/amend/list/detail services, with deep-copy transaction fake state for contracts/orders/events/file grants. Cover:
- outsider and missing-read/write permission rejection before private lookup; each actual side reads signed history; private template/draft isolation;
- exact requisite fields, entire nested future Terms JSON, integer Money and real allocated VINs; future line without VIN stays so; mutations of source objects cannot change saved snapshot;
- template edit/body override/external-only draft, empty rejection, exact special characters and nonexecuting placeholder-like values;
- persisted edit, omitted/empty semantics, explicit refresh and stale accepted terms rejection;
- same ID consumed once, stale draft and order revisions, second initial signature rejection, new-draft amendment preserving old signed bytes and file IDs;
- file owner/counterparty/resource/context exactness; pending draft causes no counterparty grant; retained external file plus scan union;
- failures at file sharing, contract CAS, order CAS and event write leave no partial contract/order/grant/event changes; fake rollback cannot just call fn(s) on shared pointers;
- all documentary actions leave Order.Terms, economic addendum records, invoice/payment/stock state unchanged.

Add pure TestContractStoreWiring in module root: construct inert repository store without opening DB and exercise storeAdapter.Contracts availability/type; no database calls. Compile-time Store assertion additionally protects production wiring.

Run affected units once after relevant change, from repo root:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestContract'`
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce -run '^TestContractStoreWiring$'`

No unrelated service reruns, migration execution, DB/build/lint/review/QA. Stop for genuine missing product rule or unowned production path; otherwise report exact interfaces, snapshot schema, source IDs, transaction/CAS rules, meaningful unit evidence and exclusions in result-12a1.md. Normalize newly edited Go normally, no formatting-check loop.

