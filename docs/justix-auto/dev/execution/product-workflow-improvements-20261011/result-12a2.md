# P12a2 result — DONE

Preflight passed with `bash tools/check-git.sh` on
`feature/company-commerce-workflows-20261011`. Dirty predecessor changes were
preserved. This packet changed only its owned commerce handler/module/app paths
and the assigned focused tests.

## Production authorization seam

`commerce.New` now receives a typed `ContractPartiesFactory`, which receives an
already initialized `*commerce.ActualOrderParties`. The app factory builds
`commerceContractParties`: it calls `OrderPartiesFor(ctx, viewerCompanyID,
orderID)` before identity, then calls `ContractRequisitesFor(ctx, target,
target)` separately for the actual buyer and supplier. It maps every private
requisite field to `commerce.ContractRequisites`. The adapter accepts no client
party identifier, does not use `CompanyProfile`, and retains the request
context. The module exposes only the narrow aliases needed for this composition
shape. There is no partially initialized contract service or unchecked type
assertion.

## Endpoint contract

All paths are under `/api/v1/commerce`, protected by `RequireCompany` plus
`commerce.read` for GET/preview and `commerce.trade` for mutations.

| Method/path | Body / precondition | Success |
|---|---|---|
| `GET /contract-templates` | paging | `200` `httpx.List` templates |
| `GET /contract-templates/:templateID` | — | `200` template `httpx.Data`/template ETag |
| `POST /contract-templates` | `{name,content}` | `201` template |
| `PATCH /contract-templates/:templateID` | `{name,content}` + template `If-Match` | `200` template |
| `GET /orders/:id/contracts` | paging | `200` `httpx.List` visible contracts |
| `GET /orders/:id/contracts/:contractID` | — | `200` contract |
| `POST /orders/:id/contracts/preview` | `{templateId?,content?,bindingIds?,refreshSnapshot?}` | `200`, read-only preview; envelope revision/orderRevision are the source order revision |
| `POST /orders/:id/contracts` | `{templateId?,content?,bindingIds?,refreshSnapshot?}` + order `If-Match` | `201` draft |
| `PATCH /orders/:id/contracts/:contractID` | `{draftRevision,templateId?,content?,bindingIds?,refreshSnapshot?}` + order `If-Match` | `200` draft |
| `POST /orders/:id/contracts/:contractID/sign` | `{draftRevision,bindingIds,signedOn,reference}` + order `If-Match` | `200` same-ID signed version |
| `POST /orders/:id/contracts/:contractID/amendments` | `{draftRevision,priorSignedVersionId,bindingIds,signedOn,reference}` + order `If-Match` | `201` same-ID signed amendment |

`signedOn` is an ISO date. Command/detail envelopes have the contract
revision/ETag; DTOs additionally expose `contractRevision` and the current
`orderRevision` as decimal strings. Templates expose `revision`. Contract DTOs
use lower-camel explicit fields: `id`, `orderId`, `recordingCompanyId`, state,
kind, template/prior IDs and revisions, `content`, `fileIds`, structured
`partySnapshot`, exact raw structured `termsSnapshot`, `vehicleSnapshot`,
`acceptedTermsDigest`, `sourceOrderRevision`, signature metadata and timestamps.
Unknown nested accepted-term fields and large JSON numbers remain JSON, rather
than bytes/base64 or decoded float values. `fileIds` is the deduplicated union
of retained external bindings and signed scans; storage implementation fields
are never returned.

## Focused unit evidence

Passed once after the final relevant changes:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort12($|_)'
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/handler -run '^TestContract'
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce -run '^TestContractStoreWiring$'
```

`TestWorkflowPort12` proves outsider denial occurs before identity, verifies
actual buyer/supplier targets and full private-field mapping, context retention,
and no public projection. Handler units prove decimal revisions, unchanged
contract ID representation, ISO signature parsing, explicit-empty binding
arrays, plain text, file arrays, structured unknown large-number terms, and no
raw storage-field leakage.

No service selector was rerun because P12a2 did not change the service domain.
No migration execution, database-concurrency validation, integration/browser
checks, review/QA, build/lint/vet/race, Git operation, or deployment occurred.
