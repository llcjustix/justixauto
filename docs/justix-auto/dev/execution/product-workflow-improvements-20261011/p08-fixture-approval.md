# P08 existing file-port fake adaptation

Application hub grants the current writer one additional test path:
internal/modules/retail/service/installment_payment_test.go. Extend only its
paymentFiles fake to satisfy the new Files.BindPassport signature; retain
payment behavior and assertions. No production ownership expansion.

The first affected command passed documents/service but retail/service failed
to compile because of this fake and an owned passportEvents test helper.
Repair the helpers and run only retail/service; do not repeat documents/service.
The planned pure app TestWorkflowPort08 selector still runs separately.
