# go-user-system

基于 Go + Gin + GORM + MySQL 的用户认证系统。项目重点不是堆功能，而是把一个后端服务做成可运行、可测试、可部署、可复盘的工程化样板。

## 当前状态

- 单元测试与核心业务测试已系统补齐。
- 当前全量测试覆盖率：`96.2%`。
- 核心业务包覆盖率基本达到 `100%`：`service`、`handler`、`middleware`、`dao`、`utils`、`response`、`router` 等。
- 覆盖率流程与测试内容说明：`docs/testing/test-coverage.md`。
- CI 已覆盖：依赖下载、测试、`go vet`、goose migration 校验、二进制构建、Docker 镜像构建。

## 功能范围

- 用户注册、登录、当前用户查询、昵称修改
- bcrypt 密码哈希存储，接口不返回 `password_hash`
- JWT 签发、解析和鉴权中间件
- 统一响应结构、业务错误码、应用错误封装
- `/ping`、`/livez`、`/readyz` 健康检查
- Goose SQL migration 替代 GORM `AutoMigrate`
- MySQL 集成测试安全保护：测试库名称必须包含 `test`
- Dockerfile、Docker Compose、本地部署与生产检查文档

## 技术栈

| 类型 | 技术 |
| --- | --- |
| Web 框架 | Gin |
| ORM | GORM |
| 数据库 | MySQL |
| 数据库迁移 | goose |
| 认证 | JWT + bcrypt |
| 配置 | `config.yml` + `.env` + 环境变量覆盖 |
| 测试 | Go testing、httptest、fake SQL driver、MySQL integration test |
| 部署 | Docker、Docker Compose |
| CI | GitHub Actions |

## 工程化设计

- **依赖注入**：`*gorm.DB` 从 `cmd/main.go` 显式传入 router、handler、service，避免全局 DB。
- **上下文传递**：HTTP request context 从 handler 传到 service 和 dao，DAO 使用 `WithContext`。
- **错误处理**：用 `internal/apperror` 封装 HTTP 状态码、业务码、消息和底层 cause。
- **数据库迁移**：使用 goose 管理 `migrations/*.sql`，CI 校验迁移文件，执行记录存入 `goose_db_version`。
- **测试可测性**：对启动流程、token 解析、数据库打开、DAO adapter 等位置做可注入设计，方便覆盖失败分支。
- **部署安全**：容器健康检查、非 root 用户运行、SIGTERM 优雅关闭。

## 项目结构

