# Pennypinch

A self-hosted pennypinch with three views: a spreadsheet-style table, a monthly category overview, and a yearly Sankey cash-flow visualization. Designed for homelab use — single binary, SQLite persistence, no authentication.

## Features

- **Table view** — search, filter (date range, category, amount), sort, and paginate your transactions with HTMX (no full page reloads).
- **Monthly overview** — a donut chart of top-level category spending plus a per-category breakdown with percentages, income/expense/net totals, and prev/next month navigation.
- **Yearly Sankey** — an interactive cash-flow diagram mapping income sources to top-level expense categories, with any remainder flowing into savings (or a deficit source when spending exceeds income).
- **Slash-separated categories** — unlimited nesting (`Food/Groceries/Supermarket`), case-insensitive, whitespace-normalized. Top-level aggregation means `Food/Groceries` and `Food/Dining Out` both roll up to `Food`.
- **CSV import** — the primary data entry method, with strict row-by-row validation and atomic config + database updates (rolls back on any error).
- **Auto-discovered categories** — new categories encountered on import are appended to the config with the next color from a 12-color ColorBrewer Set3 palette.
- **YAML config** — category metadata (name, description, color) plus settings (theme, currency symbol, date format), hot-reloaded on file change via fsnotify.
- **Light/dark themes** — respects OS preference and persists the manual override.
- **Signed amounts** — negative for expenses, positive for income; the type is derived from the sign.

## Tech Stack

