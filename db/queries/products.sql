-- name: ListProducts :many
SELECT id, ref, name, description, price, currency, unit, created_at
FROM products ORDER BY ref;

-- name: GetProduct :one
SELECT id, ref, name, description, price, currency, unit, created_at
FROM products WHERE id = ?;

-- name: CreateProduct :exec
INSERT INTO products (id, ref, name, description, price, currency, unit)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateProduct :execrows
UPDATE products
SET ref = ?, name = ?, description = ?, price = ?, currency = ?, unit = ?
WHERE id = ?;

-- name: DeleteProduct :execrows
DELETE FROM products WHERE id = ?;
