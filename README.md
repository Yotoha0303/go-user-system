# go-user-system

基于 Go + Gin + GORM + MySQL 的用户认证系统。项目重点不是堆功能，而是把一个后端服务做成可运行、可测试、可部署、可复盘的工程化样板。

## 功能范围

- 用户注册、登录、当前用户查询、昵称修改
- bcrypt 密码哈希存储，接口不返回 `password_hash`
- JWT 签发、解析和鉴权中间件
- 统一响应结构、业务错误码和应用错误封装
- `/ping`、`/livez`、`/readyz` 健康检查
- SQL migration 替代 GORM `AutoMigrate`

## 工程化基本盘

- **配置**：`config.yml` 保存非敏感默认配置，`.env` 注入敏感配置，`.env.example` 提供模板。
- **依赖注入**：`*gorm.DB` 从 `cmd/main.go` 显式传入 router、service、handler，不再使用全局 DB。
- **数据库迁移**：启动时自动执行 `migrations/*.up.sql`，执行记录保存在 `schema_migrations`。
- **测试**：包含单元测试、Handler 测试、JWT/中间件测试、DAO/Service/MySQL 集成测试、migration 幂等测试。
- **部署**：提供 `Dockerfile`、`compose.yaml`、容器健康检查、非 root 用户运行、SIGTERM 优雅停止。
- **质量门禁**：提供 `Makefile`、GitHub Actions CI、`.editorconfig`、`.gitattributes`、`.dockerignore`。

## 项目结构

```text
cmd/                    程序入口
config/                 配置加载：config.yml + 环境变量覆盖
internal/
  apperror/             应用错误模型
  dao/                  数据访问
  handler/              HTTP Handler
  middleware/           JWT 鉴权中间件
  model/                GORM 模型
  request/              请求 DTO
  response/             响应结构和业务错误码
  service/              业务逻辑
  testutil/             集成测试工具
  utils/                JWT 工具
pkg/
  database/             MySQL / GORM 初始化
  migration/            SQL migration 执行器
router/                 路由注册
migrations/             数据库迁移脚本
docs/
  deploy/               部署说明和生产检查清单
  http/                 REST Client 手动测试
  sql/                  本地 SQL 辅助脚本
```

## 快速启动

### Docker Compose 启动

```bash
cp .env.example .env
docker compose up -d --build
docker compose ps
```

Windows PowerShell：

```powershell
Copy-Item .env.example .env
docker compose up -d --build
docker compose ps
```

启动前必须修改 `.env`：

```dotenv
DB_PASSWORD=your_mysql_password
JWT_SECRET=replace_with_a_32_plus_chars_random_secret
```

验证：

```bash
curl http://127.0.0.1:8082/ping
curl http://127.0.0.1:8082/livez
curl http://127.0.0.1:8082/readyz
```

更多 Compose 说明见 `docs/deploy/local-compose.md`。

### 本地 Go 启动

前置条件：

- 已安装 Go
- 已启动 MySQL
- 已创建数据库 `go_user_system`
- 已配置 `.env`

创建数据库：

```sql
CREATE DATABASE go_user_system
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;
```

启动：

```bash
go mod download
go run ./cmd
```

## 配置说明

| 来源 | 作用 | 是否提交 |
| --- | --- | --- |
| `config.yml` | 非敏感默认配置 | 是 |
| `.env.example` | 本地和 Compose 配置模板 | 是 |
| `.env` | 本地真实配置 | 否 |
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

项目不使用 GORM `AutoMigrate`，启动时执行 `migrations/` 下尚未执行的 `.up.sql` 文件。

| 文件 | 作用 |
| --- | --- |
| `migrations/001_create_users.up.sql` | 创建 `users` 表 |
| `migrations/001_create_users.down.sql` | 删除 `users` 表 |
| `migrations/002_add_user_audit_fields.up.sql` | 增加 `last_login_at`、`deleted_at` |
| `migrations/002_add_user_audit_fields.down.sql` | 回滚用户审计字段 |

执行规则：

- 自动创建 `schema_migrations` 表
- 只执行后缀为 `.up.sql` 的文件
- 按文件名升序执行
- 每个版本执行前检查是否已记录，已执行则跳过
- 每个 migration 在事务中执行，失败不会记录版本

新增表结构变更时，新增一组 migration 文件：

```text
migrations/003_your_change.up.sql
migrations/003_your_change.down.sql
```

## 用户模型

| 字段 | 说明 |
| --- | --- |
| `id` | 用户主键 |
| `username` | 登录用户名，唯一索引 |
| `password_hash` | bcrypt 密码哈希，不对外返回 |
| `nickname` | 用户昵称 |
| `status` | 用户状态，`1` 正常，`2` 禁用 |
| `created_at` | 创建时间 |
| `updated_at` | 更新时间 |
| `last_login_at` | 最近一次登录时间 |
| `deleted_at` | GORM 软删除字段 |

## 测试与质量门禁

默认测试不依赖外部服务：

```bash
go test ./...
go vet ./...
go build -o bin/go-user-system ./cmd
```

或使用 Makefile：

```bash
make test
make vet
make build
```

集成测试需要专用 MySQL 测试库，数据库名必须包含 `test`，避免误删开发或生产库：

```sql
CREATE DATABASE go_user_system_test
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_0900_ai_ci;
```

PowerShell 示例：

```powershell
$env:TEST_DATABASE_DSN="root:your_mysql_password@tcp(127.0.0.1:3306)/go_user_system_test?charset=utf8mb4&parseTime=True&loc=Local"
go test ./... -run Integration -v
```

CI 位于 `.github/workflows/ci.yml`，包含：

1. `go mod download`
2. `go test ./...`
3. `go vet ./...`
4. `go build -o bin/go-user-system ./cmd`
5. `docker build -t go-user-system:ci .`

## 部署检查

生产部署前至少确认：

- `JWT_SECRET` 使用 32 位以上强随机字符串
- 生产数据库不使用 MySQL `root` 用户连接业务库
- `/readyz` 返回 200
- migration 已执行，`schema_migrations` 中存在版本记录
- 镜像以非 root 用户运行
- SIGTERM 能触发服务优雅关闭

完整清单见 `docs/deploy/production-checklist.md`。

## 常见问题

### 找不到 `.env`

从项目根目录运行：

```bash
go run ./cmd
```

或确认 `.env` 位于项目根目录，且保存为 UTF-8 无 BOM。

### JWT 初始化失败

检查：

```dotenv
JWT_SECRET=replace_with_a_32_plus_chars_random_secret
JWT_EXPIRE_HOURS=24
```

`JWT_SECRET` 不能为空，长度不能少于 32 个字符。

### Compose 中应用连不上数据库

容器内部使用 `DB_HOST=mysql`，本机直连使用 `DB_HOST=127.0.0.1`。优先检查：

```bash
docker compose ps
docker compose logs mysql
docker compose logs app
```
