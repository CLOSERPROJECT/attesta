#!/usr/bin/env bash
# Red-capable loop for: hard-refresh does not show saved draft Formata array rows.
# GREEN only when Mongo has nested rows AND (if COOKIE_JAR set) SSR FormValue has them.
# Client UI emptiness is asserted separately via browser CDP (see AGENT notes).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "${ROOT}/.env.worktree"
INSTANCE_ID="${1:-6ac9157dc8acd5d3db318a49}"
SUBSTEP_KEY="${2:-1_1}"
ARRAY_KEY="${3:-batches}"

echo "== Mongo progress for ${INSTANCE_ID} / ${SUBSTEP_KEY} =="
mongo_json="$(mongosh --quiet --port "${MONGODB_PORT}" closer_demo --eval "
const p = db.processes.findOne({_id: ObjectId('${INSTANCE_ID}')});
if (!p) { print('MISSING_PROCESS'); quit(2); }
const step = (p.progress || {})['${SUBSTEP_KEY}'];
print(JSON.stringify(step && step.data || null));
" 2>/dev/null | tail -n 1)"
echo "progress.data=${mongo_json}"

row_count="$(ARRAY_KEY="${ARRAY_KEY}" MONGO_JSON="${mongo_json}" python3 - <<'PY'
import json, os
data=json.loads(os.environ["MONGO_JSON"])
arr=(data or {}).get(os.environ["ARRAY_KEY"]) or []
print(len(arr) if isinstance(arr, list) else -1)
PY
)"
if [[ "${row_count}" -lt 1 ]]; then
  echo "RED: Mongo has no nested ${ARRAY_KEY} rows (save path broken or wrong instance)"
  exit 1
fi
echo "OK: Mongo has ${row_count} ${ARRAY_KEY} row(s)"

if [[ -n "${COOKIE_JAR:-}" ]]; then
  STREAM_KEY="${STREAM_KEY:-6ac90e24ebf16384b69efab0}"
  url="http://localhost:${PORT}/my/streams/${STREAM_KEY}/instance/${INSTANCE_ID}"
  html="$(curl -sS -b "${COOKIE_JAR}" -c "${COOKIE_JAR}" "${url}")"
  if ! grep -q 'js-formata-value' <<<"${html}"; then
    echo "RED: no .js-formata-value in HTML (auth/route)"
    exit 1
  fi
  if ! grep -q "loadId" <<<"${html}" && ! grep -q "&#34;loadId&#34;" <<<"${html}"; then
    echo "RED: SSR HTML FormValue missing saved row fields"
    exit 1
  fi
  echo "OK: SSR HTML includes FormValue row fields"
  echo "NOTE: If UI still empty, client Formata restore is RED (no setValue/value prop on formata-form@main)."
else
  echo "SKIP SSR check (set COOKIE_JAR to an authenticated cookie file)"
fi

echo "Loop seam Mongo(+SSR) GREEN for nested draft rows; UI restore is a separate client assertion."
