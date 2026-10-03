UPDATE identity_permissions SET deleted_at = NULL WHERE key = 'inventory.models.edit';
DELETE FROM identity_permissions WHERE key = 'platform.catalog.manage';
