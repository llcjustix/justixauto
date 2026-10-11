# P21b3 existing-deal authority classification

Application hub accepts the final b1 API consumption identified in b3's
preparation. No extra retail model flag or commerce source ownership is needed.

Inside the existing bound retail transaction, listing/Create use the b1
Check/Reserve operation and ordinary flow only for its true `(nil, nil)` absence.
For Deliver/Cancel, Check establishes actual live absence or the specific
`consignment_retail_reserved` error. Only that exact code permits proceeding to
ConsignmentSaleForDeal for the same seller, VIN and deal; arbitrary NotFound,
permission/read errors and wrong association never permit ordinary fallback.
The shared per-VIN lock stays in the outer transaction across the physical
action, authority update and retail writes. The narrow adapter/port may express
this classification; no generic error-swallowing fallback is authorized.

Preserve existing state/revision/payment/contract/insurance/partner-finance
guards before actual delivery/cancellation. Both company types retain ordinary
owned-stock behavior. Source grant still waits for final b2. Original b3 paths
and service/app unit selectors cover these branches without another check.
