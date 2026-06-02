[Unit]
Description=WUST Algo Agent Service
After=network.target docker.service wust-user.service wust-core-data.service

[Service]
User=${APP_USER}
WorkingDirectory=${APP_ROOT}/tracker
EnvironmentFile=-${APP_ROOT}/tracker/deploy/.env
ExecStart=${APP_ROOT}/bin/agent -conf ${APP_ROOT}/conf/agent
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
