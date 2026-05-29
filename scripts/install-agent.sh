#!/usr/bin/env bash
set -Eeuo pipefail

VERSION="v0.4.0"
SERVICE_NAME="ou-ui-agent"
ENV_DIR="/etc/ou-ui"
ENV_FILE="$ENV_DIR/agent.env"
SERVICE_FILE="/etc/systemd/system/$SERVICE_NAME.service"

DEFAULT_INSTALL_DIR="/opt/ou-ui-agent"
DEFAULT_DATA_DIR="/var/lib/ou-ui-agent"
DEFAULT_DOCKER_IMAGE="ou-ui-agent:0.4.0"
DEFAULT_CONTAINER_NAME="ou-ui-agent"

DEFAULT_HOSTNAME="$(hostname 2>/dev/null || true)"
DEFAULT_HOSTNAME="${DEFAULT_HOSTNAME:-ou-ui-agent}"

PANEL_URL="${1:-${OUUI_SERVER_URL:-}}"
JOIN_TOKEN="${2:-${OUUI_AGENT_JOIN_TOKEN:-}}"
AGENT_NAME="${3:-${OUUI_AGENT_NAME:-$DEFAULT_HOSTNAME}}"
INSTALL_DIR="${OUUI_AGENT_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"
DATA_DIR="${OUUI_AGENT_DATA_DIR:-$DEFAULT_DATA_DIR}"
RUN_MODE="${OUUI_AGENT_RUN_MODE:-binary}"
BINARY_PATH="${OUUI_AGENT_BINARY_PATH:-}"
AGENT_EXTRA_ARGS="${OUUI_AGENT_EXTRA_ARGS:-}"
DOCKER_IMAGE="${OUUI_AGENT_DOCKER_IMAGE:-$DEFAULT_DOCKER_IMAGE}"
DOCKER_ARGS="${OUUI_AGENT_DOCKER_ARGS:-}"
CONTAINER_NAME="${OUUI_AGENT_CONTAINER_NAME:-$DEFAULT_CONTAINER_NAME}"
LOCAL_COMMAND="${OUUI_AGENT_LOCAL_COMMAND:-}"
RUNNER_FILE=""

print_line() {
  printf '%s\n' "------------------------------------------------------------"
}

warn() {
  printf '[提醒] %s\n' "$1"
}

fail() {
  printf '[错误] %s\n' "$1" >&2
  exit 1
}

on_error() {
  local exit_code
  local line_no
  exit_code=$?
  line_no=${1:-unknown}
  printf '\n[错误] Agent 安装脚本在第 %s 行失败，退出码 %s。\n' "$line_no" "$exit_code" >&2
  printf '[排查] 可重新执行：bash scripts/install-agent.sh，或查看 systemd 日志：journalctl -u %s -n 100 --no-pager。\n' "$SERVICE_NAME" >&2
  exit "$exit_code"
}
trap 'on_error $LINENO' ERR

is_interactive() {
  [[ -t 0 ]]
}

require_command() {
  local command_name="$1"
  local hint="$2"
  if ! command -v "$command_name" >/dev/null 2>&1; then
    fail "缺少命令：$command_name。$hint"
  fi
}

ask() {
  local prompt="$1"
  local default_value="${2:-}"
  local answer

  if ! is_interactive; then
    printf '%s' "$default_value"
    return 0
  fi

  if [[ -n "$default_value" ]]; then
    printf '%s [%s]: ' "$prompt" "$default_value" >&2
  else
    printf '%s: ' "$prompt" >&2
  fi

  IFS= read -r answer
  printf '%s' "${answer:-$default_value}"
}

ask_secret() {
  local prompt="$1"
  local default_value="${2:-}"
  local answer

  if [[ -n "$default_value" ]]; then
    printf '%s' "$default_value"
    return 0
  fi

  if ! is_interactive; then
    printf '%s' "$default_value"
    return 0
  fi

  printf '%s: ' "$prompt" >&2
  IFS= read -r -s answer
  printf '\n' >&2
  printf '%s' "$answer"
}

