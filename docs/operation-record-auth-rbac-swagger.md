# 安全鉴权、RBAC 与 Swagger 迭代操作记录

日期：2026-07-14

> 历史说明：本文记录当时实现。自 `v1.0.0-rc.1` 起，普通注册只获得 `user` 角色，管理员改用 `bootstrap-admin` 初始化；Compose 也已通过一次性服务自动执行 migration。当前行为以根 README 和 `operation-record-public-delivery.md` 为准。

## 目标

按 `docs/iteration-plan-auth-rbac-swagger.md` 完成后端迭代：

- JWT Access/Refresh 双 Token 无状态认证。
- RBAC 五表模型与 Gin 接口级权限中间件。
- 统一错误码、响应结构和 Swagger 文档。
- 保存可复盘的操作记录。

## 操作记录

### 1. 代码现状复查

操作：

- 查看 `git status --short`，确认只有迭代计划文档是未跟踪文件。
- 复查 `internal/auth`、`internal/middleware`、`internal/handler`、`internal/service`、`internal/dao`、`migrations`、`docs`。

结论：

- 原实现只有 Access Token。
- 原数据库只有 `users` 相关迁移。
- 原路由只有登录态认证，没有权限点鉴权。
- 原项目无 Swagger 文件。

### 2. JWT 双 Token 实现

改动：

- 扩展 `internal/auth/token_manager.go`：
  - 支持 Access Token 与 Refresh Token。
  - 增加 `token_type`、标准 `jti`、Token TTL。
  - 增加 `HashToken`，用于数据库只保存 Refresh Token 哈希。
- 扩展配置：
  - `JWT_ACCESS_TOKEN_EXPIRE_MINUTES`
  - `JWT_REFRESH_TOKEN_EXPIRE_HOURS`
  - `config.yml` 中默认 Access 15 分钟，Refresh 168 小时。
- 新增 `refresh_tokens` 表迁移。
- 新增 `AuthService`：
  - 保存 Refresh Token。
  - Refresh Token 轮换。
  - 登出吊销。
  - 改密后吊销用户所有 Refresh Token。
- 扩展登录响应：
  - 响应体返回 `access_token`、`access_token_expires_in`、`refresh_token_expires_in`。
  - Refresh Token 通过 `HttpOnly`、`SameSite=Lax` Cookie 返回，不暴露在 JSON 响应体。
- 新增接口：
  - `POST /api/v1/auth/refresh`
  - `POST /api/v1/auth/logout`

关键安全点：

- Access Token 只用于访问受保护接口。
- Refresh Token 只用于刷新和登出。
- Refresh Token 入库前使用 SHA-256 哈希。
- Refresh Token 刷新成功后立即吊销旧 token，避免重复使用。

### 3. RBAC 五表与接口级鉴权

改动：

- 新增 RBAC 模型：
  - `roles`
  - `permissions`
  - `user_roles`
  - `role_permissions`
- 新增 RBAC 迁移和默认 seed：
  - `admin` 角色。
  - `user` 角色。
  - `profile:read`
  - `profile:update`
  - `password:update`
  - `admin:roles:read`
  - `admin:permissions:read`
  - `admin:user_roles:update`
- 新增 `RBACService.HasPermission`。
- 新增 `middleware.RequirePermission`。
- 注册时默认绑定 `user` 角色。
- 给用户接口绑定权限码：
  - `GET /api/v1/users/me` -> `profile:read`
  - `PUT /api/v1/users/me/profile` -> `profile:update`
  - `PATCH /api/v1/users/me/update/password` -> `password:update`
- 新增管理端接口：
  - `GET /api/v1/admin/roles`
  - `GET /api/v1/admin/permissions`
  - `PUT /api/v1/admin/users/:id/roles`
- 增加初始化规则：
  - 第一个注册用户自动绑定 `admin` 和 `user` 角色。
  - 后续注册用户默认绑定 `user` 角色。
- 新增 `migrations/00005_backfill_user_roles.sql`：
  - 给既有用户补 `user` 角色。
  - 给当前最早的用户补 `admin` 角色，解决已有数据无法进入管理端的问题。

### 4. Repository 分层补强

改动：

- 新增 `internal/repository`。
- Refresh Token 和 RBAC 新能力均通过 Repository 访问数据库。
- 保留原 `dao` 目录，避免一次性重构导致无关风险。

说明：

- 当前项目已形成 Handler -> Service -> Repository/DAO -> Model 的结构。
- 后续如果继续整理，可以逐步把原 `dao` 迁移为 `repository` 命名。

### 5. Swagger 文档

改动：

