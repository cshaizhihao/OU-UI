#!/usr/bin/env bash
set -Eeuo pipefail

VERSION="v0.3.0"
REPO_URL="https://github.com/cshaizhihao/OU-UI"
DEFAULT_INSTALL_DIR="/opt/ou-ui"
DEFAULT_PANEL_PORT="3000"
DEFAULT_AGENT_INTERVAL="10s"

INSTALL_DIR=""
SOURCE_DIR=""
DOMAIN=""
PANEL_PORT=""
SECURE_PATH=""
ENABLE_SSL="no"
ACCESS_HOST="server-ip"
START_SERVICE="no"

print_line() { printf '%s\n' "------------------------------------------------------------"; }
info() { printf '[信息] %s\n' "$1"; }
warn() { printf '[提醒] %s\n' "$1"; }
fail() { printf '[错误] %s\n' "$1" >&2; exit 1; }

on_error() {
  local exit_code=$?
  local line_no=${1:-unknown}
  printf '\n[错误] 安装脚本在第 %s 行失败，退出码 %s。\n' "$line_no" "$exit_code" >&2
  printf '[排查] 请查看上方命令输出；常见原因包括 Docker 未启动、端口被占用、DNS token 未导出、证书路径不可写。\n' >&2
  printf '[排查] 修复后可重新执行：bash scripts/install.sh\n' >&2
  exit "$exit_code"
}
trap 'on_error $LINENO' ERR

ask() {
  local prompt="$1"
  local default_value="${2:-}"
  local answer
  if [[ -n "$default_value" ]]; then
    read -r -p "$prompt [$default_value]: " answer
    printf '%s' "${answer:-$default_value}"
  else
    read -r -p "$prompt: " answer
    printf '%s' "$answer"
  fi
}

ask_yes_no() {
  local prompt="$1"
  local default_value="${2:-n}"
  local answer
  while true; do
    read -r -p "$prompt (y/n) [$default_value]: " answer
    answer="${answer:-$default_value}"
    case "$answer" in
      y|Y|yes|YES|Yes) return 0 ;;
      n|N|no|NO|No) return 1 ;;
      *) warn "请输入 y 或 n。" ;;
    esac
  done
}

normalize_path() {
  local raw_path="$1"
  raw_path="${raw_path/#\~/$HOME}"
  printf '%s' "$raw_path"
}

