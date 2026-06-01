[Unit]
Description=WUST Algo Gateway
After=network.target docker.service wust-user.service wust-core-data.service

[Service]
User=${APP_USER}
WorkingDirectory=${APP_ROOT}/tracker/app/gateway
ExecStart=${APP_ROOT}/bin/gateway -addr ${GATEWAY_ADDR} -conf ${APP_ROOT}/conf/gateway/config.yaml -discovery.dsn consul://${CONSUL_HOST}:${CONSUL_PORT}
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
