#!/usr/bin/env bash
# Symlink the primary checkout's .env into a worktree (or any target root).
# Usage:
#   deployment/scripts/worktree-env.sh              # current git toplevel
#   deployment/scripts/worktree-env.sh /path/to/wt  # explicit worktree root
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

main_root="$(git worktree list --porcelain | awk '/^worktree / { print substr($0, 10); exit }')"
if [[ -z "${main_root}" || ! -d "${main_root}" ]]; then
  echo "error: could not resolve primary worktree root" >&2
  exit 1
fi

target_root="${1:-$(git rev-parse --show-toplevel)}"
if [[ -z "${target_root}" || ! -d "${target_root}" ]]; then
  echo "error: target root does not exist: ${target_root:-<empty>}" >&2
  exit 1
fi

src="${main_root}/.env"
dst="${target_root}/.env"

if [[ ! -f "${src}" ]]; then
  echo "error: no .env in primary checkout (${main_root})" >&2
  echo "copy .env.example to .env there first" >&2
  exit 1
fi

if [[ -L "${dst}" ]]; then
  current="$(readlink "${dst}")"
  if [[ "${current}" == "${src}" ]]; then
    echo "ok: ${dst} already linked to ${src}"
    bash "$SCRIPT_DIR/worktree-ports.sh" "$target_root"
    exit 0
  fi
  echo "error: ${dst} is a symlink to ${current}, not ${src}" >&2
  exit 1
fi

if [[ -e "${dst}" ]]; then
  echo "ok: ${dst} already exists (not replacing a real file)"
  bash "$SCRIPT_DIR/worktree-ports.sh" "$target_root"
  exit 0
fi

ln -s "${src}" "${dst}"
echo "linked ${dst} -> ${src}"
bash "$SCRIPT_DIR/worktree-ports.sh" "$target_root"
