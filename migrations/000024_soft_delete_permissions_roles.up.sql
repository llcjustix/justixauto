-- User decision 2026-09-27: permissions and roles are soft-deleted. A deleted
-- permission leaves the catalog and stops granting through every role; a
-- deleted role leaves every list, cannot be assigned and stops granting to
-- the users who held it. Rows stay for history and the audit log.

ALTER TABLE identity.permissions ADD COLUMN deleted_at timestamptz;
ALTER TABLE identity.roles ADD COLUMN deleted_at timestamptz;

-- A deleted role's name can be used again.
DROP INDEX identity.roles_name_key;
DROP INDEX identity.roles_company_name_key;
CREATE UNIQUE INDEX roles_name_key ON identity.roles (lower(name))
    WHERE company_id IS NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX roles_company_name_key ON identity.roles (company_id, lower(name))
    WHERE company_id IS NOT NULL AND deleted_at IS NULL;
