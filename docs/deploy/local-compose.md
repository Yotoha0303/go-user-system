# 本地 Docker Compose 部署说明

本文档说明如何在本地使用 Docker Compose 启动 `go-user-system` 和 MySQL。

## 1. 前置条件

- 已安装 Docker Desktop，或 Docker Engine + Docker Compose
- Docker 能拉取 `golang:1.25.5-alpine`、`alpine:3.22`、`mysql:8.4`
- 已复制 `.env.example` 为 `.env`
- `.env` 中已设置 `DB_PASSWORD` 和 `JWT_SECRET`
- 本地端口 `8082`、`3306` 未被占用

## 2. 启动服务

首次启动前：

```bash
cp .env.example .env
```

Windows PowerShell：

```powershell
Copy-Item .env.example .env
```

修改 `.env`：

```dotenv
DB_PASSWORD=your_mysql_password
JWT_SECRET=replace_with_a_32_plus_chars_random_secret
```

启动：

```bash
docker compose up -d --build
make migrate-up
```

或：

```bash
make compose-up
make migrate-up
```

以上命令会：

- 构建 Go 应用镜像
- 启动 MySQL `8.4`
- 创建数据库 `go_user_system`
- 等待 MySQL healthcheck 通过
- 启动应用容器
- 使用 goose 执行 `migrations/*.sql`

## 3. 查看状态

```bash
docker compose ps
```

期望结果：

- `go-user-system-mysql` 为 `healthy`
- `go-user-system-app` 为 `running` 或 `healthy`

查看日志：

```bash
docker compose logs -f app
docker compose logs -f mysql
```

## 4. 验证服务

```bash
curl http://127.0.0.1:8082/ping
curl http://127.0.0.1:8082/livez
curl http://127.0.0.1:8082/readyz
```

`/readyz` 返回 200 表示应用进程已启动，且数据库可连接。

完整接口验证可使用：

- `docs/http/test.http`
- README 中的 API 概览

## 5. MySQL 连接信息

| 配置项 | 值 |
| --- | --- |
| 主机 | `127.0.0.1` |
| 端口 | `3306` |
| 用户 | `root` |
| 密码 | `.env` 中的 `DB_PASSWORD` |
| 数据库 | `go_user_system` |

应用容器内部访问 MySQL 时使用：

```dotenv
DB_HOST=mysql
DB_PORT=3306
```

原因：Compose 会创建内部 DNS，`mysql` 是数据库服务名。

## 6. 停止服务

停止并删除容器：

```bash
docker compose down
```

删除容器和 MySQL 数据卷：

```bash
docker compose down -v
```

注意：`docker compose down -v` 会删除本地 MySQL 数据卷，执行前确认数据可以丢弃。

## 7. 常见问题

### 端口被占用

问题：`8082` 或 `3306` 已被其他进程占用。

原因：Compose 需要把容器端口映射到宿主机端口。

修改建议：停止占用端口的进程，或修改 `compose.yaml` 端口映射。

示例：

```yaml
ports:
  - "8083:8082"
```

### 应用连接数据库失败

问题：应用日志出现数据库连接失败。

原因：常见原因包括 MySQL 尚未健康、`.env` 密码不一致、`DB_HOST` 配置错误。

修改建议：依次检查服务状态、MySQL 日志、应用日志和 `.env`。

```bash
docker compose ps
docker compose logs mysql
docker compose logs app
```

### 修改代码后容器行为没有变化

问题：修改 Go 代码后，容器行为没有更新。

原因：应用镜像需要重新构建。

修改建议：

```bash
docker compose up -d --build
```
