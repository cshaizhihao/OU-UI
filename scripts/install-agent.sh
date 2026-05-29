#!/usr/bin/env bash
set -Eeuo pipefail

PANEL_URL="${1:-${OUUI_SERVER_URL:-}}"
JOIN_TOKEN="${2:-${OUUI_AGENT_JOIN_TOKEN:-}}"
AGENT_NAME="${OUUI_AGENT_NAME:-$(hostname)}"
INSTALL_DIR="${OUUI_AGENT_INSTALL_DIR:-/opt/ou-ui-agent}"
DATA_DIR="${OUUI_AGENT_DATA_DIR:-/var/lib/ou-ui-agent}"

fail() {
  printf '[错误] %s\n' "$1" >&2
  exit 1
}

[[ "$(id -u)" -eq 0 ]] || fail "请使用 root 运行 Agent 安装脚本。"
[[ -n "$PANEL_URL" ]] || fail "缺少主控地址。用法：bash install-agent.sh https://面板域名/安全路径 注册令牌"
[[ -n "$JOIN_TOKEN" ]] || fail "缺少 Agent 注册令牌。"

mkdir -p "$INSTALL_DIR" "$DATA_DIR"
chmod 700 "$DATA_DIR"

cat >"$INSTALL_DIR/ou-ui-agent.env" <<EOF_ENV
OUUI_SERVER_URL=$PANEL_URL
OUUI_AGENT_JOIN_TOKEN=$JOIN_TOKEN
OUUI_AGENT_NAME=$AGENT_NAME
OUUI_AGENT_DATA_DIR=$DATA_DIR
EOF_ENV
chmod 600 "$INSTALL_DIR/ou-ui-agent.env"

cat >/etc/systemd/system/ou-ui-agent.service <<EOF_SERVICE
[Unit]
Description=OU-UI Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$INSTALL_DIR/ou-ui-agent.env
ExecStart=$INSTALL_DIR/ou-ui-agent
Restart=always
RestartSec=5
WorkingDirectory=$DATA_DIR

[Install]
WantedBy=multi-user.target
EOF_SERVICE

systemctl daemon-reload

printf 'OU-UI Agent 配置已完成。\n'
printf '主控地址：%s\n' "$PANEL_URL"
printf 'Agent 名称：%s\n' "$AGENT_NAME"
printf '安装目录：%s\n' "$INSTALL_DIR"
printf '数据目录：%s\n' "$DATA_DIR"
printf 'v0.3.0 暂未发布预编译 Agent 二进制，请先构建：go build -o %s/ou-ui-agent ./apps/agent\n' "$INSTALL_DIR"
printf '二进制就绪后执行：systemctl enable --now ou-ui-agent\n'
