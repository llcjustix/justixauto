-- User decision 2026-09-27: a company orders from a partner without an offer
-- (offers are promotions/discounts), so an order can also be "direct".
ALTER TABLE commerce.orders DROP CONSTRAINT orders_source_check;
ALTER TABLE commerce.orders ADD CONSTRAINT orders_source_check CHECK (source IN ('rfq', 'offer', 'direct'));
