# Research 摘要

版本：`v0.4.0`

仓库：<https://github.com/cshaizhihao/OU-UI>

## 背景

OU-UI 参考了 3x-ui、x-ui、s-ui 和 h-ui 的面板能力边界：主控面板负责用户交互、节点模型、协议配置与任务调度；被控 Agent 负责本机运行时探测、配置渲染、执行状态回传和后续 runtime apply。

## v0.4.0 研究结论

本轮重点从“可演示的任务闭环”推进到“可长期运行的控制链路”：

- Agent 必须持久化本机身份，否则进程重启会不断产生重复 Agent。
- 任务必须有 lease、attempt 和最大重试次数，否则 Agent 崩溃会让任务永久卡在 `running`。
- 在线状态应由服务端基于 `LastSeenAt` 推导，不能完全信任 Agent 自报。
- 节点下发前应在服务端校验 Agent freshness、runtime capability 和 provider spec，减少无意义失败。
- API 返回任务 payload、node spec 时需要对常见敏感字段脱敏。

## CI 设计结论

GitHub Actions 拆成独立 job：

- Go：`go test ./...`，并分别构建 server 与 agent。
- Frontend：使用 pnpm 安装依赖，执行 typecheck 与 build。
- Shell：对 `scripts/` 与 `deploy/` 下的 shell 脚本执行 `bash -n` 和 ShellCheck。
- Docker：通过 Buildx 构建 `web` 与 `server` target，不推送镜像。
- Secret scan：使用 Gitleaks 扫描仓库，避免误提交 token、私钥、密码等敏感内容。

CI 不依赖真实业务 token，也不会把部署凭据写入仓库。

## 安全边界

- 仓库只保存示例值，不保存真实 token、密钥、证书私钥或生产密码。
- `CF_Token`、`ACME_EMAIL` 仅从当前 shell 环境读取，不写入 `.env`。
- 生成的 `.env` 默认 `600` 权限。
- Agent 注册令牌只在安装完成回显和安装目录文档中出现，由目标机器本地保存。

## 后续建议

1. 将全局 join token 升级为限时 enrollment token，并补充撤销/轮换接口。
2. 将 `node.deploy` 从 render-only 扩展为 render、apply、health check、rollback 四阶段。
3. 为 Xray/Hysteria2 增加真实二进制安装、版本探测和 runtime reload。
4. 补充 DB 加密或 secret store，避免长期明文保存节点私密参数。
