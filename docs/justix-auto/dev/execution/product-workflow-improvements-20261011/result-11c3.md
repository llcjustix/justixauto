# P11c3 result — DONE

Preflight passed (`GIT CHECK OK`). I preserved predecessor changes and changed
only P11c3-owned retail service/handler/test paths.

Implemented before the bounded test-fixture stop: legacy `RecordContract`
creates the same initial immutable signed version as draft signing and retains
the legacy `200` deal DTO seam; the shared signer updates compatibility fields,
finalizes the saved installment draft, then updates the deal and audits inside
the same transaction. A repeated base record now returns
`contract_amendment_required`; amendments remain explicit. New contract/template
handlers use lower-camel DTOs, ID arrays, typed structured snapshots,
decimal-string revisions, `httpx.List` lists, and a GET contract-detail route.
Malformed stored snapshots/bindings return controlled errors.

Endpoints currently pinned: `GET /retail/contract-templates`,
`POST /retail/contract-templates`, `PATCH /retail/contract-templates/:id`,
`GET /retail/deals/:id/contracts`, `GET /retail/deals/:id/contracts/:contractID`,
`POST /retail/deals/:id/contracts/drafts`, `PATCH /retail/deals/:id/contracts/:contractID`,
`POST /retail/deals/:id/contracts/:contractID/sign`, and
`POST /retail/deals/:id/contracts/:contractID/amendments`. Template POST/PATCH
bodies are `{name,body}`; PATCH uses template `If-Match`. Draft create is
`{templateId?,body?,bindingIds?}` with deal `If-Match`. Draft PATCH is
`{draftRevision,templateId?,body?,bindingIds?}` with deal `If-Match`; sign is
`{signedOn,reference,bindingIds,draftRevision}` with deal `If-Match`; amendment
adds required `priorSignedVersionId`. Contract command envelopes use the
contract revision/ETag and include `contractRevision` plus `dealRevision`; list
items include the current deal revision. Legacy POST
`/retail/deals/:id/contract-records` remains `{bindingIds,signedOn,reference}`
with deal `If-Match` and returns the existing `dealDTO` with its deal revision.

The explicit contract DTO keys are `id`, `dealId`, `state`, `kind`, `templateId`,
`templateVersion`, `priorSignedVersionId`, `reference`, `signedOn`, `bindingIds`,
`content`, `partySnapshot`, `termsSnapshot`, `contractRevision`, and
`dealRevision`; it has no `fileIds`, raw model JSON, storage key, URL, or base64
field. Template DTO keys are `id`, `name`, `body`, and `revision`.

Draft refresh is explicit only through PATCH: omitted template/body/binding IDs
preserve stored values; explicit empty values clear them if the draft remains
valid. Sign never regenerates content or economics and rejects a differing
current terms snapshot. Snapshot schema is `partySnapshot` `{seller,customer,
vehicle}` where seller has `id,name,legalName,registrationNumber,taxId,
bankRequisites,address,phone,email`, customer has `id,name,phone,passportSeries,
passportNumber,passportIssuer,personalId,registeredAddress,passportIssuedOn,
passportExpiresOn,birthDate`, and vehicle is the saved vehicle snapshot.
`termsSnapshot` is `{price,paymentScheme,installment?,legacy?}`; installment
is draft or plan evidence with source/policy/economic fields and immutable
obligation rows (`invoiceId,number,purpose,amount,dueDate`). Plain-text
replacement supports `seller.name`, `seller.legalName`, `seller.registrationNumber`,
`seller.taxId`, `seller.bankRequisites`, `seller.address`, `seller.phone`,
`seller.email`; `customer.name`, `customer.phone`, `customer.passportSeries`,
`customer.passportNumber`, `customer.passportIssuer`, `customer.passportIssuedOn`,
`customer.passportExpiresOn`, `customer.birthDate`, `customer.personalId`,
`customer.registeredAddress`; `vehicle.id`, `vehicle.vin`, `vehicle.modelId`,
`vehicle.modelSpecificationVersion`, `vehicle.exteriorColor`,
`vehicle.interiorColor`; and `price.amountMinor`, `price.currency`,
`payment.scheme`. It excludes file storage URLs/keys and passport-copy bytes.

Evidence: the affected service selector passed after legacy/finalization work:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/service -run '^(TestContract|TestInstallmentFinalization)'`.
The handler selector passed: `... ./internal/modules/retail/handler -run '^TestContract'`.
The direct Sign→saved-plan fake initially exposed two fixture defects (a
handmade unsupported policy, then an absent `issued` status on the first
invoice). The authorized narrow correction uses `GenerateInstallmentDraft` and
an issued first invoice; `^TestContractSignFinalizesExistingInstallmentDraft$`
passed. Its assertions retain the saved-plan reference, exact row count and
source-generated amount/date/row facts. No migration/runtime/database
concurrency guarantee is established.
