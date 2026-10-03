DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM retail_installment_payments)
       OR EXISTS (SELECT 1 FROM retail_payment_evidence WHERE payment_group_id IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot remove retail installment payments while grouped financial facts exist';
    END IF;
END $$;

DROP INDEX retail_payment_evidence_group_invoice_key;
ALTER TABLE retail_payment_evidence DROP COLUMN payment_group_id;
DROP INDEX retail_installment_payments_plan_idx;
DROP INDEX retail_installment_payments_reference_key;
DROP TABLE retail_installment_payments;
