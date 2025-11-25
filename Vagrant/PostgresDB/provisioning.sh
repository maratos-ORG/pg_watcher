#!/usr/bin/env bash
set -euxo pipefail

# --- Configuration ---
POSTGRES_VERSION="${POSTGRES_VERSION:-17}"
POSTGRES_INSTALL="${POSTGRES_INSTALL:-true}"

# --- 0) Basic packages ---
  sudo apt-get update -y
  sudo apt-get install -y ca-certificates curl gnupg lsb-release
  
if [ "$POSTGRES_INSTALL" = true ]; then
  echo "[INFO] PostgreSQL installation enabled."

  # --- 1) Add the official PostgreSQL repository for Ubuntu 24.04 (noble) ---
  sudo install -d -m 0755 /usr/share/keyrings
  curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc \
    | sudo gpg --dearmor -o /usr/share/keyrings/postgresql.gpg

  echo "deb [arch=arm64 signed-by=/usr/share/keyrings/postgresql.gpg] \
  https://apt.postgresql.org/pub/repos/apt noble-pgdg main" \
    | sudo tee /etc/apt/sources.list.d/pgdg.list > /dev/null

  sudo apt-get update -y

  # --- 2) Install PostgreSQL packages ---
  sudo apt-get install -y \
    postgresql-${POSTGRES_VERSION} \
    postgresql-client-${POSTGRES_VERSION} \
    postgresql-server-dev-${POSTGRES_VERSION}

  # --- 3) Enable and start the service ---
  sudo systemctl enable postgresql
  sudo systemctl restart postgresql

  # --- 4) Adjust configuration files ---
  PGCONF="/etc/postgresql/${POSTGRES_VERSION}/main/postgresql.conf"
  PGHBA="/etc/postgresql/${POSTGRES_VERSION}/main/pg_hba.conf"

  # Listen on all addresses
  sudo sed -ri "s|^[# ]*listen_addresses *=.*|listen_addresses = '*'|" "$PGCONF"

  # Enable pg_stat_statements
  sudo sed -ri "s|^[# ]*shared_preload_libraries *=.*|shared_preload_libraries = 'pg_stat_statements'|" "$PGCONF"
  grep -q "^pg_stat_statements.track =" "$PGCONF" \
    || echo "pg_stat_statements.track = all" | sudo tee -a "$PGCONF" >/dev/null
  grep -q "^pg_stat_statements.track_planning =" "$PGCONF" \
    || echo "pg_stat_statements.track_planning = on" | sudo tee -a "$PGCONF" >/dev/null

  # Allow all remote connections (demo)
  echo "host all all 0.0.0.0/0 md5" | sudo tee -a "$PGHBA" >/dev/null

  sudo systemctl restart postgresql

  # --- 5) Create role, database and extension (idempotent) ---
  sudo -u postgres psql -v ON_ERROR_STOP=1 <<'SQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='marat') THEN
    CREATE ROLE marat WITH SUPERUSER LOGIN PASSWORD 'wolfik';
  END IF;
END $$;
SQL

  sudo -u postgres psql -v ON_ERROR_STOP=1 -tc \
    "SELECT 1 FROM pg_database WHERE datname='testdb'" \
    | grep -q 1 || sudo -u postgres createdb -O marat testdb

  sudo -u postgres psql -v ON_ERROR_STOP=1 -d testdb -c \
    "CREATE EXTENSION IF NOT EXISTS pg_stat_statements;"

  # --- 6) Print version info ---
  echo "[INFO] PostgreSQL installed successfully."
  psql --version
  sudo -u postgres psql -c "SELECT version();"

else
  echo "[INFO] PostgreSQL installation skipped because POSTGRES_INSTALL=false."
fi