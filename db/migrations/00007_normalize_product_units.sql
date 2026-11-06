-- +goose Up
-- Map common human-readable unit names to the official GOBL unit keys.
UPDATE products SET unit = 'h'     WHERE lower(trim(unit)) = 'hour';
UPDATE products SET unit = 'min'   WHERE lower(trim(unit)) = 'minute';
UPDATE products SET unit = 'mon'   WHERE lower(trim(unit)) = 'month';
UPDATE products SET unit = 'wk'    WHERE lower(trim(unit)) = 'week';
UPDATE products SET unit = 'yr'    WHERE lower(trim(unit)) = 'year';
UPDATE products SET unit = 'piece' WHERE lower(trim(unit)) IN ('pcs', 'pc', 'each');
UPDATE products SET unit = 'item'  WHERE lower(trim(unit)) = 'items';

-- +goose Down
SELECT 1;
