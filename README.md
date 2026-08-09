# go-user-system

基于 Go + Gin + GORM + MySQL + Redis 的用户认证系统。项目重点不是堆功能，而是把一个后端服务做成可运行、可测试、可部署、可复盘的工程化样板。

## 当前状态

- 已实现用户注册、登录、双 Token 刷新、登出、当前用户查询、昵称修改、密码修改。
- Access/Refresh Token 携带用户 `auth_version`；改密会在同一事务中递增版本并吊销全部 Refresh Token，禁用用户和旧版本 Token 会在鉴权时被拒绝。
- 使用 **JWT Access/Refresh 双 Token** 做接口鉴权，**Refresh Token 只存哈希并支持 Rotation、Token Family 重放检测和 Family 级吊销**。
- Redis 保存按 JTI 标识的 Access Token 吊销状态，并实现账号/IP 双维度登录失败限流；生产配置下 Redis 故障采用 fail-closed。
- 基于 RBAC 五表模型实现角色、权限、用户角色、角色权限，并通过 Gin 中间件做接口级鉴权。
- 使用统一响应结构、业务错误码和 `internal/apperror` 应用错误模型。
- 使用 `swaggo/swag` 注解生成 Swagger JSON、YAML 文档，并通过 `gin-swagger` 提供 `/swagger/index.html` 文档入口。
- 使用 Goose 管理 SQL migration，不使用 GORM `AutoMigrate`。
- 已接入 `Request ID`、结构化 access log、panic recovery 日志。
- 已配置 HTTP server 超时、请求 context timeout、数据库连接池，以及 MySQL/Redis 启动 Ping timeout。
- GitHub Actions CI 覆盖 golangci-lint、单元测试、race 测试、go vet、migration 校验、二进制构建和 Docker 镜像构建；本地 `make ci` 覆盖 lint、test、race-test、vet、build 和 docker-build。

## 技术栈

| 类型 | 技术 |
| --- | --- |
| Web 框架 | Gin |
| ORM | GORM |
| 数据库 | MySQL |
| 认证状态 | Redis 7（本地测试可使用线程安全内存实现） |
| Migration | goose |
| 认证 | JWT Access/Refresh Token + bcrypt |
| 权限 | RBAC 五表模型 |
| 接口文档 | swaggo / gin-swagger |
| 配置 | `config.yml` + `.env` + 环境变量覆盖 |
| 日志 | `log/slog` JSON 结构化日志 |
| 测试 | Go testing、httptest、fake SQL driver、miniredis、MySQL integration test |
| 质量门禁 | golangci-lint v2、go test、go test -race、go vet |
| 部署 | Docker、Docker Compose、Kubernetes、GitHub Actions |

## 项目结构

```text
cmd/                    程序入口和启动流程
config/                 配置加载、默认值、环境变量覆盖和校验
internal/
  apperror/             应用错误模型
  auth/                 JWT 签发和解析
  authstate/            Access 吊销与登录限流的 Redis/内存实现
  dao/                  数据访问层
  handler/              HTTP handler 和错误响应映射
  middleware/           Request ID、Access Log、Recovery、Timeout、Auth
  model/                GORM 模型
  repository/           Refresh Token、RBAC Repository
  request/              请求 DTO
  response/             统一响应结构和业务错误码
  service/              业务逻辑
  testutil/             MySQL 集成测试工具
pkg/
  database/             MySQL / GORM 初始化
  redisclient/          Redis 客户端初始化和启动健康检查
router/                 路由注册
migrations/             goose SQL migration
docs/
  deploy/               本地 Compose 与生产部署检查文档
  http/                 REST Client 手动测试文件
  sql/                  本地 SQL 辅助脚本
  docs.go               swaggo 生成文件
  swagger.json          Swagger JSON 文档
  swagger.yaml          Swagger YAML 文档
  backend-callgraph.gv  go-callvis DOT 调用图
  backend-callgraph.svg go-callvis SVG 调用图
```

## 快速启动

### Docker Compose

首次启动前复制配置：

```bash
cp .env.example .env
cp .env.goose.example .env.goose
```

Windows PowerShell：