```text
cmd/                    程序入口与启动流程
config/                 配置加载：config.yml + 环境变量覆盖
internal/
  apperror/             应用错误模型
  dao/                  数据访问层
  handler/              HTTP Handler
  middleware/           JWT 鉴权中间件
  model/                GORM 模型
  request/              请求 DTO
  response/             统一响应结构和业务错误码
  service/              业务逻辑
  testutil/             MySQL 集成测试工具
  utils/                JWT 工具
pkg/
  database/             MySQL / GORM 初始化
router/                 路由注册
migrations/             数据库迁移脚本
docs/
  deploy/               本地 Compose 与生产检查文档
  http/                 REST Client 手动测试文件
  sql/                  本地 SQL 辅助脚本
  testing/              测试覆盖率与测试流程
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

启动服务并执行迁移：

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

更多 Compose 说明：`docs/deploy/local-compose.md`。

### 本地 Go 启动

前置条件：

- 安装 Go
- 安装 goose：`go install github.com/pressly/goose/v3/cmd/goose@latest`
- 启动 MySQL
- 创建数据库 `go_user_system`
- 从 `.env.example` 复制并配置 `.env`
- 从 `.env.goose.example` 复制并配置 `.env.goose`

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

或使用 Makefile：

```bash
make migrate-up
make run
```

## 配置说明

| 来源 | 作用 | 是否提交 |
| --- | --- | --- |
| `config.yml` | 非敏感默认配置 | 是 |
| `.env.example` | 本地和 Compose 配置模板 | 是 |
| `.env` | 本地真实配置 | 否 |
| `.env.goose.example` | goose 本地迁移配置模板 | 是 |
| `.env.goose` | goose 本地真实迁移配置 | 否 |
| shell 环境变量 | CI、容器、服务器运行时注入 | 否 |

配置加载规则：

- 启动时从当前工作目录向上查找 `.env` 和 `config.yml`
- 环境变量优先级高于 `config.yml`
- `.env`、`.env.*`、`config.local.yml` 不提交到 Git

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

goose 迁移配置：

```dotenv
GOOSE_DRIVER=mysql
GOOSE_DBSTRING=root:your_mysql_password@tcp(127.0.0.1:3306)/go_user_system?parseTime=true&multiStatements=true
GOOSE_MIGRATION_DIR=./migrations
```

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

手动测试文件：`docs/http/test.http`。

## 数据库迁移

项目不使用 GORM `AutoMigrate`，使用 goose 管理 `migrations/` 下的 SQL migration 文件。

当前 migration 采用 goose 单文件格式：

- 文件名使用顺序编号：`00001_create_users.sql`
- 文件内使用 `-- +goose Up` 和 `-- +goose Down` 区分正向迁移与回滚
- 执行记录由 goose 写入 `goose_db_version`

| 文件 | 作用 |
| --- | --- |
| `migrations/00001_create_users.sql` | 创建 / 回滚 `users` 表 |
| `migrations/00002_add_user_audit_fields.sql` | 增加 / 回滚 `last_login_at`、`deleted_at` |

常用命令：

```bash
make goose-version
make migrate-validate
make migrate-status
make migrate-version
make migrate-up
make migrate-down
```

新增表结构变更时：

```bash
make migrate-create name=your_change
```

然后在生成的 SQL 文件中补充 `-- +goose Up` 和 `-- +goose Down` 对应 SQL。

## 测试与质量门禁

默认测试不依赖真实 MySQL：

```bash
go test ./...
go vet ./...
go build -o bin/go-user-system ./cmd
```

Makefile：

```bash
make test
make coverage
make coverage-html
make vet
make build
```

覆盖率命令：

```bash
go test ./... "-coverprofile=coverage.out" -covermode=atomic
go tool cover "-func=coverage.out"
go tool cover "-html=coverage.out" -o coverage.html
```

当前覆盖率：

- 全量语句覆盖率：`96.2%`
- 建议 CI 覆盖率门槛：`95%`
- 不建议为了追求绝对 `100%` 强行构造不稳定的 runtime/数据库故障分支

详细测试内容：`docs/testing/test-coverage.md`。

## 集成测试

集成测试需要专用 MySQL 测试库，数据库名必须包含 `test`，避免误删开发库或生产库。

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

流程：

1. `go mod download`
2. `go test ./...`
3. `go vet ./...`
4. `go install github.com/pressly/goose/v3/cmd/goose@latest`
5. `goose -dir migrations validate`
6. `go build -o bin/go-user-system ./cmd`
7. `docker build -t go-user-system:ci .`

## 部署检查

生产部署前至少确认：

- `JWT_SECRET` 使用 32 位以上强随机字符串
- 生产数据库不使用 MySQL `root` 用户连接业务库
- `/readyz` 返回 200
- migration 已执行，`goose_db_version` 中存在版本记录
- 镜像以非 root 用户运行
- SIGTERM 能触发服务优雅关闭

完整清单：`docs/deploy/production-checklist.md`。

## 可写入简历的亮点

- 实现 Go + Gin + GORM 的用户认证系统，包含 JWT 鉴权、bcrypt 密码哈希、统一错误码和健康检查。
- 使用 goose 管理数据库结构变更，替代 `AutoMigrate`，并通过版本表保证幂等执行。
- 构建完整测试体系，覆盖 service、handler、middleware、DAO、启动流程等模块，全量覆盖率达到 `96.2%`。
- 支持 Docker Compose 本地启动、GitHub Actions CI、容器健康检查和 SIGTERM 优雅关闭。

## 常见问题

### 找不到 `.env`

从项目根目录运行：

```bash
go run ./cmd
```

或确认 `.env` 位于项目根目录，且保存为 UTF-8。

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
