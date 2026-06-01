#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${script_dir}/lib.sh"

load_env
require_command envsubst

mkdir -p "${APP_ROOT}/conf/user" "${APP_ROOT}/conf/core_data" "${APP_ROOT}/conf/agent" "${APP_ROOT}/conf/gateway"

envsubst < "${deploy_dir}/config/user.config.yaml.tpl" > "${APP_ROOT}/conf/user/config.yaml"
envsubst < "${deploy_dir}/config/core_data.config.yaml.tpl" > "${APP_ROOT}/conf/core_data/config.yaml"
envsubst < "${deploy_dir}/config/agent.config.yaml.tpl" > "${APP_ROOT}/conf/agent/config.yaml"
envsubst < "${deploy_dir}/config/gateway.config.yaml.tpl" > "${APP_ROOT}/conf/gateway/config.yaml"

echo "Rendered configs under ${APP_ROOT}/conf"