ask_yes_no() {
  local prompt="$1"
  local default_value="${2:-n}"
  local answer

  while true; do
    if is_interactive; then
      printf '%s (y/n) [%s]: ' "$prompt" "$default_value" >&2
      IFS= read -r answer
      answer="${answer:-$default_value}"
    else
      answer="$default_value"
    fi

    case "$answer" in
      y|Y|yes|YES|Yes) return 0 ;;
      n|N|no|NO|No) return 1 ;;
      *) warn "请输入 y 或 n。" ;;
    esac
  done
}

normalize_path() {
  local raw_path="$1"
  raw_path="${raw_path/#\~/${HOME:-/root}}"
  printf '%s' "$raw_path"
}

validate_no_newline() {
  local label="$1"
  local value="$2"
  [[ "$value" != *$'\n'* && "$value" != *$'\r'* ]] || fail "$label 不能包含换行。"
}

validate_required() {
  local label="$1"
  local value="$2"
  validate_no_newline "$label" "$value"
  [[ -n "$value" ]] || fail "$label 不能为空。"
}

validate_no_whitespace() {
  local label="$1"
  local value="$2"
  [[ "$value" != *[[:space:]]* ]] || fail "$label 不能包含空格或制表符。"
}

validate_url() {
  local value="$1"
  validate_required "主控面板地址" "$value"
  validate_no_whitespace "主控面板地址" "$value"
  [[ "$value" =~ ^https?:// ]] || fail "主控面板地址必须以 http:// 或 https:// 开头。"
}

validate_absolute_path() {
  local label="$1"
  local value="$2"
  validate_required "$label" "$value"
  validate_no_whitespace "$label" "$value"
  [[ "$value" = /* ]] || fail "$label 必须是绝对路径。"
}

normalize_run_mode() {
  local value="$1"
  case "$value" in
    1|bin|binary|二进制) printf 'binary' ;;
    2|docker|Docker|容器) printf 'docker' ;;
    3|local|dev|本地|开发) printf 'local' ;;
    *) return 1 ;;
  esac
}

ask_run_mode() {
  local current="$1"
  local answer
  local normalized

  while true; do
    if is_interactive; then
      print_line >&2
      printf '请选择 Agent 运行方式：\n' >&2
      printf '  1) 二进制：预留或使用 Agent 二进制路径\n' >&2
      printf '  2) Docker：通过 docker run 启动 Agent 镜像\n' >&2
      printf '  3) 本地命令：用于源码构建或开发环境\n' >&2
    fi

    answer="$(ask "运行方式" "$current")"
    if normalized="$(normalize_run_mode "$answer")"; then
      printf '%s' "$normalized"
      return 0
    fi

    if ! is_interactive; then
      fail "运行方式无效：$answer。可选值：binary、docker、local。"
    fi
    warn "运行方式无效，请输入 1/2/3 或 binary/docker/local。"
  done
}

quote_env_value() {
  local value="$1"
  validate_no_newline "环境变量值" "$value"
  local backtick
  backtick='`'
  value="${value//\\/\\\\}"
  value="${value//\"/\\\"}"
  value="${value//\$/\\\$}"
  value="${value//$backtick/\\$backtick}"
  printf '"%s"' "$value"
}

write_env_kv() {
  local key="$1"
  local value="$2"
  printf '%s=%s\n' "$key" "$(quote_env_value "$value")" >>"$ENV_FILE"
}

write_env_file() {
  local old_umask
  old_umask="$(umask)"
  umask 077
  : >"$ENV_FILE"
  write_env_kv "OUUI_VERSION" "$VERSION"
  write_env_kv "OUUI_SERVER_URL" "$PANEL_URL"
  write_env_kv "OUUI_AGENT_JOIN_TOKEN" "$JOIN_TOKEN"
  write_env_kv "OUUI_AGENT_NAME" "$AGENT_NAME"
  write_env_kv "OUUI_AGENT_RUN_MODE" "$RUN_MODE"
  write_env_kv "OUUI_AGENT_INSTALL_DIR" "$INSTALL_DIR"
  write_env_kv "OUUI_AGENT_DATA_DIR" "$DATA_DIR"
  write_env_kv "OUUI_AGENT_BINARY_PATH" "$BINARY_PATH"
  write_env_kv "OUUI_AGENT_EXTRA_ARGS" "$AGENT_EXTRA_ARGS"
  write_env_kv "OUUI_AGENT_DOCKER_IMAGE" "$DOCKER_IMAGE"
  write_env_kv "OUUI_AGENT_DOCKER_ARGS" "$DOCKER_ARGS"
  write_env_kv "OUUI_AGENT_CONTAINER_NAME" "$CONTAINER_NAME"
  write_env_kv "OUUI_AGENT_LOCAL_COMMAND" "$LOCAL_COMMAND"
  chmod 600 "$ENV_FILE"
  umask "$old_umask"
}

write_runner_file() {
  cat >"$RUNNER_FILE" <<'EOF_RUNNER'
#!/usr/bin/env bash
set -Eeuo pipefail

ENV_FILE="/etc/ou-ui/agent.env"

if [[ -r "$ENV_FILE" ]]; then
  set -a
  # shellcheck disable=SC1090
  source "$ENV_FILE"
  set +a
else
  printf '[错误] 未找到 Agent 环境文件：%s\n' "$ENV_FILE" >&2
  exit 1
fi

case "${OUUI_AGENT_RUN_MODE:-binary}" in
  binary)
    binary_path="${OUUI_AGENT_BINARY_PATH:-/opt/ou-ui-agent/ou-ui-agent}"
    if [[ ! -x "$binary_path" ]]; then
      printf '[错误] Agent 二进制不可执行：%s\n' "$binary_path" >&2
      printf '[提示] 请先下载或构建 Agent，并放到该路径后再启动服务。\n' >&2
      exit 127
    fi
    # shellcheck disable=SC2086
    exec "$binary_path" ${OUUI_AGENT_EXTRA_ARGS:-}
    ;;
  docker)
    image="${OUUI_AGENT_DOCKER_IMAGE:-ou-ui-agent:0.4.0}"
    container_name="${OUUI_AGENT_CONTAINER_NAME:-ou-ui-agent}"
    data_dir="${OUUI_AGENT_DATA_DIR:-/var/lib/ou-ui-agent}"
    if ! command -v docker >/dev/null 2>&1; then
      printf '[错误] 未找到 docker 命令，无法使用 Docker 模式。\n' >&2
      exit 127
    fi
    docker rm -f "$container_name" >/dev/null 2>&1 || true
    # shellcheck disable=SC2086
    exec docker run --rm --name "$container_name" \
      --env OUUI_VERSION \
      --env OUUI_SERVER_URL \
      --env OUUI_AGENT_JOIN_TOKEN \
      --env OUUI_AGENT_NAME \
      --env OUUI_AGENT_RUN_MODE \
      --env OUUI_AGENT_INSTALL_DIR \
      --env OUUI_AGENT_DATA_DIR \
      --env OUUI_AGENT_BINARY_PATH \
      --env OUUI_AGENT_EXTRA_ARGS \
      --env OUUI_AGENT_DOCKER_IMAGE \
      --env OUUI_AGENT_DOCKER_ARGS \
      --env OUUI_AGENT_CONTAINER_NAME \
      --env OUUI_AGENT_LOCAL_COMMAND \
      --volume "$data_dir:$data_dir" \
      ${OUUI_AGENT_DOCKER_ARGS:-} \
      "$image"
    ;;
  local)
    local_command="${OUUI_AGENT_LOCAL_COMMAND:-}"
    if [[ -z "$local_command" ]]; then
      printf '[错误] 本地模式缺少 OUUI_AGENT_LOCAL_COMMAND。\n' >&2
      exit 1
    fi
    exec bash -lc "$local_command"
    ;;
  *)
    printf '[错误] 未知 Agent 运行方式：%s\n' "${OUUI_AGENT_RUN_MODE:-}" >&2
    exit 1
    ;;
esac
EOF_RUNNER
  chmod 750 "$RUNNER_FILE"
}

write_service_file() {
  cat >"$SERVICE_FILE" <<EOF_SERVICE
[Unit]
Description=OU-UI Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
EnvironmentFile=$ENV_FILE
ExecStart=$RUNNER_FILE
Restart=always
RestartSec=5
WorkingDirectory=$DATA_DIR

[Install]
WantedBy=multi-user.target
EOF_SERVICE
  chmod 644 "$SERVICE_FILE"
}

collect_inputs() {
  PANEL_URL="$(ask "请输入主控面板地址，例如 https://panel.example.com/secure-path" "$PANEL_URL")"
  validate_url "$PANEL_URL"

  JOIN_TOKEN="$(ask_secret "请输入 Agent 注册 token（输入时不会回显）" "$JOIN_TOKEN")"
  validate_required "Agent 注册 token" "$JOIN_TOKEN"
  validate_no_whitespace "Agent 注册 token" "$JOIN_TOKEN"

  AGENT_NAME="$(ask "请输入 Agent 名称" "$AGENT_NAME")"
  validate_required "Agent 名称" "$AGENT_NAME"

  INSTALL_DIR="$(normalize_path "$(ask "请输入 Agent 安装目录" "$INSTALL_DIR")")"
  validate_absolute_path "Agent 安装目录" "$INSTALL_DIR"

  DATA_DIR="$(normalize_path "$(ask "请输入 Agent 数据目录" "$DATA_DIR")")"
  validate_absolute_path "Agent 数据目录" "$DATA_DIR"

  RUN_MODE="$(ask_run_mode "$RUN_MODE")"

  case "$RUN_MODE" in
    binary)
      BINARY_PATH="${BINARY_PATH:-$INSTALL_DIR/ou-ui-agent}"
      BINARY_PATH="$(normalize_path "$(ask "请输入 Agent 二进制路径（可先占位）" "$BINARY_PATH")")"
      validate_absolute_path "Agent 二进制路径" "$BINARY_PATH"
      AGENT_EXTRA_ARGS="$(ask "请输入二进制附加参数（可留空）" "$AGENT_EXTRA_ARGS")"
      validate_no_newline "二进制附加参数" "$AGENT_EXTRA_ARGS"
      ;;
    docker)
      DOCKER_IMAGE="$(ask "请输入 Agent Docker 镜像" "$DOCKER_IMAGE")"
      validate_required "Agent Docker 镜像" "$DOCKER_IMAGE"
      validate_no_whitespace "Agent Docker 镜像" "$DOCKER_IMAGE"
      CONTAINER_NAME="$(ask "请输入 Docker 容器名" "$CONTAINER_NAME")"
      validate_required "Docker 容器名" "$CONTAINER_NAME"
      validate_no_whitespace "Docker 容器名" "$CONTAINER_NAME"
      DOCKER_ARGS="$(ask "请输入 docker run 附加参数（可留空）" "$DOCKER_ARGS")"
      validate_no_newline "docker run 附加参数" "$DOCKER_ARGS"
      BINARY_PATH="${BINARY_PATH:-$INSTALL_DIR/ou-ui-agent}"
      ;;
    local)
      LOCAL_COMMAND="$(ask "请输入本地运行命令，例如 cd /srv/OU-UI && ./ou-ui-agent" "$LOCAL_COMMAND")"
      validate_required "本地运行命令" "$LOCAL_COMMAND"
      BINARY_PATH="${BINARY_PATH:-$INSTALL_DIR/ou-ui-agent}"
      ;;
    *)
      fail "未知运行方式：$RUN_MODE"
      ;;
  esac
}

print_summary() {
  print_line
  printf '安装确认：\n'
  printf '版本：%s\n' "$VERSION"
  printf '主控面板地址：%s\n' "$PANEL_URL"
  printf 'Agent 名称：%s\n' "$AGENT_NAME"
  printf '运行方式：%s\n' "$RUN_MODE"
  printf '安装目录：%s\n' "$INSTALL_DIR"
  printf '数据目录：%s\n' "$DATA_DIR"
  printf '环境文件：%s\n' "$ENV_FILE"
  printf 'systemd 服务：%s\n' "$SERVICE_FILE"

  case "$RUN_MODE" in
    binary)
      printf 'Agent 二进制路径：%s\n' "$BINARY_PATH"
      ;;
    docker)
      printf 'Docker 镜像：%s\n' "$DOCKER_IMAGE"
      printf 'Docker 容器名：%s\n' "$CONTAINER_NAME"
      ;;
    local)
      printf '本地运行命令：%s\n' "$LOCAL_COMMAND"
      ;;
  esac

  printf '注册 token：已输入，安装完成后不会回显明文。\n'
  print_line
}

print_next_steps() {
  print_line
  printf 'OU-UI Agent %s 安装配置已完成。\n' "$VERSION"
  printf '环境文件：%s\n' "$ENV_FILE"
  printf '运行脚本：%s\n' "$RUNNER_FILE"
  printf '服务文件：%s\n' "$SERVICE_FILE"
  printf '运行方式：%s\n' "$RUN_MODE"
  printf 'Agent 名称：%s\n' "$AGENT_NAME"
  printf '主控面板地址：%s\n' "$PANEL_URL"
  printf '\n常用命令：\n'
  printf '启动：systemctl enable --now %s\n' "$SERVICE_NAME"
  printf '状态：systemctl status %s --no-pager\n' "$SERVICE_NAME"
  printf '日志：journalctl -u %s -f\n' "$SERVICE_NAME"
  printf '重载配置：systemctl daemon-reload && systemctl restart %s\n' "$SERVICE_NAME"

  if [[ "$RUN_MODE" == "binary" && ! -x "$BINARY_PATH" ]]; then
    printf '\n'
    warn "尚未找到可执行 Agent 二进制：$BINARY_PATH"
    warn "请先将 v0.4.0 Agent 二进制放到该路径并 chmod +x，再执行启动命令。"
  fi

  if [[ "$RUN_MODE" == "docker" ]] && ! command -v docker >/dev/null 2>&1; then
    printf '\n'
    warn "当前未检测到 docker 命令；Docker 模式启动前请先安装并启动 Docker。"
  fi

  printf '\n安全提醒：不要把 %s、注册 token 或任何真实凭据提交到仓库、工单、截图或日志。\n' "$ENV_FILE"
  print_line
}

main() {
  print_line
  printf 'OU-UI Agent %s 中文交互式安装脚本\n' "$VERSION"
  print_line
  printf '本脚本会写入 %s，创建运行脚本，并生成 systemd 服务 %s。\n' "$ENV_FILE" "$SERVICE_NAME"
  printf '注册 token 仅写入本机环境文件，不会在安装完成回显明文。\n'
  print_line

  [[ "$(id -u)" -eq 0 ]] || fail "请使用 root 运行 Agent 安装脚本。"
  require_command systemctl "Agent 服务需要 systemd 管理。"
  require_command mkdir "系统缺少基础命令 mkdir。"
  require_command chmod "系统缺少基础命令 chmod。"

  collect_inputs
  RUNNER_FILE="$INSTALL_DIR/run-agent.sh"
  print_summary

  if ! ask_yes_no "是否写入配置并创建 systemd 服务" "y"; then
    fail "用户取消，未写入配置。"
  fi

  mkdir -p "$ENV_DIR" "$INSTALL_DIR" "$DATA_DIR"
  chmod 755 "$ENV_DIR"
  chmod 750 "$INSTALL_DIR"
  chmod 700 "$DATA_DIR"

  write_env_file
  write_runner_file
  write_service_file
  systemctl daemon-reload

  print_next_steps
}

main "$@"
