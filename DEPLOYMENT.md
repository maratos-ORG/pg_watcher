# Deployment Guide

## Build

```bash
make build
```

Binary will be created at `bin/pg_watcher`.

---

## Test

```bash
# Run all tests (unit + pg_watcher)
make test_all

# Run unit tests only (with race detection and coverage)
make test

# Test pg_watcher only (build, start PostgreSQL, test pg_watcher, cleanup)
make test_pg_watcher

# Test full stack (PostgreSQL + Telegraf + pg_watcher)
make test_telegraf
```

---

## Production Deployment

### 1. Build Release Binary

```bash
git tag v1.0.0
make build
```

Version will be embedded from git tag.

### 2. Deploy Binary

```bash
# Copy binary to target location
sudo cp bin/pg_watcher /usr/local/bin/
sudo chmod +x /usr/local/bin/pg_watcher
```

### 3. Configure Telegraf

Example configuration for `/etc/telegraf/telegraf.conf`:

```toml
[[inputs.exec]]
  commands = [
    "/usr/local/bin/pg_watcher -sql-file /etc/telegraf/sql/db_stats.sql -conn 'user=monitor host=localhost port=5432' -db-name=all"
  ]
  timeout = "30s"
  interval = "1m"
  data_format = "prometheus"
```

### 4. Create SQL Files

Example `/etc/telegraf/sql/db_stats.sql`:

```sql
SELECT
  datname,
  numbackends as active_connections,
  xact_commit as transactions_committed,
  xact_rollback as transactions_rolled_back,
  blks_read as blocks_read,
  blks_hit as blocks_hit,
  tup_returned as tuples_returned,
  tup_fetched as tuples_fetched,
  tup_inserted as tuples_inserted,
  tup_updated as tuples_updated,
  tup_deleted as tuples_deleted
FROM pg_stat_database
WHERE datname NOT IN ('template0', 'template1');
```

### 5. Create Monitoring User

```sql
CREATE USER monitor WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE postgres TO monitor;
GRANT pg_monitor TO monitor;
```

### 6. Test Configuration

```bash
/usr/local/bin/pg_watcher \
  -sql-file /etc/telegraf/sql/db_stats.sql \
  -conn 'user=monitor password=secure_password host=localhost port=5432' \
  -db-name=all
```

Expected output: Prometheus-formatted metrics.

### 7. Restart Telegraf

```bash
systemctl restart telegraf
systemctl status telegraf
```

---

## Version Management

Check binary version:

```bash
./bin/pg_watcher -version
```

Version format:
- Git tag: `v1.2.3`
- Git commit: `72a3e18`
- Dirty working tree: `72a3e18-dirty`
- No git: `dev`
