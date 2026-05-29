# OU-UI

OU-UI 是一个面向自托管和多机节点运维的分布式代理节点管理面板。当前版本 `v0.3.0` 已完成主控面板骨架、Agent 注册与任务闭环、Xray/Hysteria2 Provider 预览、中文安装脚本、CI 和节点下发控制台。

- 版本：`v0.3.0`
- 仓库：<https://github.com/cshaizhihao/OU-UI>
- 默认端口：`3000`
- 默认入口：安装时自动生成随机安全路径，例如 `/ds8a9f`
- License：`AGPL-3.0`

## 已完成

- `v0.1.0`：项目骨架、Go/Gin 后端、React/Vite 前端、Agent 注册/心跳、中文 README/SOP/Research、安装脚本。
- `v0.1.1`：GitHub Actions CI，覆盖 Go、前端、Docker、Shell 和密钥扫描。
- `v0.1.2`：安装脚本加固，包含系统检测、Docker Compose 检测、端口占用检测、acme.sh 路径、证书校验和 Agent 安装入口。
- `v0.2.0`：Agent 任务闭环，支持 `noop`、`runtime.status`、`node.deploy`，任务拉取、执行、结果回传。
- `v0.2.0`：Xray Provider 预览，支持 `VLESS`、`VMess`、`Trojan`、`Shadowsocks` 和 `VLESS Reality` 配置渲染。
- `v0.3.0`：Hysteria2 Provider 预览，支持监听端口、TLS、password auth、bandwidth、masquerade 和流量限制字段预留。
- `v0.3.0`：前端升级为 OU-UI 代理节点控制台，包含 Agent 监控、上下行、累计流量/限额、节点下发和任务队列。

## 快速开始

```bash
git clone https://github.com/cshaizhihao/OU-UI.git
cd OU-UI
bash scripts/install.sh
```

安装脚本会检测系统、Docker Compose、端口占用、安装路径和证书文件；输入域名后会强制启用 HTTPS，并可通过 `acme.sh + Cloudflare DNS` 签发证书。

需要自动签发证书时，只在当前 shell 中导出 DNS API 变量：

```bash
export CF_Token="your-cloudflare-token"
export ACME_EMAIL="admin@example.com"
bash scripts/install.sh
```

不要把真实 token、`.env`、证书私钥或运行日志提交到仓库。

## Agent 安装

运行 `scripts/install.sh` 后，安装目录会生成：

```text
/opt/ou-ui/docs/agent-install.md
```

主控面板也提供 Agent 一键安装脚本接口：

```text
/安全路径/api/v1/agents/install-script
```

Agent 注册后会定期心跳、采集基础系统指标、拉取任务、渲染节点配置并回传执行结果。

## 协议预览

- Xray：`VLESS`、`VLESS Reality`、`VMess`、`Trojan`、`Shadowsocks`
- Hysteria2：password auth、TLS cert/key、bandwidth、masquerade
- Sing-Box：已保留 Runtime 枚举，后续版本补充 Provider

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

## 许可证

本项目使用 AGPL-3.0 License，详见 [LICENSE](LICENSE)。
