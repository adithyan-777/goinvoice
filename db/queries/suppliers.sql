-- name: ListSuppliers :many
SELECT * FROM suppliers ORDER BY name;

-- name: GetSupplier :one
SELECT * FROM suppliers WHERE id = ?;

-- name: UpsertSupplier :exec
INSERT INTO suppliers (
  id, uuid, regime, label, name, alias, tax_country, tax_code, tax_scheme, tax_type,
  agent_id, email, address_line, locality, post_code, country, ext, meta
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  uuid = excluded.uuid,
  regime = excluded.regime,
  label = excluded.label,
  name = excluded.name,
  alias = excluded.alias,
  tax_country = excluded.tax_country,
  tax_code = excluded.tax_code,
  tax_scheme = excluded.tax_scheme,
  tax_type = excluded.tax_type,
  agent_id = excluded.agent_id,
  email = excluded.email,
  address_line = excluded.address_line,
  locality = excluded.locality,
  post_code = excluded.post_code,
  country = excluded.country,
  ext = excluded.ext,
  meta = excluded.meta,
  updated_at = CURRENT_TIMESTAMP;

-- name: DeleteSupplier :execrows
DELETE FROM suppliers WHERE id = ?;
