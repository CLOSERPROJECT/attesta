#!/usr/bin/env bash
# Run the mise-pinned djLint for this checkout (portable across machines).
# Used by Taskfile and by the monosans.djlint VS Code/Cursor extension
# via .vscode/settings.json → djlint.executablePath.
#
# Cursor/VS Code launched from the Dock often have a stripped PATH that
# omits Homebrew and mise shims; probe common install locations too.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "${root}"

find_cmd() {
  local name="$1"
  local candidate
  if command -v "${name}" >/dev/null 2>&1; then
    command -v "${name}"
    return 0
  fi
  for candidate in \
    "${HOME}/.local/bin/${name}" \
    "${HOME}/.local/share/mise/shims/${name}" \
    "/opt/homebrew/bin/${name}" \
    "/usr/local/bin/${name}"; do
    if [[ -x "${candidate}" ]]; then
      printf '%s\n' "${candidate}"
      return 0
    fi
  done
  return 1
}

if mise_bin="$(find_cmd mise)"; then
  exec "${mise_bin}" exec -- djlint "$@"
fi

if djlint_bin="$(find_cmd djlint)"; then
  exec "${djlint_bin}" "$@"
fi

echo "djlint not found. Install mise tools from the repo root:" >&2
echo "  mise install" >&2
echo "(requires pipx:djlint from mise.toml)" >&2
exit 127
