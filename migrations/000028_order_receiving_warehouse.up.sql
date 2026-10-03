-- User decision 2026-10-01: the buyer chooses the receiving warehouse when
-- ordering; shipped vehicles enter it at once, without a manual receipt.
-- NULL = an older order (or one from an RFQ): the buyer still receives by hand.
ALTER TABLE commerce_orders ADD COLUMN receiving_warehouse_id uuid;
