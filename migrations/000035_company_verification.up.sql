-- Company commercial type and verification are independent from platform access.
ALTER TABLE identity_companies
    ADD COLUMN user_type text CHECK (user_type IN ('dealer', 'distributor')),
    ADD COLUMN verification text NOT NULL DEFAULT 'unverified' CHECK (verification IN ('unverified', 'submitted', 'verified', 'returned')),
    ADD COLUMN verification_reason text NOT NULL DEFAULT '' CHECK (length(verification_reason) <= 1000),
    ADD COLUMN verification_actor_id uuid REFERENCES identity_users(id),
    ADD COLUMN verification_at timestamptz,
    ADD COLUMN legal_form text NOT NULL DEFAULT '' CHECK (legal_form IN ('', 'llc', 'sole_proprietor', 'other')),
    ADD COLUMN tax_id text NOT NULL DEFAULT '' CHECK (length(tax_id) <= 100),
    ADD COLUMN bank_requisites text NOT NULL DEFAULT '' CHECK (length(bank_requisites) <= 2000);
