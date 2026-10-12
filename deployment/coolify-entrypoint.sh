#!/bin/sh
set -e

# App root (not WORKDIR /app/server): prepare/server read/write $ATTESTA_ROOT_DIR/.env.appwrite
export ATTESTA_ROOT_DIR="${ATTESTA_ROOT_DIR:-/app}"

# Hard-fail prepare (ADR-0004). Server loads $ATTESTA_ROOT_DIR/.env.appwrite itself.
./attesta prepare
exec "$@"
