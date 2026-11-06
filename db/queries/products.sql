-- name: ListProducts :many
SELECT * FROM products ORDER BY ref;

-- name: GetProduct :one
SELECT * FROM products WHERE id = ?;

-- name: UpsertProduct :exec
INSERT INTO products (
  id, uuid, ref, key, name, description, currency, price, list, discount, per, unit, origin, ext, meta
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  uuid = excluded.uuid,
  ref = excluded.ref,
  key = excluded.key,
  name = excluded.name,
  description = excluded.description,
  currency = excluded.currency,
  price = excluded.price,
  list = excluded.list,
  discount = excluded.discount,
  per = excluded.per,
  unit = excluded.unit,
  origin = excluded.origin,
  ext = excluded.ext,
  meta = excluded.meta,
  updated_at = CURRENT_TIMESTAMP;

-- name: DeleteProduct :execrows
DELETE FROM products WHERE id = ?;
