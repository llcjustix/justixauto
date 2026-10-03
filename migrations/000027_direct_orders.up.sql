-- User decision 2026-09-27: a company orders from a partner without an offer
-- (offers are promotions/discounts), so an order can also be "direct".
ALTER TABLE commerce_orders DROP CONSTRAINT IF EXISTS commerce_orders_source_check, DROP CONSTRAINT IF EXISTS orders_source_check;
ALTER TABLE commerce_orders ADD CONSTRAINT orders_source_check CHECK (source IN ('rfq', 'offer', 'direct'));
