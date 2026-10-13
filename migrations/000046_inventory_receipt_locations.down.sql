-- A receipt whose unidentified units reside away from its original receiving
-- warehouse cannot be represented by the legacy projection.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM inventory_receipt_locations l
        JOIN inventory_receipt_batches b ON b.id = l.receipt_batch_id
        WHERE l.unidentified_count > 0 AND l.warehouse_id <> b.warehouse_id
    ) THEN
        RAISE EXCEPTION 'cannot remove receipt locations while stock is away from its receiving warehouse';
    END IF;
END $$;

DROP TABLE inventory_receipt_locations;
