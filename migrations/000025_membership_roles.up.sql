-- User decisions 2026-09-27: the company selector is a business requirement,
-- so a user may belong to several companies (e.g. a company admin opens a
-- new company from Realization). Roles are per company: company roles attach
-- to the membership; platform roles stay on the user.

CREATE TABLE identity_membership_roles (
    membership_id uuid NOT NULL REFERENCES identity_memberships (id) ON DELETE CASCADE,
    role_id       uuid NOT NULL REFERENCES identity_roles (id),
    PRIMARY KEY (membership_id, role_id)
);

-- Company roles held by a user move to each of the user's active memberships
-- (until now a user had at most one).
INSERT INTO identity_membership_roles (membership_id, role_id)
SELECT m.id, ur.role_id
FROM identity_user_roles ur
JOIN identity_roles r ON r.id = ur.role_id AND r.scope = 'company'
JOIN identity_memberships m ON m.user_id = ur.user_id AND m.status = 'active';

DELETE FROM identity_user_roles ur
USING identity_roles r
WHERE r.id = ur.role_id AND r.scope = 'company';

-- One user may now belong to several companies.
DROP INDEX memberships_one_active_company_key;
