-- Restores the pre-2026-09-26 constraints. This fails if any row now
-- violates them, e.g. a company with an empty country/registration number or
-- a user with an empty e-mail (or two users who share one): such rows must be
-- fixed or removed by hand before rolling back.

DROP INDEX users_email_key;
CREATE UNIQUE INDEX users_email_key ON identity_users (lower(email));

ALTER TABLE identity_users DROP CONSTRAINT users_email_check;
ALTER TABLE identity_users
    ADD CONSTRAINT users_email_check CHECK (length(email) BETWEEN 3 AND 254);

DROP INDEX companies_country_registration_key;
CREATE UNIQUE INDEX companies_country_registration_key ON identity_companies (
    (CASE WHEN country_key <> '' THEN country_key ELSE lower(country) END),
    lower(registration_number));

ALTER TABLE identity_companies DROP CONSTRAINT companies_registration_number_check;
ALTER TABLE identity_companies
    ADD CONSTRAINT companies_registration_number_check CHECK (length(registration_number) BETWEEN 1 AND 64);

ALTER TABLE identity_companies DROP CONSTRAINT companies_country_check;
ALTER TABLE identity_companies
    ADD CONSTRAINT companies_country_check CHECK (length(country) BETWEEN 1 AND 100);
