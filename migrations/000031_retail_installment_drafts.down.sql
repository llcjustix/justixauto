DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM retail_deals WHERE installment_draft IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot remove retail installment drafts while persisted drafts exist';
    END IF;
END $$;

ALTER TABLE retail_deals DROP COLUMN installment_draft;