```powershell
Copy-Item .env.example .env
Copy-Item .env.goose.example .env.goose
```

修改 `.env`：

```dotenv
DB_PASSWORD=your_mysql_password
JWT_SECRET=replace_with_a_32_plus_chars_random_secret
JWT_ACCESS_TOKEN_EXPIRE_MINUTES=15
JWT_REFRESH_TOKEN_EXPIRE_HOURS=168
REDIS_ENABLED=false
REDIS_ADDR=127.0.0.1:6379
# REDIS_PASSWORD=
REDIS_DB=0
```

修改 `.env.goose`，确保数据库密码与 `.env` 一致：

```dotenv
GOOSE_DRIVER=mysql
GOOSE_DBSTRING=root:your_mysql_password@tcp(127.0.0.1:3306)/go_user_system?parseTime=true&multiStatements=true
GOOSE_MIGRATION_DIR=./migrations
```

启动服务，然后手动执行 migration。应用启动流程不会自动执行 migration，必须显式运行 `make migrate-up`：

```bash
docker compose up -d --build
make migrate-up
docker compose ps
```

验证：

```bash
curl http://127.0.0.1:8082/ping
curl http://127.0.0.1:8082/livez
curl http://127.0.0.1:8082/readyz
```

更多说明见 `docs/deploy/local-compose.md`。

### 本地 Go 启动

前置条件：

- 安装 Go。
- 安装 goose：`go install github.com/pressly/goose/v3/cmd/goose@v3.27.3`。
- 启动 MySQL；启用 `REDIS_ENABLED=true` 时还需启动 Redis。
- 创建数据库 `go_user_system`。
- 复制并配置 `.env` 和 `.env.goose`。

创建数据库：

```sql
CREATE DATABASE go_user_system
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;
```

启动：

```bash
go mod download
make migrate-up
go run ./cmd
```

注意：`cmd/main.go` 只负责加载配置、初始化数据库连接、初始化 JWT 和启动 HTTP server，不会自动执行 migration。

## 配置说明

| 来源 | 作用 | 是否提交 |
| --- | --- | --- |
| `config.yml` | 非敏感默认配置；不建议保存密钥 | 是 |
| `.env.example` | 本地和 Compose 环境变量模板 | 是 |
| `.env` | 本地真实环境变量 | 否 |
| `.env.goose.example` | goose 本地迁移模板 | 是 |
| `.env.goose` | goose 本地真实迁移配置 | 否 |
| shell 环境变量 | CI、容器、服务器运行时注入 | 否 |

关键环境变量：

```dotenv
APP_PORT=8082
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=your_mysql_password
DB_NAME=go_user_system
JWT_SECRET=replace_with_a_32_plus_chars_random_secret
JWT_EXPIRE_HOURS=24
JWT_ACCESS_TOKEN_EXPIRE_MINUTES=15
JWT_REFRESH_TOKEN_EXPIRE_HOURS=168
REDIS_ENABLED=false
REDIS_ADDR=127.0.0.1:6379
# REDIS_PASSWORD=
REDIS_DB=0
```

配置加载规则：

- 启动时加载 `.env`，再加载 `config.yml`。
- `APP_PORT`、数据库变量、JWT 变量以及 `REDIS_ENABLED`、`REDIS_ADDR`、`REDIS_PASSWORD`、`REDIS_DB` 可覆盖 `config.yml`。
- `APP_PORT`、`JWT_EXPIRE_HOURS`、`JWT_ACCESS_TOKEN_EXPIRE_MINUTES` 和 `JWT_REFRESH_TOKEN_EXPIRE_HOURS` 如果存在但格式错误，启动会失败。
- `JWT_SECRET` 会被加载进 `cfg.JWT.Secret` 后再初始化 TokenManager；环境变量或 `.env` 中的 `JWT_SECRET` 优先级高于 `config.yml` 的 `jwt.secret`。
- `DB_PASSWORD` 不在 `config.yml` 中保存，必须通过环境变量或 `.env` 注入。
- `REDIS_ENABLED=true` 时启动阶段必须完成 Redis Ping；运行中 Redis 读取失败会拒绝鉴权，不会静默放行。关闭 Redis 只适用于本地开发和单元测试。
- `JWT_SECRET` 长度必须至少 32 个字符。生产环境推荐只通过运行时环境变量或 `.env` 注入，不要提交到 `config.yml`。

