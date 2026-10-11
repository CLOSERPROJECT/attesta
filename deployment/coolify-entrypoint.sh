#!/bin/sh
set -e

# Hard-fail prepare (ADR-0004). Server loads /tmp/attesta-appwrite.env itself.
./attesta prepare
exec "$@"
