-- +goose Up
-- The license product has no meaningful unit of measure; clear it instead of
-- the old (invalid) "seat" value.
UPDATE products SET unit = NULL WHERE id = 'prod-license' OR ref = 'LIC-002';

-- +goose Down
UPDATE products SET unit = 'unit' WHERE id = 'prod-license' OR ref = 'LIC-002';
