-- +goose Up
INSERT OR IGNORE INTO suppliers (id, name, tax_country, tax_code, email, address_line, locality, post_code, country) VALUES
  ('sup-acme', 'Acme Consultancy SL', 'ES', 'B98602642', 'billing@acme.example', 'Calle Mayor 1', 'Madrid', '28001', 'ES'),
  ('sup-nordica', 'Nordica Design Oy', 'FI', 'FI12345678', 'hello@nordica.example', 'Mannerheimintie 10', 'Helsinki', '00100', 'FI');

INSERT OR IGNORE INTO customers (id, name, tax_country, tax_code, email, address_line, locality, post_code, country) VALUES
  ('cus-globex', 'Globex Ibérica SA', 'ES', 'A12345674', 'ap@globex.example', 'Avenida de la Innovación 42', 'Madrid', '28020', 'ES'),
  ('cus-initech', 'Initech SL', NULL, NULL, 'accounts@initech.example', 'Calle de Alcalá 100', 'Madrid', '28009', 'ES'),
  ('cus-hooli', 'Hooli Systems Inc', 'US', NULL, 'billing@hooli.example', '1 Market Street', 'San Francisco', '94105', 'US');

INSERT OR IGNORE INTO products (id, ref, name, description, price, currency, unit) VALUES
  ('prod-consult', 'CONS-001', 'Backend consulting', 'Backend consulting per day', '100.00', 'EUR', 'day'),
  ('prod-license', 'LIC-002', 'Invoice generator license', 'Annual license per seat', '250.00', 'EUR', 'seat'),
  ('prod-support', 'SUP-003', 'Support hour', 'Remote support per hour', '80.00', 'EUR', 'hour'),
  ('prod-hosting', 'HOS-004', 'Hosting', 'Managed hosting per month', '49.00', 'EUR', 'month');

-- +goose Down
DELETE FROM products WHERE id IN ('prod-consult', 'prod-license', 'prod-support', 'prod-hosting');
DELETE FROM customers WHERE id IN ('cus-globex', 'cus-initech', 'cus-hooli');
DELETE FROM suppliers WHERE id IN ('sup-acme', 'sup-nordica');
