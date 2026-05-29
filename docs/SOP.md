# OU-UI SOP

版本：`v0.5.0`

仓库：<https://github.com/cshaizhihao/OU-UI>

## 1. 安装前检查

1. 目标服务器建议使用 Linux，并确认当前用户可以创建安装目录，例如 `/opt/ou-ui`。
2. 安装脚本会检测 Docker daemon 与 Docker Compose v2；如只生成配置，可暂不启动 Docker。
3. 默认 Web 端口为 `3000`，脚本会用 `ss`、`lsof` 或 `netstat` 检测端口占用。
4. 如需域名访问，先确认 DNS A/AAAA 记录已指向当前服务器。
5. 如需自动签发 SSL，只在当前 shell 导出 DNS API 环境变量，例如 `CF_Token`；不要把真实 token 写入仓库、文档、日志或截图。

## 2. 标准安装

```bash
bash scripts/install.sh
```

脚本会依次完成：

1. 显示安装声明并要求确认。
2. 检测系统、基础命令、安装目录和端口占用。
3. 生成随机安全路径、管理员账号、管理员密码、JWT Secret 和 Agent 注册令牌。
4. 按域名选择 HTTP 或 HTTPS 分支；输入域名后强制启用 HTTPS。
5. 可选调用 `acme.sh + Cloudflare DNS` 签发证书，并校验 `fullchain.pem` 与 `privkey.pem`。
6. 写入 `.env`、`docker-compose.yml` 和 `docs/agent-install.md`。
7. 可选执行 `docker compose up -d --build`。

安装完成后，立即保存脚本回显的登录链接、管理员账号和密码。不要提交生成的 `.env`、证书私钥或运行日志。

## 3. Agent 接入

面板可生成 Agent 一键安装脚本：

```text
/安全路径/api/v1/agents/install-script
```

也可在被控机运行：

```bash
bash scripts/install-agent.sh "https://panel.example.com/安全路径" "Agent注册Token"
```

v0.5.0 的 Agent 安装脚本会写入 `/etc/ou-ui/agent.env`，创建 systemd 服务，并把运行状态、日志命令回显给用户。Agent 进程会把本机身份保存到数据目录，避免重启后重复注册。

## 4. 任务控制链路

v0.4.0 起，Server 和 Agent 对任务执行采用以下约定：

1. Server 下发任务前校验 Agent 是否在线、是否支持 `task-polling` 和目标 runtime capability。
2. Agent 拉取任务时，Server 将任务从 `queued` CAS 更新为 `running`，并写入 `leaseExpiresAt`。
3. Agent 回传结果时必须带当前 `attempt`，Server 只接受仍处于 `running` 且 attempt 匹配的更新。
4. 超过租约的任务会被重新排队；超过最大尝试次数后标记为 `failed`。
5. 终态任务不可被旧 Agent 回包覆盖。

## 5. Runtime Deploy

v0.5.0 起，`node.deploy` 任务在 Agent 本机执行以下阶段：

1. `render`：由 Provider 渲染 Xray JSON 或 Hysteria2 YAML。
2. `install`：检查目标 runtime 二进制是否存在，不存在则失败。
3. `apply`：写入 active config，并在旧配置存在时生成备份。
4. `reload`：通过 systemd reload/restart 目标 runtime 服务。
5. `health`：采集 runtime version、service status；systemd 返回非 `active` 时判定失败，Xray 额外执行 TCP 端口健康检查。
6. `rollback`：apply/reload/health 失败时尽量恢复上一版配置；首次部署无备份时移除新写入的配置。

当前 v0.5.0 不会自动安装 Xray/Hysteria2 二进制，也不会自动接管 runtime 的 systemd unit。目标机器需要先准备 runtime 和 systemd 服务，并确保该服务配置会加载 OU-UI 写入的数据目录；后续版本会补齐自动安装器和 unit 生成器。

## 6. CI

GitHub Actions 工作流位于 `.github/workflows/ci.yml`，覆盖：

- `go test ./...`
- 构建 `apps/server` 与 `apps/agent`
- `pnpm typecheck` 与 `pnpm build`
- `bash -n` 与 ShellCheck
- Docker `web`、`server` target build
- Gitleaks secret scan

CI 不需要真实业务 token。不要新增项目级真实凭据。

## 7. 升级

1. 备份安装目录中的 `.env` 与 `data/`。
2. 拉取新版本代码。
3. 对比 `.env.example` 是否新增变量。
4. 在安装目录执行：

```bash
docker compose up -d --build
docker compose ps
```

5. 验证 Web 页面、Agent 注册、日志和健康状态。
