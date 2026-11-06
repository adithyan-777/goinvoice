-- +goose Up
-- Extend the customer/supplier catalog to hold the full GOBL org.Party shape.
-- Scalars live on the parent rows; repeated collections live in FK child tables.
-- Each child row belongs to exactly one customer or supplier (CHECK), and the
-- two nullable FKs keep referential integrity with ON DELETE CASCADE.

ALTER TABLE customers ADD COLUMN uuid TEXT;
ALTER TABLE customers ADD COLUMN regime TEXT;
ALTER TABLE customers ADD COLUMN label TEXT;
ALTER TABLE customers ADD COLUMN alias TEXT;
ALTER TABLE customers ADD COLUMN tax_scheme TEXT;
ALTER TABLE customers ADD COLUMN tax_type TEXT;
ALTER TABLE customers ADD COLUMN agent_id TEXT REFERENCES customers(id) ON DELETE SET NULL;
ALTER TABLE customers ADD COLUMN ext TEXT;
ALTER TABLE customers ADD COLUMN meta TEXT;
ALTER TABLE customers ADD COLUMN updated_at DATETIME;

ALTER TABLE suppliers ADD COLUMN uuid TEXT;
ALTER TABLE suppliers ADD COLUMN regime TEXT;
ALTER TABLE suppliers ADD COLUMN label TEXT;
ALTER TABLE suppliers ADD COLUMN alias TEXT;
ALTER TABLE suppliers ADD COLUMN tax_scheme TEXT;
ALTER TABLE suppliers ADD COLUMN tax_type TEXT;
ALTER TABLE suppliers ADD COLUMN agent_id TEXT REFERENCES suppliers(id) ON DELETE SET NULL;
ALTER TABLE suppliers ADD COLUMN ext TEXT;
ALTER TABLE suppliers ADD COLUMN meta TEXT;
ALTER TABLE suppliers ADD COLUMN updated_at DATETIME;

