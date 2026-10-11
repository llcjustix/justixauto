CREATE TABLE commerce_consignment_agreements (
 id uuid PRIMARY KEY,
 order_id uuid NOT NULL UNIQUE REFERENCES commerce_orders(id),
 reservation_id uuid NOT NULL UNIQUE REFERENCES commerce_reservations(id),
 supplier_company_id uuid NOT NULL,
 dealer_company_id uuid NOT NULL,
 booking_digest text NOT NULL,
 continuation_digest text NOT NULL,
 continuation_revision bigint NOT NULL CHECK (continuation_revision > 0),
 version bigint NOT NULL CHECK (version >= 1),
 created_at timestamptz NOT NULL,
 CHECK (supplier_company_id <> dealer_company_id)
);
CREATE INDEX commerce_consignment_agreements_parties_idx ON commerce_consignment_agreements (supplier_company_id, dealer_company_id);

CREATE TABLE commerce_consignment_units (
 id uuid PRIMARY KEY,
 agreement_id uuid NOT NULL REFERENCES commerce_consignment_agreements(id),
 order_id uuid NOT NULL REFERENCES commerce_orders(id),
 order_line_id uuid NOT NULL,
 vehicle_id uuid NOT NULL,
 vin text NOT NULL,
 supplier_company_id uuid NOT NULL,
 dealer_company_id uuid NOT NULL,
 status text NOT NULL CHECK (status IN ('prepared','held','sold','returned')),
 handover_fact_id uuid,
 handover_operation_ref uuid,
 handed_over_at timestamptz,
 handed_over_by uuid,
 current_retail_deal_id uuid,
 pending_return_id uuid,
 retail_delivery_ref text NOT NULL DEFAULT '',
 sold_at timestamptz,
 sold_by uuid,
 returned_at timestamptz,
 return_fact_id uuid,
 version bigint NOT NULL CHECK (version >= 1),
 created_at timestamptz NOT NULL,
 updated_at timestamptz NOT NULL,
 UNIQUE (agreement_id, vehicle_id),
 CHECK (supplier_company_id <> dealer_company_id),
 CHECK (status <> 'held' OR (handed_over_at IS NOT NULL AND handover_fact_id IS NOT NULL)),
 CHECK (status <> 'sold' OR (sold_at IS NOT NULL AND sold_by IS NOT NULL AND current_retail_deal_id IS NOT NULL)),
 CHECK (status <> 'returned' OR (returned_at IS NOT NULL AND return_fact_id IS NOT NULL))
);
CREATE UNIQUE INDEX commerce_consignment_units_live_vehicle_idx ON commerce_consignment_units (vehicle_id) WHERE status IN ('prepared','held');
CREATE UNIQUE INDEX commerce_consignment_units_pending_return_idx ON commerce_consignment_units (pending_return_id) WHERE pending_return_id IS NOT NULL;
CREATE INDEX commerce_consignment_units_agreement_idx ON commerce_consignment_units (agreement_id, id);
