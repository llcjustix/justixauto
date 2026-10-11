# P11 bounded completion plan

Application hub /root; canonical task product-workflow-improvements-20261011, requirement R9, original packet P11. Base identifier supplied by hub remains HEAD 4fa (original full base `4fa683ba4708bb71031534260ec5cb8742ce9977`) with current uncommitted feature work preserved. No Git inspection/writes. Source writer was free at assignment; hub owns dispatch and one-writer lease. This continuation does not rewrite accepted plan/packet, board or approved contracts and does not add canonical backlog tasks.

Status: three ready sequential implementation continuations. Original result-11 RESLICE_REQUIRED reflects worker context exhaustion; this decomposition stays within existing P11 acceptance and does not change canonical product scope. This is implementation preparation, not an independent review/QA gate.

## Targeted findings and bounded completion

| Packet | Coherent behavior | Principal unfinished acceptance addressed |
|---|---|---|
| 11c1 | Production repository wiring and sign-once transaction | Current contracts(st) unchecked assertion lacks production storeAdapter.Contracts; Sign clones without consuming original draft; service list lacks read guard. |
| 11c2 | Persisted editable drafts and complete private plain-text snapshot | Draft lacks upload/edit; snapshots omit installment terms and most requisite/profile fields; current HTML escaping is wrong for approved plain-text rendering. |
| 11c3 | Legacy compatibility, installment finalization and DTO seam | RecordContract still overwrites metadata; new Sign never invokes installment finalization; handlers return raw model/[]byte; old response must remain dealDTO. |

11c1 → 11c2 → 11c3. Context budgets 14k/16k/16k, fresh worker contexts, exact paths in each packet. Shared contract/service/DTO files serialize. If a further concrete missing fixture path appears, hub authorizes only that path; do not load full retail module or whole diff. Normalize edited minified files as normal editing without lint/build/check loops.

## Chosen lifecycle and technical seams

Existing 000038 schema already supports mutable draft → immutable signed. Convert the SAME draft row to signed through company/deal/id/state/revision compare-and-set; signed ID equals draft ID. Never modify/delete an already-signed row. A new amendment uses a new draft row and explicit priorSignedVersionId. This prevents reusable source drafts without adding an unnecessary schema field or rewriting migration. Unique stale/conflict behavior and transaction rollback are service/repository responsibilities; fakes do not prove deployed SQL locking.

Actual service.Store declares Contracts; production adapter implements it. Existing fixture Stores receive minimal interface additions. No optional unchecked runtime assertion or bypass. Repository predicate must use actual `revision` contract column, not assume generic version-column helper applies. Template rows retain their existing version column.

External uploaded contracts and signed scans use current private Files.Share with bound transaction context. Draft file set is retained when signed scans are added; required new scan is explicit. Snapshot includes existing optional profile fields but does not automatically copy/share passport-copy attachments. No public URL/storage key/logged personal payload.

Plain-text template/content preserves exact source characters. Literal replacements only; no HTML rendering or expression evaluation. Snapshot uses authoritative generated installment draft or immutable plan/obligations, never newly recomputed terms/current paid balances. Signed records preserve their snapshot even after company/customer/template edits. Economic terms changed before signing require explicit draft refresh.

Legacy RecordContract creates signed version through shared transaction logic, preserves response and finalizes schedule using current established prerequisites. Already-signed changes require explicit amendment, not repeated base-contract overwrite. Historical signed rows with missing scans/empty old snapshot remain known legacy records; new signatures require scan. Migration 000038 remains unchanged; no invented history.

## Explicit HTTP/revision handoff to P13

- Legacy POST /retail/deals/:id/contract-records retains request {bindingIds,signedOn,reference}, If-Match deal revision, success 200 dealDTO through dealResponse and its deal revision/ETag.
- Template PATCH uses template revision.
- New deal-owned draft create uses If-Match deal revision.
- Draft PATCH/sign/amendment use If-Match deal revision plus explicit body draftRevision. Commands return distinct contractRevision and current dealRevision so frontend never uses contract revision as the deal action header. Amendment adds priorSignedVersionId.
- New contract/template list/detail/command responses use explicit lower-camel DTOs. Files are ID arrays, snapshots structured JSON objects, decimal-string revisions; no raw []byte base64 or GORM models. Contract response envelope revision is contract resource revision; dealRevision is explicitly available for next action.
- Add authenticated contract detail route if absent. List/detail permission and company/branch enforcement occurs in service. New list envelopes use httpx.List.
- Final worker result must record actual method/path/body/DTO/revision semantics and source IDs; P12/P13 consume that result. No assumption that old result-11 model serialization is a finished contract.

## Prior evidence versus affected units

Reported green in result-11: identity TestContractRequisites, retail service package, app TestWorkflowPort11. They are historical results, not P11 acceptance.

- Identity private requisites and app authorized-company adapter remain unchanged; those green cases remain applicable and are not rerun.
- Prior render-only TestContractPreviewEscapesSnapshotValues established old HTML-escaped output; it does not establish lifecycle, persistence or plain-text semantics. Replace its expectation for explicit current plain-text instruction in 11c2.
- Prior unrelated retail service tests remain valid where code paths/fixtures are unchanged; no blanket repeat merely because continuation exists.
- 11c1 adds actual transactional contract lifecycle and production adapter unit; selectors are TestContract in service and exact TestContractStoreWiring in module root. No app/e2e construction.
- 11c2 changes shared contract behavior, so affected TestContract units run once after its change. Add pure DTO selector if mapping starts there.
- 11c3 affects contract/finalization behavior and existing fixtures, so run TestContract and TestInstallmentFinalization once after those changes, plus pure handler TestContract selectors. Earlier nil-scan/overwrite fixture expectations must become valid scanned initial-signature or explicit-amendment cases; do not delete schedule preservation assertions.
- Transaction fakes deep-copy persisted contract/deal/plan/invoice/event/file-binding state and inject CAS/file/event/plan failures. No no-tests output, render-only suite, or earlier service green can substitute for lifecycle cases.

