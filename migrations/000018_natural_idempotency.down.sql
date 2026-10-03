DROP INDEX IF EXISTS files_content_key;
DROP INDEX IF EXISTS document_requests_open_title_key;
DROP INDEX IF EXISTS shipment_milestones_key;
DROP INDEX IF EXISTS retail_payment_evidence_reference_key;
DROP INDEX IF EXISTS payment_evidence_reference_key;
CREATE TABLE platform_idempotency_keys (
    actor_id      uuid        NOT NULL,
    key           uuid        NOT NULL,
    request_hash  bytea       NOT NULL,
    status        text        NOT NULL CHECK (status IN ('in_progress', 'completed')),
    response_code integer,
    response_etag text        NOT NULL DEFAULT '',
    response_body bytea,
    created_at    timestamptz NOT NULL,
    completed_at  timestamptz,
    PRIMARY KEY (actor_id, key)
);
