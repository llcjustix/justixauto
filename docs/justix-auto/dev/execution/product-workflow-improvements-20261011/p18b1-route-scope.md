# P18b1 route registration seam

The final handler composition registers reservation routes directly in
internal/modules/commerce/handler/handler.go. The originally owned
handler/reservation.go cannot expose the approved extension/cancellation
actions by itself. The application hub adds exactly handler/handler.go to
P18b1 ownership for those two authenticated, permission-scoped registrations.

This completes existing P18b1 HTTP acceptance; no public expiry, payment-state
or stock-release endpoint is introduced. Existing route guards and If-Match
semantics remain. No broader route refactor or extra verification gate.
