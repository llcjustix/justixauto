# P12a2 proposal — Production party authorization and structured contract HTTP

Canonical R9/P12; depends on accepted result-12a1.md and final P11c3 DTO/revision handoff. Read p12-execution-plan.md common rules. One coherent behavior: expose the completed B2B contract lifecycle through production-authorized routes and explicit JSON. Context ceiling 15k. Exclusive source-writer lease; no runtime resources.

## Exact owned paths

- internal/modules/commerce/handler/contract.go (new; include contract DTOs here)
- internal/modules/commerce/handler/handler.go
- internal/modules/commerce/handler/contract_test.go (new pure handler/unit fixture addition)
- internal/modules/commerce/module.go
- internal/modules/commerce/contract_wiring_test.go (a1 unit addition, construction/alias checks only if needed)
- internal/app/app.go
- internal/app/workflow_ports_test.go

No handler/dto.go production extension: define this feature's DTOs in the original-owned contract.go. No changes to identity, public company projections, existing commerce order DTOs or economics. If a1 interface is insufficient, report exact seam to hub rather than widening this packet silently.

## Production composition and authorization

1. Wire real Contract service and handler; do not accept a construction that leaves Contracts missing or uninitialized. module.go exports only necessary aliases for private ContractParties, CompanyRequisites and actual-order party lookup. No app import of commerce/service or model.
2. Adapter must validate against actual stored order: accept viewerCompanyID + orderID; call a narrow commerce-root OrderPartiesFor source returning actual buyer/supplier; independently require viewer is one of those IDs before invoking identity. Do not validate against request-supplied party arrays or saved client draft values. The service also performs its own order-party check.
3. Use existing identity.ContractRequisitesFor for each validated target with authorizedCompanyID=target and companyID=target ONLY after actual-order authorization. Map every requisite field exactly into commerce snapshot type. No public Directory expansion, identity table query, arbitrary company lookup or caller-chosen target forwarding. Preserve ambient bound transaction context.
4. Avoid a circular half-constructed Module: a typed factory constructor argument may receive the module-created narrow order lookup and return the app's ContractParties adapter before NewContract/handler construction. module.New creates lookup from existing Deps; app passes a factory closing over identity. This is a small composition seam, not new infrastructure. Use compile-time interface guarantees; no unchecked type assertions. If an equivalent already-initialized noncircular typed shape is simpler, preserve these invariants and record exact exported signature.
5. Routes use RequireCompany and commerce.read/commerce.trade; service guards from a1 remain authoritative. Template ownership and actual party access cannot depend only on route guards.

## Exact proposed endpoint/revision contract

All paths are below /api/v1/commerce. Pin final implemented method/path/inputs in result-12a2.md for P13; follow accepted P11 vocabulary where possible without changing existing order endpoints.

- GET /contract-templates; GET /contract-templates/:templateID; POST /contract-templates; PATCH /contract-templates/:templateID. PATCH If-Match is template revision. Create/edit name + plain-text content (map internal Body explicitly).
- GET /orders/:id/contracts; GET /orders/:id/contracts/:contractID. Return only actor-visible own drafts and both-party signed history; no outsider detail leakage.
- POST /orders/:id/contracts/preview: input optional templateId/content and explicit desired snapshot source; service authorization and actual order snapshot, plain-text filled result, structured snapshots. Preview is read-only, creates no grants/rows/order revision; use the service preview operation from a1. It cannot accept client-supplied requisite/economic snapshots as truth.
- POST /orders/:id/contracts: draft input optional templateId/content/bindingIds; If-Match=order revision.
- PATCH /orders/:id/contracts/:contractID: body draftRevision plus optional content/templateId/bindingIds/refreshSnapshot; If-Match=order revision. Preserve omitted versus explicit-empty semantics. Only own draft editable.
- POST /orders/:id/contracts/:contractID/sign: body draftRevision,bindingIds,signedOn,reference; If-Match=order revision. Initial signed ID is this same contractID, no second source row.
- POST /orders/:id/contracts/:contractID/amendments: consumes this NEW draft contractID, body draftRevision,priorSignedVersionId,bindingIds,signedOn,reference; If-Match=order revision. priorSignedVersionId belongs to same order and is already signed. No economic terms input/change.

Contract response includes contractRevision and current orderRevision as decimal strings, ID/orderId/recordingCompanyId, state/kind, templateId/templateRevision, priorSignedVersionId, content, fileIds[], partySnapshot object, termsSnapshot object (complete accepted JSON), vehicleSnapshot array, acceptedTermsDigest/sourceOrderRevision, signed/reference metadata and timestamps. A saved draft and its signed result have same id; priorSignedVersionId is documentary ancestry. Use explicit DTO mapping with json.RawMessage or typed JSON for structured snapshot values, never []byte base64/GORM fields. File IDs are JSON arrays, empty arrays where appropriate, no storage implementation details. Contract command/detail httpx.Data revision/ETag is contract revision; explicit orderRevision is what P13 sends on next order-owned action. Template response envelope uses template revision. httpx.List for lists; preview envelope revision may use source order revision but name it orderRevision in data and document distinction.

Do not silently alter any preexisting order response, action If-Match, or economic-addendum request/response. New POST create returns created contract DTO, mutation/detail returns contract DTO; signed click records paper signature only. Plain text remains exact bytes/characters; downstream React uses normal text rendering, no dangerouslySetInnerHTML.

## Meaningful pure units and report

TestWorkflowPort12 in existing app test file: fake actual-order lookup + identity fake. Test viewer outsider denied before requisite query; tampered requested order/returned parties not used as arbitrary targets; both actual sides receive exact full buyer/supplier requisites; identity error propagation; context marker retained; public CompanyProfile is never used. Test module constructor factory receives typed initialized lookup if changed, without opening DB.

Pure TestContract handler cases invoke thin request handlers/service fakes or pure parser/DTO mapper as appropriate: distinguish header order revision from explicit draftRevision; invalid/missing/stale revision behavior; unchanged contract ID on sign; priorSignedVersionId on amendment; own versus cross-order routing; special plain text preserved; structured unknown nested accepted-term keys and large exact numeric values survive JSON; arrays are arrays; no base64/raw storage fields. Validate route/auth registration with inert handlers/ports in process if needed, never app.New/e2e.DB or HTTP server/integration. Domain authorization/rollback is covered by a1 actual service units; do not pretend mapper tests cover it.

Run once after relevant change, from repo root:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort12($|_)'`
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/handler -run '^TestContract'`
If construction/wiring pure test changed: `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce -run '^TestContractStoreWiring$'`.
No service rerun if service unchanged. No broad app tests (OpenAPI uses DB), lint/build/integration or review loop.

Result must supply actual endpoint table, success shape/status, If-Match resource, explicit revision fields, draft/signed/prior source IDs, exact placeholders/snapshot schema, pure unit evidence and known deployed-concurrency limitations. Stop on unowned production changes or unresolved product rules, not a generic approval checkpoint.

