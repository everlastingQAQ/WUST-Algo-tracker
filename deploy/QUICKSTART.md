# WUST Algo Server Quickstart

This quickstart assumes Ubuntu 22.04 and the default deployment root `/opt/wust-algo`.

## 1. Prepare System Packages

```bash
sudo apt update
sudo apt install -y git curl nginx gettext-base postgresql-client build-essential
```

Create the deployment user:

```bash
sudo adduser acm_tracker
sudo usermod -aG sudo acm_tracker
getent group docker >/dev/null && sudo usermod -aG docker acm_tracker
```

Install Docker with the Compose plugin if it is not already available:

```bash
docker compose version
```

Install Node.js 22 for frontend builds:

```bash
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt install -y nodejs
```

Install Go matching `go.mod`:

```bash
cd /tmp
wget https://go.dev/dl/go1.25.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.3.linux-amd64.tar.gz
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
source ~/.bashrc
```

## 2. Clone Repositories

```bash
sudo mkdir -p /opt/wust-algo
sudo chown -R acm_tracker:acm_tracker /opt/wust-algo

sudo -iu acm_tracker
cd /opt/wust-algo
git clone https://github.com/everlastingQAQ/WUST-Algo-tracker.git tracker
git clone https://github.com/everlastingQAQ/WUST-Algo-Frontend.git frontend
```

## 3. Deploy Backend

```bash
cd /opt/wust-algo/tracker
cp deploy/.env.example deploy/.env
nano deploy/.env
bash deploy/scripts/deploy-backend.sh
```

Keep `ENABLE_AGENT=0` until OpenAI-compatible AI settings and SMTP settings are real.
For DeepSeek, set `AI_BASE_URL=https://api.deepseek.com`, `AI_MODEL=deepseek-chat`, and `AI_API_KEY`.

## 4. Deploy Frontend

```bash
cd /opt/wust-algo/frontend
cp deploy/.env.example deploy/.env
nano deploy/.env
bash deploy/scripts/deploy-frontend.sh
```

Set `DOMAIN` in `deploy/.env` before running the frontend script.

## 5. Create First Admin

Register a normal account from the website, then run:

```bash
cd /opt/wust-algo/tracker
bash deploy/scripts/init-admin.sh your_username
```

## 6. Check Status

```bash
cd /opt/wust-algo/tracker
bash deploy/scripts/status.sh
curl http://127.0.0.1:8080/v1/user/group/list
```
