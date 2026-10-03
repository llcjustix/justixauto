ALTER TABLE retail_invoices
    DROP CONSTRAINT IF EXISTS retail_invoices_purpose_check, DROP CONSTRAINT IF EXISTS invoices_purpose_check,
    ADD CONSTRAINT invoices_purpose_check CHECK (
        purpose IN ('vehicle-payment', 'first-installment', 'registration', 'monthly-installment')
    );
