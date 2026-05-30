# OU-UI

OU-UI 是一个面向自托管和多机节点运维的分布式代理节点管理面板。当前版本 `v0.6.0` 已在 v0.5.0 runtime deploy lifecycle 基础上，新增 OU-UI 托管 runtime systemd service，让配置写入、服务生成、重启和健康检查形成闭环。

- 版本：`v0.6.0`
- 仓库：<https://github.com/cshaizhihao/OU-UI>
- 默认端口：`3000`
- 默认入口：安装时自动生成随机安全路径，例如 `/ds8a9f`
- License：`AGPL-3.0`

## 已完成

- `v0.1.0`：项目骨架、Go/Gin 后端、React/Vite 前端、Agent 注册/心跳、中文 README/SOP/Research、安装脚本。
- `v0.1.1`：GitHub Actions CI，覆盖 Go、前端、Docker、Shell 和密钥扫描。
- `v0.1.2`：安装脚本加固，包含系统检测、Docker Compose 检测、端口占用检测、acme.sh 路径、证书校验和 Agent 安装入口。
- `v0.2.0`：Agent 任务闭环，支持 `noop`、`runtime.status`、`node.deploy`，任务拉取、执行、结果回传。
- `v0.2.0`：Xray Provider 预览，支持 `VLESS`、`VLESS Reality`、`VMess`、`Trojan`、`Shadowsocks` 配置渲染。
- `v0.3.0`：Hysteria2 Provider 预览，支持监听端口、TLS、password auth、bandwidth、masquerade 和流量限制字段预留。
- `v0.3.0`：前端升级为 OU-UI 代理节点控制台，包含 Agent 监控、上下行、累计流量/限额、节点下发和任务队列。
- `v0.4.0`：Agent 通过 `installId` 持久绑定身份，重启后复用本地 `agent-state.json`，避免重复注册。
- `v0.4.0`：任务执行加入租约、超时回收、最大尝试次数、失败原因和 CAS 状态更新，避免卡死或重复覆盖终态。
- `v0.4.0`：服务端按 `LastSeenAt` 推导在线/降级/离线状态，下发前校验 Agent 在线状态和 runtime capability。
- `v0.4.0`：前端展示注册状态、认证状态、最后心跳、任务状态、失败原因、重试次数和 runtime capabilities。
- `v0.5.0`：新增 runtime deploy lifecycle，节点下发按 `render -> install -> apply -> reload -> health` 执行，失败时尝试 rollback。
- `v0.5.0`：Xray/Hysteria2 Provider 支持二进制探测、active config 写入、systemd reload/restart、服务活跃度检查和备份恢复；Xray 额外执行 TCP 端口健康检查。
- `v0.5.0`：Node 模型记录 `runtimeVersion`、`serviceName`、`serviceStatus`、`configPath`、`lastError` 和 `lastDeployedAt`。
- `v0.5.0`：前端展示部署阶段、runtime version、service status、config path、rollback available 和失败阶段。
- `v0.6.0`：Agent 会为 Xray/Hysteria2 生成 OU-UI 托管的每节点 systemd unit，并用 `systemctl enable --now/restart` 让 runtime 直接加载 active config。
- `v0.6.0`：部署结果新增 `configDir`、`unitPath`、`serviceMode` 和 `managedByOuui`，Node 状态同步这些托管服务字段。
- `v0.6.0`：runtime 配置与 unit 写入改为临时文件 + rename，rollback 会检查并回报 systemd 停止、重载、重启错误。
- `v0.6.0`：前端升级为 Service Control 视图，展示 managed/external、unit path、config dir、reload/restart/health 信息。

## V3.0.0 Update Log

- Added the V3 control-plane data model foundation: per-node traffic samples, visual routing rules, HA load-balancer groups, alert webhooks, external subscriptions, Clash profiles, tenants, panel users, API keys, and Copilot incidents.
- Added `host.optimize` tasks for Agent-side BBR/sysctl network tuning and exposed one-click network optimization APIs from the panel.
- Added Agent-side per-generated-node traffic reporting and panel APIs for latest node traffic plus historical samples.
- Added visual routing APIs and Xray routing export for GeoIP, GeoSite, ads, domain, protocol, direct, proxy, and block rules.
- Added HA group decision logic based on latency, packet loss, and weight, with persisted health decisions.
- Added Telegram, ServerChan, and generic webhook alert delivery for CPU, traffic quota, node connection, and Agent error events.
- Added external V2Ray/Clash-style subscription import, mixed external/self-built node management, and generated Clash YAML hosting.
- Added RBAC groundwork for tenants, sub-users, node access lists, monthly traffic quota, connection limits, and scoped API keys.
- Added built-in operations Copilot APIs with a local diagnosis fallback and optional OpenAI-compatible `/chat/completions` integration.
- Added pnpm lockfile and workspace build approval for deterministic frontend installs.
- Added background maintenance sweeps for task lease recovery, offline Agent alerting, tenant traffic/connection quota checks, and HA decision refresh.
- Added scoped API-key enforcement, `routing.apply` task dispatch, Agent-side routing payload persistence, and tests for API scopes, subscription parsing, and Clash YAML generation.
- Added a V3 SaaS console surface for routing, host tuning, HA groups, alert webhooks, external subscriptions, Clash profiles, tenants, users, API keys, Copilot incidents, and recent tasks.
- Updated the Vite build base to relative assets so the panel works under `/ou-ui/` or a future randomized secure path.
- Remote DNS/HTTPS deployment: created `ou-ui-1879d0cb.zze.cc -> 154.217.255.192`, issued a Let's Encrypt certificate with Cloudflare DNS-01, and configured nginx TLS.
- Remote smoke test: `https://ou-ui-1879d0cb.zze.cc/healthz`, `https://ou-ui-1879d0cb.zze.cc/ou-fe6a2bcd05/`, SPA assets, and login API returned HTTP 200 on the OU-UI test server.

