# goinvoice

An e-invoicing service built with [Gin](https://gin-gonic.com), [GOBL](https://github.com/invopop/gobl)
and [gobl.html](https://github.com/invopop/gobl.html).

You POST the business intent of an invoice (who, what, at what price, which taxes).
GOBL fills in the truth: computed totals, per-rate tax breakdown, regime-based validation
(including tax ID checksums), and a tamper-evident digest. `gobl.html` renders the stored
envelope into standalone printable HTML you can convert to PDF with any headless browser.

## Architecture

```
cmd/server/          entrypoint (Gin)
internal/handlers/   HTTP routes
internal/invoicer/   GOBL pipeline: build → calculate → validate → summarize → render
internal/store/      in-memory envelope store (swap with Postgres later)
examples/            sample request bodies
```

## Run

```bash
go run ./cmd/server            # listen on :8080
```

## API

### Create — `POST /invoices`

Body: a `bill.Invoice`-shaped JSON document, e.g.
[`examples/sample-invoice.json`](examples/sample-invoice.json).

| Query param   | Effect                                                        |
| ------------- | ------------------------------------------------------------- |
| `dry_run=true`| calculate + validate, **don't** store                         |
| `html=true`   | store, and include the HTML rendering URL in the response      |

Responses:

- `201` — `{ "id": <envelope uuid>, "envelope": {...} }` — the envelope is the **source of
  truth**: it contains the regime, calculated totals, tax breakdown and SHA-256 digest.
- `400` — malformed JSON.
- `422` — GOBL validation failures, as a readable, JSON-pathed message:

```json
{"error":"validation: [GOBL-ES-TAX-IDENTITY-01] ($.doc.supplier.tax_id.code, ...) invalid Spanish VAT identity code format or checksum"}
```

```bash
curl -s -X POST http://localhost:8080/invoices \
  -H 'Content-Type: application/json' \
  -d @examples/sample-invoice.json
```

### Read — `GET /invoices`, `GET /invoices/:id`, `GET /invoices/:id/envelope.json`

List returns compact summaries (id, code, currency, totals, digest); `/:id` returns the
full envelope; `/envelope.json` downloads it.

### Render — `GET /invoices/:id/html[?...]`

Returns `text/html` with stylesheets embedded — no network needed, safe to feed to
headless Chrome/Playwright for PDF.

| Query param    | Effect                                     |
| -------------- | ------------------------------------------ |
| `locale=es`    | translate labels (en, es, fr, pl, …)       |
| `watermark=`   | stamp text over the page (e.g. `DRAFT`)    |
| `sandbox=true` | simplified rendering for non-legal copies  |

```bash
curl -s "http://localhost:8080/invoices/$ID/html?locale=es&watermark=DRAFT" -o inv.html
```

### Delete — `DELETE /invoices/:id`

Removes the entry (no legal cancellation semantics — use GOBL credit notes for that).

## Client rules (things GOBL is strict about)

- **Amounts are JSON strings**, never numbers: `"price": "100.00"`, `"quantity": "10"`.
  Floats introduce rounding errors, which is exactly what GOBL exists to prevent.
- **Percentages need the `%` suffix**: `"percent": "10.0%"`. A bare `"10.0"` is treated as
  already pre-scaled (÷100) and means 1000 %. There will be no error — you'll just get
  gloriously wrong totals.
- **Tax IDs must pass checksum validation** for the regime (inferred from the supplier's
  tax ID country). Spanish CIF examples valid at time of writing: supplier `B98602642`,
  customer `A12345674` (control character is a *digit* for `A`/`B` prefixes).
- Use `dry_run=true` to validate before storing.

## Roadmap ideas

- Signing: `env.Sign(key)` requires an invoice `code` and produces JWS signatures.
- Postgres persistence (pgx + golang-migrate) behind the same `store` interface.
- `/invoices/:id/pdf` via chromedp.
- Country format conversion (CFDI, FatturaPA, VERI*FACTU, …) — in the ES case the
  VERI*FACTU addon is already validated against the invoice automatically.
