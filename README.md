# pg_watcher
### Plugin for Telegraf

**pg_watcher** executes SQL queries against PostgreSQL and prints the results in Prometheus-compatible format (`data_format = "prometheus"`).
Errors are printed to `stderr` — they can be captured by Telegraf and logged to `/var/log/messages`.

---

## Arguments

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| **`-conn`** | `string` | `user=telegraf host=127.0.0.1 port=5435` | PostgreSQL connection string in libpq format. The tool appends `dbname=<DB>` internally. |
| **`-db-name`** | `string` | — | Databases to target: `all` or comma-separated list (`db1,db2,...`). If `all`, the list is resolved from `pg_database` (excluding `template0/1` and `postgres`). |
| **`-sql-cmd`** | `string` | — | SQL text (wrap in quotes!). Mutually exclusive with `-sql-file`. |
| **`-sql-file`** | `string` | — | Path to a file with SQL text. Mutually exclusive with `-sql-cmd`. |
| **`-SQLSpliter`** | `string` | `""` | Delimiter to split multiple SQL statements inside `-sql-cmd` / file. Example: `-SQLSpliter=";"`. |
| **`-labels`** | `string` | `""` | Comma-separated columns to **force as labels**. By default **all string columns** become labels; **numeric** columns (int/float/numeric) become metrics. This flag only *adds/forces* label behavior. |
| **`-ignoredColumns`** | `string` | `""` | Comma-separated columns to exclude completely from output. |
| **`-prefixMetric`** | `string` | `pgwatch` | Prefix added to every metric name: `<prefix>_<column>`. |
| **`-master-only`** | `bool` | `false` | Execute only if node is **primary** (not in recovery). |
| **`-replica-only`** | `bool` | `false` | Execute only if node is **replica** (in recovery). |
| **`-j`** | `int` | `1` | Max concurrent databases to process (parallelism). |
| **`-pg-timeout`** | `duration` | `5s` | Global timeout applied to **connect** and **each query** (per-query context). Go duration syntax (e.g. `250ms`, `3s`, `1m`). |
| **`-version`** | `bool` | — | Print build version and exit. |

> **Important quoting note:**  
> When using `-sql-cmd`, always quote the SQL to prevent shell splitting:
> -sql-cmd="YOURS SQL QUERY"

---

## Execution model

- **Parallel per-database:** each database is processed in parallel (bounded by `-j`) using a **separate PostgreSQL connection** per DB.
- **Sequential per database:** within a single database, all SQL statements (from `-sql-file` or `-sql-cmd` split by `-SQLSpliter`) run **sequentially on the same connection**.
- **Per-query timeout:** every SQL statement is executed with its **own timeout context** derived from the parent (`-pg-timeout`), so slow queries don’t stall others.
- **Role gate (optional):** if `-master-only` or `-replica-only` is set, the node role is checked once via `pg_is_in_recovery()` before running queries.

---

## Example — Telegraf configuration

```toml
[[inputs.exec]]
  commands = ["/data/scripts/pg_watcher -sql-file /data/scripts/statements.sql -conn 'user=telegraf port=5432' -db-name=postgres"]
  timeout = "10s"
  interval = "30m"
  data_format = "prometheus"

[[inputs.exec]]
  commands = ["/data/scripts/pg_watcher -j 3 -pg-timeout=11s -sql-file /data/scripts/table_stats.sql -conn 'telegraf port=5432' -db-name=all -master-only"]
  timeout = "10s"
  interval = "30m"
  data_format = "prometheus"

[[inputs.exec]]
  commands = ["/data/scripts/pg_watcher -sql-cmd="select datname, session_time, xact_commit from pg_stat_database;" -conn 'user=telegraf port=5432' -db-name=postgres"]
  timeout = "10s"
  interval = "1m"
  data_format = "prometheus"
```

---

## CLI Example

```bash
# Get metrics from odyssey pooler 
./pg_watcher   -db-name=console   -sql-cmd="show stats;show pools;"   -conn="user=telegraf password=XXX host=10.203.97.94 port=6432 sslmode=disable"   -labels=database,user   -ignoredColumns=pool_mode,avg_req,avg_recv,avg_sent,avg_query   -SQLSpliter=";"

# Get a metric from DB
./pg_watcher -db-name=testdb -conn="user=marat password=wolfik host=127.0.0.1 port=5440 sslmode=disable" -sql-cmd="select datname, session_time, xact_commit from pg_stat_database"
```

---

## Metric logic

- **All string columns** automatically become **labels**
- **Numeric columns** (`int`, `float`, `numeric`) automatically become **metrics**
- `--labels` can override and force specific columns to be labels
- `--ignoredColumns` removes columns entirely from the output

---

## Output example

```text
pgwatch_active_sessions{db="postgres",user="replica"} 5
pgwatch_db_size_bytes{db="postgres"} 2.409e+08
pgwatch_last_vacuum_age{db="postgres",table="users"} 1234
```
