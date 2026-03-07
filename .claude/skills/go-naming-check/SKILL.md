---
name: go-naming-check
description: Check Go naming conventions for the pg_watcher project. Use when adding new identifiers, reviewing code, or renaming things. Based on Effective Go + golangci-lint (revive var-naming rule) used in this project.
---

# Go Naming Convention Checker

Verify names follow Go conventions as used in this project (`golangci-lint` with `revive` + `var-naming` rule).

## Quick Reference

| Entity | Convention | Examples from this project |
|--------|------------|---------------------------|
| Package | lowercase, single word, no underscores | `watcher`, `main` |
| Exported type/struct | PascalCase | `FlagParam`, `ConnectionString` |
| Unexported type/struct | camelCase | `colMeta`, `mval` |
| Exported function | PascalCase | `Run`, `ParseFlags` |
| Unexported function | camelCase | `resolveDBList`, `connectDB`, `processDB` |
| Exported struct field | PascalCase | — (avoid, prefer unexported) |
| Unexported struct field | camelCase | `sqlQuery`, `masterOnly`, `pgTimeout` |
| Local variable | camelCase | `dbList`, `flagParam`, `connParam` |
| Error variable (local) | `err` | `err`, `err2` |
| Error variable (package) | `ErrXxx` | `ErrNotFound` |
| Constant | PascalCase (exported) / camelCase (unexported) | `kMaxSize` style NOT used in Go |
| Interface | PascalCase, often `-er` suffix | `Reader`, `Closer` |
| Receiver | 1-2 char abbreviation, consistent per type | `c *pgx.Conn` → use `c` |

## Acronym Rules (important!)

Go treats acronyms as all-caps or all-lowercase — never mixed:

```go
// Exported → all caps acronym
ParseSQL()       // ✅
ResolveDBList()  // ✅
ConnectDB()      // ✅

// Unexported → all lowercase acronym
parseSQL()       // ✅
resolveDBList()  // ✅ (used in this project)
connectDB()      // ✅ (used in this project)

// Wrong
ParseSql()       // ❌ SQL acronym must be all-caps
CheckDbRole()    // ⚠️  inconsistent — prefer checkDBRole
```

Common acronyms in this project: `DB`, `SQL`, `ID`, `URL`, `PG`

## Common Mistakes → Corrections

```go
// Types
type flag_param struct     → type FlagParam struct       // no underscores
type connectionstring      → type ConnectionString struct // PascalCase

// Functions
func Process_db()          → func processDB()            // no underscores
func getDBList()           → func resolveDBList()        // no Get- prefix for non-getters
func SQLsplitter()         → func sqlSplitter() / SQLSplitter() // consistent acronym case

// Variables
var FlagParam FlagParam    → var flagParam FlagParam     // unexported var = camelCase
var pg_timeout             → var pgTimeout               // no underscores

// Struct fields
SQLSpliter string          → sqlSplitter string          // unexported field + fix typo
m_value int                → value int                   // no Hungarian notation

// Errors
var err_not_found = ...    → var ErrNotFound = ...       // package-level errors: ErrXxx

// Interfaces
type DataGetter interface   → type DataGetter interface   // ✅ -er suffix
type IDataGetter interface  → type DataGetter interface   // ❌ no I- prefix
```

## Project Structure Conventions

```
cmd/pg_watcher/main.go          ← thin entrypoint, no business logic
internal/watcher/watcher.go     ← all logic lives here
```

- **`cmd/`**: only flag parsing + `os.Exit`. No logic.
- **`internal/`**: all business logic. Not importable outside the module.
- **Package name must not stutter**: `watcher.Watcher` ❌ → `watcher.Runner` ✅

## Linter Rules Active (from `.golangci.yml`)

- `revive` / `var-naming` — enforces Go naming conventions, will flag violations
- `gofmt` — enforces formatting (run `gofmt -w .` or `make lint`)
- `gocritic` with `style` tag — flags style issues including naming

Run before committing:
```bash
make lint
```

## Check Process

1. Identify the entity type (package, type, function, variable, field)
2. Is it exported (uppercase first letter) or unexported?
3. Apply the PascalCase / camelCase rule accordingly
4. Check acronyms: all-caps when exported, all-lowercase when unexported
5. No underscores (except `_` blank identifier), no Hungarian notation
6. Run `make lint` to confirm
