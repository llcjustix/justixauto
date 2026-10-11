# Application contract: custody movement and physical holders

P20 preparation proposed preserving the active order holder after custody
handover. That would leave a completed supply hold blocking the required P21b
retail reservation. The application hub clarifies the technical lifecycle
within the approved consignment/return workflow; no new commercial policy.

- Consignment handover requires the exact current commerce-order holder and
  accepted agreement, verified by the commerce caller and rechecked by the
  inventory custody primitive. A successful physical handover consumes that
  matching supply hold in the same transaction as custody/placement/facts.
  This is explicit fulfilment of the expected holder, not silent release of
  another reservation. Preserve its durable history; failure rolls it back.
- A handed-over unsold consignment may later be retail-reserved by the dealer
  only through the checked P21b consignment authority. Owner remains supplier
  until actual customer delivery. Mere warehouse placement grants no sale
  permission.
- Return custody movement requires no active competing physical hold. Recheck
  under inventory's vehicle locks and reject retail/order/other reservations;
  do not consume an unrelated hold. The completed original order hold is not
  required to remain active for a return. No new public return-hold endpoint
  or automatic release is introduced.
- P21b/P22 must serialize sale authorization and pending-return eligibility on
  the same live consignment authority in their bound transaction. A pending
  return request is not a movement, payment/refund or debt cancellation.

P20 may express expected-holder requirements by movement purpose in its
trusted module-root input. Report the actual signature so P21a2/P22 use it
exactly. Existing purchase Transfer behavior and same-company Move visibility
remain unchanged. No inventory HTTP route, new dependency or runtime action.
