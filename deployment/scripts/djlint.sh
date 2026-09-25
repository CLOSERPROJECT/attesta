#!/usr/bin/env bash
# Run the mise-pinned djLint for this checkout (portable across machines).
# Used by Taskfile and by the monosans.djlint VS Code/Cursor extension
# via .vscode/settings.json → djlint.executablePath.
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"
cd "${root}"

if command -v mise >/dev/null 2>&1; then
  exec mise exec -- djlint "$@"
fi

if command -v djlint >/dev/null 2>&1; then
  exec djlint "$@"
fi

echo "djlint not found. Install mise tools from the repo root:" >&2
echo "  mise install" >&2
echo "(requires pipx:djlint from mise.toml)" >&2
exit 127
