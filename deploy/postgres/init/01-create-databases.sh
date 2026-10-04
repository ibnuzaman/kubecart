#!/bin/sh
# Creates one database and one login role per service.
# The postgres image runs this only on first start, when the data volume is
# empty. Changing it later has no effect until the volume is removed.
set -eu

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres \
  -v orders_pw="$ORDERS_DB_PASSWORD" \
  -v inventory_pw="$INVENTORY_DB_PASSWORD" <<'SQL'
CREATE ROLE orders LOGIN PASSWORD :'orders_pw';
CREATE DATABASE orders OWNER orders;
-- By default every role may connect to every database. Revoke that, so the
-- owner is the only non-superuser that can connect (database-per-service).
REVOKE CONNECT ON DATABASE orders FROM PUBLIC;

CREATE ROLE inventory LOGIN PASSWORD :'inventory_pw';
CREATE DATABASE inventory OWNER inventory;
REVOKE CONNECT ON DATABASE inventory FROM PUBLIC;
SQL
