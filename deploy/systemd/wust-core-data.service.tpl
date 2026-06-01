[Unit]
Description=WUST Algo Core Data Service
After=network.target docker.service wust-user.service

[Service]
User=${APP_USER}
WorkingDirectory=${APP_ROOT}/tracker
ExecStart=${APP_ROOT}/bin/core_data -conf ${APP_ROOT}/conf/core_data
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
