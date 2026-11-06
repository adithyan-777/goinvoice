# Ittisal Invoice

An e-invoicing service built with [Chi](https://go-chi.io), [templ](https://templ.guide),
[GOBL](https://github.com/invopop/gobl) and [gobl.html](https://github.com/invopop/gobl.html).

You POST the business intent of an invoice (who, what, at what price, which taxes).
GOBL fills in the truth: computed totals, per-rate tax breakdown, regime-based validation
(including tax ID checksums), and a tamper-evident digest. `gobl.html` renders the stored
envelope into standalone printable HTML you can convert to PDF with any headless browser.

## Architecture

```
cmd/server/          entrypoint (Chi)
internal/handlers/   HTTP routes + templ pages (index.templ → index_templ.go)
internal/invoicer/   GOBL pipeline: build → calculate → validate → summarize → render
internal/store/      in-memory envelope store (swap with Postgres later)
examples/            sample request bodies
```

`GET /` is rendered with templ (`IndexPage`) as a dashboard: one card per
bill document kind (invoices, orders, deliveries, payments, statuses) with
stored counts, plus the invoice table.
`GET /invoices/{id}/html`
delegates to `gobl.html` (itself templ-based) for the printable invoice document.
`GET /customers`, `/suppliers` and `/products` render the seeded catalog
tables (`catalog.templ`); demo rows come from `db/migrations/00003_*` and
demo invoices are built via GOBL on first boot (`internal/seed`).
`GET /orders`, `/deliveries`, `/payments`, `/statuses` render the matching
document list (`documents.templ`) with view/JSON-download links and inline
delete per row; the same URLs serve the JSON API to non-browser clients
(content negotiation on `Accept: text/html`).

Design: the app shell (sidebar, topbar, cards, tables, modal drawer,
dark mode, theme engine) follows `../qatar/pattika` structure and
conventions (`internal/handlers/static/pattika/`, see its `AGENTS.md`);
goinvoice wears its own identity on top — deep-teal on warm paper/pine
ink (`static/goinvoice.css` token overrides plus KPI stats, series
badges and tabular money figures).
`POST /products` creates a product (JSON API or the form on `/products`;
duplicate `ref` → `409`). `POST /products/{id}` updates,
`DELETE /products/{id}` deletes (`204`); the list page links an edit form
per row with inline delete. Customers expose CRUD under `/customers`
(unique `tax_code` → `409`). The issuing company is a single record edited at
`/company` (stored in the `suppliers` table); `/suppliers` redirects there, and
the supplier CRUD API routes remain available for integrations.

### Catalog parties

Customers and suppliers store the full GOBL `org.Party` shape. Scalars
(`uuid`, `$regime`, `label`, `name`, `alias`, `tax_id.{country,code,scheme,type}`,
`agent_id`, `ext`, `meta`) live on the `customers`/`suppliers` rows. Repeated
collections live in child tables with foreign keys and `ON DELETE CASCADE`:

| Table               | GOBL field                   |
| ------------------- | ---------------------------- |
| `party_addresses`   | `addresses[]` (+ `coords`)   |
| `party_emails`      | `emails[]`                   |
| `party_telephones`  | `telephones[]`               |
| `party_websites`    | `websites[]`                 |
| `party_identities`  | `identities[]`               |
| `party_endpoints`   | `endpoints[]`                |
| `party_inboxes`     | `inboxes[]`                  |
| `party_logos`       | `logos[]` (`org.Image`)      |
| `party_people`      | `people[]` (`org.Person`)    |

Each child row has nullable `customer_id`/`supplier_id` columns with a `CHECK`
so it belongs to exactly one party. `people` keeps its own nested
identities/addresses/emails/telephones/avatars as JSON text (third-level data,
rarely queried directly).

`POST /customers` and `POST /suppliers` accept either a full `org.Party` JSON
document (nested collections included) or the flat HTML form (name, tax id,
primary email/address, phone, website). When a document form selects a party
from the catalog, the **whole party** is loaded into the invoice/order/…, so
addresses, emails, identities, endpoints and logos flow through to GOBL and
`gobl.html`.

### Catalog products

Products store the full GOBL `org.Item` shape. Scalars (`uuid`, `ref`, `key`,
`name`, `description`, `currency`, `price`, `list`, `discount`, `per`, `unit`,
`origin`, `ext`, `meta`) live on the `products` row. Repeated collections live
in FK child tables with `ON DELETE CASCADE`:

| Table                  | GOBL field       |
| ---------------------- | ---------------- |
| `product_identities`   | `identities[]`   |
| `product_attributes`   | `attributes[]`   |
| `product_images`       | `images[]`       |
| `product_alt_prices`   | `alt_prices[]`   |

`POST /products` accepts either a full `org.Item` JSON document or the flat
HTML form. Selecting a product in a document line embeds the whole item
(ref, description, unit, origin, identities, attributes, images, alt prices,
extensions) while the line name/price remain editable.

Customers and products use the same modal workflow as the bill documents:
**New** opens a modal form and the row edit button opens the same form
pre-filled (non-JS fallbacks: `/customers/new`, `/products/new`). The single
issuing company is edited directly on `/company`, and document forms prefill it
as **Company** with no supplier picker clutter.

Pages are SPA-like via [Datastar](https://data-star.dev) v1
(`internal/handlers/static/datastar.js`, vendored, no CDN): nav swaps
`#main` in place, edits open in a modal, and creates/updates/deletes
re-render the section without reloads. The same endpoints serve plain
browsers (full pages + PRG redirects) and JSON API clients — Datastar
requests are detected by their `Accept: text/event-stream` header and get
HTML fragments with `Datastar-Selector`/`Datastar-Mode` patch headers.

## Run

```bash
go run ./cmd/server            # listen on :8080
```

## Dev

templ files must be regenerated after editing (generated `*_templ.go` is committed):

```bash
go install github.com/a-h/templ/cmd/templ@v0.3.1001  # must match gobl.html's templ
templ generate
# or: go generate ./...
```

Pinned to templ v0.3.1001 — that's what `gobl.html` v0.135.0 was generated with;
newer templ runtimes break its generated code (`templ.JoinURLErrs` undefined).

## API

All five `bill` documents share the same shape — replace `/invoices` with
`/orders`, `/deliveries`, `/payments` or `/statuses`:

| Document   | Types                              | Example                                              |
| ---------- | ---------------------------------- | ---------------------------------------------------- |
| `invoice`  | `standard\|proforma\|corrective\|credit-note\|debit-note\|other` | [`sample-invoice.json`](examples/sample-invoice.json) |
| `order`    | `purchase\|sale\|quote`             | [`sample-order.json`](examples/sample-order.json)     |
| `delivery` | `advice\|note\|waybill\|receipt\|other` | [`sample-delivery.json`](examples/sample-delivery.json) |
| `payment`  | `request\|receipt\|advice`          | [`sample-payment.json`](examples/sample-payment.json) |
| `status`   | `response\|update\|system`          | [`sample-status.json`](examples/sample-status.json)   |

HTML pages live at `/orders`, `/deliveries`, `/payments`, `/statuses`
(Datastar partials under `/partials/orders`, …). Summaries carry a
`document` field (`invoice|order|…`) plus `type`, totals and digest;
each collection only lists its own kind (cross-kind `GET` → `404`).

### Create — `POST /invoices`

Body: a `bill.Invoice`-shaped JSON document, e.g.
[`examples/sample-invoice.json`](examples/sample-invoice.json).
Same for `POST /orders` (`bill.Order`), `/deliveries` (`bill.Delivery`),
`/payments` (`bill.Payment`), `/statuses` (`bill.Status`).

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

### Read — `GET /invoices`, `GET /invoices/{id}`, `GET /invoices/{id}/envelope.json`

List returns compact summaries (document, id, code, currency, totals, digest);
`/{id}` returns the full envelope; `/envelope.json` downloads it.
Same under `/orders`, `/deliveries`, `/payments`, `/statuses`.

### Render — `GET /invoices/{id}/html[?...]` (also `/orders/{id}/html`, …)

Returns `text/html` with stylesheets embedded plus a floating
**Download PDF** button. Clicking it converts the invoice to PDF entirely
in the browser via [html2pdf.js](https://cdnjs.com/libraries/html2pdf.js)
(html2pdf.bundle from CDN) — no server-side converter needed.

| Query param    | Effect                                     |
| -------------- | ------------------------------------------ |
| `locale=es`    | translate labels (en, es, fr, pl, …)       |
| `watermark=`   | stamp text over the page (e.g. `DRAFT`)    |
| `sandbox=true` | simplified rendering for non-legal copies  |

```bash
curl -s "http://localhost:8080/invoices/$ID/html?locale=es&watermark=DRAFT" -o inv.html
```

### Delete — `DELETE /invoices/{id}` (also `/orders/{id}`, …)

Removes the entry (no legal cancellation semantics — use GOBL credit notes for that).

### Update — `PUT /invoices/{id}` (also `POST`, also `/orders/{id}`, …)

Body: the full replacement document. It is rebuilt through calculate +
validate and stored under the **same id** (links stay stable).

- `200` — `{ "id": <same uuid>, "envelope": {...} }`.
- `400` — malformed JSON; `404` — unknown id or wrong kind; `422` — GOBL validation failures.

```bash
curl -s -X PUT http://localhost:8080/orders/$ID \
  -H 'Content-Type: application/json' \
  -d @examples/sample-order.json
```

### UI — modal forms for every kind

Every kind — invoices, orders, deliveries, payments, statuses — has a
**New** button and an **Edit** button per row. Both open a centered modal
with a structured form (type dropdown, series/code, date, currency,
supplier/customer with tax IDs, and up to three line rows; the customer
field is a searchable dropdown fed by the customers catalog, and selecting
one fills its tax ID (also resolved server-side on save); invoices support
all invoice types including credit/debit notes; payments also get method +
amount, statuses get status keys). Submitting posts form fields to the same
`POST` endpoints — JSON API clients are unaffected.

Behavior:

- Success closes the modal and refreshes the list in place.
- Validation failures keep the modal open and re-render the form with the
  error and the user's input intact.
- Without JavaScript the same forms are served as full pages at
  `/{route}/new` and `/{route}/{id}/edit` and follow PRG redirects.
- Datastar fragments are streamed as SSE `datastar-patch-elements` /
  `datastar-patch-signals` events (what the vendored Datastar v1.0.4 client
  understands), under `/{route}` and `/partials/*`.

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
- Country format conversion (CFDI, FatturaPA, VERI*FACTU, …) — in the ES case the
  VERI*FACTU addon is already validated against the invoice automatically.
