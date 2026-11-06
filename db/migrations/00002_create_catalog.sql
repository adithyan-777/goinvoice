-- +goose Up
CREATE TABLE IF NOT EXISTS customers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  tax_country TEXT,
  tax_code TEXT UNIQUE,
  email TEXT,
  address_line TEXT,
  locality TEXT,
  post_code TEXT,
  country TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS suppliers (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  tax_country TEXT,
  tax_code TEXT UNIQUE,
  email TEXT,
  address_line TEXT,
  locality TEXT,
  post_code TEXT,
  country TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS products (
  id TEXT PRIMARY KEY,
  ref TEXT NOT NULL UNIQUE,
  name TEXT NOT NULL,
  description TEXT,
  price TEXT NOT NULL,
  currency TEXT NOT NULL DEFAULT 'EUR',
  unit TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS suppliers;
DROP TABLE IF EXISTS customers;
