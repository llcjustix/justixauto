-- Fails if two companies' roles share a name: rename them first.
DROP INDEX identity.roles_company_name_key;
DROP INDEX identity.roles_name_key;
CREATE UNIQUE INDEX roles_name_key ON identity.roles (lower(name));

ALTER TABLE identity.roles
    DROP CONSTRAINT roles_company_scope_check,
    DROP COLUMN company_id;

DROP TABLE identity.permissions;
