-- Child collections for the GOBL org.Party shape. Each row belongs to exactly
-- one customer or supplier; list ordering is preserved by `position`.

-- ---------- addresses ----------

-- name: ListPartyAddressesByCustomer :many
SELECT id, customer_id, supplier_id, position, uuid, label, po_box, num, floor, block, door,
       street, street_extra, locality, region, state, code, country, coords, meta
FROM party_addresses WHERE customer_id = ? ORDER BY position, id;

-- name: ListPartyAddressesBySupplier :many
SELECT id, customer_id, supplier_id, position, uuid, label, po_box, num, floor, block, door,
       street, street_extra, locality, region, state, code, country, coords, meta
FROM party_addresses WHERE supplier_id = ? ORDER BY position, id;

-- name: InsertPartyAddress :exec
INSERT INTO party_addresses (
  id, customer_id, supplier_id, position, uuid, label, po_box, num, floor, block, door,
  street, street_extra, locality, region, state, code, country, coords, meta
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeletePartyAddressesByCustomer :exec
DELETE FROM party_addresses WHERE customer_id = ?;

-- name: DeletePartyAddressesBySupplier :exec
DELETE FROM party_addresses WHERE supplier_id = ?;

-- ---------- emails ----------

-- name: ListPartyEmailsByCustomer :many
SELECT id, customer_id, supplier_id, position, uuid, label, addr, meta
FROM party_emails WHERE customer_id = ? ORDER BY position, id;

-- name: ListPartyEmailsBySupplier :many
SELECT id, customer_id, supplier_id, position, uuid, label, addr, meta
FROM party_emails WHERE supplier_id = ? ORDER BY position, id;

-- name: InsertPartyEmail :exec
INSERT INTO party_emails (id, customer_id, supplier_id, position, uuid, label, addr, meta)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeletePartyEmailsByCustomer :exec
DELETE FROM party_emails WHERE customer_id = ?;

-- name: DeletePartyEmailsBySupplier :exec
DELETE FROM party_emails WHERE supplier_id = ?;

-- ---------- telephones ----------

-- name: ListPartyTelephonesByCustomer :many
SELECT id, customer_id, supplier_id, position, uuid, label, num
FROM party_telephones WHERE customer_id = ? ORDER BY position, id;

-- name: ListPartyTelephonesBySupplier :many
SELECT id, customer_id, supplier_id, position, uuid, label, num
FROM party_telephones WHERE supplier_id = ? ORDER BY position, id;

-- name: InsertPartyTelephone :exec
INSERT INTO party_telephones (id, customer_id, supplier_id, position, uuid, label, num)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: DeletePartyTelephonesByCustomer :exec
DELETE FROM party_telephones WHERE customer_id = ?;

-- name: DeletePartyTelephonesBySupplier :exec
DELETE FROM party_telephones WHERE supplier_id = ?;

-- ---------- websites ----------

-- name: ListPartyWebsitesByCustomer :many
SELECT id, customer_id, supplier_id, position, uuid, label, title, url
FROM party_websites WHERE customer_id = ? ORDER BY position, id;

-- name: ListPartyWebsitesBySupplier :many
SELECT id, customer_id, supplier_id, position, uuid, label, title, url
FROM party_websites WHERE supplier_id = ? ORDER BY position, id;

-- name: InsertPartyWebsite :exec
INSERT INTO party_websites (id, customer_id, supplier_id, position, uuid, label, title, url)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeletePartyWebsitesByCustomer :exec
DELETE FROM party_websites WHERE customer_id = ?;

-- name: DeletePartyWebsitesBySupplier :exec
DELETE FROM party_websites WHERE supplier_id = ?;

-- ---------- identities ----------

-- name: ListPartyIdentitiesByCustomer :many
SELECT id, customer_id, supplier_id, position, uuid, label, scope, country, key, type,
       code, description, ext
FROM party_identities WHERE customer_id = ? ORDER BY position, id;

-- name: ListPartyIdentitiesBySupplier :many
SELECT id, customer_id, supplier_id, position, uuid, label, scope, country, key, type,
       code, description, ext
FROM party_identities WHERE supplier_id = ? ORDER BY position, id;

-- name: InsertPartyIdentity :exec
INSERT INTO party_identities (
  id, customer_id, supplier_id, position, uuid, label, scope, country, key, type,
  code, description, ext
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeletePartyIdentitiesByCustomer :exec
DELETE FROM party_identities WHERE customer_id = ?;

-- name: DeletePartyIdentitiesBySupplier :exec
DELETE FROM party_identities WHERE supplier_id = ?;

-- ---------- endpoints ----------

-- name: ListPartyEndpointsByCustomer :many
SELECT id, customer_id, supplier_id, position, uuid, label, uri
FROM party_endpoints WHERE customer_id = ? ORDER BY position, id;

-- name: ListPartyEndpointsBySupplier :many
SELECT id, customer_id, supplier_id, position, uuid, label, uri
FROM party_endpoints WHERE supplier_id = ? ORDER BY position, id;

-- name: InsertPartyEndpoint :exec
INSERT INTO party_endpoints (id, customer_id, supplier_id, position, uuid, label, uri)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: DeletePartyEndpointsByCustomer :exec
DELETE FROM party_endpoints WHERE customer_id = ?;

-- name: DeletePartyEndpointsBySupplier :exec
DELETE FROM party_endpoints WHERE supplier_id = ?;

-- ---------- inboxes ----------

-- name: ListPartyInboxesByCustomer :many
SELECT id, customer_id, supplier_id, position, uuid, label, key, scheme, code, url, email
FROM party_inboxes WHERE customer_id = ? ORDER BY position, id;

-- name: ListPartyInboxesBySupplier :many
SELECT id, customer_id, supplier_id, position, uuid, label, key, scheme, code, url, email
FROM party_inboxes WHERE supplier_id = ? ORDER BY position, id;

-- name: InsertPartyInbox :exec
INSERT INTO party_inboxes (id, customer_id, supplier_id, position, uuid, label, key, scheme, code, url, email)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeletePartyInboxesByCustomer :exec
DELETE FROM party_inboxes WHERE customer_id = ?;

-- name: DeletePartyInboxesBySupplier :exec
DELETE FROM party_inboxes WHERE supplier_id = ?;

-- ---------- logos (org.Image) ----------

-- name: ListPartyLogosByCustomer :many
SELECT id, customer_id, supplier_id, position, uuid, label, url, data, mime, description,
       alt, height, width, digest, meta
FROM party_logos WHERE customer_id = ? ORDER BY position, id;

-- name: ListPartyLogosBySupplier :many
SELECT id, customer_id, supplier_id, position, uuid, label, url, data, mime, description,
       alt, height, width, digest, meta
FROM party_logos WHERE supplier_id = ? ORDER BY position, id;

-- name: InsertPartyLogo :exec
INSERT INTO party_logos (
  id, customer_id, supplier_id, position, uuid, label, url, data, mime, description,
  alt, height, width, digest, meta
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeletePartyLogosByCustomer :exec
DELETE FROM party_logos WHERE customer_id = ?;

-- name: DeletePartyLogosBySupplier :exec
DELETE FROM party_logos WHERE supplier_id = ?;

-- ---------- people ----------

-- name: ListPartyPeopleByCustomer :many
SELECT id, customer_id, supplier_id, position, uuid, label, key, role, name_uuid, name_alias,
       name_prefix, name_given, name_middle, name_surname, name_surname2, name_suffix,
       name_meta, identities, addresses, emails, telephones, avatars, meta
FROM party_people WHERE customer_id = ? ORDER BY position, id;

-- name: ListPartyPeopleBySupplier :many
SELECT id, customer_id, supplier_id, position, uuid, label, key, role, name_uuid, name_alias,
       name_prefix, name_given, name_middle, name_surname, name_surname2, name_suffix,
       name_meta, identities, addresses, emails, telephones, avatars, meta
FROM party_people WHERE supplier_id = ? ORDER BY position, id;

-- name: InsertPartyPerson :exec
INSERT INTO party_people (
  id, customer_id, supplier_id, position, uuid, label, key, role, name_uuid, name_alias,
  name_prefix, name_given, name_middle, name_surname, name_surname2, name_suffix,
  name_meta, identities, addresses, emails, telephones, avatars, meta
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: DeletePartyPeopleByCustomer :exec
DELETE FROM party_people WHERE customer_id = ?;

-- name: DeletePartyPeopleBySupplier :exec
DELETE FROM party_people WHERE supplier_id = ?;
