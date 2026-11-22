#!/usr/bin/env bash
#
# End-to-end smoke test for LAN Clipboard.
#
# Starts a throwaway server (temp data dir, ephemeral port), exercises the REST
# API and the static frontend, then cleans everything up. Run via `make smoke`.
#
set -euo pipefail

cd "$(dirname "$0")/.."

PORT="${SMOKE_PORT:-18080}"
BASE="http://127.0.0.1:${PORT}"
BIN="./bin/lan-clipboard"
STATIC_DIR="$(pwd)/server/static"
TMP_DIR="$(mktemp -d)"
SERVER_PID=""

cleanup() {
    if [[ -n "${SERVER_PID}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
        kill "${SERVER_PID}" 2>/dev/null || true
        wait "${SERVER_PID}" 2>/dev/null || true
    fi
    rm -rf "${TMP_DIR}"
}
trap cleanup EXIT

fail() {
    echo "FAIL: $*" >&2
    exit 1
}

pass() {
    echo "  ✓ $*"
}

json_field() {
    # Extract a top-level-ish JSON string value without requiring jq.
    python3 -c 'import json,sys; print(json.load(sys.stdin)["'"$1"'"])'
}

[[ -x "${BIN}" ]] || fail "missing ${BIN} — run: make build-server"
[[ -f "${STATIC_DIR}/index.html" ]] || fail "missing ${STATIC_DIR}/index.html — run: make static"

echo "==> starting server on ${BASE} (data: ${TMP_DIR})"
CLIP_PORT="${PORT}" \
CLIP_DATA_DIR="${TMP_DIR}/data" \
CLIP_STATIC_DIR="${STATIC_DIR}" \
CLIP_RETENTION_DAYS=30 \
    "${BIN}" > "${TMP_DIR}/server.log" 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 100); do
    if curl -fsS "${BASE}/api/stats" > /dev/null 2>&1; then break; fi
    if ! kill -0 "${SERVER_PID}" 2>/dev/null; then
        cat "${TMP_DIR}/server.log" >&2
        fail "server exited during startup"
    fi
    sleep 0.1
done
curl -fsS "${BASE}/api/stats" > /dev/null || fail "server did not become ready"

echo "==> API"
STATS="$(curl -fsS "${BASE}/api/stats")"
[[ "$(echo "${STATS}" | json_field totalClips)" == "0" ]] || fail "expected empty stats, got ${STATS}"
pass "GET /api/stats"

CREATE="$(curl -fsS -X POST "${BASE}/api/clips" -H 'Content-Type: application/json' \
    -d '{"content":"smoke-test hello world"}')"
CLIP_ID="$(echo "${CREATE}" | json_field id)"
[[ -n "${CLIP_ID}" ]] || fail "create clip returned no id: ${CREATE}"
pass "POST /api/clips"

LIST="$(curl -fsS "${BASE}/api/clips")"
python3 - "${LIST}" "${CLIP_ID}" <<'PY' || fail "created clip missing from list"
import json, sys
data = json.loads(sys.argv[1])
assert data["clips"][0]["id"] == sys.argv[2], data
assert data["hasMore"] is False, data
PY
pass "GET /api/clips"

curl -fsS -X PATCH "${BASE}/api/clips/${CLIP_ID}" -H 'Content-Type: application/json' \
    -d '{"pinned":true}' > /dev/null
PINNED="$(curl -fsS "${BASE}/api/clips/pinned")"
python3 - "${PINNED}" "${CLIP_ID}" <<'PY' || fail "pinned clip missing"
import json, sys
data = json.loads(sys.argv[1])
assert data["clips"][0]["id"] == sys.argv[2], data
assert data["clips"][0]["pinned"] is True, data
PY
pass "PATCH pinned + GET /api/clips/pinned"

SEARCH="$(curl -fsS "${BASE}/api/clips/search?q=smoke")"
python3 - "${SEARCH}" <<'PY' || fail "search did not find the clip"
import json, sys
data = json.loads(sys.argv[1])
assert len(data["clips"]) == 1, data
PY
pass "GET /api/clips/search"

echo "smoke file $(date +%s)" > "${TMP_DIR}/hello.txt"
UPLOAD="$(curl -fsS -X POST "${BASE}/api/clips/upload" -F "file=@${TMP_DIR}/hello.txt")"
FILE_URL="$(python3 - "${UPLOAD}" <<'PY'
import json, sys
data = json.loads(sys.argv[1])
clip = data["clips"][0]
assert clip["type"] == "file", clip
print(clip["fileUrl"])
PY
)" || fail "upload failed: ${UPLOAD}"
curl -fsS "${BASE}${FILE_URL}" | grep -q "smoke file" || fail "uploaded file could not be downloaded"
pass "POST /api/clips/upload + download"

echo "==> static frontend"
curl -fsS "${BASE}/" | grep -q 'id="root"' || fail "index.html not served"
pass "GET / (SPA shell)"
curl -fsS "${BASE}/some/client/route" | grep -q 'id="root"' || fail "SPA fallback not working"
pass "GET /some/client/route (SPA fallback)"
ASSET="$(grep -o '/assets/[^"]*\.js' "${STATIC_DIR}/index.html" | head -1 || true)"
[[ -n "${ASSET}" ]] || fail "no JS bundle referenced by index.html"
curl -fsS "${BASE}${ASSET}" > /dev/null || fail "JS bundle not served: ${ASSET}"
pass "GET ${ASSET}"

echo "==> cleanup endpoints"
curl -fsS -X DELETE "${BASE}/api/clips/${CLIP_ID}" > /dev/null
curl -fsS -X DELETE "${BASE}/api/clips" > /dev/null
STATS="$(curl -fsS "${BASE}/api/stats")"
[[ "$(echo "${STATS}" | json_field totalClips)" == "0" ]] || fail "clips remain after clear: ${STATS}"
pass "DELETE /api/clips/:id + DELETE /api/clips"

echo
echo "SMOKE OK"