validate_safe_path() {
  local target="$1"
  [[ "$target" = /* ]] || fail "安装目录必须是绝对路径。"
  [[ "$target" != *[[:space:]]* ]] || fail "安装目录不能包含空格或制表符。"
  [[ "$target" != "/" ]] || fail "不能安装到根目录 /。"
  [[ "$target" != "/root" ]] || fail "不能直接安装到 /root。"
  [[ "$target" != "/home" ]] || fail "不能直接安装到 /home。"
  [[ "$target" != "/etc" ]] || fail "不能直接安装到 /etc。"
  [[ "$target" != "/usr" ]] || fail "不能直接安装到 /usr。"
  [[ "$target" != "/var" ]] || fail "不能直接安装到 /var。"
  [[ "$target" != "/opt" ]] || fail "不能直接安装到 /opt，请使用 /opt/ou-ui 这类子目录。"
  [[ "$target" != *".."* ]] || fail "安装目录不能包含 ..。"
}

validate_port() {
  local port="$1"
  [[ "$port" =~ ^[0-9]+$ ]] || fail "端口必须是数字。"
  (( port >= 1 && port <= 65535 )) || fail "端口必须在 1 到 65535 之间。"
}

random_string() {
  local length="$1"
  local bytes
  local value
  if command -v openssl >/dev/null 2>&1; then
    bytes=$(( (length + 1) / 2 ))
    value="$(openssl rand -hex "$bytes")"
    printf '%s' "${value:0:length}"
  else
    set +o pipefail
    value="$(tr -dc 'A-Za-z0-9' </dev/urandom | head -c "$length")"
    set -o pipefail
    printf '%s' "$value"
  fi
}

detect_os() {
  local os_name="unknown"
  local os_version="unknown"
  if [[ -r /etc/os-release ]]; then
    os_name="$(awk -F= '$1=="PRETTY_NAME"{gsub(/^"|"$/, "", $2); print $2}' /etc/os-release)"
    os_version="$(awk -F= '$1=="VERSION_ID"{gsub(/^"|"$/, "", $2); print $2}' /etc/os-release)"
    os_name="${os_name:-unknown}"
    os_version="${os_version:-unknown}"
  fi
  info "系统检测：$(uname -srm 2>/dev/null || printf 'unknown')；发行版：${os_name} (${os_version})"
  case "$(uname -s 2>/dev/null || true)" in
    Linux) ;;
    *) warn "当前脚本主要面向 Linux 服务器，其他系统请优先使用 Docker Compose 手动部署。" ;;
  esac
}

require_command() {
  local command_name="$1"
  local hint="$2"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    fail "缺少命令：$command_name。$hint"
  fi
}

detect_docker_compose() {
  require_command docker "请先安装 Docker Engine。"
  if ! docker info >/dev/null 2>&1; then
    fail "Docker daemon 未运行或当前用户无权限访问 Docker。请启动 Docker，或将当前用户加入 docker 组后重新登录。"
  fi
  if docker compose version >/dev/null 2>&1; then
    info "Docker Compose 检测通过：$(docker compose version --short 2>/dev/null || docker compose version)"
  else
    fail "未检测到 Docker Compose v2。请安装 docker compose 插件后重试。"
  fi
}

is_port_in_use() {
  local port="$1"
  if command -v ss >/dev/null 2>&1; then
    ss -ltnH "( sport = :$port )" 2>/dev/null | grep -q .
  elif command -v lsof >/dev/null 2>&1; then
    lsof -iTCP:"$port" -sTCP:LISTEN -Pn >/dev/null 2>&1
  elif command -v netstat >/dev/null 2>&1; then
    netstat -ltn 2>/dev/null | awk '{print $4}' | grep -Eq "[:.]$port$"
  else
    warn "未检测到 ss/lsof/netstat，无法自动确认端口占用。"
    return 1
  fi
}

check_port_available() {
  local port="$1"
  if is_port_in_use "$port"; then
    fail "端口 $port 已被占用。请换一个端口，或先停止占用该端口的服务。"
  fi
  info "端口检测通过：$port 当前未监听。"
}

detect_public_ip() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsS --max-time 5 https://api.ipify.org || true
  fi
}

find_acme_bin() {
  if command -v acme.sh >/dev/null 2>&1; then
    command -v acme.sh
    return 0
  fi
  if [[ -x "$HOME/.acme.sh/acme.sh" ]]; then
    printf '%s\n' "$HOME/.acme.sh/acme.sh"
    return 0
  fi
  return 1
}

install_acme_if_needed() {
  if find_acme_bin >/dev/null 2>&1; then
    return 0
  fi
  require_command curl "acme.sh 自动安装需要 curl。"
  info "正在安装 acme.sh 到当前用户目录。"
  curl -fsSL https://get.acme.sh | sh -s email="${ACME_EMAIL:-admin@$DOMAIN}"
}

validate_cert_pair() {
  local cert_file="$1"
  local key_file="$2"
  [[ -s "$cert_file" ]] || fail "证书文件不存在或为空：$cert_file"
  [[ -s "$key_file" ]] || fail "私钥文件不存在或为空：$key_file"
  if command -v openssl >/dev/null 2>&1; then
    openssl x509 -in "$cert_file" -noout -subject -issuer -dates >/dev/null
    openssl rsa -in "$key_file" -check -noout >/dev/null 2>&1 || openssl pkey -in "$key_file" -check -noout >/dev/null
  else
    warn "未检测到 openssl，已跳过证书内容校验，仅检查文件存在性。"
  fi
  info "证书校验通过：$cert_file 与 $key_file"
}

run_acme_cloudflare() {
  [[ -n "${CF_Token:-}" ]] || fail "未检测到 CF_Token 环境变量。请先执行 export CF_Token='Cloudflare API Token' 后重试；不要把 token 写入仓库。"
  [[ -n "${ACME_EMAIL:-}" ]] || warn "未设置 ACME_EMAIL，将使用 admin@$DOMAIN 注册 acme.sh 账号。"

  install_acme_if_needed
  local acme_bin
  acme_bin="$(find_acme_bin)" || fail "acme.sh 安装后仍未找到可执行文件。"
  info "acme.sh 路径：$acme_bin"

  "$acme_bin" --set-default-ca --server letsencrypt
  "$acme_bin" --register-account -m "${ACME_EMAIL:-admin@$DOMAIN}" || true
  "$acme_bin" --issue --dns dns_cf -d "$DOMAIN"
  "$acme_bin" --install-cert -d "$DOMAIN" \
    --key-file "$INSTALL_DIR/certs/privkey.pem" \
    --fullchain-file "$INSTALL_DIR/certs/fullchain.pem" \
    --reloadcmd "cd $INSTALL_DIR && docker compose restart ou-ui-web || true"
  validate_cert_pair "$INSTALL_DIR/certs/fullchain.pem" "$INSTALL_DIR/certs/privkey.pem"
}

write_env_file() {
  local env_file="$1"
  umask 077
  cat >"$env_file" <<EOF_ENV
OUUI_VERSION=$VERSION
OUUI_DOMAIN=$DOMAIN
OUUI_PANEL_PORT=$PANEL_PORT
OUUI_SECURE_PATH=$SECURE_PATH
OUUI_ENABLE_SSL=$ENABLE_SSL
OUUI_ADMIN_USER=$ADMIN_USER
OUUI_ADMIN_PASSWORD=$ADMIN_PASSWORD
OUUI_JWT_SECRET=$JWT_SECRET
OUUI_AGENT_JOIN_TOKEN=$AGENT_JOIN_TOKEN
OUUI_DB=/app/data/ou-ui.db
OUUI_TLS_CERT_FILE=/app/certs/fullchain.pem
OUUI_TLS_KEY_FILE=/app/certs/privkey.pem
EOF_ENV
  chmod 600 "$env_file"
}

write_compose_file() {
  local compose_file="$1"
  cat >"$compose_file" <<EOF_COMPOSE
services:
  ou-ui-web:
    image: ou-ui-web:local
    build:
      context: "$SOURCE_DIR"
      dockerfile: Dockerfile
      target: web
    container_name: ou-ui-web
    restart: unless-stopped
    env_file:
      - .env
    ports:
      - "\${OUUI_PANEL_PORT:-3000}:3000"
    volumes:
      - ./certs:/app/certs:ro
    depends_on:
      - ou-ui-server

  ou-ui-server:
    image: ou-ui-server:local
    build:
      context: "$SOURCE_DIR"
      dockerfile: Dockerfile
      target: server
    container_name: ou-ui-server
    restart: unless-stopped
    env_file:
      - .env
    environment:
      OUUI_HOST: "0.0.0.0"
      OUUI_PORT: "8080"
      OUUI_DB: "/app/data/ou-ui.db"
    volumes:
      - ./data:/app/data
      - ./certs:/app/certs:ro
EOF_COMPOSE
}

write_agent_install_doc() {
  local doc_file="$1"
  cat >"$doc_file" <<EOF_DOC
# OU-UI Agent 安装入口

面板安装脚本已生成 Agent 注册令牌。请在需要接入的节点上使用下面的命令构建并运行 Agent。

```bash
git clone $REPO_URL
cd OU-UI
go build -o ou-ui-agent ./apps/agent
./ou-ui-agent \\
  -server "$LOGIN_URL" \\
  -token "$AGENT_JOIN_TOKEN" \\
  -name "\$(hostname)" \\
  -interval "$DEFAULT_AGENT_INTERVAL"
```

注意事项：
- 不要把 Agent 注册令牌写入仓库、工单、截图或日志。
- 如果面板启用了 HTTPS，请先确认证书链可信。
- 注册完成后，后续心跳使用服务端返回的 Agent Token。
EOF_DOC
}

print_line
printf 'OU-UI %s 中文交互式安装脚本\n' "$VERSION"
printf '仓库：%s\n' "$REPO_URL"
print_line
printf '安装前声明：\n'
printf '1. 本脚本会创建 OU-UI 运行目录、.env、Docker Compose 文件和 Agent 安装说明。\n'
printf '2. 输入域名时会强制启用 HTTPS，并可通过 acme.sh + Cloudflare DNS 签发证书。\n'
printf '3. 脚本会生成随机安全路径、管理员账号、管理员密码和 Agent 注册令牌。\n'
printf '4. 请勿把 .env、证书私钥、日志或任何真实凭据提交到仓库。\n'
print_line

if ! ask_yes_no "是否确认安装声明并继续" "n"; then
  fail "用户未确认安装声明，已退出。"
fi

require_command bash "请使用 bash 执行本脚本。"
require_command mkdir "系统缺少基础命令 mkdir。"
require_command chmod "系统缺少基础命令 chmod。"
require_command grep "系统缺少基础命令 grep。"
require_command awk "系统缺少基础命令 awk。"
detect_os

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

INSTALL_DIR="$(normalize_path "$(ask "请输入安装目录" "$DEFAULT_INSTALL_DIR")")"
validate_safe_path "$INSTALL_DIR"

PANEL_PORT="$(ask "请输入面板运行端口" "$DEFAULT_PANEL_PORT")"
validate_port "$PANEL_PORT"
check_port_available "$PANEL_PORT"

SECURE_PATH="/$(random_string 6)"
ADMIN_USER="admin_$(random_string 8)"
ADMIN_PASSWORD="$(random_string 28)"
JWT_SECRET="$(random_string 48)"
AGENT_JOIN_TOKEN="ouj_$(random_string 32)"

if ask_yes_no "是否已经完成域名 DNS 解析" "n"; then
  DOMAIN="$(ask "请输入面板绑定域名，例如 ouui.example.com")"
  [[ -n "$DOMAIN" ]] || fail "域名不能为空。"
  ACCESS_HOST="$DOMAIN"
  ENABLE_SSL="yes"
  print_line
  printf '检测到域名：%s\n' "$DOMAIN"
  printf 'OU-UI 将强制使用 HTTPS，并尝试通过 acme.sh 签发证书。\n'
  printf 'Cloudflare DNS 模式需要你先在当前 shell 设置：export CF_Token=你的令牌\n'
  printf '如需指定邮箱，可设置：export ACME_EMAIL=你的邮箱\n'
  print_line
  if ask_yes_no "是否现在使用 acme.sh + Cloudflare DNS 自动签发证书" "y"; then
    mkdir -p "$INSTALL_DIR/certs"
    run_acme_cloudflare
  else
    warn "你选择暂不签发证书。域名模式仍强制 HTTPS，请在启动前放置证书到 $INSTALL_DIR/certs。"
  fi
else
  DETECTED_IP="$(detect_public_ip)"
  if [[ -n "$DETECTED_IP" ]]; then
    ACCESS_HOST="$DETECTED_IP"
  fi
  warn "未配置域名，将降级使用 HTTP://IP:端口 访问。"
fi

print_line
printf '安装确认：\n'
printf '版本：%s\n' "$VERSION"
printf '安装目录：%s\n' "$INSTALL_DIR"
printf '面板端口：%s\n' "$PANEL_PORT"
printf '域名：%s\n' "${DOMAIN:-未配置}"
printf 'HTTPS：%s\n' "$ENABLE_SSL"
printf '安全路径：%s\n' "$SECURE_PATH"
print_line

if ! ask_yes_no "是否创建配置并继续" "y"; then
  fail "用户取消，未写入配置。"
fi

mkdir -p "$INSTALL_DIR/data" "$INSTALL_DIR/certs" "$INSTALL_DIR/docs"
if [[ -f "$INSTALL_DIR/.env" ]]; then
  warn "$INSTALL_DIR/.env 已存在。覆盖会生成新的安全路径、账号和密码。"
  if ask_yes_no "是否覆盖现有 .env" "n"; then
    write_env_file "$INSTALL_DIR/.env"
  else
    warn "保留现有 .env，跳过随机凭据写入。"
  fi
else
  write_env_file "$INSTALL_DIR/.env"
fi

if [[ ! -f "$INSTALL_DIR/docker-compose.yml" ]]; then
  write_compose_file "$INSTALL_DIR/docker-compose.yml"
else
  warn "$INSTALL_DIR/docker-compose.yml 已存在，未覆盖。"
fi

if [[ "$ENABLE_SSL" == "yes" ]]; then
  if [[ -f "$INSTALL_DIR/certs/fullchain.pem" || -f "$INSTALL_DIR/certs/privkey.pem" ]]; then
    validate_cert_pair "$INSTALL_DIR/certs/fullchain.pem" "$INSTALL_DIR/certs/privkey.pem"
  else
    warn "HTTPS 已启用，但证书尚未放置。启动前需要提供 fullchain.pem 和 privkey.pem。"
  fi
fi

if [[ "$ENABLE_SSL" == "yes" ]]; then
  ACCESS_URL="https://$ACCESS_HOST"
  [[ "$PANEL_PORT" != "443" ]] && ACCESS_URL="$ACCESS_URL:$PANEL_PORT"
else
  ACCESS_URL="http://$ACCESS_HOST:$PANEL_PORT"
fi
LOGIN_URL="$ACCESS_URL$SECURE_PATH"

write_agent_install_doc "$INSTALL_DIR/docs/agent-install.md"

if ask_yes_no "是否现在检测 Docker Compose 并执行 docker compose up -d --build" "n"; then
  detect_docker_compose
  START_SERVICE="yes"
  (cd "$INSTALL_DIR" && docker compose up -d --build)
else
  warn "已跳过启动。稍后可执行：cd $INSTALL_DIR && docker compose up -d --build"
fi

print_line
printf 'OU-UI %s 安装配置已完成。\n' "$VERSION"
printf '面板登录链接：%s\n' "$LOGIN_URL"
printf '管理员账号：%s\n' "$ADMIN_USER"
printf '管理员密码：%s\n' "$ADMIN_PASSWORD"
printf 'Agent 注册令牌：%s\n' "$AGENT_JOIN_TOKEN"
printf 'Agent 安装说明：%s/docs/agent-install.md\n' "$INSTALL_DIR"
printf '安装目录：%s\n' "$INSTALL_DIR"
printf '环境文件：%s/.env\n' "$INSTALL_DIR"
printf '服务已启动：%s\n' "$START_SERVICE"
printf '\n请立即妥善保存登录链接、账号和密码；不要提交 .env、证书私钥或任何真实凭据。\n'
print_line
