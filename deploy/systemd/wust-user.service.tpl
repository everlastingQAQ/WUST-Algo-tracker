[Unit]
Description=WUST Algo User Service
After=network.target docker.service

[Service]
User=${APP_USER}
WorkingDirectory=${APP_ROOT}/tracker
ExecStart=${APP_ROOT}/bin/user -conf ${APP_ROOT}/conf/user
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
