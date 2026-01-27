#!/usr/bin/env bash
set -euo pipefail

cd /app

export API_PORT="${API_PORT:-80}"
export NGINX_CONF_GEN_ENABLED="${NGINX_CONF_GEN_ENABLED:-1}"
export NGINX_GENERATED_CONF="${NGINX_GENERATED_CONF:-/etc/nginx/conf.d/switchboard.generated.conf}"

mkdir -p "$(dirname "$NGINX_GENERATED_CONF")"
touch "$NGINX_GENERATED_CONF"

exec switchboard
