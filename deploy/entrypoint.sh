#!/usr/bin/env bash
set -euo pipefail

cd /app

export API_PORT="${API_PORT:-8069}"

export NGINX_ENABLED="${NGINX_ENABLED:-1}"
export NGINX_GENERATED_CONF="${NGINX_GENERATED_CONF:-/etc/nginx/conf.d/switchboard.generated.conf}"

mkdir -p "$(dirname "$NGINX_GENERATED_CONF")"
touch "$NGINX_GENERATED_CONF"

switchboard &
SB_PID=$!

nginx -g 'daemon off;' &
NGINX_PID=$!

shutdown() {
  # Best-effort, don't hang forever.
  set +e

  kill -TERM "$SB_PID" 2>/dev/null

  nginx -s quit 2>/dev/null || true

  for _ in $(seq 1 30); do
    if ! kill -0 "$SB_PID" 2>/dev/null && ! kill -0 "$NGINX_PID" 2>/dev/null; then
      return 0
    fi
    sleep 0.2
  done

  kill -KILL "$SB_PID" 2>/dev/null
  kill -KILL "$NGINX_PID" 2>/dev/null
}

trap shutdown INT TERM

while kill -0 "$SB_PID" 2>/dev/null && kill -0 "$NGINX_PID" 2>/dev/null; do
  sleep 1
done

shutdown
wait || true
