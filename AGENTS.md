# AGENTS.md

Instructions for AI coding agents working on this project.

## Project

Pennypinch is a self-hosted expense tracker. Single Go binary, SQLite persistence, HTMX frontend with Chart.js, no authentication. Designed for homelab use.

## Build & Run

```sh
go run ./cmd/pennypinch
go build -o pennypinch ./cmd/pennypinch
go test ./...
```

Live reload during development via [Air](https://github.com/air-verse/air) (configured in `.air.toml`):

```sh
go tool air
```

## Project Layout

```
cmd/pennypinch/        entrypoint (flags, wiring, graceful shutdown)
internal/
  color/               color parsing/validation, palette
  config/              YAML config manager, fsnotify watcher, atomic writes
  handler/             HTTP handlers (pages + REST API)
  model/               Expense, analytics, and config types
  service/             category parsing, CSV parsing, aggregation, import
  storage/             storage interface + SQLite implementation
  web/                 embedded templates & static assets (go:embed)
samples/               example CSV import
```

Templates and static assets are embedded via `go:embed`. The binary is self-contained.

## Code Style

- **Comments**: Minimal. Brief comments on exported types and functions only. No inline comments.
- **Formatting**: `gofmt` standard formatting.
- **Linting**: `golangci-lint` (run before committing).
- **Error handling**: Return errors up the stack. Log only in `main` or when irrecoverable.

## Testing

- **Framework**: Ginkgo with Gomega matchers only.
- **Command**: `go test ./...`
- **Patterns**: Use `Describe`/`It` blocks. In-memory SQLite for storage tests.

## Git Workflow

- **Branches**: Feature branches off `main`. Open a PR to merge.
- **Commits**: Follow the [Chris Beams rules](https://chris.beams.io/posts/git-commit/):
  1. Separate subject from body with a blank line
  2. Limit subject line to 50 characters
  3. Capitalize the subject line
  4. Do not end subject line with a period
  5. Use imperative mood in the subject line
  6. Wrap body at 72 characters
  7. Use body to explain what and why vs. how

## No CI

All build, test, and lint checks are run locally. No CI pipeline is configured.