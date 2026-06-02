# WUST Algo Tracker

WUST Algo Tracker 是 WUST ACM 算法训练数据平台的后端仓库，包含用户服务、核心数据服务、网关、AI 总结服务以及一套适用于 Ubuntu 服务器的部署脚本。

前端仓库：`https://github.com/everlastingQAQ/WUST-Algo-Frontend`

## 功能概览

- 用户系统：注册邀请码、登录、JWT 鉴权、角色管理、个人资料、头像链接、邮件通知。
- 后台管理：用户列表、角色调整、分组调整、密码重置、用户软删除。
- 团队系统：创建团队、队长管理团队、邀请成员、处理邀请、团队成员刷题统计。
- OJ 数据抓取：支持 AtCoder、NowCoder、LuoGu、CodeForces、QOJ。
- 数据统计：总提交、总 AC、时间段统计、平台拆分统计、热力图、用户排名、团队排名。
- 比赛数据：比赛列表、比赛详情、比赛排行榜。
- AI 总结：支持 OpenAI-compatible API，可配置 DeepSeek 等模型服务。
- 部署资产：Docker Compose 中间件、systemd 服务、配置渲染、状态检查脚本。

## 技术栈

- Go 1.25.3
- Kratos
- PostgreSQL
- Redis
- RabbitMQ
- Consul
- GORM
- systemd
- Docker Compose

## 服务结构

```text
app/
├── user/       # 用户、角色、团队、注册邀请码
├── core_data/  # OJ 绑定、爬虫、提交记录、统计、比赛
├── gateway/    # API gateway
└── agent/      # AI 总结服务
```

默认部署目录：

```text
/opt/wust-algo/
├── tracker/
├── frontend/
├── bin/
├── conf/
└── infra/
```

## 快速部署

以下步骤适用于 Ubuntu 22.04。

### 1. 创建运行用户

```bash
sudo adduser acm_tracker
sudo usermod -aG sudo acm_tracker
getent group docker >/dev/null && sudo usermod -aG docker acm_tracker
```

### 2. 准备目录

```bash
sudo mkdir -p /opt/wust-algo
sudo chown -R acm_tracker:acm_tracker /opt/wust-algo
sudo -iu acm_tracker
cd /opt/wust-algo
```

### 3. 获取代码

```bash
git clone https://github.com/everlastingQAQ/WUST-Algo-tracker.git tracker
git clone https://github.com/everlastingQAQ/WUST-Algo-Frontend.git frontend
```

也可以将打包好的源码传到服务器后解压到 `tracker` 和 `frontend`。

### 4. 安装 Go

```bash
cd /tmp
wget https://go.dev/dl/go1.25.3.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.25.3.linux-amd64.tar.gz
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
source ~/.bashrc
go version
```

如果服务器访问 `go.dev` 较慢，可以使用可信镜像源下载同版本 Go。

### 5. 部署后端

```bash
cd /opt/wust-algo/tracker
cp deploy/.env.example deploy/.env
nano deploy/.env
bash deploy/scripts/deploy-backend.sh
```

脚本会启动 PostgreSQL、Redis、RabbitMQ、Consul，并安装以下 systemd 服务：

- `wust-user`
- `wust-core-data`
- `wust-gateway`
- `wust-agent`，仅当 `ENABLE_AGENT=1` 时启用

### 6. 部署前端

```bash
cd /opt/wust-algo/frontend
cp deploy/.env.example deploy/.env
nano deploy/.env
bash deploy/scripts/deploy-frontend.sh
```

## 配置说明

后端配置位于 `deploy/.env`。常用字段：

- `POSTGRES_*`：PostgreSQL 连接信息。
- `REDIS_*`：Redis 连接信息。
- `RABBITMQ_*`：RabbitMQ 连接信息。
- `CONSUL_*`：Consul 服务发现地址。
- `USER_HTTP_ADDR` / `CORE_HTTP_ADDR`：用户服务和核心数据服务 HTTP 地址。
- `GATEWAY_ADDR`：网关监听地址。
- `ENABLE_AGENT`：是否启用 AI 总结服务。
- `AI_BASE_URL` / `AI_MODEL` / `AI_API_KEY`：OpenAI-compatible AI 配置。
- `SMTP_*`：邮件发送配置。

DeepSeek 示例：

```env
ENABLE_AGENT=1
AI_BASE_URL=https://api.deepseek.com
AI_MODEL=deepseek-chat
AI_API_KEY=replace-with-your-api-key
```

不要提交真实 `.env`、API Key、SMTP 授权码或数据库密码。

## 注册和管理员

注册默认需要邀请码，默认值在后端系统配置中初始化为：

```text
wustacm666
```

管理员可在后台系统设置中修改邀请码。

首次创建管理员：

```bash
cd /opt/wust-algo/tracker
bash deploy/scripts/init-admin.sh your_username
```

## 常用维护命令

查看状态：

```bash
cd /opt/wust-algo/tracker
bash deploy/scripts/status.sh
```

重启服务：

```bash
sudo systemctl restart wust-user
sudo systemctl restart wust-core-data
sudo systemctl restart wust-gateway
```

查看日志：

```bash
sudo journalctl -u wust-user -n 100 --no-pager
sudo journalctl -u wust-core-data -n 100 --no-pager
```

重新部署后端：

```bash
cd /opt/wust-algo/tracker
git pull
bash deploy/scripts/deploy-backend.sh
```

## 本地构建

```bash
go build -o /tmp/wust-user ./app/user/cmd/user
go build -o /tmp/wust-core-data ./app/core_data/cmd/core_data
go build -o /tmp/wust-agent ./app/agent/cmd/agent
```

## 统计口径

AC 统计按 `user_id + platform + problem` 去重，避免同一用户在同一平台同一题多次 AC 被重复计入。系统账号 `admin` 不参与公开排名，其他管理员账号正常计入排名。

团队排名中，成员数显示团队真实人数；团队刷题数按团队成员贡献汇总。

## 安全注意事项

- 修改生产环境 JWT secret 后再公开暴露服务。
- 不要将 `deploy/.env` 提交到 Git。
- 管理员账号不应使用弱密码。
- 生产环境建议只开放 Nginx 入口，数据库、Redis、RabbitMQ、Consul 不要直接暴露到公网。

## 致谢

本项目基于无锡学院相关开源项目继续开发，感谢原项目在 GitHub 上贡献的源码。
