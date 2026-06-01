#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <username>"
  exit 1
fi

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib.sh
source "${script_dir}/lib.sh"

load_env
require_command psql

username="$1"
PGPASSWORD="${POSTGRES_PASSWORD}" psql -h "${POSTGRES_HOST}" -p "${POSTGRES_PORT}" -U "${POSTGRES_USER}" -d "${POSTGRES_USER_DB}" \
  -v username="${username}" \
  -c "UPDATE users SET role_id = 1 WHERE username = :'username';"

echo "Promoted ${username} to admin. Log out and log back in on the frontend."
