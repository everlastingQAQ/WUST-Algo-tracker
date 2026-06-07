#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${script_dir}/lib.sh"

load_env

export PATH="/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:${PATH}"

FRONTEND_DIR="${FRONTEND_DIR:-${APP_ROOT}/frontend}"
BACKUP_ROOT="${BACKUP_ROOT:-${APP_ROOT}/backups}"
HEALTH_URL="${HEALTH_URL:-http://${DOMAIN:-127.0.0.1}:${NGINX_PORT:-8088}/}"
STAMP="$(date +%Y%m%d%H%M%S)"
BACKUP_DIR="${BACKUP_ROOT}/release-${STAMP}"

require_command curl
require_command docker
require_command envsubst
require_command go
require_command nginx
require_command npm
require_command psql
require_command sudo
require_command systemctl

if [[ -n "${SUDO_PASSWORD:-}" ]]; then
  printf "%s\n" "${SUDO_PASSWORD}" | sudo -S -v >/dev/null
else
  sudo -v
fi

if [[ ! -d "${FRONTEND_DIR}" ]]; then
  echo "Frontend directory not found: ${FRONTEND_DIR}"
  echo "Set FRONTEND_DIR=/path/to/WUST-Algo-Frontend or keep the default ${APP_ROOT}/frontend."
  exit 1
fi

backup_path() {
  local src="$1"
  local dst="$2"
  if [[ -e "${src}" ]]; then
    mkdir -p "$(dirname "${dst}")"
    cp -a "${src}" "${dst}"
  fi
}

echo "Creating release backup at ${BACKUP_DIR}..."
mkdir -p "${BACKUP_DIR}/tracker" "${BACKUP_DIR}/frontend" "${BACKUP_DIR}/systemd" "${BACKUP_DIR}/nginx"
backup_path "${APP_ROOT}/bin" "${BACKUP_DIR}/bin"
backup_path "${APP_ROOT}/conf" "${BACKUP_DIR}/conf"
backup_path "${APP_ROOT}/infra/docker-compose.yml" "${BACKUP_DIR}/infra/docker-compose.yml"
backup_path "${FRONTEND_DIR}/dist" "${BACKUP_DIR}/frontend/dist"
backup_path "/etc/nginx/sites-available/${NGINX_SITE_NAME:-wust-algo}" "${BACKUP_DIR}/nginx/${NGINX_SITE_NAME:-wust-algo}"
for unit in wust-user.service wust-core-data.service wust-gateway.service wust-agent.service; do
  backup_path "/etc/systemd/system/${unit}" "${BACKUP_DIR}/systemd/${unit}"
done

echo "Deploying backend services..."
bash "${script_dir}/deploy-backend.sh"

echo "Deploying frontend..."
(
  cd "${FRONTEND_DIR}"
  bash deploy/scripts/deploy-frontend.sh
)

echo "Running health checks..."
run_sudo systemctl --no-pager --full status wust-user.service wust-core-data.service wust-gateway.service >/dev/null
if [[ "${ENABLE_AGENT}" == "1" ]]; then
  run_sudo systemctl --no-pager --full status wust-agent.service >/dev/null
fi
run_sudo nginx -t >/dev/null
curl -fsS --max-time 10 -o /dev/null "${HEALTH_URL}"

echo "Release deployment finished."
echo "Backup: ${BACKUP_DIR}"
echo "Health URL: ${HEALTH_URL}"
