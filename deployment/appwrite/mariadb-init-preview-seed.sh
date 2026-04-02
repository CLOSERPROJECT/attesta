#!/bin/sh
set -e

case "${APPWRITE_RESTORE_SEED_SQL:-false}" in
  1|true|TRUE|yes|YES|on|ON)
    echo "restoring preview seed into MariaDB"
    mariadb \
      --user=root \
      --password="${MYSQL_ROOT_PASSWORD}" \
      "${MYSQL_DATABASE}" < /seed/appwrite-seed.sql
    echo "cleaning Appwrite preview runtime state"
    mariadb \
      --user=root \
      --password="${MYSQL_ROOT_PASSWORD}" \
      "${MYSQL_DATABASE}" <<'SQL'
DELETE FROM _console_certificates;
DELETE FROM _console_rules;
DELETE FROM _console_sessions;
DELETE FROM _1_sessions;
SQL
    ;;
  *)
    echo "skipping MariaDB preview seed restore"
    ;;
esac
