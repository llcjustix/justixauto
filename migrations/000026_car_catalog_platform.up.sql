-- User decision 2026-09-27: only the platform admin maintains the shared car
-- catalog (Admin → «Каталог автомобилей»); companies pick models from it.
INSERT INTO identity_permissions (key, scope, name, assignable)
VALUES ('platform.catalog.manage', 'platform', 'Платформа: каталог автомобилей', true)
ON CONFLICT (key) DO UPDATE SET scope = EXCLUDED.scope, name = EXCLUDED.name, deleted_at = NULL;

-- The company permission to edit models is retired (soft-deleted): it grants
-- nothing and leaves role pickers.
UPDATE identity_permissions SET deleted_at = now()
WHERE key = 'inventory.models.edit' AND deleted_at IS NULL;