CREATE TABLE IF NOT EXISTS party_addresses (
  id TEXT PRIMARY KEY,
  customer_id TEXT REFERENCES customers(id) ON DELETE CASCADE,
  supplier_id TEXT REFERENCES suppliers(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  uuid TEXT,
  label TEXT,
  po_box TEXT,
  num TEXT,
  floor TEXT,
  block TEXT,
  door TEXT,
  street TEXT,
  street_extra TEXT,
  locality TEXT,
  region TEXT,
  state TEXT,
  code TEXT,
  country TEXT,
  coords TEXT,
  meta TEXT,
  CHECK ((customer_id IS NOT NULL) <> (supplier_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS party_addresses_customer_idx ON party_addresses(customer_id);
CREATE INDEX IF NOT EXISTS party_addresses_supplier_idx ON party_addresses(supplier_id);

CREATE TABLE IF NOT EXISTS party_emails (
  id TEXT PRIMARY KEY,
  customer_id TEXT REFERENCES customers(id) ON DELETE CASCADE,
  supplier_id TEXT REFERENCES suppliers(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  uuid TEXT,
  label TEXT,
  addr TEXT NOT NULL,
  meta TEXT,
  CHECK ((customer_id IS NOT NULL) <> (supplier_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS party_emails_customer_idx ON party_emails(customer_id);
CREATE INDEX IF NOT EXISTS party_emails_supplier_idx ON party_emails(supplier_id);

CREATE TABLE IF NOT EXISTS party_telephones (
  id TEXT PRIMARY KEY,
  customer_id TEXT REFERENCES customers(id) ON DELETE CASCADE,
  supplier_id TEXT REFERENCES suppliers(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  uuid TEXT,
  label TEXT,
  num TEXT NOT NULL,
  CHECK ((customer_id IS NOT NULL) <> (supplier_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS party_telephones_customer_idx ON party_telephones(customer_id);
CREATE INDEX IF NOT EXISTS party_telephones_supplier_idx ON party_telephones(supplier_id);

CREATE TABLE IF NOT EXISTS party_websites (
  id TEXT PRIMARY KEY,
  customer_id TEXT REFERENCES customers(id) ON DELETE CASCADE,
  supplier_id TEXT REFERENCES suppliers(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  uuid TEXT,
  label TEXT,
  title TEXT,
  url TEXT NOT NULL,
  CHECK ((customer_id IS NOT NULL) <> (supplier_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS party_websites_customer_idx ON party_websites(customer_id);
CREATE INDEX IF NOT EXISTS party_websites_supplier_idx ON party_websites(supplier_id);

CREATE TABLE IF NOT EXISTS party_identities (
  id TEXT PRIMARY KEY,
  customer_id TEXT REFERENCES customers(id) ON DELETE CASCADE,
  supplier_id TEXT REFERENCES suppliers(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  uuid TEXT,
  label TEXT,
  scope TEXT,
  country TEXT,
  key TEXT,
  type TEXT,
  code TEXT NOT NULL,
  description TEXT,
  ext TEXT,
  CHECK ((customer_id IS NOT NULL) <> (supplier_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS party_identities_customer_idx ON party_identities(customer_id);
CREATE INDEX IF NOT EXISTS party_identities_supplier_idx ON party_identities(supplier_id);

CREATE TABLE IF NOT EXISTS party_endpoints (
  id TEXT PRIMARY KEY,
  customer_id TEXT REFERENCES customers(id) ON DELETE CASCADE,
  supplier_id TEXT REFERENCES suppliers(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  uuid TEXT,
  label TEXT,
  uri TEXT NOT NULL,
  CHECK ((customer_id IS NOT NULL) <> (supplier_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS party_endpoints_customer_idx ON party_endpoints(customer_id);
CREATE INDEX IF NOT EXISTS party_endpoints_supplier_idx ON party_endpoints(supplier_id);

CREATE TABLE IF NOT EXISTS party_inboxes (
  id TEXT PRIMARY KEY,
  customer_id TEXT REFERENCES customers(id) ON DELETE CASCADE,
  supplier_id TEXT REFERENCES suppliers(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  uuid TEXT,
  label TEXT,
  key TEXT,
  scheme TEXT,
  code TEXT,
  url TEXT,
  email TEXT,
  CHECK ((customer_id IS NOT NULL) <> (supplier_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS party_inboxes_customer_idx ON party_inboxes(customer_id);
CREATE INDEX IF NOT EXISTS party_inboxes_supplier_idx ON party_inboxes(supplier_id);

CREATE TABLE IF NOT EXISTS party_logos (
  id TEXT PRIMARY KEY,
  customer_id TEXT REFERENCES customers(id) ON DELETE CASCADE,
  supplier_id TEXT REFERENCES suppliers(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  uuid TEXT,
  label TEXT,
  url TEXT,
  data BLOB,
  mime TEXT,
  description TEXT,
  alt TEXT,
  height INTEGER,
  width INTEGER,
  digest TEXT,
  meta TEXT,
  CHECK ((customer_id IS NOT NULL) <> (supplier_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS party_logos_customer_idx ON party_logos(customer_id);
CREATE INDEX IF NOT EXISTS party_logos_supplier_idx ON party_logos(supplier_id);

-- People keep their nested collections (identities/addresses/emails/
-- telephones/avatars) as JSON text: they are third-level data and rarely
-- queried on their own.
CREATE TABLE IF NOT EXISTS party_people (
  id TEXT PRIMARY KEY,
  customer_id TEXT REFERENCES customers(id) ON DELETE CASCADE,
  supplier_id TEXT REFERENCES suppliers(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  uuid TEXT,
  label TEXT,
  key TEXT,
  role TEXT,
  name_uuid TEXT,
  name_alias TEXT,
  name_prefix TEXT,
  name_given TEXT,
  name_middle TEXT,
  name_surname TEXT,
  name_surname2 TEXT,
  name_suffix TEXT,
  name_meta TEXT,
  identities TEXT,
  addresses TEXT,
  emails TEXT,
  telephones TEXT,
  avatars TEXT,
  meta TEXT,
  CHECK ((customer_id IS NOT NULL) <> (supplier_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS party_people_customer_idx ON party_people(customer_id);
CREATE INDEX IF NOT EXISTS party_people_supplier_idx ON party_people(supplier_id);

-- Backfill the legacy single email/address columns into the child tables so
-- existing rows expose the same data through the new structure.
INSERT INTO party_emails (id, customer_id, position, addr)
SELECT id || '-email-0', id, 0, email FROM customers
WHERE email IS NOT NULL AND email <> '';

INSERT INTO party_emails (id, supplier_id, position, addr)
SELECT id || '-email-0', id, 0, email FROM suppliers
WHERE email IS NOT NULL AND email <> '';

INSERT INTO party_addresses (id, customer_id, position, street, locality, code, country)
SELECT id || '-addr-0', id, 0, address_line, locality, post_code, country FROM customers
WHERE (address_line IS NOT NULL AND address_line <> '')
   OR locality IS NOT NULL OR post_code IS NOT NULL OR country IS NOT NULL;

INSERT INTO party_addresses (id, supplier_id, position, street, locality, code, country)
SELECT id || '-addr-0', id, 0, address_line, locality, post_code, country FROM suppliers
WHERE (address_line IS NOT NULL AND address_line <> '')
   OR locality IS NOT NULL OR post_code IS NOT NULL OR country IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS party_people;
DROP TABLE IF EXISTS party_logos;
DROP TABLE IF EXISTS party_inboxes;
DROP TABLE IF EXISTS party_endpoints;
DROP TABLE IF EXISTS party_identities;
DROP TABLE IF EXISTS party_websites;
DROP TABLE IF EXISTS party_telephones;
DROP TABLE IF EXISTS party_emails;
DROP TABLE IF EXISTS party_addresses;

ALTER TABLE customers DROP COLUMN updated_at;
ALTER TABLE customers DROP COLUMN meta;
ALTER TABLE customers DROP COLUMN ext;
ALTER TABLE customers DROP COLUMN agent_id;
ALTER TABLE customers DROP COLUMN tax_type;
ALTER TABLE customers DROP COLUMN tax_scheme;
ALTER TABLE customers DROP COLUMN alias;
ALTER TABLE customers DROP COLUMN label;
ALTER TABLE customers DROP COLUMN regime;
ALTER TABLE customers DROP COLUMN uuid;

ALTER TABLE suppliers DROP COLUMN updated_at;
ALTER TABLE suppliers DROP COLUMN meta;
ALTER TABLE suppliers DROP COLUMN ext;
ALTER TABLE suppliers DROP COLUMN agent_id;
ALTER TABLE suppliers DROP COLUMN tax_type;
ALTER TABLE suppliers DROP COLUMN tax_scheme;
ALTER TABLE suppliers DROP COLUMN alias;
ALTER TABLE suppliers DROP COLUMN label;
ALTER TABLE suppliers DROP COLUMN regime;
ALTER TABLE suppliers DROP COLUMN uuid;
