# OU-UI SOP

版本：`v0.1.0`

仓库：<https://github.com/cshaizhihao/OU-UI>

## 1. 安装前检查

1. 确认目标机器已安装 Docker 和 Docker Compose。
2. 确认当前用户有权限创建安装目录，例如 `/opt/ou-ui`。
3. 确认目标端口未被占用，默认端口为 `3000`。
4. 如需域名访问，确认域名所有权和 DNS 管理权限。
5. 如需 SSL 自动签发，先阅读 acme.sh 和 DNS 服务商的官方文档，在本机安全地配置 DNS API 环境变量。

禁止事项：

- 不要把真实 token、密码、证书私钥写入仓库。
- 不要把 `.env`、`data/`、`certs/`、日志或备份文件提交到 Git。
- 不要在公共聊天、工单或截图中展示安装脚本生成的密码。

## 2. 标准安装

```bash
bash scripts/install.sh
```

按提示完成：

1. 阅读并确认安装声明。
2. 输入安装目录。
3. 输入 Web 端口。
4. 选择是否绑定域名。
5. 如绑定域名，确认 DNS 是否已经指向当前服务器。
6. 如果输入域名，面板强制 HTTPS，并尝试使用 acme.sh + Cloudflare DNS 自动签发证书。
7. 如果不输入域名，降级使用 `HTTP://IP:端口`。
8. 脚本自动生成随机安全路径，例如 `/ds8a9f`。
9. 确认是否生成 Docker Compose 配置并启动。

安装完成后，记录脚本回显的管理员账号和密码。该密码不会再次从仓库中恢复。

## 3. 域名、DNS、SSL 分支

### 3.1 不使用域名

适合内网或本地调试。访问地址格式：

```text
http://服务器IP:端口
```

### 3.2 使用域名并强制 HTTPS

确认 DNS A/AAAA 记录已经指向服务器。只要输入域名，安装脚本就会强制使用 HTTPS，并优先走 acme.sh + Cloudflare DNS 自动签发证书。访问地址格式：

```text
https://域名:端口/安全路径
```

### 3.3 手动放置证书

适合已有证书管理流程的团队。建议路径：

```text
/opt/ou-ui/certs/fullchain.pem
/opt/ou-ui/certs/privkey.pem
```

证书私钥必须只保存在目标机器，不得提交到仓库。

`v0.1.0` 的 Nginx 容器会在 `OUUI_ENABLE_SSL=yes` 时读取证书并启用 TLS 监听。

### 3.4 acme.sh

适合自动签发和续期。建议步骤：

1. 安装 acme.sh。
2. 在当前 shell 会话中配置 DNS 服务商要求的环境变量。
3. 执行安装脚本并选择 acme.sh 分支。
4. 安装脚本会执行签发和安装证书流程。

注意：安装脚本从当前 shell 读取 `CF_Token`，不会把 DNS token 写入仓库或 `.env` 文件。

## 4. 升级

1. 备份安装目录中的 `.env` 和 `data/`。
2. 拉取新版本代码。
3. 对比 `.env.example` 是否新增变量。
4. 在安装目录执行：

```bash
docker compose pull
docker compose up -d --build
docker compose ps
```

5. 验证 Web 页面、日志和健康状态。

## 5. 回滚

1. 停止当前服务：

```bash
docker compose down
```

2. 切回上一版本镜像或上一份 Compose 文件。
3. 恢复对应版本的数据备份。
4. 启动服务并验证日志。

## 6. 日常运维

查看状态：

```bash
docker compose ps
```

查看日志：

```bash
docker compose logs -f --tail=200
```

重启：

```bash
docker compose restart
```

停止：

```bash
docker compose down
```

## 7. 安全检查清单

- `.env` 权限是否为 `600`。
- `certs/` 是否未进入 Git。
- 端口是否只暴露必要范围。
- 管理员密码是否已妥善保存。
- 日志中是否没有 token、密码和证书私钥。
- DNS token 是否只存在于用户受控的安全环境中。

## 8. 故障排查

端口被占用：

```bash
ss -lntp | grep 3000
```

容器启动失败：

```bash
docker compose logs --tail=200
```

DNS 未生效：

```bash
dig +short example.com
```

证书签发失败：

1. 检查域名解析是否正确。
2. 检查 DNS API 环境变量是否只在当前安全会话中配置。
3. 查看 acme.sh 输出日志。
4. 不要把失败日志中可能出现的敏感值提交或公开。
