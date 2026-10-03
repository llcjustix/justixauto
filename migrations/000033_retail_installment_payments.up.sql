CREATE TABLE retail_installment_payments (
    id                 uuid        PRIMARY KEY,
    company_id         uuid        NOT NULL,
    deal_id            uuid        NOT NULL REFERENCES retail_deals,
    installment_plan_id uuid       NOT NULL REFERENCES retail_installment_plans,
    amount_minor       text        NOT NULL,
    currency           text        NOT NULL,
    paid_on            date        NOT NULL,
    external_reference text        NOT NULL,
    attachment_ids     jsonb       NOT NULL,
    status             text        NOT NULL CHECK (status IN ('submitted', 'accepted', 'rejected')),
    decision_reason    text        NOT NULL DEFAULT '',
    submitted_by       uuid        NOT NULL,
    decided_by         uuid,
    version            bigint      NOT NULL CHECK (version >= 1),
    created_at         timestamptz NOT NULL,
    decided_at         timestamptz
);

CREATE UNIQUE INDEX retail_installment_payments_reference_key
    ON retail_installment_payments (deal_id, external_reference) WHERE status <> 'rejected';
CREATE INDEX retail_installment_payments_plan_idx
    ON retail_installment_payments (installment_plan_id);

ALTER TABLE retail_payment_evidence
    ADD COLUMN payment_group_id uuid REFERENCES retail_installment_payments;
CREATE UNIQUE INDEX retail_payment_evidence_group_invoice_key
    ON retail_payment_evidence (payment_group_id, invoice_id) WHERE payment_group_id IS NOT NULL;
