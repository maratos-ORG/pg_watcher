# Coding and Documentation Standards for pg_watcher

## For AI Assistants (Claude) and Human Contributors

When working on this codebase, **strictly adhere** to the following coding and documentation standards.

---

## 1. Language and Style

- **Language:** Go 1.23.1+
- **Code Style:** Follow `golangci-lint` configuration ([.golangci.yml](../.golangci.yml))
- **Formatting:** Always run `gofmt` (enforced via `make lint`)
- **Build before commit:** Run `make all` (lint + test + build) before committing

---

## 2. Naming Conventions

| Element | Convention | Examples |
|---------|-----------|----------|
| Exported types | PascalCase | `FlagParam`, `ConnectionString` |
| Private types | camelCase | `flagParam`, `connParam` |
| Exported functions | PascalCase | `Run()`, `ParseFlags()` |
| Private functions | camelCase | `resolveDBList()`, `connectDB()`, `processDB()` |
| Variables | Short, clear | `dbname`, `ctx`, `err`, `conn`, `rows` |
| Constants | PascalCase or UPPER_SNAKE | `SQLSpliter`, `DEFAULT_TIMEOUT` |

---

## 3. Code Organization

### Project Structure
```
pg_watcher/
├── cmd/pg_watcher/        # Entry point (main.go)
├── internal/watcher/      # Core business logic
├── .claude/               # AI assistant instructions
├── Makefile               # Build automation
├── .golangci.yml          # Linter rules
└── README.md              # User documentation
```

### Function Design
- Small, focused functions with single responsibility
- Early return pattern for errors
- Descriptive names that explain purpose
- Keep functions under 50 lines when possible

### File Organization
- Package-level constants and globals at top
- Type definitions before functions
- Public API before private helpers
- Related functions grouped together

---

## 4. Error Handling

### Required Patterns

```go
// Early return with context
if err != nil {
    return fmt.Errorf("descriptive context: %w", err)
}

// Error to stderr for user-facing messages
if err := someOperation(); err != nil {
    fmt.Fprintln(os.Stderr, err)
    os.Exit(1)
}
```

### Never
- Ignore errors silently
- Use `panic()` outside recovery handlers
- Return generic error messages without context

---

## 5. Context Management

### Required
- Use `context.Context` for all database operations
- Create timeout contexts with `context.WithTimeout()`
- Defer `cancel()` immediately after creating context
- Pass parent context through function chains

### Example
```go
ctx, cancel := context.WithTimeout(ctxParent, timeout)
defer cancel()
```

---

## 6. Concurrency Patterns

### When using goroutines
- Use `golang.org/x/sync/semaphore` for bounded parallelism
- Implement panic recovery in goroutine wrappers
- Ensure proper resource cleanup with defer
- Use channels or sync primitives for coordination

### Example
```go
go func(dbname string) {
    defer func() {
        if r := recover(); r != nil {
            fmt.Fprintf(os.Stderr, "panic in %s: %v\n", dbname, r)
        }
        sem.Release(1)
    }()
    processDB(ctx, dbname)
}(db)
```

---

## 7. Documentation Standards

### Code Comments

**Package comments:** At top of primary file (currently optional per linter config)

**Function comments:** For all exported functions, describe purpose and key behaviors

**Inline comments:** Explain "why" not "what", use sparingly

**TODO comments:** Include reasoning and context

### Comment Style
```go
// connectDB establishes a connection with timeout applied only to the dial phase.
// The returned connection must be closed by the caller.
func connectDB(ctx context.Context, dbname string) (*pgx.Conn, error) {
    // Timeout context prevents hanging on unreachable hosts
    ctx, cancel := context.WithTimeout(ctx, flagParam.pgTimeout)
    defer cancel()
    ...
}
```

### README Documentation
- Keep [README.md](../README.md) synchronized with CLI flags
- Use tables for structured information (arguments, options)
- Include real-world examples for Telegraf and CLI usage
- Add output examples showing Prometheus format
- Document execution model and concurrency behavior

---

## 8. Testing Requirements

### When adding tests
- Place tests in `*_test.go` files
- Use table-driven tests for multiple cases
- Test error paths, not just happy paths
- Use `-race` flag (`make test` includes this)
- Aim for meaningful coverage, not 100% coverage

### Example structure
```go
func TestToFloat64(t *testing.T) {
    tests := []struct {
        name    string
        input   interface{}
        want    float64
        wantErr bool
    }{
        {"int64", int64(42), 42.0, false},
        {"string", "not a number", 0, true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := toFloat64(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("toFloat64() error = %v, wantErr %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("toFloat64() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

---

## 9. Dependency Management

- Minimal dependencies (currently: `pgx/v5`, `golang.org/x/sync`)
- Use `go.mod` and `go.sum` (run `make tidy` after changes)
- Prefer standard library when possible
- Vet new dependencies for security and maintenance

---

## 10. Build and Release

### Version Management
- Versions injected via `ldflags` during build
- Use git tags for releases: `git tag v1.2.3`
- Version string format: `v<major>.<minor>.<patch>` or `dev`

### Build Commands
```bash
make all        # Lint + test + build (run before commit)
make build      # Compile binary only
make lint       # Run golangci-lint
make test       # Run tests with race detection
make install    # Install to $GOBIN
```

---

## 11. Git Workflow

### Before Committing
1. Run `make all` to ensure code passes linting and builds
2. Update README.md if CLI flags changed
3. Add tests for new functionality
4. Commit with clear, descriptive messages

### Commit Message Style
```
Short summary (50 chars or less)

