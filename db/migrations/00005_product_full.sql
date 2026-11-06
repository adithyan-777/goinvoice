-- +goose Up
-- Extend the product catalog to cover the full GOBL org.Item shape.
-- Scalars live on the products row; repeated collections live in FK child
-- tables with ON DELETE CASCADE.

ALTER TABLE products ADD COLUMN uuid TEXT;
ALTER TABLE products ADD COLUMN key TEXT;
ALTER TABLE products ADD COLUMN list TEXT;
ALTER TABLE products ADD COLUMN discount TEXT;
ALTER TABLE products ADD COLUMN per TEXT;
ALTER TABLE products ADD COLUMN origin TEXT;
ALTER TABLE products ADD COLUMN ext TEXT;
ALTER TABLE products ADD COLUMN meta TEXT;
ALTER TABLE products ADD COLUMN updated_at DATETIME;

CREATE TABLE IF NOT EXISTS product_identities (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  uuid TEXT,
  label TEXT,
  scope TEXT,
  country TEXT,
  key TEXT,
  type TEXT,
  code TEXT NOT NULL,
  description TEXT,
  ext TEXT
);
CREATE INDEX IF NOT EXISTS product_identities_product_idx ON product_identities(product_id);

CREATE TABLE IF NOT EXISTS product_attributes (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  label TEXT,
  key TEXT,
  type TEXT,
  text TEXT,
  code TEXT,
  amount TEXT,
  unit TEXT,
  date TEXT,
  ext TEXT
);
CREATE INDEX IF NOT EXISTS product_attributes_product_idx ON product_attributes(product_id);

CREATE TABLE IF NOT EXISTS product_images (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
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
  meta TEXT
);
CREATE INDEX IF NOT EXISTS product_images_product_idx ON product_images(product_id);

CREATE TABLE IF NOT EXISTS product_alt_prices (
  id TEXT PRIMARY KEY,
  product_id TEXT NOT NULL REFERENCES products(id) ON DELETE CASCADE,
  position INTEGER NOT NULL DEFAULT 0,
  label TEXT,
  currency TEXT NOT NULL,
  value TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS product_alt_prices_product_idx ON product_alt_prices(product_id);

-- +goose Down
DROP TABLE IF EXISTS product_alt_prices;
DROP TABLE IF EXISTS product_images;
DROP TABLE IF EXISTS product_attributes;
DROP TABLE IF EXISTS product_identities;

ALTER TABLE products DROP COLUMN updated_at;
ALTER TABLE products DROP COLUMN meta;
ALTER TABLE products DROP COLUMN ext;
ALTER TABLE products DROP COLUMN origin;
ALTER TABLE products DROP COLUMN per;
ALTER TABLE products DROP COLUMN discount;
ALTER TABLE products DROP COLUMN list;
ALTER TABLE products DROP COLUMN key;
ALTER TABLE products DROP COLUMN uuid;