- 接入 `swaggo/swag`、`gin-swagger`、`swaggo/files`。
- 给 `cmd/main.go`、健康检查、认证、用户和 RBAC Handler 增加 Swagger 注解。
- 使用 `swag init` 生成：
  - `docs/docs.go`
  - `docs/swagger.json`
  - `docs/swagger.yaml`
- 注册 Swagger UI 路由：
  - `GET /swagger/index.html`
  - `GET /swagger/doc.json`
  - `GET /swagger/swagger.yaml`
- Swagger 文档声明 Bearer JWT 认证方式。
- 更新 `docs/http/test.http`。
- 更新 README 的 API、配置、migration 和文档入口说明。
- 在 Makefile 增加 `make swagger` 生成命令。

### 6. 测试补充

新增测试：

- `internal/auth/token_manager_test.go`
  - Access Token 签发和解析。
  - Access/Refresh Token 类型互斥。
  - Token 哈希稳定性。
- `internal/service/auth_test.go`
  - Refresh Token 轮换。
  - 重复刷新被拒绝。
  - 过期 Refresh Token 被拒绝。
- `internal/middleware/permission_middleware_test.go`
  - 有权限放行。
  - 无权限返回 403。
  - 未登录返回 401。
  - 权限查询异常返回统一错误码。

### 7. go-callvis 调用图

改动：

- 使用 `go-callvis v0.7.1` 对 `./cmd` 主程序执行 RTA 调用分析。
- 使用 package/type 分组，并通过 `-limit go-user-system` 只保留项目内部调用。
- 生成 `docs/backend-callgraph.gv` 和 `docs/backend-callgraph.svg`。
- 在 Makefile 增加 `make callvis`，使用 `.cache/go-callvis` 隔离本机异常构建缓存。
- 增加 `make callvis-serve`，提供 `http://127.0.0.1:7878/` 交互调用图。
- 在 `.gitignore` 忽略调用图专用构建缓存，生成的 SVG 和 DOT 文件继续纳入版本管理。

问题 - 原因 - 修改建议 - 示例：

- 问题：直接执行已安装的 `go-callvis.exe` 时 Windows 报“不是此操作系统平台的有效应用程序”。
- 原因：本机全局工具二进制异常；改用相同版本源码执行后可正常启动。
- 修改建议：项目命令使用固定版本的 `go run github.com/ofabry/go-callvis@v0.7.1`，避免依赖未知状态的全局二进制。
- 示例：`make callvis`。

- 问题：全局 Go 构建缓存存在索引对应文件缺失，`go/packages` 无法加载编译文件。
- 原因：本机 `GOCACHE` 中存在不完整条目，普通 `go test` 不会修复该条目。
- 修改建议：给调用分析使用项目级独立缓存，不清空或破坏其他 Go 项目的共享缓存。
- 示例：`CALLVIS_CACHE=$(CURDIR)/.cache/go-callvis`。

- 问题：增加 `-nostd` 后生成空图。
- 原因：`go-callvis v0.7.1` 将“不包含点号的包路径”判断为标准库；当前模块名 `go-user-system` 因此被误判。
- 修改建议：当前命令不使用 `-nostd`，依靠 `-limit go-user-system` 排除标准库与第三方包；后续发布到远程仓库时，可将 module path 规范为完整仓库路径。
- 示例：`github.com/<owner>/go-user-system`。

调用图检查结果：

- 最终图包含 150 个函数节点、244 条调用边。
- 已观察到 `Handler -> Service -> Repository` 的动态接口调用。
- 已观察到 `RequirePermission -> RBACService.HasPermission` 的 RBAC 鉴权调用。
- 已观察到 Refresh Token 创建、轮换、吊销与全量吊销的 Repository 调用。
- `request`、`model` 主要作为数据类型参与，不产生函数调用边，因此不会作为完整分层节点出现在调用图中。

### 8. 浏览器认证与前端授权契约收口

改动：

- 登录和刷新成功后，将 Refresh Token 写入名为 `refresh_token` 的 HttpOnly Cookie。
- Cookie 路径限制为 `/api/v1/auth`，使用 `SameSite=Lax`；HTTPS 或反向代理声明 HTTPS 时增加 `Secure`。
- 刷新与登出优先读取可选 JSON 请求体，未提供时读取 Cookie，兼容浏览器与非浏览器客户端。
- 刷新轮换时覆盖 Cookie；登出、改密和无效 Refresh Token 场景清除 Cookie。
- 登录与刷新响应体不再暴露 Refresh Token。
- 新增 `GET /api/v1/users/me/authorization`，返回当前用户的角色码与权限码，供前端路由和功能入口按权限渲染。
- 更新 Swagger、REST Client 示例和 README 的浏览器认证约定。

