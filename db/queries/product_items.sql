-- Child collections for the GOBL org.Item shape. Ordering is preserved by
-- `position`; rows cascade with their product.

-- ---------- identities ----------

-- name: ListProductIdentities :many
SELECT id, product_id, position, uuid, label, scope, country, key, type,
       code, description, ext
FROM product_identities WHERE product_id = ? ORDER BY position, id;

-- name: InsertProductIdentity :exec
INSERT INTO product_identities (
  id, product_id, position, uuid, label, scope, country, key, type,
  code, description, ext
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteProductIdentities :exec
DELETE FROM product_identities WHERE product_id = ?;

-- ---------- attributes ----------

-- name: ListProductAttributes :many
SELECT id, product_id, position, label, key, type, text, code, amount, unit,
       date, ext
FROM product_attributes WHERE product_id = ? ORDER BY position, id;

-- name: InsertProductAttribute :exec
INSERT INTO product_attributes (
  id, product_id, position, label, key, type, text, code, amount, unit,
  date, ext
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteProductAttributes :exec
DELETE FROM product_attributes WHERE product_id = ?;

-- ---------- images ----------

-- name: ListProductImages :many
SELECT id, product_id, position, uuid, label, url, data, mime, description,
       alt, height, width, digest, meta
FROM product_images WHERE product_id = ? ORDER BY position, id;

-- name: InsertProductImage :exec
INSERT INTO product_images (
  id, product_id, position, uuid, label, url, data, mime, description,
  alt, height, width, digest, meta
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeleteProductImages :exec
DELETE FROM product_images WHERE product_id = ?;

-- ---------- alternative prices ----------

-- name: ListProductAltPrices :many
SELECT id, product_id, position, label, currency, value
FROM product_alt_prices WHERE product_id = ? ORDER BY position, id;

-- name: InsertProductAltPrice :exec
INSERT INTO product_alt_prices (id, product_id, position, label, currency, value)
VALUES (?, ?, ?, ?, ?, ?);

-- name: DeleteProductAltPrices :exec
DELETE FROM product_alt_prices WHERE product_id = ?;
