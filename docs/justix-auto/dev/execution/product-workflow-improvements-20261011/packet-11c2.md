# P11c2 — Editable private contract drafts and full plain-text snapshots

Canonical R9/P11. Dependency P11c1. Source-writer exclusive; no runtime resources. Context ceiling 16k. Read common continuation plan plus targeted contract/model/installment definitions; no full historical packet set.

## Exact owned paths

- internal/modules/retail/service/contract.go
- internal/modules/retail/service/contract_test.go
- internal/modules/retail/service/contract_test_helpers_test.go
- internal/modules/retail/service/ports.go
- internal/modules/retail/repository/contract.go
- internal/modules/retail/handler/contract.go
- internal/modules/retail/handler/handler.go
- internal/modules/retail/handler/dto.go
- internal/modules/retail/handler/contract_dto_test.go (new)
- internal/modules/retail/model/contract.go

No migration required for existing draft/signed states, Content, FileIDs or JSON snapshot columns. Do not repurpose economics snapshots to store file metadata.

## Implement

1. Draft supports configurable template, edited body and/or external file. Add optional bindingIds; permit external-document-only draft with empty content; reject draft with neither content/template nor file. Use existing private Files.Share owner validation/resource binding in the same transaction. Template input and body override must work together: explicit edited body wins, while template ID/version remain provenance. Do not silently discard body whenever templateId is set.
2. Add persisted draft edit endpoint PATCH /retail/deals/:id/contracts/:contractID, using guarded repository draft CAS. Never edit a signed record. Full edited input explicitly defines desired body/source files; document omitted-versus-empty behavior (preserve omitted fields, explicit empty clears if resulting draft still valid). Preserve actor/time provenance and increment draft/deal revisions once. No template mutation rewrites previously saved draft or signed content.
3. New draft edit and sign inputs include draftRevision as a canonical decimal string. Continue If-Match=deal revision for these deal-owned actions, compare explicit draftRevision against persisted revision, and include current dealRevision in command responses. Draft creation If-Match is deal revision. Successful create/edit/sign bumps deal revision; stale draft or deal rejects. P11c3 finalizes/document-pins response DTOs; do not make P13 infer which revision to send.
4. Snapshot all available seller CompanyRequisites fields (name/legal/registration/tax/bank/address/phone/email), customer profile data (name/phone plus optional passport series/number/issuer/dates, birth date, personal ID/address), and captured vehicle facts. Do not automatically attach/share customer passport-copy files or place their bytes/URLs in contract text. Explicit file grants remain separate.
5. Terms snapshot is structured exact Money plus scheme AND complete authoritative installment information: when draft exists, deep-copy policy/version, price/down payment, months/start date, scheduled total/regular payment and every due row. When immutable plan already exists, capture original plan/down-payment facts and invoice obligations' amounts/currency/due dates/numbers without deriving a new schedule from current paid/outstanding balances. Keep base signed plan unchanged for documentary amendments. Missing legacy terms are marked absent/legacy, never reconstructed from current data.
6. Template content and filled output are PLAIN TEXT. Literal placeholder replacement only, preserve A&B, <characters> etc exactly; React text rendering handles presentation escape. No html.EscapeString in stored plain text, HTML interpretation or executable expression engine. Extend supported literal placeholders to available requisite/profile/installment fields with exact string formatting; no invented legal language.
7. Signed content/party/terms arrays are deep snapshots. Signing must not silently pair a stale draft economic snapshot with different current installment terms: under transaction compare authoritative economic snapshot and require explicit draft refresh on mismatch. Preserve the reviewed document rather than silently regenerate during sign. Later customer/template changes cannot mutate signed snapshot. Final signed FileIDs preserve valid draft external files plus supplied paper scan bindings, deduplicated; require at least one explicitly supplied signed-scan binding. Do not discard external files on sign.

## Meaningful units

Actual create/edit/sign via fake Store: external-only draft accepted, empty rejected, cross-tenant template/file denied, binding failure rolls back, body overrides template, persisted edit changes revision, stale draft/deal edit fails, signed edit fails. Snapshot exact installment draft and existing plan schedule; mutate live customer/template/terms after signing and assert bytes unchanged. Stale economics signing requires refresh; never silently recalculates saved plan. File union preserves external document+signed scans. Plain text retains exact special characters; placeholder-like input is never executed.

Run once: `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/service -run '^TestContract'`. This includes c1 lifecycle because shared contract behavior changed. Run new affected pure DTO selector only if mapping is introduced here: `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/handler -run '^TestContractDTO'`. No DB/build/lint checks. Identity/app source is unchanged; their prior green units need no repeat.

Stop before adding a template engine, public files, legal clauses or changing schedule policy. Return exact request optionality, placeholder set, revisions and structured snapshot schema for c3/P13.

