# 本地 Docker Compose 部署

Compose 提供 MySQL、一次性 migration、Redis、Go 后端和 React 前端组成的完整本地栈。

## 前置条件

- Docker Engine 或 Docker Desktop，支持 `docker compose`。
- 默认端口 `8080` 和 `8082` 未被占用；也可用 `FRONTEND_PORT`、`BACKEND_PORT` 修改。MySQL、Redis 不映射到宿主机。
- 可以拉取 Dockerfile 与 `compose.yaml` 中固定的基础镜像。

## 配置

```bash
cp .env.example .env
```

PowerShell：

```powershell
Copy-Item .env.example .env
```

替换以下值，root 密码与应用密码应不同：

```dotenv
DB_ROOT_PASSWORD=replace_with_a_strong_root_password
DB_PASSWORD=replace_with_a_different_app_password
JWT_SECRET=replace_with_a_32_plus_chars_random_secret
REGISTRATION_ENABLED=true
APP_ENV=development
COOKIE_SECURE=false
FRONTEND_PORT=8080
BACKEND_PORT=8082
```

本地 Compose 只监听 HTTP，因此保持 `COOKIE_SECURE=false`；Compose 仍强制使用 Redis，并将固定容器网段配置为可信代理。通过 HTTPS 公开部署时必须改用 `APP_ENV=production`、`COOKIE_SECURE=true`，并把 `TRUSTED_PROXIES` 收窄到真实入口代理 CIDR。

`.env` 已被 Git 忽略，不得提交真实凭据。

## 启动

```bash
docker compose up -d --build --wait
docker compose ps
```

启动顺序：

1. MySQL 健康检查通过。
2. `migrate` 服务使用应用账号执行 `migrations/*.sql` 后正常退出。
3. Redis 健康检查通过。
4. 后端启动并通过 `/readyz`。
5. 前端 Nginx 启动，并把 `/api` 请求代理到后端。

无需运行本机 Goose，也无需复制 `.env.goose.example`。`.env.goose` 仅用于后端脱离 Compose 开发时手动执行 migration。

## 初始化管理员

普通注册永远只获得 `user` 角色。第一个管理员必须通过一次性命令创建：

```bash
export BOOTSTRAP_ADMIN_USERNAME=admin
export BOOTSTRAP_ADMIN_PASSWORD='replace-with-a-strong-password'
docker compose run --rm -e BOOTSTRAP_ADMIN_USERNAME -e BOOTSTRAP_ADMIN_PASSWORD app bootstrap-admin
```

PowerShell：

```powershell
$env:BOOTSTRAP_ADMIN_USERNAME="admin"
$env:BOOTSTRAP_ADMIN_PASSWORD="replace-with-a-strong-password"
docker compose run --rm -e BOOTSTRAP_ADMIN_USERNAME -e BOOTSTRAP_ADMIN_PASSWORD app bootstrap-admin
```

系统已有管理员时命令会拒绝再次初始化。不要把管理员密码写进 Compose 文件或 Git。

## 验证

```bash
curl http://127.0.0.1:8082/ping
curl http://127.0.0.1:8082/livez
curl http://127.0.0.1:8082/readyz
```

访问：

- Web：`http://127.0.0.1:8080`
- Swagger：`http://127.0.0.1:8082/swagger/index.html`

浏览器端到端测试：

```bash
cd frontend
npx playwright install chromium
npm run test:e2e
```

## 日志与维护

```bash
docker compose logs -f app
docker compose logs -f frontend
docker compose logs migrate
docker compose logs mysql
docker compose logs redis
```

重新执行 migration 时可运行：

```bash
docker compose run --rm migrate
```

停止服务但保留数据：

```bash
docker compose down
```

删除服务和本地数据卷：

```bash
docker compose down -v
```

`down -v` 会永久删除本地 MySQL 和 Redis 数据，仅用于可丢弃环境。

## 常见问题

### migration 失败

问题：`migrate` 退出，后端没有启动。

原因：常见原因是两个数据库密码未设置、旧数据卷使用了不同密码，或 SQL migration 无法执行。

修改建议：

```bash
docker compose logs migrate
docker compose logs mysql
docker compose config
```

测试环境可在确认数据可删除后执行 `docker compose down -v` 再重建。

### 修改注册开关后未生效

问题：修改 `.env` 后注册路由行为未变化。

原因：运行中的后端容器仍使用创建时的环境变量。

修改建议：

```bash
docker compose up -d --force-recreate app frontend
```

### `/readyz` 不是 200

问题：应用进程存在，但就绪检查失败。

原因：MySQL 或 Redis 不可访问时，认证服务采用 fail-closed。

修改建议：先检查 `docker compose ps`，再查看 app、mysql、redis 日志和 `.env`。
