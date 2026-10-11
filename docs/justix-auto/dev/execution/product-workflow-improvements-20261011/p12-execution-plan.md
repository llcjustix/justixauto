# P12 bounded execution proposal

Application hub /root; canonical task product-workflow-improvements-20261011, requirement R9, original P12. Supplied base HEAD remains 4fa683ba4708bb71031534260ec5cb8742ce9977, feature/company-commerce-workflows-20261011, with current shared uncommitted work preserved. No Git operation was performed. This is a proposed execution split for hub acceptance, not a new canonical task, source review or QA gate. Original plan/packet and product decisions are unchanged.

## Scope and sequence

| Packet | Behavior | Context ceiling | Dependency |
|---|---|---|---|
| 12a1 | Complete private B2B draft→signed lifecycle, exact snapshots and transactional persistence | 17k | Accepted P11c1–c3 result interfaces; source-writer lease free |
| 12a2 | Production actual-order authorization adapter, authenticated HTTP and structured DTOs | 15k | Accepted result-12a1 and final P11 DTO handoff |

Two sequential fresh worker contexts, one source writer. P11c2 currently owns source; these artifacts do not authorize concurrent writing. Read root AGENTS.md, agent-workflow.md, internal/AGENTS.md, this common plan and assigned packet plus only targeted source. Before application edits worker runs bash tools/check-git.sh. No Git writes by workers/slicer; hub/version_control owns branch operations. Preserve dirty state.

Both packets remain within original P12 ownership. Explicit proposed unit-only additions are internal/modules/commerce/service/contract_test_helpers_test.go, internal/modules/commerce/contract_wiring_test.go and internal/modules/commerce/handler/contract_test.go. No new production path beyond original P12. Current existing service fixtures embed Store, so no fixture edit elsewhere is presently necessary. New DTO definitions fit owned handler/contract.go; do not add handler/dto.go ownership.

Service-only a1 is deliberately incomplete at product level. R9/P12 cannot be called complete before a2 production authorization/wiring/HTTP and its affected units. No broad source inspection, infrastructure change, live DB, migration execution, browser or deploy required.

## Targeted findings and exact boundaries

- Current service.Store/repository.Store/storeAdapter have no Contracts. Add the typed method in a1 across the actual production seam, with compile-time conformance and an inert repository unit. Do not copy P11's initial unchecked assertion defect.
- Current model.Order.Terms is JSON bytes; DecodeTerms returns fixed model.Terms and would discard future fields. Contract capture keeps complete accepted structured JSON, typed known values only for text formatting, and no float64 conversion. Include real counting allocation VINs, not a current ownership lookup or fabricated future vehicle.
- Existing Deals().Order scopes actual buyer/supplier; UpdateOrder performs optimistic version CAS. Use unchanged order economics plus version bump in the same transaction as draft CAS/event/file grants. Actual simultaneous database behavior is not proven by fake units.
- Identity ContractRequisitesFor only permits equal authorized/target IDs. App must first resolve and validate actual order parties through a narrow commerce-root source; only then request each target's private projection. Public CompanyProfile remains untouched. A typed adapter factory injected at construction avoids a circular partially initialized module and unchecked runtime assertions.
- Existing Files.Share checks owner and joins bound context. Draft external documents bind to author only; signed files grant the actual counterparty. Grants, contract state and order revision rollback together. New signed scans are explicit, retained external IDs are not discarded.
- Final P11 instruction supersedes original P12 phrase “escaped content”: templates and filled preview are plain text, preserving A&B and angle brackets exactly. Literal replacement, normal React text presentation, no HTML/expression engine/legal drafting.
- Technical lifecycle follows accepted P11 semantics: same draft row becomes signed once, signed ID equals draft ID; documentary amendments consume a new draft with explicit priorSignedVersionId. No edit to signed records, no mutation/acceptance of economic addenda. Existing economic order acceptance stays authoritative.

Private drafts/templates belong to recording company. Actual buyer and supplier read signed history; authorized recording side records an externally signed paper fact. No browser e-signature, supplier/dealer exclusivity change, new financial or legal policy is introduced. Existing-obligation documentation and history stay accessible under existing authority rather than a new verification gate.

## Acceptance coverage

| Original P12/R9 acceptance | Implementation/evidence |
|---|---|
| Configurable company templates, filled preview, editable/external draft | a1 actual service lifecycle/plain-text/ownership units; a2 endpoints and input semantics |
| Both actual order-party private requisites | a1 typed snapshot port and service party guard; a2 independently authorized actual-order adapter + TestWorkflowPort12 |
| Full accepted terms, future fields, model lines/VINs, source provenance | a1 deep-copy raw JSON and exact snapshot units; a2 structured DTO preservation |
| Same-row consumed draft, immutable signed versions, amendments | a1 state/revision CAS and rollback units; a2 explicit source IDs/revisions |
| Private files, scan requirement, actual counterparty sharing | a1 bound-context owner/grant/failure units; a2 IDs-only arrays and no storage data |
| No economic-order/invoice/payment/inventory mutation | a1 existing order CAS with unchanged terms, service assertions; a2 no terms mutation input |
| Buyer/supplier history; outsider/read/write permission denial | a1 guarded services; a2 route requirements and structured list/detail |
| Production construction and UI-ready interface | a1 compile-time Store/adapter pure unit; a2 actual adapter/DTO/If-Match units and final endpoint manifest |