`.env` 是可选的本地开发文件。容器和生产环境可以只通过运行时环境变量注入配置，不需要挂载 `.env`。

## API 概览

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| `GET` | `/ping` | 基础健康检查 | 否 |
| `GET` | `/livez` | 进程存活检查 | 否 |
| `GET` | `/readyz` | 服务就绪检查，包含 MySQL 和认证状态存储 Ping | 否 |
| `POST` | `/api/v1/auth/register` | 用户注册 | 否 |
| `POST` | `/api/v1/auth/login` | 返回 Access Token，并通过 HttpOnly Cookie 写入 Refresh Token；限流时返回 429 和 `Retry-After` | 否 |
| `POST` | `/api/v1/auth/refresh` | 使用 Refresh Cookie 轮换双 Token | 否 |
| `POST` | `/api/v1/auth/logout` | 吊销 Refresh Token；携带 Access Token 时同时按 JTI 立即吊销 | 否 |
| `GET` | `/api/v1/users/me` | 当前用户信息，需要 `profile:read` | 是 |
| `GET` | `/api/v1/users/me/authorization` | 当前用户角色码与权限码 | 是 |
| `PUT` | `/api/v1/users/me/profile` | 修改当前用户昵称，需要 `profile:update` | 是 |
| `PATCH` | `/api/v1/users/me/update/password` | 修改当前用户密码，需要 `password:update` | 是 |
| `GET` | `/api/v1/admin/roles` | 查询角色列表，需要 `admin:roles:read` | 是 |
| `GET` | `/api/v1/admin/permissions` | 查询权限列表，需要 `admin:permissions:read` | 是 |
| `PUT` | `/api/v1/admin/users/:id/roles` | 给用户分配角色，需要 `admin:user_roles:update` | 是 |

Swagger 文档：

- 页面入口：`/swagger/index.html`
- JSON：`/swagger/doc.json`
- YAML：`/swagger/swagger.yaml`
- 重新生成：`make swagger`

手动测试文件：`docs/http/test.http`。

后端调用图：

- SVG：`docs/backend-callgraph.svg`
- DOT 源文件：`docs/backend-callgraph.gv`
- 重新生成：`make callvis`
- 交互查看：运行 `make callvis-serve`，访问 `http://127.0.0.1:7878/`。
- 分析方式：RTA，按 package/type 分组，仅保留 `go-user-system` 模块内调用。

`make callvis` 固定使用 `go-callvis v0.7.1`，首次执行会下载工具并在 `.cache/go-callvis` 建立独立 Go 构建缓存。当前模块路径不含域名，不能给命令增加 `-nostd`，否则该版本会把项目包误判为标准库并生成空图。

RBAC 初始化规则：

- 第一个注册用户会自动绑定 `admin` 和 `user` 角色，用于系统初始化。
- 后续注册用户默认绑定 `user` 角色。
- 管理员可通过 `/api/v1/admin/users/:id/roles` 调整用户角色。

浏览器端认证约定：

- Access Token 只保存在前端内存状态中，通过 `Authorization: Bearer <token>` 发送。
- Refresh Token 仅存储在 `HttpOnly`、`SameSite=Lax` Cookie 中，不出现在登录和刷新响应体。
- `/api/v1/auth/refresh` 和 `/api/v1/auth/logout` 仍允许可选 JSON 请求体，便于非浏览器客户端调用。

## 数据库迁移

项目使用 goose 管理 `migrations/*.sql`。当前 migration：

应用启动不会自动执行 migration。部署或本地启动前需要通过 `make migrate-up` 或等价 goose 命令显式执行。

