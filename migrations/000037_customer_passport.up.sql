ALTER TABLE retail_customers
    ADD COLUMN passport_series text NOT NULL DEFAULT '' CHECK (length(passport_series) <= 100),
    ADD COLUMN passport_number text NOT NULL DEFAULT '' CHECK (length(passport_number) <= 100),
    ADD COLUMN passport_issuer text NOT NULL DEFAULT '' CHECK (length(passport_issuer) <= 500),
    ADD COLUMN passport_issued_on date,
    ADD COLUMN passport_expires_on date,
    ADD COLUMN birth_date date,
    ADD COLUMN personal_id text NOT NULL DEFAULT '' CHECK (length(personal_id) <= 100),
    ADD COLUMN registered_address text NOT NULL DEFAULT '' CHECK (length(registered_address) <= 1000),
    ADD COLUMN passport_copy_file_ids jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT retail_customers_passport_copy_file_ids_array CHECK (jsonb_typeof(passport_copy_file_ids) = 'array');
