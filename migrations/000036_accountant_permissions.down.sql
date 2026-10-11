-- Remove only P04's deterministic backfill rows. Future company presets use
-- random IDs and are intentionally left untouched by a migration rollback.
WITH seeded AS (
    SELECT (
        substr(md5('justixauto:p04:accountant:' || c.id::text), 1, 8) || '-' ||
        substr(md5('justixauto:p04:accountant:' || c.id::text), 9, 4) || '-' ||
        substr(md5('justixauto:p04:accountant:' || c.id::text), 13, 4) || '-' ||
        substr(md5('justixauto:p04:accountant:' || c.id::text), 17, 4) || '-' ||
        substr(md5('justixauto:p04:accountant:' || c.id::text), 21, 12)
    )::uuid AS id
    FROM identity_companies c
    WHERE c.kind = 'seller'
)
DELETE FROM identity_roles r
USING seeded s
WHERE r.id = s.id AND r.system_key IS NULL AND r.scope = 'company';

DELETE FROM identity_permissions
WHERE key IN ('retail.invoices.manage', 'commerce.invoices.manage');
