-- name: UpsertInvoice :exec
INSERT INTO invoices (id, envelope) VALUES (?, ?)
ON CONFLICT (id) DO UPDATE SET envelope = excluded.envelope, updated_at = CURRENT_TIMESTAMP;

-- name: GetInvoice :one
SELECT id, envelope, created_at, updated_at FROM invoices WHERE id = ?;

-- name: ListInvoices :many
SELECT id, envelope, created_at, updated_at FROM invoices ORDER BY created_at DESC;

-- name: DeleteInvoice :execrows
DELETE FROM invoices WHERE id = ?;