| Layer      | Choice                                  |
| ---------- | --------------------------------------- |
| Backend    | Go 1.27, [chi](https://github.com/go-chi/chi) router, `html/template` |
| Storage    | SQLite (WAL mode), `modernc.org/sqlite` (pure-Go, no CGO) |
| Frontend   | [HTMX](https://htmx.org) + minimal vanilla JS |
| Charts     | [Chart.js](https://www.chartjs.org) + [chartjs-chart-sankey](https://github.com/kurkle/chartjs-chart-sankey) |
| Config     | YAML (`gopkg.in/yaml.v3`) with fsnotify hot-reload |
| Tests      | [Ginkgo](https://onsi.github.io/ginkgo/) + [Gomega](https://onsi.github.io/gomega/) |

## Requirements

- Go 1.27+ (the SQLite driver is pure Go, so no CGO or external SQLite installation is required)

## Getting Started

```sh
go run ./cmd/pennypinch
```

Then open <http://localhost:8080>. On first run the data directory is created automatically with an empty database and default config.

### Command-line flags

| Flag        | Default                     | Description                                    |
| ----------- | --------------------------- | ---------------------------------------------- |
| `-addr`     | `:8080`                     | HTTP listen address                            |
| `-data-dir` | `$EXPENSE_DATA_DIR` or `./data` | Where `expenses.db` and `config.yaml` live |

```sh
go run ./cmd/pennypinch -addr :9000 -data-dir /var/lib/pennypinch
```

To start fresh, delete (or point `-data-dir` away from) the data directory.

## Data Model

An expense is a single row with a signed amount and a slash-separated category path:

| Field       | Type     | Notes                                                        |
| ----------- | -------- | ------------------------------------------------------------ |
| `date`      | `YYYY-MM-DD` | Stored as UTC                                            |
| `amount`    | `number` | Negative = expense, positive = income, max 2 decimal places  |
| `subject`   | `string` | Where the money went to / came from                          |
| `description` | `string` | Optional notes                                             |
| `category`  | `string` | Canonical slash-separated path, e.g. `food/groceries`        |

Category paths are normalized on import: outer whitespace is trimmed, internal whitespace is collapsed, components are lowercased, and empty components are rejected. Original casing is preserved for display in the config where present, but matching is case-insensitive.

## CSV Import

Download a template from the UI ("Template" button) or `GET /api/import/template`. The fixed column order is:

```
date,amount,subject,description,category
2024-01-05,-42.50,Grocery Mart,Weekly groceries,Food/Groceries
```

Rules:

- `date` must be `YYYY-MM-DD`.
- `amount` is a signed number with at most 2 decimal places.
- `category` is a slash-separated path (header row optional).
- The whole file is rejected if any row fails validation, with a line-by-line error report.

A working example with 31 rows is available at [`samples/expenses.csv`](samples/expenses.csv).

## Configuration

The config file lives at `{data-dir}/config.yaml` and is hot-reloaded on change:

```yaml
categories:
  - name: food/groceries
    description: ""
    color: "#8dd3c7"
  - name: transportation/fuel
    description: Gasoline and diesel
    color: blue            # web color names also accepted
settings:
  theme: dark              # or light
  currency_symbol: "$"
  date_format: YYYY-MM-DD
```

Colors accept `#RGB`, `#RRGGBB`, or CSS color names (`blue`, `red`, …). New categories discovered during import are appended automatically with the next palette color and an empty description.

## REST API

### Expenses

| Method | Endpoint                       | Description                                    |
| ------ | ------------------------------ | ---------------------------------------------- |
| GET    | `/api/expenses`                | List with pagination & filtering               |
| POST   | `/api/expenses`                | Create an expense                              |
| GET    | `/api/expenses/export`         | Export all expenses as CSV                     |
| GET    | `/api/expenses/{id}`           | Get a single expense                           |
| PUT    | `/api/expenses/{id}`           | Update an expense                              |
| DELETE | `/api/expenses/{id}`           | Delete an expense                              |
| POST   | `/api/expenses/batch-delete`   | Delete many (JSON body `{"ids":[...]}`)        |

`GET /api/expenses` supports query params `search`, `category`, `date_from`, `date_to`, `amount_min`, `amount_max`, `sort`, `dir`, `page`, and `page_size`.

### Config

| Method | Endpoint                  | Description                              |
| ------ | ------------------------- | ---------------------------------------- |
| GET    | `/api/config/categories`  | List categories with metadata (read-only)|

### CSV Import

| Method | Endpoint               | Description                     |
| ------ | ---------------------- | ------------------------------- |
| POST   | `/api/import/csv`      | Upload and import a CSV file    |
| GET    | `/api/import/template` | Download a CSV template         |

### Analytics

| Method | Endpoint                                | Description                        |
| ------ | --------------------------------------- | ---------------------------------- |
| GET    | `/api/analytics/monthly/{year}/{month}` | Monthly top-level category overview |
| GET    | `/api/analytics/yearly/{year}`          | Yearly cash flow for the Sankey   |
| GET    | `/api/analytics/summary`                | Quick summary statistics          |

### Health

| Method | Endpoint   | Description      |
| ------ | ---------- | ---------------- |
| GET    | `/health`  | Liveness probe   |

## Project Layout

```
cmd/pennypinch/     entrypoint (flags, wiring, graceful shutdown)
internal/
  color/                 color parsing/validation, palette
  config/                YAML config manager, fsnotify watcher, atomic writes
  handler/               HTTP handlers (pages + REST API)
  model/                 Expense, analytics, and config types
  service/               category parsing, CSV parsing, aggregation, import
  storage/               storage interface + SQLite implementation
  web/                   embedded templates & static assets
samples/                 example CSV import
```

Templates and static assets (HTMX, Chart.js, the Sankey plugin, CSS, and JS) are embedded via `go:embed`, so the binary is fully self-contained.

## Testing

Tests use [Ginkgo](https://onsi.github.io/ginkgo/) + [Gomega](https://onsi.github.io/gomega/):

```sh
go test ./...
```

Coverage includes category/color parsing, CSV validation, config atomic writes and rollback, SQLite CRUD/filtering/aggregation, and service-level import, monthly overview, and Sankey generation.

## Building

```sh
go build -o pennypinch ./cmd/pennypinch
```

## Deployment

- Single binary with an embedded SQLite database and static assets.
- Mount a volume for `--data-dir` (or set `EXPENSE_DATA_DIR`) to persist data.
- Put it behind a reverse proxy (nginx/Caddy) for HTTPS — there is no built-in authentication, so expose it only on trusted networks.
- Back up by copying the data directory (database + config), or use `GET /api/expenses/export` for CSV portability.

## Notes & Limitations

- No authentication — intended for trusted/homelab networks only.
- No inline editing in the table view; data entry is via CSV import or the JSON REST API.
- Schema is created on startup with no migration system yet.

## Contributing

We are happy to have other people contributing to the project. If you decide to do that, here's how to:

- get Go (Pennypinch requires Go version 1.27 or greater)
- fork the project
- create a new branch
- make your changes
- open a PR.

Git commit messages should be meaningful and follow the rules nicely written down by [Chris Beams](https://chris.beams.io/posts/git-commit/):

> The seven rules of a great Git commit message
>
> 1. Separate subject from body with a blank line
> 1. Limit the subject line to 50 characters
> 1. Capitalize the subject line
> 1. Do not end the subject line with a period
> 1. Use the imperative mood in the subject line
> 1. Wrap the body at 72 characters
> 1. Use the body to explain what and why vs. how
