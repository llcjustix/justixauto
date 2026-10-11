# P11 private requisite projection unit

Application hub adds the exact test path
internal/modules/identity/service/company_contract_test.go to P11 ownership.
The new private company-requisite method in the already owned company.go must
have a meaningful pure unit for its authorization/projection contract and exact
fields. Do not expose bank/tax/address data through public Directory/Profile.

Run the new identity selector once after that change, in addition to P11's
retail service and pure app selector:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/identity/service -run '^TestContractRequisites($|_)'`.
Name the cases to match this selector, confirm tests are discovered. Reuse
existing fakes; no database, runtime, build or broader verification.
