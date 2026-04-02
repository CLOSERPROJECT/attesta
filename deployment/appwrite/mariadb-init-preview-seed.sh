#!/bin/sh
set -e

case "${APPWRITE_RESTORE_SEED_SQL:-false}" in
  1|true|TRUE|yes|YES|on|ON)
    echo "restoring preview seed into MariaDB"
    mariadb \
      --user=root \
      --password="${MYSQL_ROOT_PASSWORD}" \
      "${MYSQL_DATABASE}" < /seed/appwrite-seed.sql
    ;;
  *)
    echo "skipping MariaDB preview seed restore"
    ;;
esac
