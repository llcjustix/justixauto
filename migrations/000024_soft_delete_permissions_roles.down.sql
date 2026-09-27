-- Fails if a deleted and a live role share a name: rename one first.
-- Deleted permissions and roles become visible again after rollback.
DROP INDEX identity.roles_company_name_key;
DROP INDEX identity.roles_name_key;
CREATE UNIQUE INDEX roles_name_key ON identity.roles (lower(name)) WHERE company_id IS NULL;
CREATE UNIQUE INDEX roles_company_name_key ON identity.roles (company_id, lower(name)) WHERE company_id IS NOT NULL;

ALTER TABLE identity.roles DROP COLUMN deleted_at;
ALTER TABLE identity.permissions DROP COLUMN deleted_at;
