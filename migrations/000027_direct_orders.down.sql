-- Fails while direct orders exist, so no order silently loses its source.
ALTER TABLE commerce.orders DROP CONSTRAINT orders_source_check;
ALTER TABLE commerce.orders ADD CONSTRAINT orders_source_check CHECK (source IN ('rfq', 'offer'));
