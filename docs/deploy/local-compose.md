# 本地 Docker Compose 部署说明

本文档说明如何使用 Docker Compose 在本地启动 `go-user-system` 和 MySQL。

## 1. 前置条件

- 已安装 Docker Desktop，或 Docker Engine + Docker Compose
- Docker 可以拉取 `golang:1.25.5-alpine`、`alpine:3.22`、`mysql:8.4`
- 已复制 `.env.example` 为 `.env`，并配置 `DB_PASSWORD`、`JWT_SECRET`
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

`compose.yaml` 会使用 `.env` 中的 `DB_PASSWORD` 作为 MySQL root 密码和应用连接密码，并使用 `JWT_SECRET` 初始化 JWT 签名密钥。

在项目根目录执行：

```bash
make compose-up
```

未安装 `make` 时执行：

```bash
docker compose up -d --build
```

该命令会：

- 构建 Go 应用镜像
- 启动 MySQL `8.4`
- 创建数据库 `go_user_system`
- 应用启动时自动执行 `migrations/*.up.sql`
- 启动应用容器并监听 `8082`

## 3. 查看运行状态

```bash
docker compose ps
```

期望结果：

- `go-user-system-mysql` 状态为 healthy
- `go-user-system-app` 状态为 running 或 healthy

查看应用日志：

```bash
make compose-logs
```

未安装 `make` 时执行：

```bash
docker compose logs -f app
```

## 4. 验证接口

健康检查：

```bash
curl http://127.0.0.1:8082/ping
curl http://127.0.0.1:8082/livez
curl http://127.0.0.1:8082/readyz
```

期望返回：

```json
{
  "code": 0,
  "msg": "success",
  "data": {
    "message": "success"
  }
}
```

完整接口验证可使用：

- `docs/http/test.http`
- README 第 9 节手动测试流程

## 5. 连接 MySQL

Compose 中 MySQL 配置：

| 配置项 | 值 |
| --- | --- |
| 主机 | `127.0.0.1` |
| 端口 | `3306` |
| 用户 | `root` |
| 密码 | `.env` 中的 `DB_PASSWORD` |
| 数据库 | `go_user_system` |

应用容器内部访问 MySQL 时使用：

```text
DB_HOST=mysql
DB_PORT=3306
```

原因是 Compose 会为服务创建内部 DNS，`mysql` 是数据库服务名。

## 6. 停止服务

停止并删除容器：

```bash
make compose-down
```

未安装 `make` 时执行：

```bash
docker compose down
```

默认不会删除 `mysql_data` 数据卷，因此数据库数据会保留。

如果需要删除数据库数据：

```bash
docker compose down -v
```

注意：该命令会删除本地 MySQL 数据卷，执行前确认数据可以丢弃。

## 7. 常见问题

### 端口被占用

问题：`8082` 或 `3306` 已被本机其他进程占用。

原因：Compose 需要把容器端口映射到宿主机端口。

修改建议：停止占用端口的进程，或修改 `compose.yaml` 中的端口映射。

示例：

```yaml
ports:
  - "8083:8082"
```

### 应用连接数据库失败

问题：应用日志出现数据库连接失败。

原因：常见原因包括 MySQL 尚未健康、数据库密码不一致、`DB_HOST` 配置错误。

修改建议：优先检查 `docker compose ps`、`docker compose logs mysql`、`docker compose logs app` 和 `.env` 中的环境变量。

示例：

```bash
docker compose logs mysql
docker compose logs app
```

### 修改代码后没有生效

问题：修改 Go 代码后容器行为没有变化。

原因：应用镜像需要重新构建。

修改建议：重新执行带 `--build` 的启动命令。

示例：

```bash
docker compose up -d --build
```
