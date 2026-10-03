DROP TABLE IF EXISTS identity_audit_events;
DROP TABLE IF EXISTS identity_session_branches;
DROP TABLE IF EXISTS identity_sessions;
DROP TABLE IF EXISTS identity_membership_branches;
DROP TABLE IF EXISTS identity_memberships;
DROP TABLE IF EXISTS identity_branches;
DROP TABLE IF EXISTS identity_user_roles;
DROP TABLE IF EXISTS identity_role_permissions;
DROP TABLE IF EXISTS identity_roles;
DROP TABLE IF EXISTS identity_users;

DROP INDEX companies_country_registration_key;
ALTER TABLE identity_companies
    DROP COLUMN legal_name, DROP COLUMN country_key, DROP COLUMN region_key,
    DROP COLUMN email, DROP COLUMN address, DROP COLUMN phone,
    DROP CONSTRAINT companies_kind_check;
UPDATE identity_companies SET kind = 'insurer' WHERE kind = 'insurance';
ALTER TABLE identity_companies
    ADD CONSTRAINT companies_kind_check CHECK (kind IN ('seller', 'bank', 'mfo', 'insurer'));
CREATE UNIQUE INDEX companies_country_registration_key
    ON identity_companies (lower(country), registration_number);
