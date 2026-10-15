-- Receipt batches retain receipt-wide totals. This table records where their
-- still-unidentified units are physically stored.
CREATE TABLE inventory_receipt_locations (
    receipt_batch_id   uuid    NOT NULL REFERENCES inventory_receipt_batches,
    warehouse_id       uuid    NOT NULL REFERENCES inventory_warehouses,
    unidentified_count integer NOT NULL CHECK (unidentified_count >= 0),
    PRIMARY KEY (receipt_batch_id, warehouse_id)
);
CREATE INDEX inventory_receipt_locations_warehouse_idx
    ON inventory_receipt_locations (warehouse_id) WHERE unidentified_count > 0;

INSERT INTO inventory_receipt_locations (receipt_batch_id, warehouse_id, unidentified_count)
SELECT id, warehouse_id, unidentified_count
FROM inventory_receipt_batches;
