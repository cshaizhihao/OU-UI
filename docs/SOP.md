# OU-UI SOP

版本：`v0.3.0`

仓库：<https://github.com/cshaizhihao/OU-UI>

## 1. 安装前检查

1. 目标服务器建议使用 Linux，并确认当前用户可以创建安装目录，例如 `/opt/ou-ui`。
2. 安装脚本会检测 Docker daemon 与 Docker Compose v2；如果只生成配置，可暂不启动 Docker。
3. 默认 Web 端口为 `3000`，脚本会用 `ss`、`lsof` 或 `netstat` 检测端口占用。
4. 如需域名访问，先确认 DNS A/AAAA 记录已指向当前服务器。
5. 如需自动签发 SSL，先在当前 shell 中导出 DNS API 环境变量，例如 `CF_Token`；不要把真实 token 写入仓库、文档、日志或截图。

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

## 3. SSL 与 acme.sh

自动签发使用当前 shell 中的环境变量：

```bash
export CF_Token="your-cloudflare-token"
export ACME_EMAIL="admin@example.com"
bash scripts/install.sh
```

脚本查找 acme.sh 的真实路径顺序：

1. `PATH` 中的 `acme.sh`
2. `$HOME/.acme.sh/acme.sh`
3. 未找到时通过 `https://get.acme.sh` 安装到当前用户目录

证书默认安装到：

```text
/opt/ou-ui/certs/fullchain.pem
/opt/ou-ui/certs/privkey.pem
```

手动证书也必须放到同一路径，或在 `.env` 中设置 `OUUI_TLS_CERT_FILE` 与 `OUUI_TLS_KEY_FILE` 对应容器内路径。

## 4. Agent 安装入口

面板安装脚本会在安装目录生成：

```text
/opt/ou-ui/docs/agent-install.md
```

该文件包含面向节点的 Agent 构建和启动命令，会引用本次生成的面板地址与 Agent 注册令牌。令牌只应在受控机器上使用，不要复制到仓库、工单或公共聊天中。

## 5. CI

GitHub Actions 工作流位于 `.github/workflows/ci.yml`，覆盖：

- `go test ./...`
- 构建 `apps/server` 与 `apps/agent`
- `pnpm typecheck` 与 `pnpm build`
- `bash -n` 与 ShellCheck
- Docker `web`、`server` target build
- Gitleaks secret scan

CI 不需要真实业务 token。secret scan 使用仓库上下文内的 `GITHUB_TOKEN`，不要新增项目级真实凭据。

## 6. 升级

1. 备份安装目录中的 `.env` 与 `data/`。
2. 拉取新版本代码。
3. 对比 `.env.example` 是否新增变量。
4. 在安装目录执行：

```bash
docker compose up -d --build
docker compose ps
```

5. 验证 Web 页面、Agent 注册、日志和健康状态。

## 7. 回滚

1. 停止当前服务：

```bash
docker compose down
```

2. 切回上一版本镜像或上一份 Compose 文件。
3. 恢复对应版本的数据备份。
4. 重新启动并查看日志：

```bash
docker compose logs --tail=200
```

## 8. 故障排查

端口被占用：

```bash
ss -lntp | grep 3000
```

Docker 未启动或权限不足：

```bash
docker info
docker compose version
```

证书签发失败：

1. 检查域名解析是否正确。
2. 确认 `CF_Token` 只在当前安全 shell 会话中导出。
3. 查看 acme.sh 输出，不要公开包含敏感值的日志。
4. 重新运行安装脚本或手动放置证书后再启动服务。
