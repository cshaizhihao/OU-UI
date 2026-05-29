# Research 摘要

版本：`v0.3.0`

仓库：<https://github.com/cshaizhihao/OU-UI>

## 背景

v0.3.0 的目标是在 DevOps 加固基础上补齐 Agent 任务闭环、Xray Provider 预览、Hysteria2 Provider 预览和前端节点下发界面。

## CI 设计结论

GitHub Actions 拆成独立 job：

- Go：`go test ./...`，并分别构建 server 与 agent。
- Frontend：使用 pnpm 安装依赖，执行 typecheck 与 build。
- Shell：对 `scripts/` 与 `deploy/` 下的 shell 脚本执行 `bash -n` 和 ShellCheck。
- Docker：通过 Buildx 构建 `web` 与 `server` target，不推送镜像。
- Secret scan：使用 Gitleaks 扫描仓库，避免误提交 token、私钥、密码等敏感内容。

CI 不依赖真实业务 token，也不会把部署凭据写入仓库。

## 安装脚本加固结论

`scripts/install.sh` 增强点：

1. `trap ERR` 输出失败行号和常见排查方向。
2. 检测 Linux 发行版、基础命令、Docker daemon 和 Docker Compose v2。
3. 检查安装目录是否为安全绝对路径，拒绝 `/`、`/opt`、`/root` 等危险目标。
4. 检查端口范围和监听占用，优先使用 `ss`，回退到 `lsof` 或 `netstat`。
5. acme.sh 查找真实可执行路径：`PATH`、`$HOME/.acme.sh/acme.sh`，未找到才安装。
6. HTTPS 分支校验 `fullchain.pem` 与 `privkey.pem` 是否存在、非空，并在有 openssl 时解析证书与私钥。
7. 安装目录生成 `docs/agent-install.md`，作为节点安装 Agent 的入口。

## 安全边界

- 仓库只保存示例值，不保存真实 token、密钥、证书私钥或生产密码。
- `CF_Token`、`ACME_EMAIL` 仅从当前 shell 环境读取，不写入 `.env`。
- 生成的 `.env` 默认 `600` 权限。
- Agent 注册令牌只在安装完成回显和安装目录文档中出现，由目标机器本地保存。

## 后续建议

1. 发布二进制产物后，将 Agent 安装入口从本地构建命令升级为 release 下载命令。
2. 增加镜像推送工作流，并通过环境保护规则控制发布。
3. 为 `deploy/nginx/entrypoint.sh` 增加更多容器启动测试。
