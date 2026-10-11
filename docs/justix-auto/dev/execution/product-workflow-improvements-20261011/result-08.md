status: DONE

P08 implements optional private CRM passport profiles. Customer input and DTO
profile fields are `passportSeries`, `passportNumber`, `passportIssuer`,
`passportIssuedOn`, `passportExpiresOn`, `birthDate`, `personalId`,
`registeredAddress`, and `passportCopyFileIds`; customer create/update routes
are unchanged (`POST /api/v1/retail/customers`, `PATCH /api/v1/retail/customers/:id`). Dates are strict
`YYYY-MM-DD`; expiry cannot precede issue and issue cannot precede birth.

`passport-copy` is server-marked sensitive at upload. `documents.Service.BindPassport`
first resolves an owned file and requires both purpose `passport-copy` and
`Sensitive`, then creates the existing same-company `retail.customer` share.
The app `fileShares` adapter exposes the narrow retail `Files.BindPassport`
port. CRM invokes it with `Store.Bind(ctx)` inside its create/update transaction,
so a bind failure rolls back the customer write. Customer events retain empty
details and contain no passport data or file URLs; document metadata remains the
existing key-free view and download still uses `documents.sensitive.download`.

Approved exact-scope extensions: documents service `Owned` store port and GORM
implementation, needed to establish ownership before sharing; the existing
`installment_payments_test.go` `paymentFiles` fake now has an explicitly
unsupported `BindPassport` method after the shared retail Files port grew.

Units passed:

* `GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/documents/service` — PASS.
* `GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/retail/service` — PASS.
* `GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort08($|_)'` — PASS.

No DB, HTTP, browser, migration-runtime, lint, typecheck, build, vet, race,
review, or QA checks were run. Pure units do not establish deployed storage,
database transaction isolation, or migration execution.
