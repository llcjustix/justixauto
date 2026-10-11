CREATE TABLE commerce_consignment_returns (
 id uuid PRIMARY KEY,
 unit_id uuid NOT NULL REFERENCES commerce_consignment_units(id),
 agreement_id uuid NOT NULL REFERENCES commerce_consignment_agreements(id),
 order_id uuid NOT NULL REFERENCES commerce_orders(id),
 vehicle_id uuid NOT NULL,
 supplier_company_id uuid NOT NULL,
 dealer_company_id uuid NOT NULL,
 destination_warehouse_id uuid NOT NULL,
 reason text NOT NULL CHECK (length(trim(reason)) > 0),
 status text NOT NULL CHECK (status IN ('pending','confirmed','rejected')),
 requested_by uuid NOT NULL,
 requested_at timestamptz NOT NULL,
 decided_by uuid,
 decided_at timestamptz,
 decision_reason text NOT NULL DEFAULT '',
 moved_at timestamptz,
 movement_fact_id uuid,
 movement_operation_ref uuid,
 version bigint NOT NULL CHECK (version >= 1),
 updated_at timestamptz NOT NULL,
 CHECK (supplier_company_id <> dealer_company_id),
 CHECK ((status = 'pending' AND decided_by IS NULL AND decided_at IS NULL AND moved_at IS NULL AND movement_fact_id IS NULL AND movement_operation_ref IS NULL)
     OR (status = 'confirmed' AND decided_by IS NOT NULL AND decided_at IS NOT NULL AND moved_at IS NOT NULL AND movement_fact_id IS NOT NULL AND movement_operation_ref IS NOT NULL)
     OR (status = 'rejected' AND decided_by IS NOT NULL AND decided_at IS NOT NULL AND length(trim(decision_reason)) > 0 AND moved_at IS NULL AND movement_fact_id IS NULL AND movement_operation_ref IS NULL))
);
CREATE UNIQUE INDEX commerce_consignment_returns_pending_unit_idx ON commerce_consignment_returns (unit_id) WHERE status = 'pending';
CREATE INDEX commerce_consignment_returns_order_idx ON commerce_consignment_returns (order_id, requested_at, id);
