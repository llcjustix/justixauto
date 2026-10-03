-- Fails if two companies' roles share a name: rename them first.
DROP INDEX roles_company_name_key;
DROP INDEX roles_name_key;
CREATE UNIQUE INDEX roles_name_key ON identity_roles (lower(name));

ALTER TABLE identity_roles
    DROP CONSTRAINT roles_company_scope_check,
    DROP COLUMN company_id;

DROP TABLE identity_permissions;