## 快速开始

```bash
git clone https://github.com/cshaizhihao/OU-UI.git
cd OU-UI
bash scripts/install.sh
```

需要自动签发证书时，只在当前安全 shell 中导出 DNS API 变量：

```bash
export CF_Token="your-cloudflare-token"
export ACME_EMAIL="admin@example.com"
bash scripts/install.sh
```

不要把真实 token、`.env`、证书私钥或运行日志提交到仓库。

## Agent 安装

主控面板提供 Agent 一键安装脚本接口：

```text
/安全路径/api/v1/agents/install-script
```

也可以在被控机上使用仓库脚本：

```bash
bash scripts/install-agent.sh "https://panel.example.com/安全路径" "Agent注册Token"
```

v0.6.0 的 Agent 会把 `installId`、`agentId` 和 `agentToken` 保存到数据目录的 `agent-state.json`，后续重启优先复用身份；如服务端拒绝旧 token，且环境中仍有注册 token，会自动尝试重新绑定。

## Runtime Deploy

`node.deploy` 现在不再只写入渲染文件。Agent 会在被控机本地执行分阶段部署，并生成 OU-UI 托管的 runtime systemd service：

```text
render -> install -> apply -> reload -> health
```

- `install`：检查目标 runtime 二进制和 `systemctl` 是否存在，例如 `xray`、`hysteria`。
- `apply`：原子写入 active config，生成 managed unit，并在已有配置时生成备份。
- `reload`：执行 `systemctl enable --now` 和 `systemctl restart`，让 runtime 直接加载 OU-UI 写入的配置。
- `health`：记录 runtime version、service status；systemd 返回非 `active` 时判定失败，Xray 会额外对监听端口做 TCP 健康检查。
- `rollback`：apply/reload/health 失败时，如存在备份则恢复上一版配置；首次部署失败且没有备份时会移除新写入的配置。

当前 v0.6.0 不会自动下载和安装 Xray/Hysteria2 二进制；被控机仍需预先准备 `xray` 或 `hysteria`/`hysteria2` 命令。OU-UI 会生成每节点托管服务，例如 `ou-ui-xray-nod_xxx.service`、`ou-ui-hysteria2-nod_xxx.service`，服务文件带有基础 systemd hardening，并记录在任务结果和 Node 状态中。

## 协议预览

- Xray：`VLESS`、`VLESS Reality`、`VMess`、`Trojan`、`Shadowsocks`
- Hysteria2：password auth、TLS cert/key、bandwidth、masquerade
- Sing-Box：已保留 Runtime 枚举，后续版本补入 Provider

详细说明：

- [Xray Provider](docs/protocols/xray.md)
- [Hysteria2 Provider](docs/protocols/hysteria2.md)

## 手动 Docker Compose

```bash
cp .env.example .env
docker compose up -d --build
```

请先修改 `.env` 中的示例值，尤其是管理员密码、JWT Secret、Agent 注册令牌和安全路径。

## CI

GitHub Actions 工作流位于 `.github/workflows/ci.yml`，覆盖：

- Go test/build
- 前端 typecheck/build
- Docker web/server target build
- `bash -n` 与 ShellCheck
- Gitleaks secret scan

## 文档

- [Research 摘要](docs/Research.md)
- [SOP](docs/SOP.md)

## 安全约定

- 不要在 issue、日志、截图或提交中泄露 token、密码、证书私钥。
- `.env`、`data/`、`certs/`、运行日志和备份文件默认被 `.gitignore` 忽略。
- Agent 注册令牌只应在受控节点上使用。
- 面板 API 会对任务、节点 spec 中常见敏感字段做脱敏返回；后续版本会补充专用 secret store。

## 许可证

本项目使用 AGPL-3.0 License，详见 [LICENSE](LICENSE)。
