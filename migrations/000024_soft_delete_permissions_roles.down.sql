-- Fails if a deleted and a live role share a name: rename one first.
-- Deleted permissions and roles become visible again after rollback.
DROP INDEX roles_company_name_key;
DROP INDEX roles_name_key;
CREATE UNIQUE INDEX roles_name_key ON identity_roles (lower(name)) WHERE company_id IS NULL;
CREATE UNIQUE INDEX roles_company_name_key ON identity_roles (company_id, lower(name)) WHERE company_id IS NOT NULL;

ALTER TABLE identity_roles DROP COLUMN deleted_at;
ALTER TABLE identity_permissions DROP COLUMN deleted_at;
