DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM inventory_vehicle_units WHERE exterior_color IS NOT NULL OR interior_color IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot remove vehicle colors while vehicle selections exist';
    END IF;
    IF EXISTS (SELECT 1 FROM inventory_receipt_batches WHERE exterior_color IS NOT NULL OR interior_color IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot remove vehicle colors while receipt selections exist';
    END IF;
    IF EXISTS (SELECT 1 FROM inventory_model_specifications WHERE jsonb_array_length(exterior_colors) > 1 OR jsonb_array_length(interior_colors) > 1) THEN
        RAISE EXCEPTION 'cannot remove multi-color palettes';
    END IF;
    IF EXISTS (SELECT 1 FROM retail_deals WHERE vehicle_snapshot IS NOT NULL) THEN
        RAISE EXCEPTION 'cannot remove retail vehicle snapshots';
    END IF;

    ALTER TABLE retail_deals DROP COLUMN vehicle_snapshot;
    ALTER TABLE inventory_receipt_batches DROP COLUMN exterior_color, DROP COLUMN interior_color;
    ALTER TABLE inventory_vehicle_units DROP COLUMN exterior_color, DROP COLUMN interior_color;
    ALTER TABLE inventory_model_specifications
        DROP CONSTRAINT model_specifications_exterior_colors_array,
        DROP CONSTRAINT model_specifications_interior_colors_array,
        DROP COLUMN exterior_colors,
        DROP COLUMN interior_colors;
END $$;
