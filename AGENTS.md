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
- **Function body layout**: Separate logical blocks with blank lines. Each guard clause, variable declaration/initialization, loop, error-checked call, and return statement should be its own visually distinct block. Do not pack unrelated statements together without a blank line between them.
- **Line wrapping**: Keep function calls on a single line when they fit within a reasonable width. Avoid splitting argument lists across multiple lines when the call can be expressed on one line. Multi-line wrapping is acceptable for long SQL strings or struct literals that genuinely exceed readable line length.
- **Nesting**: Prefer early returns (guard clauses) over deep nesting. Validate preconditions and bail out at the top so the main logic flows linearly at one indentation level. Nesting is acceptable when it genuinely simplifies the code, but as a rule keep it shallow.

### Function body style example

```go
// Preferred — blank lines between logical blocks, single-line call
func (s *SQLite) DeleteExpenses(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")

	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	res, err := s.db.ExecContext(ctx, `DELETE FROM expenses WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

// Avoid — statements packed together, call wrapped unnecessarily
func (s *SQLite) DeleteExpenses(ctx context.Context, ids []int64) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM expenses WHERE id IN (`+placeholders+`)`, args...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
```

## Testing

- **Framework**: Ginkgo with Gomega matchers only.
- **Command**: `go test ./...`
- **Patterns**: Use `Describe`/`It` blocks. In-memory SQLite for storage tests.

### Expected Ginkgo style

Use [Arrange-Act-Assert](https://automationpanda.com/2020/07/07/arrange-act-assert-a-pattern-for-writing-good-tests/):

```go
var _ = Describe("High-level description", func() {
	var veryGlobalVar any

	BeforeEach(func() {
		veryGlobalVar = setupSomeGlobalClient()
	})

	Context("description of the setup/context of this section", func() {
		var result any
		var err error

		BeforeEach(func() { // Arrange
			// Code that actually sets up exactly what the context
			// description describes. Usually a series of code blocks
			// that set it up and initialize variables for this section,
			// for example creating a test input or test files.
		})

		JustBeforeEach(func() { // Act
			// Ideally just one function/method call that exectutes
			// exactly what we want to have test assertions for.
			result, err = veryGlobalVar.Foobar()
		})

		It("does not error", func() { // Assert
			Expect(err).ToNot(HaveOccurred())
		})

		It("should describe the test assertion", func() { // Assert
			Expect(result).To(Equal("exactly what we expect"))
		})

		Context("optional sub-context", func() {
			BeforeEach(func() {
				// setting up the optional sub-context on top of the outher context
			})

			It("describe test assertion", func() {
				Expect(something).To(Equal(expectedSomething))
			})
		})
	})

	Context("next context following the same style and logic", func() {
		// ...
	})

	When("some other description", func ()  {
		// When is used when the description as a sentence sounds good
		// with the word when in it, e.g. When("there is no user input")
	})
})

// setupSomeGlobalClient is a helper function and is placed at the end of the test file
func setupSomeGlobalClient() any {
	return nil
}
```

Note: Ideally, there is *only* one `Expect` per `It`.

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