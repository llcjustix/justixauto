# Narrow P16-origin handler compile repair

P17a service units passed. Compilation of its authorized app/handler selectors
found a predecessor listSupply mapper signature mismatch at supply.go:52:
mapSlice expects a pointer mapper, while the closure accepts a value.
Existing toSupply already accepts *service.SupplyView.

Grant the active P17a completion writer exactly
internal/modules/commerce/handler/supply.go for replacing that mapper with
`mapSlice(views, toSupply)`. No supply behavior or DTO change. Run only the
previously unexecuted authorized app/handler selectors; do not repeat green
service units for this thin compile fix. Record the seam repair and limitation
in result-17a-completion.md. No other verification or source extension.
