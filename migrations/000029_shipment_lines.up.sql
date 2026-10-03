-- User decision 2026-10-01: the supplier ships an order by quantity, without
-- holding the vehicles in its own stock. Each line of such a shipment enters
-- the buyer's receiving warehouse as a receipt batch (VINs may come later).
CREATE TABLE commerce_shipment_lines (
    id               uuid        PRIMARY KEY,
    shipment_id      uuid        NOT NULL REFERENCES commerce_shipments,
    order_id         uuid        NOT NULL REFERENCES commerce_orders,
    line_id          uuid        NOT NULL,
    quantity         integer     NOT NULL CHECK (quantity >= 1),
    receipt_batch_id uuid        NOT NULL,
    created_at       timestamptz NOT NULL
);
CREATE INDEX shipment_lines_order_idx ON commerce_shipment_lines (order_id);
