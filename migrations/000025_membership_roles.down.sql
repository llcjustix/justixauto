-- Fails if a user belongs to several companies: revoke extra memberships first.
CREATE UNIQUE INDEX memberships_one_active_company_key
    ON identity_memberships (user_id) WHERE status = 'active';

INSERT INTO identity_user_roles (user_id, role_id)
SELECT DISTINCT m.user_id, mr.role_id
FROM identity_membership_roles mr
JOIN identity_memberships m ON m.id = mr.membership_id
ON CONFLICT DO NOTHING;

DROP TABLE identity_membership_roles;
