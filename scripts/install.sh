#!/usr/bin/env bash
set -Eeuo pipefail

VERSION="v0.1.0"
REPO_URL="https://github.com/cshaizhihao/OU-UI"
DEFAULT_INSTALL_DIR="/opt/ou-ui"
DEFAULT_PANEL_PORT="3000"

print_line() { printf '%s\n' "------------------------------------------------------------"; }
info() { printf '[信息] %s\n' "$1"; }
warn() { printf '[提醒] %s\n' "$1"; }
fail() { printf '[错误] %s\n' "$1" >&2; exit 1; }

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

detect_public_ip() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsS --max-time 5 https://api.ipify.org || true
  fi
}

install_acme_if_needed() {
  if command -v acme.sh >/dev/null 2>&1; then
    return 0
  fi
  if [[ -x "$HOME/.acme.sh/acme.sh" ]]; then
    return 0
  fi
  command -v curl >/dev/null 2>&1 || fail "未检测到 curl，无法自动安装 acme.sh。"
  info "正在安装 acme.sh。"
  curl https://get.acme.sh | sh -s email="${ACME_EMAIL:-admin@$DOMAIN}"
}

run_acme_cloudflare() {
  [[ -n "${CF_Token:-}" ]] || fail "未检测到 CF_Token 环境变量。请先 export CF_Token='你的 Cloudflare API Token' 后重试。"
  [[ -n "${ACME_EMAIL:-}" ]] || warn "未设置 ACME_EMAIL，将使用 admin@$DOMAIN 作为 acme.sh 注册邮箱。"

  install_acme_if_needed
  local acme_bin="acme.sh"
  if ! command -v acme.sh >/dev/null 2>&1; then
    acme_bin="$HOME/.acme.sh/acme.sh"
  fi

  "$acme_bin" --set-default-ca --server letsencrypt
  "$acme_bin" --register-account -m "${ACME_EMAIL:-admin@$DOMAIN}" || true
  "$acme_bin" --issue --dns dns_cf -d "$DOMAIN"
  "$acme_bin" --install-cert -d "$DOMAIN" \
    --key-file "$INSTALL_DIR/certs/privkey.pem" \
    --fullchain-file "$INSTALL_DIR/certs/fullchain.pem" \
    --reloadcmd "docker compose -f $INSTALL_DIR/docker-compose.yml restart || true"
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

print_line
printf 'OU-UI %s 中文交互式安装脚本\n' "$VERSION"
printf '仓库：%s\n' "$REPO_URL"
print_line
printf '安装协议确认：\n'
printf '1. 本脚本会创建 OU-UI 主控面板运行目录、配置文件和 Docker Compose 文件。\n'
printf '2. 若输入域名，脚本会要求 DNS 已解析，并自动尝试通过 acme.sh 签发证书。\n'
printf '3. 脚本会自动生成安全路径、管理员账号、管理员密码和 Agent 注册令牌。\n'
printf '4. 请不要把 .env、证书私钥、日志或任何真实凭据提交到仓库。\n'
print_line

if ! ask_yes_no "请输入 y 确认安装协议并继续" "n"; then
  fail "用户未确认安装协议，已退出。"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

INSTALL_DIR="$(normalize_path "$(ask "请输入安装目录" "$DEFAULT_INSTALL_DIR")")"
validate_safe_path "$INSTALL_DIR"

PANEL_PORT="$(ask "请输入面板运行端口" "$DEFAULT_PANEL_PORT")"
validate_port "$PANEL_PORT"

SECURE_PATH="/$(random_string 6)"
ADMIN_USER="admin_$(random_string 8)"
ADMIN_PASSWORD="$(random_string 28)"
JWT_SECRET="$(random_string 48)"
AGENT_JOIN_TOKEN="ouj_$(random_string 32)"

DOMAIN=""
ENABLE_SSL="no"
ACCESS_HOST="服务器IP"

if ask_yes_no "是否已经在 Cloudflare 等平台完成域名 DNS 解析" "n"; then
  DOMAIN="$(ask "请输入面板绑定域名，例如 ouui.zze.cc")"
  [[ -n "$DOMAIN" ]] || fail "域名不能为空。"
  ACCESS_HOST="$DOMAIN"
  ENABLE_SSL="yes"
  print_line
  printf '检测到你输入了域名：%s\n' "$DOMAIN"
  printf 'OU-UI 将强制使用 HTTPS，并尝试通过 acme.sh 自动签发证书。\n'
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

mkdir -p "$INSTALL_DIR/data" "$INSTALL_DIR/certs"
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

START_SERVICE="no"
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
  if ask_yes_no "是否现在执行 docker compose up -d --build" "n"; then
    START_SERVICE="yes"
    (cd "$INSTALL_DIR" && docker compose up -d --build)
  fi
else
  warn "未检测到 Docker Compose。配置已生成，请安装 Docker 后手动启动。"
fi

if [[ "$ENABLE_SSL" == "yes" ]]; then
  ACCESS_URL="https://$ACCESS_HOST"
  [[ "$PANEL_PORT" != "443" ]] && ACCESS_URL="$ACCESS_URL:$PANEL_PORT"
else
  ACCESS_URL="http://$ACCESS_HOST:$PANEL_PORT"
fi
LOGIN_URL="$ACCESS_URL$SECURE_PATH"

print_line
printf 'OU-UI %s 安装配置已完成。\n' "$VERSION"
printf '面板登录链接：%s\n' "$LOGIN_URL"
printf '管理员账号：%s\n' "$ADMIN_USER"
printf '管理员密码：%s\n' "$ADMIN_PASSWORD"
printf 'Agent 注册令牌：%s\n' "$AGENT_JOIN_TOKEN"
printf '安装目录：%s\n' "$INSTALL_DIR"
printf '环境文件：%s/.env\n' "$INSTALL_DIR"
printf '服务已启动：%s\n' "$START_SERVICE"
printf '\n请立即保存上面的登录链接、账号和密码；不要提交 .env、证书私钥或任何真实凭据。\n'
print_line
