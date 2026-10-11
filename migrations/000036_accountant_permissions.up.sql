INSERT INTO identity_permissions (key, scope, name, assignable) VALUES
    ('retail.invoices.manage',   'company', 'Продажи: управление счетами', true),
    ('commerce.invoices.manage', 'company', 'Закупки: управление счетами', true)
ON CONFLICT (key) DO NOTHING;

-- Each seeded role has a deterministic ID derived from its company. This is a
-- migration marker, not a system role: company administrators may edit and
-- assign the resulting private role like any other company role.
WITH accountant_roles AS (
    SELECT
        (
            substr(md5('justixauto:p04:accountant:' || c.id::text), 1, 8) || '-' ||
            substr(md5('justixauto:p04:accountant:' || c.id::text), 9, 4) || '-' ||
            substr(md5('justixauto:p04:accountant:' || c.id::text), 13, 4) || '-' ||
            substr(md5('justixauto:p04:accountant:' || c.id::text), 17, 4) || '-' ||
            substr(md5('justixauto:p04:accountant:' || c.id::text), 21, 12)
        )::uuid AS id,
        c.id AS company_id,
        CASE WHEN EXISTS (
            SELECT 1 FROM identity_roles r
            WHERE r.company_id = c.id AND r.deleted_at IS NULL AND lower(r.name) = lower('Бухгалтер')
        ) THEN 'Бухгалтер (счета и оплаты)' ELSE 'Бухгалтер' END AS name
    FROM identity_companies c
    WHERE c.kind = 'seller' AND c.deleted_at IS NULL
), inserted AS (
    INSERT INTO identity_roles (id, system_key, name, scope, company_id, version, created_at, updated_at)
    SELECT id, NULL, name, 'company', company_id, 1, now(), now()
    FROM accountant_roles
    RETURNING id
)
INSERT INTO identity_role_permissions (role_id, permission)
SELECT i.id, p.key
FROM inserted i
CROSS JOIN (VALUES
    ('retail.read'), ('retail.invoices.manage'), ('retail.payments.accept'),
    ('commerce.read'), ('commerce.invoices.manage'), ('commerce.payments.accept'),
    ('documents.read'), ('documents.upload'), ('documents.sensitive.download')
) AS p(key);
