#!/usr/bin/env bash
# Fail when stylesheet modules use literal px in @media (width …) queries.
# Canonical thresholds live in breakpoints.css as @custom-media aliases.
# Use @media (--sm-down) etc. instead. See docs/css.md.
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"
styles="${root}/web/src/styles"

scan_media_width_px() {
  if command -v rg >/dev/null 2>&1; then
    rg -n '@media \(width[^)]*[0-9]+px' "${styles}" \
      --glob '!breakpoints.css' || true
  else
    find "${styles}" -name '*.css' ! -name 'breakpoints.css' -print0 \
      | xargs -0 grep -En '@media \(width[^)]*[0-9]+px' 2>/dev/null || true
  fi
}

violations=0

while IFS= read -r match; do
  [ -z "$match" ] && continue

  echo "${match}: literal px in @media (width …) — use @custom-media from breakpoints.css" >&2
  violations=$((violations + 1))
done < <(scan_media_width_px)

if [ "$violations" -gt 0 ]; then
  echo "" >&2
  echo "${violations} disallowed @media (width … px) rule(s). See docs/css.md" >&2
  exit 1
fi

echo "css breakpoints: ok"
