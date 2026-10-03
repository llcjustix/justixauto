DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM retail_invoices WHERE purpose = 'monthly-installment') THEN
        RAISE EXCEPTION 'cannot remove monthly invoice purpose while monthly invoices exist';
    END IF;

    ALTER TABLE retail_invoices
        DROP CONSTRAINT invoices_purpose_check,
        ADD CONSTRAINT invoices_purpose_check CHECK (
            purpose IN ('vehicle-payment', 'first-installment', 'registration')
        );
END $$;
