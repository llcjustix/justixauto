# P11c2 result — DONE

Implemented the accepted persisted-draft and snapshot scope only. Changed paths:
`internal/modules/retail/model/contract.go`, `service/contract.go`,
`service/ports.go`, `service/contract_test.go`,
`service/contract_test_helpers_test.go`, `repository/contract.go`,
`handler/contract.go`, and `handler/handler.go`.

`POST /retail/deals/:id/contracts/drafts` continues to use deal `If-Match` and
now accepts optional `templateId`, `body`, and `bindingIds`. A template remains
provenance while a non-empty explicit body wins; an external-file-only draft is
valid, and no body/template/files is rejected. Creation binds every source file
privately to the new contract in the transaction and increments the deal once.

`PATCH /retail/deals/:id/contracts/:contractID` now persists a draft edit under
both deal `If-Match` and required canonical-decimal `draftRevision`. In its body,
omitted `templateId`, `body`, and `bindingIds` preserve their saved values; an
explicit empty value clears its field if the resulting draft still has content,
template provenance, or files. The guarded repository predicate includes the
draft state and revision. Edits preserve original creator/time provenance,
increment the contract and deal revisions once, and cannot modify a signed row.

Signing now requires the same canonical `draftRevision`, preserves uploaded
draft files, unions/deduplicates them with the required signed-scan bindings,
and exposes the resulting deal revision as `dealRevision` on the existing raw
contract command model pending P11c3 DTO normalization. Create, edit, and sign
all set that value after the versioned deal update. Signing rejects a draft when
its persisted exact economic snapshot differs from current authoritative terms;
it never regenerates reviewed content.

`partySnapshot` is now structured lower-camel JSON with every available seller
requisite, customer profile/passport fields, and vehicle snapshot. `termsSnapshot`
stores exact Money/currency, scheme, generated installment draft policy/rows, or
the existing immutable plan's original down-payment facts and invoice obligation
rows. A missing own-installment draft/plan is marked `legacy`; no current payment
balances or new schedule are derived. Plain-text placeholder replacement supports
seller requisite, customer profile/passport, vehicle, price, and payment fields,
preserving literal characters without HTML escaping or expression evaluation.

Evidence: `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/service -run '^TestContract'` passed. The selector covers external-only/rejected-empty drafts, edit and stale revision guards, signed-edit rejection, template body override, file union, plain text, immutable snapshot bytes, stale economics, and immutable-plan capture. No handler DTO selector was run because P11c3 owns DTO normalization and this packet did not add a DTO mapper.

Excluded: legacy `contract-records`, installment finalization, compatibility behavior, contract list/detail DTOs, and all P11c3 work. No migration, database/runtime, build, lint, review, QA, browser, or Git action was performed.