More detailed explanation if needed. Wrap at 72 characters.
Explain the problem this solves and why this approach was chosen.

- Bullet points are fine
- Use present tense: "Add feature" not "Added feature"
```

---

## 12. Code Review Checklist

Before submitting changes, verify:
- [ ] `make all` passes without errors
- [ ] Code follows naming conventions
- [ ] Error handling includes context
- [ ] Context and timeouts used correctly
- [ ] Resources properly cleaned up (defer)
- [ ] Comments explain non-obvious logic
- [ ] README updated if behavior changed
- [ ] No unnecessary dependencies added
- [ ] No hardcoded credentials or secrets

---

## 13. Prometheus Output Format

### Metric Naming
- Format: `<prefix>_<column_name>{labels} <value>`
- Default prefix: `pgwatch` (configurable via `-prefixMetric`)
- Column names normalized: lowercase, spaces→underscores

### Label Rules
- String columns automatically become labels
- Numeric columns automatically become metrics
- Use `-labels` to force specific columns as labels
- Always include `db="<database>"` label

### Example
```text
pgwatch_active_sessions{db="mydb",user="replica",state="active"} 5
pgwatch_db_size_bytes{db="mydb"} 2.409e+08
```

---

## 14. Performance Considerations

- Use `strings.Builder` for string concatenation in loops
- Pre-compute metadata when processing rows
- Avoid unnecessary allocations in hot paths
- Use semaphores to limit concurrent connections
- Close connections and cancel contexts properly

---

## 15. Security Best Practices

- Never log connection strings or credentials
- Use parameterized queries when building dynamic SQL
- Validate user inputs (file paths, connection strings)
- Set timeouts on all database operations
- Handle panics to prevent information leakage

---

## Quick Start for Contributors

```bash
# 1. Clone and setup
git clone <repository-url>
cd pg_watcher
make tidy

# 2. Make changes following guidelines above

# 3. Verify before commit
make all

# 4. Commit
git add .
git commit -m "Brief description of changes"

# 5. Create release (maintainers only)
git tag v1.2.3
make build
```

---

## For Claude Code Assistant

When assisting with this project:

1. **Always read existing code** before proposing changes
2. **Follow the patterns** already established in [internal/watcher/watcher.go](../internal/watcher/watcher.go)
3. **Run `make lint`** after code changes
4. **Update README.md** if adding/changing CLI flags
5. **Preserve the coding style** documented above
6. **Use early returns** and clear error messages
7. **Keep functions focused** and under 50 lines
8. **Test changes** with `make test` (when tests exist)
9. **Explain your reasoning** in code comments when logic is non-obvious
10. **Never sacrifice readability** for brevity
11. **Match existing patterns** in the codebase (see examples in existing code)
12. **Use semaphore pattern** for concurrency (as seen in `Run()` function)
13. **Apply timeout contexts** for all database operations
14. **Format metric output** according to Prometheus specification
15. **Keep global state minimal** (only when necessary for backward compatibility)

---

## Key Code Patterns to Follow

### Pattern 1: Resource Cleanup
```go
ctx, cancel := context.WithTimeout(parent, timeout)
defer cancel()
defer conn.Close(ctx)
```

### Pattern 2: Error Wrapping
```go
if err != nil {
    return fmt.Errorf("operation failed for %s: %w", identifier, err)
}
```

### Pattern 3: Semaphore-based Concurrency
```go
sem := semaphore.NewWeighted(int64(maxJobs))
for _, item := range items {
    sem.Acquire(ctx, 1)
    go func(i string) {
        defer sem.Release(1)
        process(i)
    }(item)
}
sem.Acquire(ctx, int64(maxJobs)) // Wait for all
```

### Pattern 4: Type-safe Conversions
```go
func toFloat64(val interface{}) (float64, error) {
    switch v := val.(type) {
    case float64:
        return v, nil
    case int64:
        return float64(v), nil
    // ... handle all expected types
    default:
        return 0, fmt.Errorf("cannot convert %T to float64", val)
    }
}
```

---

## Project-Specific Context

### Architecture
- **Purpose:** PostgreSQL metrics exporter for Telegraf/Prometheus
- **Execution Model:** Per-database parallelism with per-query timeouts
- **Output Format:** Prometheus line protocol

### Key Files
- [cmd/pg_watcher/main.go](../cmd/pg_watcher/main.go) - CLI entry point (minimal)
- [internal/watcher/watcher.go](../internal/watcher/watcher.go) - Core logic (492 lines)
- [.golangci.yml](../.golangci.yml) - Linter configuration (13 linters)
- [Makefile](../Makefile) - Build automation

### Critical Functions
- `Run()` - Main orchestration with semaphore-based parallelism
- `ParseFlags()` - CLI argument processing
- `processDB()` - Per-database query execution
- `toFloat64()` - Type conversion for metrics
- `connectDB()` - Connection establishment with timeout

### Design Decisions
- Global state (`flagParam`, `connParam`) maintained for backward compatibility
- Separate connections per database for isolation
- Per-query timeout contexts for reliability
- String columns auto-convert to Prometheus labels
- Numeric columns auto-convert to metrics
