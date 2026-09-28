#!/usr/bin/env bash
# Backward-compatible entrypoint for the current worktree's isolated stack.
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
exec bash "${ROOT_DIR}/scripts/worktree-compose.sh" up "$@"
