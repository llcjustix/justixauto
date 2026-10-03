-- Uploaded proof files attached to payment claims (documents_files IDs), and
-- a stable insertion order for claims recorded at the same instant.
ALTER TABLE commerce_payment_evidence
    ADD COLUMN attachment_ids jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN seq bigint GENERATED ALWAYS AS IDENTITY UNIQUE;
ALTER TABLE retail_payment_evidence
    ADD COLUMN attachment_ids jsonb NOT NULL DEFAULT '[]',
    ADD COLUMN seq bigint GENERATED ALWAYS AS IDENTITY UNIQUE;
