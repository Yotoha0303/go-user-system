# go-user-system

基于 Go + Gin + GORM + MySQL 的用户认证系统。项目重点不是堆功能，而是把一个后端服务做成可运行、可测试、可部署、可复盘的工程化样板。

## 当前状态

- 已实现用户注册、登录、当前用户查询、昵称修改、密码修改。
- 使用 JWT 做接口鉴权，密码使用 bcrypt 哈希存储，接口不返回 `password_hash`。
- 使用统一响应结构、业务错误码和 `internal/apperror` 应用错误模型。
- 使用 Goose 管理 SQL migration，不使用 GORM `AutoMigrate`。
- 已接入 `Request ID`、结构化 access log、panic recovery 日志。
- 已配置 HTTP server 超时、请求 context timeout、数据库连接池和启动时 DB ping timeout。
- CI 覆盖 golangci-lint、单元测试、race 测试、go vet、migration 校验、二进制构建和 Docker 镜像构建。

## 技术栈

| 类型 | 技术 |
| --- | --- |
| Web 框架 | Gin |
| ORM | GORM |
| 数据库 | MySQL |
| Migration | goose |
| 认证 | JWT + bcrypt |
| 配置 | `config.yml` + `.env` + 环境变量覆盖 |
| 日志 | `log/slog` JSON 结构化日志 |
| 测试 | Go testing、httptest、fake SQL driver、MySQL integration test |
| 质量门禁 | golangci-lint v2、go test、go test -race、go vet |
| 部署 | Docker、Docker Compose、GitHub Actions |

## 项目结构

```text
cmd/                    程序入口和启动流程
config/                 配置加载、默认值、环境变量覆盖和校验
internal/
  apperror/             应用错误模型
  auth/                 JWT 签发和解析
  dao/                  数据访问层
  handler/              HTTP handler 和错误响应映射
  middleware/           Request ID、Access Log、Recovery、Timeout、Auth
  model/                GORM 模型
  request/              请求 DTO
  response/             统一响应结构和业务错误码
  service/              业务逻辑
  testutil/             MySQL 集成测试工具
pkg/
  database/             MySQL / GORM 初始化
router/                 路由注册
migrations/             goose SQL migration
docs/
  deploy/               本地 Compose 与生产部署检查文档
  http/                 REST Client 手动测试文件
  sql/                  本地 SQL 辅助脚本
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
- 安装 goose：`go install github.com/pressly/goose/v3/cmd/goose@latest`。
- 启动 MySQL。
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
| `config.yml` | 非敏感默认配置 | 是 |
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
```

配置加载规则：

- 启动时加载 `.env`，再加载 `config.yml`。
- `APP_PORT`、`DB_HOST`、`DB_PORT`、`DB_USER`、`DB_NAME`、`JWT_EXPIRE_HOURS` 可覆盖 `config.yml`。
- `APP_PORT` 和 `JWT_EXPIRE_HOURS` 如果存在但格式错误，启动会失败。
- `DB_PASSWORD` 和 `JWT_SECRET` 不在 `config.yml` 中保存，必须通过环境变量或 `.env` 注入。
- `JWT_SECRET` 长度必须至少 32 个字符。

当前实现会在启动时加载 `.env`。如果生产环境完全依赖平台环境变量而不挂载 `.env`，需要先调整 `config.LoadEnv` 的行为，或者确保部署环境提供可读取的 `.env`。

## API 概览

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| `GET` | `/ping` | 基础健康检查 | 否 |
| `GET` | `/livez` | 进程存活检查 | 否 |
| `GET` | `/readyz` | 服务就绪检查，包含 DB ping | 否 |
| `POST` | `/api/v1/auth/register` | 用户注册 | 否 |
| `POST` | `/api/v1/auth/login` | 用户登录 | 否 |
| `GET` | `/api/v1/users/me` | 当前用户信息 | 是 |
| `PUT` | `/api/v1/users/me/profile` | 修改当前用户昵称 | 是 |
| `PATCH` | `/api/v1/users/me/update/password` | 修改当前用户密码 | 是 |

手动测试文件：`docs/http/test.http`。

## 数据库迁移

项目使用 goose 管理 `migrations/*.sql`。当前 migration：

应用启动不会自动执行 migration。部署或本地启动前需要通过 `make migrate-up` 或等价 goose 命令显式执行。

| 文件 | 作用 |
| --- | --- |
| `migrations/00001_create_users.sql` | 创建 / 回滚 `users` 表 |
| `migrations/00002_add_user_audit_fields.sql` | 增加 / 回滚 `last_login_at`、`deleted_at` |

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
7. `go install github.com/pressly/goose/v3/cmd/goose@latest`
8. `goose -dir migrations validate`
9. `go build -o bin/go-user-system ./cmd`
10. `docker build -t go-user-system:ci .`

## 生产部署检查

生产部署前至少确认：

- `JWT_SECRET` 使用 32 位以上强随机字符串。
- `DB_PASSWORD` 不使用默认值。
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
```

`JWT_SECRET` 不能为空，长度不能少于 32 个字符。

### Compose 中应用连接不上数据库

容器内部使用 `DB_HOST=mysql`，本机直连使用 `DB_HOST=127.0.0.1`。优先检查：

```bash
docker compose ps
docker compose logs mysql
docker compose logs app
```

### `golangci-lint` 报配置版本不匹配

`.golangci.yml` 是 v2 配置。如果本地是 v1，会看到类似“configuration file for golangci-lint v2 with golangci-lint v1”的错误。安装 v2 后再运行 `make lint`。
