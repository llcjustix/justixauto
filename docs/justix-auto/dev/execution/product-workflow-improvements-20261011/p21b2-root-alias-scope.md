# P21b2 checked sale input module-root alias

Application hub accepts the concrete preparation finding from
/root/consignment_stock_sale. Original b2 stock.go will define a coherent
ConsignmentSaleInput for checked owner, dealer custodian, agreement, vehicle,
retail holder and actual actor/time. Existing custody/ownership-transfer DTOs
describe different operations and should not be overloaded for retail delivery.

Add exact source ownership of internal/modules/inventory/module.go solely for
the root type alias `type ConsignmentSaleInput = service.ConsignmentSaleInput`
(matching the final implementation name if predecessor types require it).
The existing root StockService alias exposes methods without another wrapper.
Do not modify constructor wiring, public routes, permissions, repository ports
or schema. No foreign module may import inventory/service to construct inputs.

Final b1 checked-authority contract and a3 handoff plus a separate source grant
still precede implementation. The original inventory service unit selector
remains the only b2 check; b3 owns the actual application seam units. Include
the added alias path/hash in result-21b2.md. No source or test execution occurred
during preparation; the original two production/test paths otherwise stand.
