#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${script_dir}/lib.sh"

load_env

require_command docker
require_command envsubst
require_command go
require_command psql
require_command sudo
require_command systemctl

mkdir -p "${APP_ROOT}/bin" "${APP_ROOT}/infra"

echo "Installing infrastructure compose file..."
install -m 0644 "${deploy_dir}/docker-compose.infra.yml" "${APP_ROOT}/infra/docker-compose.yml"
install -m 0600 "${deploy_dir}/.env" "${APP_ROOT}/infra/.env"

echo "Starting PostgreSQL, Redis, RabbitMQ and Consul..."
docker compose --env-file "${APP_ROOT}/infra/.env" -f "${APP_ROOT}/infra/docker-compose.yml" up -d

echo "Waiting for PostgreSQL..."
until PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d postgres -c "select 1" >/dev/null 2>&1; do
  sleep 2
done

echo "Creating databases if needed..."
PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d postgres \
  -tc "SELECT 1 FROM pg_database WHERE datname = '${POSTGRES_USER_DB}'" | grep -q 1 || \
  PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d postgres -c "CREATE DATABASE ${POSTGRES_USER_DB}"

PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d postgres \
  -tc "SELECT 1 FROM pg_database WHERE datname = '${POSTGRES_CORE_DB}'" | grep -q 1 || \
  PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d postgres -c "CREATE DATABASE ${POSTGRES_CORE_DB}"

"${script_dir}/render-configs.sh"

echo "Building backend services..."
cd "${repo_dir}"
go mod download
go build -o "${APP_ROOT}/bin/user" ./app/user/cmd/user
go build -o "${APP_ROOT}/bin/core_data" ./app/core_data/cmd/core_data
go build -o "${APP_ROOT}/bin/agent" ./app/agent/cmd/agent

cd "${repo_dir}/app/gateway"
go mod download
go build -o "${APP_ROOT}/bin/gateway" ./cmd/gateway

echo "Installing systemd units..."
sudo_write_template "${deploy_dir}/systemd/wust-user.service.tpl" /etc/systemd/system/wust-user.service
sudo_write_template "${deploy_dir}/systemd/wust-core-data.service.tpl" /etc/systemd/system/wust-core-data.service
sudo_write_template "${deploy_dir}/systemd/wust-agent.service.tpl" /etc/systemd/system/wust-agent.service
sudo_write_template "${deploy_dir}/systemd/wust-gateway.service.tpl" /etc/systemd/system/wust-gateway.service

run_sudo systemctl daemon-reload
run_sudo systemctl enable wust-user.service
run_sudo systemctl enable wust-core-data.service
run_sudo systemctl enable wust-gateway.service
run_sudo systemctl restart wust-user.service
run_sudo systemctl restart wust-core-data.service
run_sudo systemctl restart wust-gateway.service

if [[ "${ENABLE_AGENT}" == "1" ]]; then
  run_sudo systemctl enable wust-agent.service
  run_sudo systemctl restart wust-agent.service
else
  run_sudo systemctl disable --now wust-agent.service >/dev/null 2>&1 || true
  echo "Agent service skipped because ENABLE_AGENT=${ENABLE_AGENT}."
fi

echo "Backend deployment finished."
systemctl --no-pager --full status wust-user.service wust-core-data.service wust-gateway.service || true
