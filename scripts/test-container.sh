#!/usr/bin/env bash
# Verify a local image with disposable data; requires Docker Compose v2, Go, curl.
# Usage: bash scripts/test-container.sh lan-clipboard:ci [volume|bind]
set -euo pipefail
cd "$(dirname "$0")/.."

TEST_DIR="$(mktemp -d)"
export CLIP_IMAGE="${1:-lan-clipboard:ci}"
export CLIP_HOST_DATA_DIR="${TEST_DIR}/data"
export CLIP_HOST_PORT="${CONTAINER_TEST_PORT:-18080}"
export CLIP_BIND_ADDRESS=127.0.0.1 CLIP_PUBLIC_ORIGIN=
STORAGE="${2:-volume}"
COMPOSE=(docker compose --env-file /dev/null -f compose.yaml -p "clipboard-test-$$")
case "$STORAGE" in
    volume) ;;
    bind) COMPOSE+=(-f compose.bind.yaml); mkdir -p "$CLIP_HOST_DATA_DIR" ;;
    *) echo 'Storage must be volume or bind' >&2; exit 2 ;;
esac

cleanup() {
    local result=$?
    if (( result != 0 )); then "${COMPOSE[@]}" logs --no-color >&2 || true; fi
    "${COMPOSE[@]}" down --timeout 15 --volumes || true
    # Remove only this test's files, including files owned by the container UID.
    if [[ "$STORAGE" == bind ]]; then
        docker run --rm --user 0:0 --entrypoint sh \
            --mount "type=bind,source=${CLIP_HOST_DATA_DIR},target=/test-data" \
            "$CLIP_IMAGE" -c 'find /test-data -mindepth 1 -delete' || true
    fi
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

"${COMPOSE[@]}" config --quiet
if [[ "$STORAGE" == bind ]]; then
    docker run --rm --user 0:0 --entrypoint sh \
        --mount "type=bind,source=${CLIP_HOST_DATA_DIR},target=/test-data" \
        "$CLIP_IMAGE" -c 'chown 10001:10001 /test-data && chmod 700 /test-data'
fi

start() {
    "${COMPOSE[@]}" up -d --no-build --pull never
    for (( attempt=0; attempt<90; attempt++ )); do
        CONTAINER_ID="$("${COMPOSE[@]}" ps -q clipboard)"
        if [[ -n "$CONTAINER_ID" ]] && \
            [[ "$(docker inspect --format '{{.State.Health.Status}}' "$CONTAINER_ID")" == healthy ]]; then
            curl --fail --silent --show-error --max-time 5 "http://127.0.0.1:${CLIP_HOST_PORT}/api/stats"
            return
        fi
        sleep 1
    done
    echo 'Container did not become healthy' >&2
    return 1
}

start
[[ "$("${COMPOSE[@]}" exec -T clipboard id -u)" == 10001 ]]
"${COMPOSE[@]}" exec -T clipboard sh -c 'test -s /usr/share/licenses/lan-clipboard/LICENSE && test -s /usr/share/licenses/lan-clipboard/npm/pdfjs-dist/LICENSE'
(cd server && "${GO:-go}" run ../scripts/check-runtime.go "http://127.0.0.1:${CLIP_HOST_PORT}" create "$TEST_DIR/state.json")
[[ "$("${COMPOSE[@]}" exec -T clipboard sqlite3 /app/data/clipboard.db 'PRAGMA journal_mode;')" == wal ]]
[[ "$("${COMPOSE[@]}" exec -T clipboard sqlite3 /app/data/clipboard.db "SELECT count(*) FROM clips_fts WHERE clips_fts MATCH 'cafe';")" == 1 ]]
"${COMPOSE[@]}" exec -T clipboard sh -c 'test -f /app/data/clipboard.db-wal && test -f /app/data/clipboard.db-shm'

"${COMPOSE[@]}" stop --timeout 15
[[ "$(docker inspect --format '{{.State.ExitCode}}' "$CONTAINER_ID")" == 0 ]]
"${COMPOSE[@]}" rm --force clipboard
start
(cd server && "${GO:-go}" run ../scripts/check-runtime.go "http://127.0.0.1:${CLIP_HOST_PORT}" verify "$TEST_DIR/state.json")
[[ "$("${COMPOSE[@]}" exec -T clipboard sh -c 'find /app/data/uploads -type f | wc -l' | tr -d '[:space:]')" == 0 ]]
echo "PASS container: non-root, health, WAL/SHM, SIGTERM, $STORAGE persistence and file deletion"
