#!/usr/bin/env bash
# Fail when server templates use disallowed inline style= attributes.
# Allowed patterns are documented in docs/css.md (ADR-0001).
set -euo pipefail

root="$(cd "$(dirname "$0")/../.." && pwd)"
templates="${root}/server/templates"

allowed_pattern='--pill-bg:|--dept-color:|--dept-border:|--stream-color:|width:[[:space:]]*\{\{'

scan_styles() {
  if command -v rg >/dev/null 2>&1; then
    rg -n 'style=' "${templates}" || true
  else
    grep -rn 'style=' "${templates}" 2>/dev/null || true
  fi
}

violations=0

while IFS= read -r match; do
  [ -z "$match" ] && continue
  file="${match%%:*}"
  rest="${match#*:}"
  line_no="${rest%%:*}"
  content="${rest#*:}"

  if [[ "$content" =~ $allowed_pattern ]]; then
    continue
  fi

  echo "${file}:${line_no}: disallowed inline style" >&2
  echo "  ${content}" >&2
  violations=$((violations + 1))
done < <(scan_styles)

if [ "$violations" -gt 0 ]; then
  echo "" >&2
  echo "${violations} disallowed inline style(s). See docs/css.md" >&2
  exit 1
fi

echo "template inline styles: ok"
