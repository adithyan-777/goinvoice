-- +goose Up
-- "seat" is not a GOBL unit key; use the generic "unit" instead. This fixes
-- the seeded license product and any user-created rows that copied it.
UPDATE products SET unit = 'unit' WHERE lower(trim(unit)) = 'seat';

-- +goose Down
UPDATE products SET unit = 'seat' WHERE ref = 'LIC-002' AND lower(trim(unit)) = 'unit';
