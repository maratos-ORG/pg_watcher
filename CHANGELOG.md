# Changelog

All notable changes to pg_watcher will be documented in this file.

## [1.0.1] - 2026-01-19

### Added
- **`-exclude-db` flag**: Ability to exclude specific databases when using `-db-name=all`
  - Accepts comma-separated list of database names
  - Example: `-exclude-db="testdb,olddb"`
  - Combines with hardcoded exclusions (template0, template1, postgres, cloudsqladmin)
  - Only takes effect when `-db-name=all` is specified

### Changed
- **`-prefixMetric` flag**: Removed default value (`pgwatch`)
  - If `-prefixMetric` is not explicitly set, metrics will be named without any prefix
  - Metric names will use only the column name (e.g., `numbackends` instead of `pgwatch_numbackends`)
  - Prefix is only added when `-prefixMetric` is manually specified via command line parameter
- **Label value formatting**: Improved label value formatting in Prometheus output
  - Labels now properly handle values containing quotes and special characters (e.g., from `quote_ident()` PostgreSQL function)

## [1.0.0] - 2025-01-12

### Added
- Initial release of pg_watcher
- Execute SQL queries against PostgreSQL databases
- Output metrics in Prometheus-compatible format
- Support for multiple databases via `-db-name=all` or comma-separated list
- Parallel database processing with `-j` flag
- Configurable timeouts with `-pg-timeout`
- Role-based execution with `-master-only` and `-replica-only` flags
- Multiple SQL statements support via `-SQLSpliter`
- Flexible label and metric handling
- Column filtering with `-ignoredColumns`
- Custom metric prefix with `-prefixMetric`
