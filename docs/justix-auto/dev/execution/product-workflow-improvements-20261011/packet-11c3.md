# P11c3 — Legacy record compatibility, finalization and contract DTO handoff

Canonical R9/P11. Dependency P11c2. Source-writer exclusive; no runtime resources. Context ceiling 16k. Read c1/c2 results and only RecordContract, finalizeInstallmentDraft, DTO/handler and listed fixtures.

## Exact owned paths

- internal/modules/retail/service/deal.go
- internal/modules/retail/service/contract.go
- internal/modules/retail/service/contract_test.go
- internal/modules/retail/service/contract_test_helpers_test.go
- internal/modules/retail/service/installment_finalization_test.go
- internal/modules/retail/service/installment_test_helpers_test.go
- internal/modules/retail/handler/contract.go
- internal/modules/retail/handler/handler.go
- internal/modules/retail/handler/dto.go
- internal/modules/retail/handler/contract_dto_test.go
- internal/modules/retail/handler/contract_compatibility_test.go (new pure)

Existing finalization fixtures: only adaptations for real contract repository/file/requisite ports, required signed scan, and replacing former overwrite expectations with explicit amendment preservation. Do not remove original amount/date/transaction assertions.

## Implement

1. Deal.RecordContract delegates to shared contract-version logic within ONE transaction, creates a known initial signed version from authenticated external scan, snapshots available approved data, updates compatibility fields, invokes existing installment finalization, and returns *model.Deal as before. No invented template text is needed for external uploaded contract; empty content is valid for this path. Require scan for NEW signature; historical migrated rows without scans remain readable/serviceable. Repeated initial record must conflict and direct caller to explicit amendment flow, rather than overwrite signed history. Legacy rows already copied by 000038 remain their known first version. Do not rewrite migration or fabricate prior versions.
2. Preserve HTTP POST /retail/deals/:id/contract-records request {bindingIds,signedOn,reference}, If-Match=deal revision, success 200 dealDTO with dealResponse revision/ETag. Ignore result-11 suggestion to change this response into contract-version DTO; original approved behavior does not authorize breaking consumers. Existing GET deal fields still work.
3. First signing through BOTH new Sign and legacy RecordContract must preserve existing finalization order: invoice→contract and contract→invoice produce the same immutable plan once prerequisites complete. Reuse the existing exact saved InstallmentDraft rows. Share one transaction for signed state, file bindings, deal update, plan/invoices and audit; inject failure at each step and roll back all. Existing saved plans are never regenerated/repriced by amendments; documentary amendment only appends a new signed version from a different draft and may update current compatibility references while preserving original plan reference/date/files/amounts.
4. New contract/template handlers map to explicit lower-camel DTOs, never serialize GORM models or []byte directly. File IDs are JSON arrays; partySnapshot/termsSnapshot are structured objects (typed data or validated json.RawMessage), not base64 strings. No storage keys, public URLs, unrequested metadata or raw passport-copy payload. Template/draft content is plain text. Lists use httpx.List with items envelope; commands use httpx.Data. Malformed stored snapshot data yields a controlled error, not silent empty replacement.
5. Pin the completed endpoint seam in result: templates list/create/update; deal contracts list/detail (add GET /retail/deals/:id/contracts/:contractID if absent), draft create, draft PATCH, sign, amendments. Company/branch/read permission at service for list/detail. Template PATCH If-Match=template revision. Deal draft create If-Match=deal revision. Draft PATCH/sign/amendment If-Match=deal revision PLUS body draftRevision; amendment body additionally priorSignedVersionId. Responses identify id/dealId/state/kind/templateId/templateVersion/priorSignedVersionId, bindingIds, content/snapshots, contractRevision and current dealRevision. Envelope revision follows returned contract resource revision; callers explicitly use dealRevision for deal-owned action header. Template response revision is template revision. Document signed ID equals its consumed draft ID (c1); amendments use a new draft ID linked to previous signed ID. GET detail reports current dealRevision without mutating signed snapshot.
6. Do not mark P11 complete until c1/c2 plus this compatibility/finalization/DTO scope is delivered. P12/P13 consume actual final result, not old model JSON or stale result-11 recommendation.

## Pure unit cases and commands

- Both signing entry points create immutable signed state and preserve first-installment lifecycle order.
- Second base recording at old/fresh revision fails, original record unchanged; explicit amendment preserves full plan and invoice rows.
- Missing scan rejected only on new signing; legacy no-scan record can be read/serviced.
- File rejection, contract CAS, stale deal update, plan insert/second-row/audit failure restores all persisted fake state.
- DTO JSON uses lower-camel keys, array bindings, object snapshots and decimal-string revisions; no base64/object model fields. Legacy command still yields deal representation/status/revision through unchanged handler seam. Use pure mapping/service fakes, no e2e.DB or complete app construction.

Run affected service units once: `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/service -run '^(TestContract|TestInstallmentFinalization)'`.
Run affected pure handler units once: `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/handler -run '^TestContract'`.
If an existing contract-calling unit has another name and is affected, report exact selector/path before broadening; no blanket suite/review loop.

Return exact endpoint/request/response/revision/source-ID handoff, unit counts/output and immutable-plan exclusions. No deployed concurrency or migration guarantee is established by fake transactions.

