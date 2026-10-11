# P20 exact existing holder fixture extension

Application hub adds `internal/modules/inventory/service/reservation_holder_test.go`
to P20 ownership solely to make the existing owned-stock fake represent its
actual current custodian as the owner. `holderTestVehicles.LockOwned` currently
returns only OwnerCompanyID. The approved new ordinary-reserve custody check
needs the true own-custody fixture; no production fallback for missing custody
is authorized by this fixture correction.

Keep the existing namespace, same-holder retry, atomic reassignment and
matching release assertions. Run its affected `TestReservationHolderNamespace`
case once with the new custody family after the source change. No additional
unowned fixture or broad package verification is implied.
