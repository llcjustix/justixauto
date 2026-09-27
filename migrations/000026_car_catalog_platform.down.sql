UPDATE identity.permissions SET deleted_at = NULL WHERE key = 'inventory.models.edit';
DELETE FROM identity.permissions WHERE key = 'platform.catalog.manage';
