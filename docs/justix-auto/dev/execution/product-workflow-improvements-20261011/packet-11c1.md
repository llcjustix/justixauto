# P11c1 — Wire contract store and consume drafts exactly once

Canonical scope R9/P11 only. Dependency: current P11 partial; first ready continuation. Authority /root; source-writer exclusive lock, no live resources. Read p11-continuation-plan.md, result-11.md and root/internal rules. Context ceiling 14k; read only listed symbols. No Git/source ownership outside this packet; normalize edited minified Go as part of editing, no formatting/lint loop.

## Exact owned paths

- internal/modules/retail/service/ports.go
- internal/modules/retail/service/contract.go
- internal/modules/retail/service/contract_test.go
- internal/modules/retail/service/contract_test_helpers_test.go (new)
- internal/modules/retail/repository/contract.go
- internal/modules/retail/module.go
- internal/modules/retail/contract_wiring_test.go (new, pure)
- internal/modules/retail/service/deal_colors_test.go
- internal/modules/retail/service/passport_test.go
- internal/modules/retail/service/installment_payment_test.go
- internal/modules/retail/service/installment_test_helpers_test.go
- internal/modules/retail/service/cash_sale_test.go

Last five files: only exact Store-interface fixture additions required for compilation; no unrelated assertion/behavior rewrites. Both singular/plural payment test filenames exist; the plural file's paymentStore embeds *installmentStore and can inherit Contracts() from installment_test_helpers_test.go. Do not edit the plural file or add redundant fixture methods. The singular file is the hub-approved exact fixture extension, used only if its dependent fixture code actually needs adaptation.

## Implement

1. Add Contracts() ContractRepository to the actual service.Store interface; remove unchecked contractStore assertion. Add production storeAdapter.Contracts() and compile-time Store conformance. Existing repository.Store already exposes Contracts. Every InTx callback receives a Store that statically has the contract repository. Do not solve missing production wiring by treating it as optional or silently skipping persistence.
2. Introduce repository guarded draft transition/update: company+deal+id+state=draft+expected revision in the UPDATE predicate; increment revision; zero affected row means conflict/stale and never success. Existing schema supports draft→signed on the SAME row. Adopt this minimal lifecycle: draft ID becomes signed version ID; once signed no edits/deletes. Do not append a signed clone while leaving its source reusable. Existing already-signed rows remain untouched. No schema change/migration rewrite is needed.
3. Sign checks service permission, tenant, branch, supplied deal revision and draft state, validates scans/reference/date, validates explicit amendment parent belongs to this deal and is signed. Bind files to the actual resulting signed-version ID, not an abandoned draft ID. Perform draft CAS, file binding, deal compatibility update and event in one Store.InTx/Bind transaction; any failure rolls all back. Existing deal Update(expected) prevents competing writers with same deal revision. Re-read same draft with a fresh deal revision must still reject because its state is signed.
4. Keep existing Sign signature/deal-level If-Match semantics during this packet. P11c2 adds explicit draft revision input; P11c3 publishes both revision fields clearly. Keep signed rows immutable even if caller supplies a new deal revision. Only first base contract may be signed without priorSignedVersionId; later documentary change goes through an explicit amendment linked to an existing signed version.
5. Contracts listing must enforce PermRead within service, then tenant/branch visibility. Preserve platform-versus-company rules; no verification block added to servicing existing deals.

## Pure units, once after changes

Use transactional fake stores that deep-copy persisted contract/deal/event state, not just call-count/render assertions. Include first sign consumes exactly one draft; repeat sign with old AND refreshed deal revision fails; wrong company/branch/missing read or write permission; amendment parent outside deal; injected contract CAS conflict; file binding failure; stale deal update/event failure roll back draft state, deal metadata and history. Assert file binding carries Store.Bind context and resulting signed ID. Fake files should model rollback through shared fake transaction state.

Run `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/service -run '^TestContract'`.
Run `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail -run '^TestContractStoreWiring$'` only for new pure adapter construction test, no app.New/e2e.DB. It exercises production storeAdapter method availability with an inert repository; no database calls.

Do not rerun identity/app tests; their seam is unchanged. These fakes establish service conflict/rollback intent, not actual concurrent PostgreSQL behavior. Stop for exact additional fixture path if compilation requires another Store implementation; never enlarge scope silently. Return source/signing ID semantics and remaining c2/c3 acceptance.

