#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

if [[ $# -ne 0 ]]; then
  echo "error: bootstrap accepts no port arguments; edit .env.worktree after generation" >&2
  exit 1
fi

# mise trusts configuration by absolute checkout path. Trust before invoking
# `wt` or `task` because either binary may itself be installed through mise.
if command -v mise >/dev/null 2>&1; then
  mise trust --yes "${ROOT_DIR}/mise.toml"
  missing_mise_tools="$(mise ls --missing --no-header)"
  if [[ -n "${missing_mise_tools}" ]]; then
    mise install
  fi
fi

primary="$(git worktree list --porcelain | awk '/^worktree / { print substr($0, 10); exit }')"
if [[ -z "${primary}" || ! -d "${primary}" ]]; then
  echo "error: could not resolve primary worktree root" >&2
  exit 1
fi

if [[ "$(pwd -P)" == "$(cd "${primary}" && pwd -P)" ]]; then
  echo "bootstrap on primary worktree: classic ports"
  ./scripts/worktree-env.sh write --classic
else
  setup_file_missing=false
  if [[ -f "${primary}/.worktreeinclude" ]]; then
    while IFS= read -r include_path || [[ -n "${include_path}" ]]; do
      include_path="${include_path%/}"
      [[ -z "${include_path}" || "${include_path}" == \#* ]] && continue
      if [[ -e "${primary}/${include_path}" && ! -e "${ROOT_DIR}/${include_path}" ]]; then
        setup_file_missing=true
        break
      fi
    done <"${primary}/.worktreeinclude"
  fi

  if [[ "${setup_file_missing}" == true ]]; then
    echo "copying allowlisted ignored setup files"
    if command -v wt >/dev/null 2>&1; then
      wt step copy-ignored --require-include || true
    elif command -v mise >/dev/null 2>&1; then
      mise exec -- wt step copy-ignored --require-include || true
    fi

    # Plain Git worktrees have no copy-ignored hook. This literal allowlist
    # fallback also covers an unavailable Worktrunk installation and never
    # overwrites files already supplied by Codex.
    while IFS= read -r include_path || [[ -n "${include_path}" ]]; do
      include_path="${include_path%/}"
      [[ -z "${include_path}" || "${include_path}" == \#* ]] && continue
      if [[ "${include_path}" == /* || "${include_path}" == *..* || "${include_path}" == *'*'* || "${include_path}" == *'?'* ]]; then
        echo "error: unsupported .worktreeinclude path: ${include_path}" >&2
        exit 1
      fi
      source_path="${primary}/${include_path}"
      target_path="${ROOT_DIR}/${include_path}"
      [[ -e "${source_path}" && ! -L "${source_path}" && ! -e "${target_path}" ]] || continue
      mkdir -p "$(dirname "${target_path}")"
      cp -R "${source_path}" "${target_path}"
    done <"${primary}/.worktreeinclude"
  fi

  ./scripts/worktree-env.sh write
  ./scripts/worktree-data.sh snapshot-primary
fi

if [[ ! -f .env ]]; then
  echo "error: .env is missing after bootstrap" >&2
  echo "create ${primary}/.env from .env.example, then rerun bootstrap" >&2
  exit 1
fi

git submodule update --init --recursive

echo
echo "bootstrap complete"
./scripts/worktree-env.sh print
