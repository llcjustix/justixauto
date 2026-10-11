# P14 scoring persistence and affected fixture scope

Application hub accepts required typed `Store.Scoring()` and the production
adapter in original P14-owned paths. No optional/unchecked persistence assertion.

Additional exact unit-only paths:

- internal/modules/retail/service/installment_test_helpers_test.go
- internal/modules/retail/service/deal_colors_test.go
- internal/modules/retail/service/passport_test.go
- internal/modules/retail/service/contract_test_helpers_test.go
- internal/modules/retail/service/contract_test.go
- internal/modules/retail/service/installment_finalization_test.go

The first four adapt real Store fixtures; the last two preserve affected
first-sign/finalization cases using an explicit current demo run where now
required. Consume final P11c3 semantics before editing: new signatures through
both public entrypoints require the shared gate, historical servicing and
documentary amendments do not. Never weaken prior amount/date/rollback or
immutability assertions, globally bypass the gate, or use test-only service
behavior. Existing payment/cash fakes inherit the shared Store fixture.

Only P14's affected retail/service unit command runs once after the relevant
change. No extra checks or runtime activity. This is fixture/source scope
authorization, not evidence that P14 is implemented or tested.
