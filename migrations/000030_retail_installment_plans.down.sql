DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM retail_installment_plans)
       OR EXISTS (SELECT 1 FROM retail_invoices WHERE purpose = 'monthly-installment') THEN
        RAISE EXCEPTION 'cannot remove retail installment plans while saved financial facts exist';
    END IF;
END $$;

DROP INDEX retail_invoices_installment_number_key;
DROP INDEX retail_invoices_purpose_key;
ALTER TABLE retail_invoices DROP CONSTRAINT retail_invoices_installment_link_check;
ALTER TABLE retail_invoices DROP COLUMN installment_number;
ALTER TABLE retail_invoices DROP COLUMN installment_plan_id;
CREATE UNIQUE INDEX retail_invoices_purpose_key ON retail_invoices (deal_id, purpose) WHERE status = 'issued';
DROP TABLE retail_installment_plans;
