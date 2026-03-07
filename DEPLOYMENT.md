# Deployment Guide

## GitHub Actions CI

### Automated Tests

- **On Pull Request to `main`**: runs `make test` (unit tests) and `make test_telegraf` (full stack)

### Workflow

```bash
# 1. Create feature branch and push
git checkout -b feature/my-changes
git add .
git commit -m "Add feature"
git push -u origin feature/my-changes
# → Create PR on GitHub → Tests run automatically

# 2. After PR merged, create release tag
git checkout main
git pull origin main
git tag v1.0.3
git push origin v1.0.3
```

---

## Test

Some tests require Docker. You can use a completely free alternative on your laptop [rancherdesktop](https://rancherdesktop.io/).
```bash
# Run all tests (unit + pg_watcher + telegraf)
make test_all

# Run unit tests only (with race detection and coverage)
make test

# Test pg_watcher only (build, start PostgreSQL, test pg_watcher, cleanup)
make test_pg_watcher

# Test full stack (PostgreSQL + Telegraf + pg_watcher)
make test_telegraf
```

---

## Telegraf Configuration
### 1. Create SQL Files

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

### 2. Configure Telegraf
Example configuration for `/etc/telegraf/telegraf.conf`:

```toml
[[inputs.exec]]
  commands = [
    "/usr/local/bin/pg_watcher -sql-file /etc/telegraf/sql/db_stats.sql -conn 'user=telegraf host=localhost port=5432' -db-name=all"
  ]
  timeout = "30s"
  interval = "1m"
  data_format = "prometheus"
```

### 3. Create Monitoring User in DB

```sql
CREATE USER telegraf WITH PASSWORD 'secure_password';
GRANT CONNECT ON DATABASE postgres TO telegraf; --Optional
GRANT pg_monitor TO telegraf;
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