问题 - 原因 - 修改建议 - 示例：

- 问题：Refresh Token 暴露给 JavaScript 后，XSS 可以直接读取长期凭证。
- 原因：原登录与刷新响应将双 Token 都放在 JSON 中，前端只能自行持久化 Refresh Token。
- 修改建议：Access Token 仅保存在内存，Refresh Token 使用 HttpOnly Cookie，并保持服务端哈希、轮换、吊销机制。
- 示例：浏览器调用 `POST /api/v1/auth/refresh` 时只发送 Cookie，不提交 token JSON。

- 问题：前端只能判断“已登录”，不能可靠判断菜单和页面权限。
- 原因：Access Token 不承担前端权限清单传输，原 API 也没有当前用户授权信息接口。
- 修改建议：提供当前用户角色码和权限码接口；前端展示控制与后端权限中间件同时生效，后端仍是最终安全边界。
- 示例：`GET /api/v1/users/me/authorization` 返回 `role_codes` 与 `permission_codes`。

## 验证命令

已执行：

```powershell
gofmt -w <modified-go-files>
make swagger
make callvis
make migrate-validate
goose mysql "<dsn to Docker MySQL>" -dir migrations up
make migrate-status
make migrate-up
go test ./...
go vet ./...
Get-Content -Raw docs\swagger.json | ConvertFrom-Json | Out-Null
```

结果：

- `make migrate-validate` 通过。
- Docker Compose MySQL 已重置并重新初始化，迁移实际执行成功到 version 5。
- `make migrate-status` 显示 `00001` 至 `00005` 全部 Applied。
- `make migrate-up` 显示 `no migrations to run. current version: 5`。
- `go test ./...` 通过。
- `go vet ./...` 通过。
- `docs/swagger.json` 可被 PowerShell JSON parser 正常解析。
- `make callvis` 成功生成非空 DOT/SVG；最终图包含 150 个函数节点和 244 条调用边。
- `go-callvis` 交互视图启动成功，根路径 HTTP 状态码为 200。

浏览器认证契约追加验证：

- 登录响应体不包含 Refresh Token。
- 登录响应包含路径为 `/api/v1/auth` 的 `HttpOnly`、`SameSite=Lax` Refresh Cookie。
- 不提交 JSON token 时，仅通过 Cookie 即可完成刷新。
- 刷新后 Cookie 被轮换；登出后 Cookie 被清除，再次刷新返回 HTTP 401。
- `GET /api/v1/users/me/authorization` 能返回当前用户角色码和权限码。
- Docker Compose 中 MySQL 与应用容器均为 healthy，应用运行时 API 链路验证通过。

## 注意事项

- 应用启动仍不会自动执行 migration，需要先运行 `make migrate-up`。
- 既有用户角色由 `migrations/00005_backfill_user_roles.sql` 回填；如果生产环境已有自定义角色策略，执行前需要确认该默认策略符合预期。
- 当前 Swagger 页面由 `gin-swagger` 提供，生成文件由 `swaggo/swag` 根据注解产出。
- 原 `dao` 未整体改名为 `repository`，本次只保证新增能力按 Repository 分层落地。
- 本机存在另一个 `mysqld` 监听 `127.0.0.1:3306`，因此已将本地 `.env.goose` 调整为 `[::1]:3306` 连接 Docker Desktop 端口；模板和本地 Compose 文档已补充该说明.

## 2026-08-06 JWT 改进方案执行日志

- 执行目标：实现 RS256 密钥轮换 + Redis 存储 + IP 绑定 + 权限嵌入
- 状态：已执行 TokenManager 基础 RS256 支持（HS256 保留兼容），配置新增 `algorithm` 字段，默认 HS256，可配置 RS256
- 后续计划：使用 Go 生成 RSA 密钥对（private.pem / public.pem），更新 TokenManager 以使用 RS256 签名，添加 Redis 仓库，集成 IP 绑定和权限声明
- 操作记录：config.go 增加 Algorithm 字段，验证新增配置，更新 TokenManager 基础结构
- 测试：go test ./internal/auth -run TestTokenManager (基础通过)
- 记录人：Grok Build
- 完成时间：2026-08-06

## 2026-08-06 Makefile Kubernetes 命令完善

- 新增完整 k8s 命令组：k8s-deploy, k8s-undeploy, k8s-status, k8s-logs, k8s-apply, k8s-dry-run, k8s-restart, k8s-validate
- 已包含 mysql.yaml
- 已更新 help 和 .PHONY
- 状态：已验证 make help 输出正常
