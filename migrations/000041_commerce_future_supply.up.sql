-- Distributor future capacity is deliberately separate from physical VIN stock.
CREATE TABLE commerce_supply_lots (
    id uuid PRIMARY KEY,
    supplier_company_id uuid NOT NULL,
    model_id uuid NOT NULL,
    model_specification_version text NOT NULL,
    exterior_color text NOT NULL,
    interior_color text NOT NULL,
    total_quantity integer NOT NULL CHECK (total_quantity > 0),
    expected_arrival timestamptz,
    version bigint NOT NULL CHECK (version >= 1),
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL
);
CREATE INDEX commerce_supply_lots_supplier_idx ON commerce_supply_lots (supplier_company_id, updated_at DESC, id);

-- Each row is one durable committed future unit. Reservation IDs intentionally
-- have no foreign key: the reservation tables are introduced in migration 42.
CREATE TABLE commerce_supply_claims (
    id uuid PRIMARY KEY,
    supply_lot_id uuid NOT NULL REFERENCES commerce_supply_lots,
    reservation_id uuid,
    reservation_line_id uuid,
    order_id uuid REFERENCES commerce_orders,
    order_line_id uuid,
    state text NOT NULL CHECK (state IN ('held', 'materialized', 'released')),
    materialization_kind text NOT NULL DEFAULT '' CHECK (materialization_kind IN ('', 'vin-allocation', 'quantity-receipt')),
    materialization_ref uuid,
    materialization_unit_ordinal integer,
    materialized_at timestamptz,
    created_at timestamptz NOT NULL,
    CHECK ((state = 'materialized' AND materialization_kind <> '' AND materialization_ref IS NOT NULL
            AND materialization_unit_ordinal IS NOT NULL AND materialization_unit_ordinal > 0 AND materialized_at IS NOT NULL)
        OR (state <> 'materialized' AND materialization_kind = '' AND materialization_ref IS NULL
            AND materialization_unit_ordinal IS NULL AND materialized_at IS NULL)),
    CHECK (materialization_kind <> 'vin-allocation' OR materialization_unit_ordinal = 1),
    CHECK (state <> 'materialized' OR reservation_id IS NOT NULL OR order_id IS NOT NULL)
);
CREATE INDEX commerce_supply_claims_lot_state_idx ON commerce_supply_claims (supply_lot_id, state);
CREATE INDEX commerce_supply_claims_reservation_idx ON commerce_supply_claims (reservation_id, reservation_line_id) WHERE reservation_id IS NOT NULL;
CREATE INDEX commerce_supply_claims_order_idx ON commerce_supply_claims (order_id, order_line_id) WHERE order_id IS NOT NULL;
CREATE UNIQUE INDEX commerce_supply_claims_materialization_once ON commerce_supply_claims (materialization_kind, materialization_ref, materialization_unit_ordinal) WHERE materialization_ref IS NOT NULL;
CREATE TABLE commerce_supply_claim_allocations (
    id uuid PRIMARY KEY,
    claim_id uuid NOT NULL REFERENCES commerce_supply_claims,
    reservation_line_id uuid NOT NULL,
    vehicle_id uuid NOT NULL,
    holder_type text NOT NULL CHECK (holder_type IN ('commerce-reservation', 'commerce-order')),
    holder_id uuid NOT NULL,
    order_line_id uuid,
    current boolean NOT NULL,
    created_at timestamptz NOT NULL,
    CHECK ((holder_type = 'commerce-reservation' AND order_line_id IS NULL) OR
           (holder_type = 'commerce-order' AND order_line_id IS NOT NULL))
);
CREATE INDEX commerce_supply_claim_allocations_claim_idx ON commerce_supply_claim_allocations (claim_id, created_at, id);
CREATE UNIQUE INDEX commerce_supply_claim_allocations_current_claim ON commerce_supply_claim_allocations (claim_id) WHERE current;
CREATE UNIQUE INDEX commerce_supply_claim_allocations_current_vehicle ON commerce_supply_claim_allocations (vehicle_id) WHERE current;
