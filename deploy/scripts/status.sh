#!/usr/bin/env bash
set -euo pipefail

systemctl --no-pager --full status wust-user.service wust-core-data.service wust-gateway.service || true
systemctl --no-pager --full status wust-agent.service || true
docker ps --filter "name=wust-algo"
