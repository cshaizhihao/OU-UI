# OU-UI

OU-UI 是一个面向自托管场景的中文 UI 面板项目。当前版本为 `v0.1.0`，目标是提供清晰、可审计、可重复执行的部署入口，让使用者在不暴露凭据的前提下完成基础安装、端口配置、域名/DNS/SSL 选择和运行环境初始化。

- 版本：`v0.1.0`
- 仓库：<https://github.com/cshaizhihao/OU-UI>
- 默认端口：`3000`
- 默认安全路径：安装时自动生成，例如 `/ds8a9f`
- 部署方式：Docker / Docker Compose

## Research 摘要

v0.1.0 的研究重点是把“能跑起来”和“不要误写敏感信息”放在第一位：

1. 安装流程必须中文交互，所有危险选项需要显式确认。
2. 不在仓库内保存真实 token、API key、证书私钥或生产密码。
3. 首次安装时自动生成随机管理员账号和密码，并只写入目标主机的运行目录。
4. 域名、DNS、SSL 采用分支式 SOP：用户输入域名后强制 HTTPS，并通过 acme.sh + DNS API 自动申请证书。
5. 面板入口必须包含随机安全路径，降低主动探测风险。
6. 路径必须限制在安全目录，避免把运行数据写到系统根目录、用户家目录或仓库源码目录之外的未知位置。

完整记录见 [docs/Research.md](docs/Research.md)。

## 快速开始

> 下面命令不会要求你输入任何 token 或密钥。需要 SSL 自动签发时，请按脚本提示在本机环境中自行配置 DNS 服务商凭据，仓库不会保存这些内容。

```bash
git clone https://github.com/cshaizhihao/OU-UI.git
cd OU-UI
bash scripts/install.sh
```

安装脚本会依次询问：

1. 是否同意安装前声明。
2. 安装目录，默认 `/opt/ou-ui`。
3. Web 端口，默认 `3000`。
4. 是否已完成域名 DNS 解析。
5. 如输入域名，自动调用 acme.sh + Cloudflare DNS 申请证书，并强制 HTTPS。
6. 如没有域名，则降级使用 `HTTP://IP:端口`。
7. 自动生成随机安全路径、管理员账号和管理员密码。
8. 是否生成 Docker Compose 配置并启动服务。

安装完成后，脚本会回显完整面板登录链接、安装目录、管理员账号、管理员密码和 Agent 注册令牌。

SSL 说明：`v0.1.0` 的安装脚本会准备证书目录并给出手动证书/acme.sh 分支提示；公网 HTTPS 建议由反向代理或后续 TLS 终止配置接入。

## 手动 Docker Compose

也可以复制示例环境文件后手动启动：

```bash
cp .env.example .env
docker compose up -d --build
```

请务必修改 `.env` 中的默认值。不要把 `.env`、证书、私钥或任何真实凭据提交到仓库。

## 目录结构

```text
.
├── docs/
│   ├── Research.md
│   └── SOP.md
├── scripts/
│   └── install.sh
├── .env.example
├── docker-compose.yml
├── Dockerfile
├── LICENSE
└── README.md
```

## SOP

部署、升级、回滚、证书处理和安全检查步骤见 [docs/SOP.md](docs/SOP.md)。

## 安全约定

- 不要在 issue、日志、截图或提交中泄露 token、密码、证书私钥。
- `.env`、`data/`、`certs/`、运行日志和备份文件默认被 `.gitignore` 忽略。
- 安装脚本只生成本地随机账号密码，不内置任何真实凭据。
- 若使用 acme.sh，请优先使用环境变量或交互式会话配置 DNS API 凭据，不要写进仓库文件。

## 许可证

本项目使用 AGPL-3.0 License，详见 [LICENSE](LICENSE)。