Proposed endpoint/DTO/revision contract is explicit in packet-12a2. Its result must pin actual paths/inputs/statuses/fields for P13. Order-owned create/edit/sign/amend header is order If-Match; edit/sign/amend additionally body draftRevision. Contract command/detail envelope revision is contract revision; orderRevision returned explicitly. Template PATCH uses template revision. Structured party/terms/vehicle values and file ID arrays must not be raw []byte/base64.

## Verification policy and prior evidence

User policy: only affected pure unit tests, once per relevant change, pinned Go via tools/go.sh and -vet=off. Use writable GOCACHE shown in packets. No automatic review/QA or final-SHA integration loop, lint/typecheck/build/vet/race/browser/E2E, broad app checks or runtime actions. Slicer ran no tests.

Prior P11/identity/app units remain historical evidence for unchanged retail/requisite behavior; do not rerun them merely for P12. P12a1 adds actual transactional service tests and pure production Contracts seam; a2 adds app TestWorkflowPort12 and handler units. a1 units stay valid if a2 does not change service/domain; no repeat solely because a2 follows. Existing unrelated commerce tests/green results are not a substitute for these new behaviors. Extra fixture/source needs return to hub for exact ownership, not speculative widening.

## Gaps and risks

No newly discovered product decision gate. Original scope stays R9/P12, no canonical backlog expansion or RESLICE_REQUIRED product-scope change. Hub acceptance of this bounded execution proposal is pending. P11 completion and its actual interface result remain technical dispatch dependencies.

Unit fakes exercise intended transactional failure/CAS logic; deployed SQL races, schema application and documents transaction behavior remain unverified under user policy. No guarantee of release readiness. Snapshot hashes below are observed working-file inputs, not a clean-tree claim; active P11 edits can change shared app source before a2. Consume predecessor's final narrow seam rather than overwrite it.

## Packet SHA-256

```text
3069ecb49ded869077c672f5b8af2c2b87446175814aff90135293fb142081ef  packet-12a1.md
75e317e3e8c1ae12ddcce903ffa74196aeffb36c4f77ace71fe04b9b855795fd  packet-12a2.md
```

## Targeted source SHA-256

New contract model/service/repository/handler and migration 000039 did not exist at preparation. Source observations below provide recoverable exact-input identifiers; original accepted artifacts are not rewritten.

```text
abf19187364f584fc0be82122bf64657d3b20613ed15db47b0cb1fa08f8f3a7b  docs/justix-auto/dev/execution/product-workflow-improvements-20261011/packet-12.md
27b0eb239a33489427fe71ffd9c16c0d1de0213731b0d370bb199a7bd181a4b3  docs/justix-auto/dev/execution/product-workflow-improvements-20261011/p11-continuation-plan.md
b22213a76ca11a4ce4aeb04323260d1d577f28ef1230d48c15367d074e67650b  internal/AGENTS.md
255fb0111651c953f8514e5a588ea6c95077c5eef5c96d882d5081b1336b5adb  internal/modules/commerce/service/ports.go
b0b81742a2478f0edce53f9db85b2eaa0a4853c92bcfcd080826ae5e56c242b6  internal/modules/commerce/service/deps.go
4b8fd87b0b74f39236d6ffaa4fef7f2e5104bed968bdc87f139a65b4d4b6aa0c  internal/modules/commerce/service/deal.go
a71e3dfd2bc378d2638c1ae33f8686f0c3026b24c3d4e0d5247075b51072c1f2  internal/modules/commerce/model/order.go
5dd4bbc7afd9030915a583449d16ec73a2ac83fe07af94f8ad163fc17872c2a2  internal/modules/commerce/model/offer.go
978e314a8e1ea32d5be1e0b0c374598cb2ae2966063746235302c90431325a41  internal/modules/commerce/model/fulfilment.go
f558adf2f4bbd43bf40fccece7f8754dc04d047ad72724e997d3d6d19181156b  internal/modules/commerce/repository/store.go
e5f8103891c55eea811d8b9b022de87a397a1bbb96822d093cb9f9dd666da65d  internal/modules/commerce/repository/deal.go
af3888c84c47b7456d5b74d3dfaa662a399fe56b25830811cf24df45fe4605e8  internal/modules/commerce/module.go
3a28f7f6222b4c6cfb6fa11d15337715a9f7488dcfd299c284e5fdaac99edc7e  internal/modules/commerce/handler/handler.go
2734ae80ce6caf536c70b5553f2113b7d528e9c01a38b41ae0a2b074de8b08c1  internal/modules/commerce/service/terms_colors_test.go
407e7faf3a144a0858b65b881b32553216bb89d2b4e34f01d6308a895019e6fc  internal/modules/commerce/service/fulfilment_colors_test.go
f0ec517a94141a6c924be5bde8d564d9fd4ef991cd5549c3040f7a8ada63bb10  internal/modules/identity/service/company.go
a21433865a477d3c713d5b0c729a94803f55ad1e7396079c864520beea4d9ab5  internal/modules/documents/service/documents.go
452f6ad517c4e75eef221792e453c9a8e4ea93eec666443a0d34bc452a839ab8  internal/app/app.go
b0d2b7cd52b869c98818ca18a4e75547658d4ef5fc76dcdaa091626965090bd4  internal/app/workflow_ports_test.go
```