User verification policy: affected pure units once per relevant change, -vet=off, approved writable GOCACHE. No tests executed by slicer. No review/QA loop, lint/typecheck/build/vet/race/browser/E2E/integration, migration execution or runtime actions.

## Acceptance coverage and remaining risks

Lifecycle/read/tenant/branch/CAS → 11c1; external upload/edit/private binding → 11c2; full parties/vehicle/economics/plain-text snapshot → 11c2; immutable signed/amendment and saved schedule → 11c1+11c3; legacy known first version/HTTP deal shape/finalization → 11c3; UI-ready version history/source IDs/structured JSON → 11c3. Original P11 cannot be marked complete before all three.

No new product policy question is introduced. Real database migration execution, concurrent UPDATE isolation and deployed file binding remain unverified. If source wiring evolves during sequential work, hand off exact predecessor result; source hashes below are current working-file observations, not a clean committed-tree claim.

## Packet SHA-256

```text
ec87105620edebb586b25dc5e6b1c473f38c90b62f3f9a741c12c07ab2af1164  packet-11c1.md
964763157f1bcfab5eaa0cc12827feed7c1eb056d48bd12ceaf92c2411af595c  packet-11c2.md
c27cb3d7c543ce1efd681a6b7544230f800f23e81a77b4903a0ba915cbf5df4e  packet-11c3.md
```

Fixture-path correction: 11c1 owns the hub-approved singular installment_payment_test.go. The observed plural paymentStore inherits from installmentStore; add Contracts() to the shared helper and avoid unnecessary plural-file edits. Both paths were inspected read-only; no test ran.

## Targeted source SHA-256

```text
2cf9dd1a0fbda3f9b643cef595c65aabc0e0bc051cd5530075586c3696805fd5  docs/justix-auto/dev/execution/product-workflow-improvements-20261011/result-11.md
4a89df838133081caaf06c333ffe1b113a214f0ef388e41bb4d7a414abc050ca  docs/justix-auto/dev/execution/product-workflow-improvements-20261011/packet-11.md
e059da4e185a1a288b694af9a31b14a2191517c40f7dc48a3466237a43594999  internal/modules/retail/model/contract.go
b3d7f502f31288060871a4ffae48eee6138a6766a7f5fcee6b2b58307a8f3df2  internal/modules/retail/service/contract.go
1394a3c39dea050b6d3d7f83aa8141cbcb6a553182d8dba4fa85feaf262357b6  internal/modules/retail/service/contract_test.go
ee01e42442b3f02b65728cbddf52925eb5fbe19e4c56e193b6499113a8edf32b  internal/modules/retail/service/ports.go
f7f8c9bbaa6c1be5417150c067aaaf4b8536cdc64b72cf9729ebda1cf04f1885  internal/modules/retail/repository/contract.go
2d7b9ba24077ec8bc2aa76e5aeed65e6cd63cebb50d603206034ff54633ad9cf  internal/modules/retail/module.go
11aa0f654c904fad07dc8c74ae7c7076ced9a5d0357ff7f794027e16571f589f  internal/modules/retail/handler/contract.go
f493855b0867033960fddef09243c85469063f8902c40996986976fff1337915  internal/modules/retail/handler/handler.go
0cf8d7a8a654723e7f2f9231531a6c4424d97001a287f5f4cfb6178d53c0cd4a  internal/modules/retail/handler/dto.go
66935f8233a1aab59eae466c291b2b49262991ccb0a279d2e5261eebce935a17  internal/modules/retail/service/deal.go
ce77df2681ffa9b4ab72abd7796a69547c562bf7efd73ee420abd9d97fecc537  internal/modules/retail/service/installment.go
755d01c69b916a9e927ee560847c27b1961e9e09edc4378bbd54e9e8e8605756  internal/modules/retail/model/installment_terms.go
316720e2dcbc4ff06b397242d23b5eb5524df985bba3c218979e0d4012cd19e7  internal/modules/retail/model/installment.go
2cc9784664783a7bf876aef55e75c117e5e54f1fe1b4305e4d2b50cb1f5dbf9e  internal/modules/retail/service/installment_test_helpers_test.go
ee8b96f18c6f723a2485322846ebf59e155fd719ab659cf6868ed878561907e6  internal/modules/retail/service/installment_finalization_test.go
badee16057682df323c65019a15d1f0f4e6bee57abc9ae15e4ad0437eabfb07d  migrations/000038_retail_contract_versions.up.sql
787dd8d9ce29e96904eb050833ac82d85c3fbf4ee2c357782fab8161fb50ef9f  internal/modules/retail/model/customer.go
7b50fc1310d09b7b7fcb944693a171bb6470d4d845ca254a4434f6e2df6a6c09  internal/modules/retail/service/installment_payment_test.go
b527e9a7a7ce898d7c8da9c910a925360db13291e064e1659f4dd57c56ece286  internal/modules/retail/service/installment_payments_test.go
```

