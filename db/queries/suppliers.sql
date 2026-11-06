-- name: ListSuppliers :many
SELECT id, name, tax_country, tax_code, email, address_line, locality, post_code, country, created_at
FROM suppliers ORDER BY name;

-- name: GetSupplier :one
SELECT id, name, tax_country, tax_code, email, address_line, locality, post_code, country, created_at
FROM suppliers WHERE id = ?;

-- name: CreateSupplier :exec
INSERT INTO suppliers (id, name, tax_country, tax_code, email, address_line, locality, post_code, country)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateSupplier :execrows
UPDATE suppliers
SET name = ?, tax_country = ?, tax_code = ?, email = ?, address_line = ?,
    locality = ?, post_code = ?, country = ?
WHERE id = ?;

-- name: DeleteSupplier :execrows
DELETE FROM suppliers WHERE id = ?;
