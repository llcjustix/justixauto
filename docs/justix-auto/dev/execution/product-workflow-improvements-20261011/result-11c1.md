# P11c1 result — DONE

Owned implementation paths completed: `internal/modules/retail/service/ports.go`,
`contract.go`, `contract_test.go`, `contract_test_helpers_test.go`,
`repository/contract.go`, `module.go`, `contract_wiring_test.go`, and the required
shared Store fixture additions in `deal_colors_test.go`, `passport_test.go`, and
`installment_test_helpers_test.go`. No canonical document, migration, handler, or
plural installment-payment fixture was edited.

`service.Store` now statically exposes `Contracts()`, the production `storeAdapter`
implements it and has compile-time conformance coverage. The unchecked runtime
store assertion is gone. `SignDraft` performs a guarded repository update on the
same row: `id`, `company_id`, `deal_id`, `state='draft'`, and `revision` must match;
it updates the row to signed and increments its revision. The signed version ID is
therefore the original draft ID. A signed row cannot match the transition again.

Signing checks `retail.deals`, company and branch scope, expected deal revision,
draft state, signed date/reference/scans, and an amendment parent that is a signed
version in the same deal. A first signature has no parent only when no signed
version already exists; later documentary work requires an explicit parent and
becomes `kind=amendment`. Contract listing now requires `retail.read` before the
existing tenant/branch deal guard. The contract CAS, file share through
`Store.Bind(ctx)` to the same signed ID, deal update, and event append are one
transaction; failures roll back the draft, deal, event history, and fake file
binding state.

Evidence: `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/service -run '^TestContract'` passed; `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail -run '^TestContractStoreWiring$'` passed. Units cover single-use draft consumption with old/refreshed deal revisions, tenant/branch/permission guards, amendment parent scope, CAS conflict, bound signed ID, and file/deal/event rollback.

Excluded for P11c2/P11c3: editable draft/upload behavior and complete plain-text
snapshots; legacy `contract-records`, installment finalization, handler DTOs and
revision publication. No database concurrency or deployed file-binding behavior
was exercised.
