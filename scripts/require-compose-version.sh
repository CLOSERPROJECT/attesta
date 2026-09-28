#!/usr/bin/env bash
# Shared Docker Compose compatibility check for worktree lifecycle scripts.

require_compose_version() {
  local version major minor patch
  if ! version="$(docker compose version --short 2>/dev/null)"; then
    echo "error: Docker Compose 2.24.4 or newer is required" >&2
    return 1
  fi
  version="${version#v}"
  if [[ ! "${version}" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+) ]]; then
    echo "error: Docker Compose 2.24.4 or newer is required (could not parse ${version})" >&2
    return 1
  fi
  major="${BASH_REMATCH[1]}"
  minor="${BASH_REMATCH[2]}"
  patch="${BASH_REMATCH[3]}"
  if ((major < 2 || (major == 2 && minor < 24) || (major == 2 && minor == 24 && patch < 4))); then
    echo "error: Docker Compose 2.24.4 or newer is required (found ${version})" >&2
    return 1
  fi
}
