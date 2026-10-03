# Business logic

Current decisions, consolidated 2026-10-03. Later explicit user decisions take
precedence. JustixAuto records vehicle transactions, documents and external
payment confirmations; it does not itself hold or transfer money.

## Applications and access

- **Realization:** purchasing, supplier/own offers, orders, stock, CRM, sales,
  invoices and own-installment servicing.
- **Financing:** bank/MFO programs, applications, decisions, terms and documents.
- **Insurance:** underwriting applications for the seller's own installments.
- **Admin:** companies, platform staff, permissions and the vehicle catalog.

A user can belong to several companies; company roles belong to each membership,
platform roles to the user. Company capability, permission, branch access, party
and object state are checked separately. Switching company resets branch scope.
Platform admins manage the catalog; company admins manage their own staff/roles.
Company administrators receive all available company permissions, subject to
capability checks. Soft deletion/suspension preserves records and audit history.
Company creation needs its name and the first admin's login/password; other
details are optional. Admin-created companies start as drafts; companies created
by a company administrator currently activate automatically. No MFA is used.

## Vehicles, purchasing and stock

- One globally unique VIN identifies one physical vehicle. Ownership, custody,
  warehouse placement and reservation are separate facts. A VIN cannot have two
  active reservations. Unidentified batch units cannot be sold or reserved.
- Each model/version offers several body and interior colors. Each order line
  chooses one pair; different pairs use separate lines. A sale shows the actual
  VIN colors. Historical versions and snapshots do not follow later catalog edits.
- An active partnership permits new B2B orders, directly or from a supplier offer.
  Orders need supplier confirmation; offer quantity does not cap ordered quantity.
  RFQ creation is absent from the current UI. Accepted terms change by addendum.
- New orders select a receiving warehouse. Supplier shipment transfers units to
  the buyer there, including batches whose VINs are entered later. Insufficient
  capacity blocks shipment. Separate receipt remains for legacy orders without
  a receiving warehouse. Occupancy includes identified units and unidentified
  batch quantities; identifying a VIN does not change occupancy.
- Purchase payment proofs are accepted/rejected by the seller's finance staff.
  Publishing an offer never reserves a vehicle. Supplier and own offers can
  prefill an editable sale price, only for an available VIN on our own stock.

## Sales and payments

A sale needs a customer and an available VIN and reserves that vehicle. CRM leads,
customers, tasks and contact history link to the sale. Documents, registration,
payments and vehicle handover are separate recorded facts.

**Full payment:** contract → one vehicle invoice for exactly the sale price and
currency → accepted payments for the full price → handover. Several partial
payments can settle that invoice. Registration and its separate fee are optional.

**Own installments:** insurer decision, positive down payment, contract,
registration and accepted registration-fee payment are required for handover.
The monthly schedule is generated automatically, interest-free: divide
price minus down payment into equal integer minor-unit amounts, putting rounding
remainder into the final payment. Dates recur monthly, clipped to month end.
Contract recording fixes the schedule; existing schedules/payments are not rebuilt.

Customers may pay before handover, cover several selected installments or settle
the entire monthly balance early. The operator chooses each row and allocation;
there is no automatic oldest-first allocation. One external proof groups these
allocations, and finance accepts/rejects it as a whole. Pending proofs reserve
available payment capacity; only accepted amounts reduce debt. Monthly settlement
does not imply handover or payment of registration fees. Cancellation is blocked
by unresolved monthly proofs or accepted monthly money without an approved refund
policy. No interest, penalties, fees or repricing are introduced by early payoff.

## Financing and insurance

Banks/MFOs publish programs and review addressed applications: draft → submitted
→ review → needs-info / terms / declined; client acceptance of terms → agreed.
Document requests and uploaded versions are reviewed separately. Submitted
snapshots are immutable; the seller cannot decide for a provider. Agreement or
document acceptance does not authorize funding or handover.

Own-installment insurance has one application per company/sale. Only the
assigned insurer reviews and decides, with reasons and versioned supplements.
Approval is an underwriting decision, not policy issuance or confirmed coverage.

## Unresolved boundaries

Do not infer these from the HTML prototypes:

- **OD-01/10:** bank/MFO funding, first-payment recipient, ownership/handover,
  actual provider tariffs, integrations, personal-data and consent requirements.
- **OD-02/03/04/13:** final vehicle-list scope, retail-only partnership setup,
  partner-stock resale/payment/cancellation rules and remaining wholesale parity.
- **OD-05/06/11:** suspended-company access to existing obligations, route-specific
  photo evidence and legal verification versus operational activation.
- **OD-07/08:** real registration charges/documents/currency and insurer approval
  versus policy, premium, coverage and claims.
- **OD-09/12:** refunds, penalties, default/collection, zero down payment and
  production account recovery/delivery/security configuration.

Four apps are the current scope; a public customer app, real provider payouts,
policy issuance and exceptional admin overrides need separate decisions.
Errors preserve form input; stale writes require refresh; history records actor,
time and reason. Monetary arithmetic is exact and never mixes currencies.
