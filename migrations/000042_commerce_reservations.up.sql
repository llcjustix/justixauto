CREATE TABLE commerce_reservations (
 id uuid PRIMARY KEY, buyer_company_id uuid NOT NULL, supplier_company_id uuid NOT NULL,
 status text NOT NULL CHECK (status IN ('requested','offered','active','expired','cancelled','converted')),
 requested_terms jsonb NOT NULL, offered_terms jsonb, current_terms jsonb,
 requested_digest text NOT NULL, offered_digest text NOT NULL DEFAULT '', current_digest text NOT NULL DEFAULT '',
 requested_revision bigint NOT NULL DEFAULT 1, offered_revision bigint NOT NULL DEFAULT 0, current_revision bigint NOT NULL DEFAULT 0,
 order_id uuid,
 continuation_proposed jsonb, continuation_accepted jsonb,
 continuation_proposed_digest text NOT NULL DEFAULT '', continuation_accepted_digest text NOT NULL DEFAULT '',
 continuation_proposed_revision bigint NOT NULL DEFAULT 0, continuation_accepted_revision bigint NOT NULL DEFAULT 0,
 continuation_proposed_by uuid, continuation_accepted_by uuid,
 expires_at timestamptz, advance_due_at timestamptz, extended_by uuid, extended_at timestamptz,
 released_at timestamptz, release_reason text NOT NULL DEFAULT '',
 version bigint NOT NULL CHECK (version >= 1), created_by uuid NOT NULL, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL
);
CREATE INDEX commerce_reservations_due_idx ON commerce_reservations (LEAST(expires_at, COALESCE(advance_due_at, expires_at)), id)
 WHERE status = 'active' AND order_id IS NULL;
CREATE TABLE commerce_reservation_lines (
 id uuid PRIMARY KEY, reservation_id uuid NOT NULL REFERENCES commerce_reservations, vehicle_id uuid, supply_lot_id uuid REFERENCES commerce_supply_lots,
 model_id uuid NOT NULL, model_specification_version text NOT NULL, exterior_color text NOT NULL, interior_color text NOT NULL, quantity integer NOT NULL CHECK (quantity > 0),
 CHECK ((vehicle_id IS NOT NULL AND supply_lot_id IS NULL AND quantity = 1) OR (vehicle_id IS NULL AND supply_lot_id IS NOT NULL))
);
CREATE INDEX commerce_reservations_party_idx ON commerce_reservations (buyer_company_id, supplier_company_id, updated_at DESC);
CREATE UNIQUE INDEX commerce_reservations_order_once ON commerce_reservations (order_id) WHERE order_id IS NOT NULL;
CREATE INDEX commerce_reservation_lines_supply_idx ON commerce_reservation_lines (supply_lot_id) WHERE supply_lot_id IS NOT NULL;
