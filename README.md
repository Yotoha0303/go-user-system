# go-user-system（Go 认证与基础用户系统）

基于 Go + Gin + GORM + MySQL 实现的用户认证系统，支持用户注册、登录、bcrypt 密码哈希、JWT 鉴权、当前用户信息查询和昵称修改。

当前项目已经补充基础工程化能力：配置分层、Docker Compose 本地部署、健康检查、数据库 migration、自动化测试、CI 质量门禁和部署检查清单。

## 1. 功能概览

- 用户注册、登录、当前用户查询、昵称修改
- bcrypt 密码哈希存储
- JWT 签发、解析和鉴权中间件
- 统一响应结构和业务错误码
- `/ping`、`/livez`、`/readyz` 健康检查
- Dockerfile + Docker Compose 本地部署
- `go test`、`go vet`、`go build`、Docker 镜像构建 CI
- 数据库建表 migration 和生产部署前检查清单

## 2. 项目结构

```text
cmd/                    程序入口
config/                 配置加载：config.yml + 环境变量覆盖
database/               MySQL / GORM 初始化
router/                 路由注册和分组
internal/
  apperror/             应用错误模型
  handler/              HTTP 处理器
  middleware/           JWT 鉴权中间件
  service/              业务逻辑
  dao/                  数据访问
  model/                数据模型
  request/              请求 DTO
  response/             响应结构和业务错误码
  utils/                JWT 工具
docs/
  http/                 REST Client 手动测试
  sql/                  本地 SQL 辅助脚本
  deploy/               本地部署与生产检查文档
migrations/             数据库迁移脚本
```

## 3. 依赖注入说明

项目不再通过 `global.DB` 暴露数据库连接，而是在 `cmd/main.go` 初始化 `*gorm.DB` 后传入 `router.SetupRouter(db)`，再由 router 统一装配 service 和 handler。

依赖链：

```text
cmd/main.go
  -> router.SetupRouter(db)
  -> service.NewUserService(db)
  -> handler.NewUserHandler(userService)
  -> dao 使用 service 持有的 db 执行查询
```

问题：全局 DB 会让 service 隐式依赖外部状态。

原因：调用 service 方法时看不出它依赖数据库，测试时也必须提前修改全局变量。

修改建议：通过构造函数显式传入依赖。

示例：

```go
userService := service.NewUserService(db)
userHandler := handler.NewUserHandler(userService)
```

## 4. 配置说明

### 配置来源

| 文件 / 来源 | 作用 | 是否提交 |
| --- | --- | --- |
| `config.yml` | 非敏感默认配置 | 是 |
| `.env.example` | 本地和 Compose 的主配置模板 | 是 |
| `.env` | 本地真实配置，应用和 Compose 默认读取 | 否 |
| shell 环境变量 | CI、服务器或容器运行时注入 | 否 |

启动时会从当前工作目录开始向上查找 `.env` 和 `config.yml`。因此无论从项目根目录执行 `go run ./cmd`，还是 IDE 以 `cmd/` 作为工作目录启动，都能加载项目根目录下的 `.env`。

### 重复配置处理

问题：如果本地运行和 Compose 分别维护不同 env 模板，容易出现同一个配置在多处不一致。

原因：Docker Compose 默认会读取项目根目录 `.env`，应用本地运行也会通过 `godotenv` 读取同一个 `.env`，因此维护两个模板会导致配置来源不清晰。

修改建议：统一只维护 `.env.example` 一个模板。日常开发复制 `.env.example` 为 `.env`，不要维护多份真实配置。

示例：

```bash
cp .env.example .env
```

Windows PowerShell：

```powershell
Copy-Item .env.example .env
```

`.env` 至少需要配置：

```dotenv
DB_PASSWORD=your_mysql_password
JWT_SECRET=replace_with_a_32_plus_chars_random_secret
```

Compose 会使用同一个 `DB_PASSWORD` 作为 MySQL root 密码和应用连接密码，并使用 `JWT_SECRET` 初始化 JWT 签名密钥。

`compose.yaml` 中的 `DB_HOST=mysql`、`DB_USER=root`、`DB_NAME=go_user_system` 是容器网络下的固定覆盖值，不是和 `config.yml` 重复维护；本地直接运行 Go 服务时仍使用 `config.yml` 和 `.env`。

## 5. 快速启动

### 4.1 Docker Compose 启动（推荐）

前置条件：

- 已安装 Docker Desktop，或 Docker Engine + Docker Compose
- 已复制 `.env.example` 为 `.env`
- `.env` 中已配置 `DB_PASSWORD` 和 `JWT_SECRET`
- 本地端口 `8082`、`3306` 未被占用

启动：

```bash
docker compose up -d --build
```

或：

```bash
make compose-up
```

查看日志：

```bash
docker compose logs -f app
```

停止：

```bash
docker compose down
```

删除本地 MySQL 数据卷：

```bash
docker compose down -v
```

更完整的 Compose 说明见 `docs/deploy/local-compose.md`。

### 4.2 本地 Go 启动

前置条件：

- 已安装 Go `1.25.5`
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

或：

```bash
make run
```

## 6. 健康检查

| 接口 | 作用 | 是否检查 DB |
| --- | --- | --- |
| `GET /ping` | 兼容旧的基础健康检查 | 否 |
| `GET /livez` | 应用进程存活检查 | 否 |
| `GET /readyz` | 服务就绪检查 | 是 |

