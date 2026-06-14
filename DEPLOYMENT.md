# WUST Algo Tracker Deployment

This repository contains the backend services and infrastructure deployment files for WUST Algo.

## Server Layout

The scripts assume this layout by default:

```text
/opt/wust-algo/
├── tracker/
├── frontend/
├── bin/
├── conf/
└── infra/
```

You can change paths and credentials in `deploy/.env`.

The default systemd runtime user is `acm_tracker`. Create it before deployment:

```bash
sudo adduser acm_tracker
sudo usermod -aG sudo acm_tracker
getent group docker >/dev/null && sudo usermod -aG docker acm_tracker
```

## One-Time Backend Deployment

```bash
cd /opt/wust-algo
git clone https://github.com/WUSTACM/WUST-Algo-tracker.git tracker
cd /opt/wust-algo/tracker

cp deploy/.env.example deploy/.env
nano deploy/.env

sudo apt update
sudo apt install -y gettext-base postgresql-client build-essential

bash deploy/scripts/deploy-backend.sh
```

The script starts these middleware containers:

- PostgreSQL
- Redis
- RabbitMQ
- Consul

It then builds and installs these systemd services:

- `wust-user`
- `wust-core-data`
- `wust-gateway`
- `wust-agent` when `ENABLE_AGENT=1`

## Preflight Before Full Release

On servers that have both backend and frontend repositories in place, run preflight before a routine release:

```bash
cd /opt/wust-algo/tracker
bash deploy/scripts/preflight.sh
```

The preflight script checks required commands, key environment variables, deployment directory permissions, the frontend repository path, and common port conflicts before the release modifies running services. `deploy/scripts/deploy-release.sh` runs it automatically.

## Re-Deploy Backend After Code Changes

```bash
cd /opt/wust-algo/tracker
git pull
bash deploy/scripts/deploy-backend.sh
```

## Promote First Admin

Register a normal user from the frontend, then run:

```bash
cd /opt/wust-algo/tracker
bash deploy/scripts/init-admin.sh your_username
```

## Check Status

```bash
cd /opt/wust-algo/tracker
bash deploy/scripts/status.sh
```

## Notes

- The backend code currently uses a hard-coded JWT secret in `app/common/const/const.go`; change it before exposing the service publicly.
- Agent features require valid OpenAI-compatible AI settings, for example DeepSeek:
  `AI_BASE_URL=https://api.deepseek.com`, `AI_MODEL=deepseek-chat`, and `AI_API_KEY=...`.
  Leave `ENABLE_AGENT=0` until those values and SMTP settings are ready.
