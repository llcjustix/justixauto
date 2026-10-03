CREATE TABLE retail_installment_plans (
    id                       uuid PRIMARY KEY,
    deal_id                  uuid NOT NULL UNIQUE REFERENCES retail_deals,
    company_id               uuid NOT NULL,
    contract_reference       text NOT NULL,
    contract_signed_on       date NOT NULL,
    contract_file_ids        jsonb NOT NULL,
    contract_total_minor     text NOT NULL,
    currency                 text NOT NULL,
    down_payment_invoice_id  uuid NOT NULL UNIQUE REFERENCES retail_invoices,
    down_payment_minor       text NOT NULL,
    created_by               uuid NOT NULL,
    created_at               timestamptz NOT NULL
);

ALTER TABLE retail_invoices
    ADD COLUMN installment_plan_id uuid REFERENCES retail_installment_plans,
    ADD COLUMN installment_number integer;

ALTER TABLE retail_invoices
    ADD CONSTRAINT retail_invoices_installment_link_check CHECK (
        (purpose = 'monthly-installment' AND installment_plan_id IS NOT NULL
            AND installment_number IS NOT NULL AND installment_number > 0 AND due_date IS NOT NULL)
        OR
        (purpose <> 'monthly-installment' AND installment_plan_id IS NULL AND installment_number IS NULL)
    );

DROP INDEX retail_invoices_purpose_key;
CREATE UNIQUE INDEX retail_invoices_purpose_key
    ON retail_invoices (deal_id, purpose) WHERE status = 'issued' AND purpose <> 'monthly-installment';
CREATE UNIQUE INDEX retail_invoices_installment_number_key
    ON retail_invoices (installment_plan_id, installment_number) WHERE installment_plan_id IS NOT NULL;
