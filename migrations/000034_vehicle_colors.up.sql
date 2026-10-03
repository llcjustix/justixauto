ALTER TABLE inventory_model_specifications
    ADD COLUMN exterior_colors jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN interior_colors jsonb NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT model_specifications_exterior_colors_array CHECK (jsonb_typeof(exterior_colors) = 'array'),
    ADD CONSTRAINT model_specifications_interior_colors_array CHECK (jsonb_typeof(interior_colors) = 'array');

UPDATE inventory_model_specifications
SET exterior_colors = jsonb_build_array(exterior_color)
WHERE btrim(exterior_color) <> '';

UPDATE inventory_model_specifications
SET interior_colors = jsonb_build_array(interior_color)
WHERE btrim(interior_color) <> '';

ALTER TABLE inventory_vehicle_units
    ADD COLUMN exterior_color text,
    ADD COLUMN interior_color text;

ALTER TABLE inventory_receipt_batches
    ADD COLUMN exterior_color text,
    ADD COLUMN interior_color text;

UPDATE inventory_vehicle_units u
SET exterior_color = NULLIF(btrim(s.exterior_color), ''),
    interior_color = NULLIF(btrim(s.interior_color), '')
FROM inventory_model_specifications s
WHERE u.model_id = s.model_id AND u.spec_version = s.spec_version;

UPDATE inventory_receipt_batches b
SET exterior_color = NULLIF(btrim(s.exterior_color), ''),
    interior_color = NULLIF(btrim(s.interior_color), '')
FROM inventory_model_specifications s
WHERE b.model_id = s.model_id AND b.spec_version = s.spec_version;

ALTER TABLE retail_deals ADD COLUMN vehicle_snapshot jsonb;
