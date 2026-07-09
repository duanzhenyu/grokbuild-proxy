#!/usr/bin/env bash
set -euo pipefail

if [[ "${GROKBUILD_LIVE_SMOKE:-}" != "1" ]]; then
  echo "Refusing live upstream test. Set GROKBUILD_LIVE_SMOKE=1 explicitly." >&2
  exit 2
fi

base_url="${GROKBUILD_BASE_URL:-http://127.0.0.1:8080}"
api_key="${GROKBUILD_API_KEY:-}"

if [[ -z "${api_key}" ]]; then
  echo "GROKBUILD_API_KEY is required." >&2
  exit 2
fi

curl --fail --silent --show-error "${base_url}/healthz" >/dev/null
curl --fail --silent --show-error "${base_url}/readyz" >/dev/null
curl --fail --silent --show-error \
  -H "Authorization: Bearer ${api_key}" \
  "${base_url}/v1/models" >/dev/null

if [[ -n "${GROKBUILD_LIVE_MODEL:-}" ]]; then
  curl --fail --silent --show-error \
    -H "Authorization: Bearer ${api_key}" \
    -H "Content-Type: application/json" \
    -d "{\"model\":\"${GROKBUILD_LIVE_MODEL}\",\"input\":\"Reply with exactly: ok\",\"max_output_tokens\":8}" \
    "${base_url}/v1/responses" >/dev/null
fi

echo "live smoke passed"