验证：

```bash
curl http://127.0.0.1:8082/ping
curl http://127.0.0.1:8082/livez
curl http://127.0.0.1:8082/readyz
```

`compose.yaml` 中的 app healthcheck 使用 `/readyz`，避免数据库不可用时误判服务健康。

## 7. API 概览

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| `GET` | `/ping` | 基础健康检查 | 否 |
| `GET` | `/livez` | 应用存活检查 | 否 |
| `GET` | `/readyz` | 服务就绪检查 | 否 |
| `POST` | `/api/v1/auth/register` | 用户注册 | 否 |
| `POST` | `/api/v1/auth/login` | 用户登录 | 否 |
| `GET` | `/api/v1/users/me` | 当前用户信息 | 是 |
| `PUT` | `/api/v1/users/me/profile` | 修改当前用户昵称 | 是 |

手动测试文件见 `docs/http/test.http`。

## 8. 数据库迁移

应用启动时不再使用 GORM `AutoMigrate`，而是自动执行 `migrations/` 下尚未执行过的 `.up.sql` 文件。执行记录保存在数据库表 `schema_migrations` 中，避免同一个 migration 被重复执行。

| 文件 | 作用 |
| --- | --- |
| `migrations/001_create_users.up.sql` | 创建 `users` 表 |
| `migrations/001_create_users.down.sql` | 删除 `users` 表 |

迁移执行规则：

- 启动时自动创建 `schema_migrations` 表
- 创建 `schema_migrations` 前会先检查表是否已存在，已存在则不重复创建
- 只执行后缀为 `.up.sql` 的文件
- 按文件名升序执行，例如 `001_...`、`002_...`
- 执行每个 migration 前会检查 `schema_migrations.version`，已执行版本会跳过
- 执行成功后写入 `schema_migrations.version`
- Docker 镜像会复制 `migrations/` 目录，容器启动时同样自动执行

问题：README 中不再重复粘贴完整建表 SQL。

原因：SQL 内容应以 `migrations/` 为唯一权威来源，避免 README 和脚本不一致。

修改建议：查看或修改表结构时，以 migration 文件为准。

示例：

```bash
cat migrations/001_create_users.up.sql
```

新增表结构变更时，新增一组 migration 文件：

```text
migrations/002_your_change.up.sql
migrations/002_your_change.down.sql
```

## 9. 工程化命令

| 命令 | 作用 |
| --- | --- |
| `make run` | 本地运行 Go 服务 |
| `make test` | 执行 `go test ./...` |
| `make vet` | 执行 `go vet ./...` |
| `make build` | 构建 Linux 二进制到 `bin/go-user-system` |
| `make docker-build` | 构建 Docker 镜像 |
| `make compose-up` | 启动 Compose 环境 |
| `make compose-down` | 停止 Compose 环境 |
| `make compose-logs` | 查看 app 容器日志 |
| `make ci` | 串联 test、vet、build、docker-build |

注意：`Makefile` 中的 `build`、`clean` 使用类 Unix 命令。Windows 原生 PowerShell 下建议使用 Git Bash、WSL 或直接运行 Go / Docker 命令。

## 10. CI 质量门禁

CI 配置位于 `.github/workflows/ci.yml`。

执行流程：

1. `actions/checkout@v4`
2. `actions/setup-go@v5`
3. `go mod download`
4. `go test ./...`
5. `go vet ./...`
6. `go build -o go-user-system ./cmd`
7. `docker build -t go-user-system:ci .`

本地可用以下命令提前验证：

```bash
go test ./...
go vet ./...
go build -o bin/go-user-system.exe ./cmd
```

## 11. 部署检查

生产部署前不要直接复用本地开发配置。检查清单见：

- `docs/deploy/production-checklist.md`

关键要求：

- `JWT_SECRET` 使用 32 位以上强随机字符串
- 生产环境不使用 MySQL `root` 用户连接业务库
- `.env`、`.env.*`、`config.local.yml` 不提交到 Git
- `/readyz` 必须返回 200
- 注册、登录、鉴权、修改昵称核心链路验证通过

## 12. 本次重复内容清理结果

问题：README 原先同时承担项目介绍、完整 SQL、详细接口说明、手动测试流程、部署说明和设计复盘，内容重复且容易过期。

原因：多个文档都在描述同一件事，例如 README 和 `docs/deploy/local-compose.md` 都写 Compose 步骤，README 和 `migrations/` 都写建表 SQL。

修改建议：README 只作为入口文档，细节放到专门文件。

示例：

- SQL 结构以 `migrations/` 为准
- 本地 Compose 细节以 `docs/deploy/local-compose.md` 为准
- 生产部署检查以 `docs/deploy/production-checklist.md` 为准
- 手动接口测试以 `docs/http/test.http` 为准

## 13. 简历表达参考

基于 Go、Gin、GORM、MySQL 实现用户认证系统，支持注册、登录、JWT 鉴权、用户信息查询和资料修改。项目采用 handler / service / dao / model 分层结构，使用 bcrypt 进行密码哈希存储，并通过统一响应结构和应用错误模型规范接口返回。工程化侧补充 Docker Compose 本地部署、健康检查、数据库 migration、自动化测试和 CI 质量门禁，具备从开发到部署验证的基础闭环。
