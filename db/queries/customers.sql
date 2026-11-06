-- name: ListCustomers :many
SELECT id, name, tax_country, tax_code, email, address_line, locality, post_code, country, created_at
FROM customers ORDER BY name;

-- name: GetCustomer :one
SELECT id, name, tax_country, tax_code, email, address_line, locality, post_code, country, created_at
FROM customers WHERE id = ?;

-- name: CreateCustomer :exec
INSERT INTO customers (id, name, tax_country, tax_code, email, address_line, locality, post_code, country)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateCustomer :execrows
UPDATE customers
SET name = ?, tax_country = ?, tax_code = ?, email = ?, address_line = ?,
    locality = ?, post_code = ?, country = ?
WHERE id = ?;

-- name: DeleteCustomer :execrows
DELETE FROM customers WHERE id = ?;