| 文件 | 作用 |
| --- | --- |
| `migrations/00001_create_users.sql` | 创建 / 回滚 `users` 表 |
| `migrations/00002_add_user_audit_fields.sql` | 增加 / 回滚 `last_login_at`、`deleted_at` |
| `migrations/00003_create_refresh_tokens.sql` | 创建 / 回滚 `refresh_tokens` 表 |
| `migrations/00004_create_rbac_tables.sql` | 创建 / 回滚 RBAC 四表并写入默认角色权限 |
| `migrations/00005_backfill_user_roles.sql` | 给既有用户补 `user` 角色，并给最早用户补 `admin` 角色 |
| `migrations/00006_harden_auth_sessions.sql` | 增加 `auth_version`、Refresh Token `family_id` 与 `revoked_reason` |

常用命令：

```bash
make migrate-validate
make migrate-status
make migrate-version
make migrate-up
make migrate-down
```

新增 migration：

```bash
make migrate-create name=your_change
```

然后在生成的 SQL 文件里补充 `-- +goose Up` 和 `-- +goose Down`。

## 测试与质量门禁

本地常用命令：

```bash
make lint
make test
make race-test
make vet
make coverage
make build
```

`make lint` 使用 `.golangci.yml`，该文件是 golangci-lint v2 配置。本地需要安装 v2，例如：

```bash
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2
```

集成测试需要专用 MySQL 测试库，数据库名必须包含 `test`，避免误删开发库或生产库：

```sql
CREATE DATABASE go_user_system_test
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;
```

PowerShell 示例：

```powershell
$env:TEST_DATABASE_DSN="root:your_mysql_password@tcp(127.0.0.1:3306)/go_user_system_test?charset=utf8mb4&parseTime=True&loc=Local"
go test ./internal/dao ./internal/service -run Integration -v
```

## CI 流程

CI 文件：`.github/workflows/ci.yml`

当前流程：

1. `go mod download`
2. `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2`
3. `golangci-lint run ./...`
4. `go test ./...`
5. `go test -race ./...`
6. `go vet ./...`
7. `go install github.com/pressly/goose/v3/cmd/goose@v3.27.3`
8. `goose -dir migrations validate`
9. `go build -o bin/go-user-system ./cmd`
10. `docker build -t go-user-system:ci .`

本地等价检查：

```bash
make ci
```

`make ci` 当前执行 `make lint`、`make test`、`make race-test`、`make vet`、`make build` 和 `make docker-build`。GitHub Actions 额外执行 `goose -dir migrations validate`。

## 生产部署检查

生产部署前至少确认：

- `JWT_SECRET` 使用 32 位以上强随机字符串。
- `DB_PASSWORD` 不使用默认值。
- Redis 必须启用、不可被公网访问，并为 `/data` 配置持久化存储。
- 生产数据库不使用 MySQL `root` 账号连接业务库。
- `make lint`、`make test`、`make race-test`、`make vet` 通过。
- `make migrate-validate` 通过，并已在目标数据库执行 migration。
- `/readyz` 返回 200。
- 应用日志没有打印密码、JWT secret、access token、password hash。
- 容器以非 root 用户运行。
- SIGTERM 能触发优雅关闭。

完整清单见 `docs/deploy/production-checklist.md`。

## 常见问题

### JWT 初始化失败

检查：

```dotenv
JWT_SECRET=replace_with_a_32_plus_chars_random_secret
JWT_EXPIRE_HOURS=24
JWT_ACCESS_TOKEN_EXPIRE_MINUTES=15
JWT_REFRESH_TOKEN_EXPIRE_HOURS=168
```

`JWT_SECRET` 不能为空，长度不能少于 32 个字符。可以放在 `.env`、shell 环境变量或 `config.yml` 的 `jwt.secret` 中；如果同时存在，环境变量优先。Access Token 和 Refresh Token 的过期配置必须是正整数。

### Compose 中应用连接不上数据库

容器内部使用 `DB_HOST=mysql`，本机直连使用 `DB_HOST=127.0.0.1`。优先检查：

```bash
docker compose ps
docker compose logs mysql
docker compose logs app
```

### `golangci-lint` 报配置版本不匹配

`.golangci.yml` 是 v2 配置。如果本地是 v1，会看到类似“configuration file for golangci-lint v2 with golangci-lint v1”的错误。安装 v2 后再运行 `make lint`。
