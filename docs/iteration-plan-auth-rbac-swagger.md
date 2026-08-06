# 后端安全鉴权、RBAC 与规范化迭代计划

## 原始审查结论

以下是实施前的审查结论，用于说明本轮迭代的来源和缺口。

> 实施状态：已在 2026-07-14 按本计划完成后端迭代。具体代码改动、验证命令和风险记录见 `docs/operation-record-auth-rbac-swagger.md`。

## 实施复核

| 目标 | 实施结果 | 关键证据 |
| --- | --- | --- |
| JWT 双 Token | 已完成 | `internal/auth/token_manager.go`、`internal/service/auth.go`、`migrations/00003_create_refresh_tokens.sql` |
| 浏览器安全会话 | 已完成 | Access Token 仅返回响应体；Refresh Token 使用 HttpOnly Cookie，并支持轮换、吊销、登出和改密失效 |
| RBAC 五表 | 已完成 | `users`、`roles`、`permissions`、`user_roles`、`role_permissions` 及默认角色权限数据 |
| 接口级权限 | 已完成 | `internal/middleware/permission_middleware.go` 与路由权限码绑定 |
| 分层边界 | 已完成本轮范围 | 新增 Auth/RBAC 使用 Handler -> Service -> Repository -> Model；原用户 DAO 保留，后续可按模块渐进迁移 |
| 统一错误和响应 | 已完成 | `internal/apperror`、`internal/response`、统一 Handler 错误映射 |
| Swagger | 已完成 | `/swagger/index.html`、`docs/swagger.json`、`docs/swagger.yaml`、`make swagger` |
| 自动化验证 | 已完成 | `go test ./...`、`go vet ./...`、运行时 Cookie 刷新/登出与授权接口验证 |

| 目标 | 当前状态 | 证据 | 结论 |
| --- | --- | --- | --- |
| JWT 双 Token 无状态认证 | 只实现 Access Token | `internal/auth/token_manager.go` 只有 `GenerateAccessToken` / `ParseAccessToken`；`internal/response/users.go` 只返回 `access_token` | 未完成 |
| RBAC 五表模型与接口级鉴权 | 未实现 RBAC 表、模型、DAO/Repository、权限中间件 | `migrations` 只有 `users` 表及用户审计字段；路由只使用 `AuthMiddleware` | 未完成 |
| 分层架构 | 基本具备 Handler -> Service -> DAO -> Model | `internal/handler`、`internal/service`、`internal/dao`、`internal/model` 已分层 | 部分完成，命名和边界仍可继续规范 |
| 统一异常和响应结构 | 已实现基础统一响应和应用错误模型 | `internal/response/response.go`、`internal/apperror/app_error.go`、`internal/handler/error_handler.go` | 基本完成 |
| Swagger 接口文档 | 未实现 | `go.mod` 无 swaggo 依赖；项目无 Swagger 注解和生成文件 | 未完成 |

## 主要问题与修改建议

### 1. JWT 只有 Access Token，没有 Refresh Token

问题：登录接口只签发 `access_token`，没有 refresh token、刷新接口、登出吊销、Refresh Token 轮换机制。

原因：当前 `TokenManager` 只有 Access Token 的生成和解析逻辑，登录响应结构也只有 `AccessToken` 字段。Access Token 过期后用户只能重新登录；如果把 Access Token TTL 调长，又会降低安全性。

修改建议：拆分 Access/Refresh 两类 Token。Access Token 保持短有效期；Refresh Token 使用长有效期并持久化哈希值，刷新时执行轮换和旧 Token 吊销。

示例：

```go
type TokenPair struct {
    AccessToken           string
    RefreshToken          string
    AccessTokenExpiresIn  int64
    RefreshTokenExpiresIn int64
}

type TokenClaims struct {
    UserID    int64  `json:"user_id"`
    Username  string `json:"username"`
    TokenType string `json:"token_type"`
    JTI       string `json:"jti"`
    jwt.RegisteredClaims
}
```

建议新增接口：

```http
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
```

建议新增表：

```sql
CREATE TABLE refresh_tokens (
    id BIGINT NOT NULL AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    jti VARCHAR(64) NOT NULL,
    token_hash CHAR(64) NOT NULL,
    expires_at DATETIME(3) NOT NULL,
    revoked_at DATETIME(3) NULL DEFAULT NULL,
    replaced_by_jti VARCHAR(64) NULL DEFAULT NULL,
    created_at DATETIME(3) NULL DEFAULT NULL,
    updated_at DATETIME(3) NULL DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_refresh_tokens_jti (jti),
    UNIQUE KEY uk_refresh_tokens_hash (token_hash),
    KEY idx_refresh_tokens_user_id (user_id),
    KEY idx_refresh_tokens_expires_at (expires_at)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;
```

验收标准：

- 登录响应体返回 `access_token` 和过期时间，Refresh Token 通过 HttpOnly Cookie 返回。
- 受保护接口只接受 Access Token。
- `/auth/refresh` 只接受 Refresh Token（浏览器使用 Cookie，非浏览器客户端可使用可选 JSON），成功后吊销旧 Refresh Token并签发新双 Token。
- `/auth/logout` 可吊销当前 Refresh Token。
- 修改密码后可吊销该用户所有 Refresh Token。
- 单元测试覆盖过期、签名错误、token_type 错误、重复刷新、已吊销刷新。

### 2. 未实现标准 RBAC 五表模型

问题：当前只有 `users` 表，没有 `roles`、`permissions`、`user_roles`、`role_permissions`，也没有接口级权限中间件。

原因：现有 `AuthMiddleware` 只验证登录态，并把 `user_id`、`username` 写入 Gin Context。它不能表达“某个用户是否允许访问某个接口”。

修改建议：以 `users` 为第一张表，新增四张 RBAC 表，并增加权限校验中间件。权限建议使用稳定编码，例如 `user:read`、`user:update`、`admin:user:disable`。

示例表结构：

```sql
CREATE TABLE roles (
    id BIGINT NOT NULL AUTO_INCREMENT,
    code VARCHAR(64) NOT NULL,
    name VARCHAR(64) NOT NULL,
    created_at DATETIME(3) NULL DEFAULT NULL,
    updated_at DATETIME(3) NULL DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_roles_code (code)
);

CREATE TABLE permissions (
    id BIGINT NOT NULL AUTO_INCREMENT,
    code VARCHAR(128) NOT NULL,
    name VARCHAR(64) NOT NULL,
    method VARCHAR(16) NOT NULL,
    path VARCHAR(255) NOT NULL,
    created_at DATETIME(3) NULL DEFAULT NULL,
    updated_at DATETIME(3) NULL DEFAULT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uk_permissions_code (code),
    UNIQUE KEY uk_permissions_method_path (method, path)
);

CREATE TABLE user_roles (
    user_id BIGINT NOT NULL,
    role_id BIGINT NOT NULL,
    created_at DATETIME(3) NULL DEFAULT NULL,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE role_permissions (
    role_id BIGINT NOT NULL,
    permission_id BIGINT NOT NULL,
    created_at DATETIME(3) NULL DEFAULT NULL,
    PRIMARY KEY (role_id, permission_id)
);
```

权限中间件示例：

```go
func RequirePermission(rbacService RBACService, permissionCode string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, ok := c.Get("user_id")
        if !ok {
            response.Fail(c, http.StatusUnauthorized, response.CodeTokenUserMissing, "用户未认证")
            c.Abort()
            return
        }

        allowed, err := rbacService.HasPermission(c.Request.Context(), userID.(int64), permissionCode)
        if err != nil {
            response.Fail(c, http.StatusInternalServerError, response.CodeInternalError, "权限校验失败")
            c.Abort()
            return
        }
        if !allowed {
            response.Fail(c, http.StatusForbidden, response.CodePermissionDenied, "无权限访问")
            c.Abort()
            return
        }

        c.Next()
    }
}
```

路由接入示例：

```go
users.GET("/me",
    middleware.AuthMiddleware(tokenManager),
    middleware.RequirePermission(rbacService, "profile:read"),
    userHandler.MeHandler,
)
```

验收标准：

- Migration 包含 `users`、`roles`、`permissions`、`user_roles`、`role_permissions` 五表。
- 有初始化数据：`admin`、`user` 角色及基础权限。
- 新用户注册默认绑定 `user` 角色。
- 受保护接口除认证外，还校验权限码。
- 无权限返回 HTTP 403 和统一业务错误码。
- 测试覆盖有权限、无权限、未登录、权限查询异常。

### 3. 分层架构基本存在，但 Repository 边界不够标准

问题：当前项目是 Handler -> Service -> DAO -> Model，和目标的 Repository 命名不完全一致；DAO 是函数集合，通过 Service 内部接口包装，边界可读性一般。

原因：函数式 DAO 能用，但项目讲解时不如 Repository 接口清晰，也不利于后续 RBAC、Refresh Token、管理端能力扩展。

修改建议：短期可以保留 `dao`，但新增能力建议使用 `internal/repository`。如果要作为简历项目，建议逐步统一为 Repository 命名，并让 Service 依赖接口。

示例：

```go
type UserRepository interface {
    Create(ctx context.Context, user *model.User) error
    FindByID(ctx context.Context, id int64) (*model.User, error)
    FindByUsername(ctx context.Context, username string) (*model.User, error)
}

type GormUserRepository struct {
    db *gorm.DB
}
```

验收标准：

- Service 层不直接依赖 GORM 查询细节。
- Repository 方法命名以业务对象为中心，避免散落函数式调用。
- Handler 只负责参数解析、调用 Service、响应输出。

### 4. Swagger/OpenAPI 未接入

问题：没有 Swagger 依赖、注解、生成文件和文档路由。

原因：项目目前只有 `docs/http/test.http` 手工请求示例，不能稳定表达接口字段、响应结构、认证方式和错误码。

修改建议：接入 `swaggo/swag`、`gin-swagger`，给 Handler 增加注解，生成 `docs/swagger.json` 和 `docs/swagger.yaml`，并注册 `/swagger/*any`。

示例：

```go
// LoginHandler godoc
// @Summary 用户登录
// @Tags auth
// @Accept json
// @Produce json
// @Param body body request.LoginRequest true "登录参数"
// @Success 200 {object} response.Response{data=response.TokenAndUserInfoResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Router /api/v1/auth/login [post]
func (h *UserHandler) LoginHandler(c *gin.Context) {}
```

验收标准：

- `go.mod` 包含 Swagger 相关依赖。
- `cmd/main.go` 有项目级 Swagger 元信息注解。
- 所有公开接口都有 `@Summary`、`@Tags`、`@Param`、`@Success`、`@Failure`、`@Router`。
- Swagger 页面可访问并支持 Bearer Token。
- 文档中的请求/响应字段与实际结构体一致。

## 推荐迭代顺序

### 第 1 轮：补齐双 Token 认证

任务：

- 扩展配置：`accessTokenExpireMinutes`、`refreshTokenExpireHours`。
- 增加 Refresh Token 模型、Repository、Migration。
- 扩展 TokenManager：支持 `token_type`、`jti`、Access/Refresh 分开签发和解析。
- Service 生成 TokenPair；Handler 仅在响应体返回 Access Token，并通过 HttpOnly Cookie 写入 Refresh Token。
- 新增 `/auth/refresh`、`/auth/logout`。
- 修改密码后吊销用户 Refresh Token。
- 补齐 Token 单元测试、Handler 测试、Service 测试。

产出：

- 可讲清楚“短 Access + 长 Refresh + Refresh 轮换 + 服务端吊销”的安全方案。

### 第 2 轮：实现 RBAC 五表与接口级鉴权

任务：

- 增加 RBAC Migration、Model、Repository。
- 增加角色和权限初始化脚本或 seed。
- 增加 `RBACService.HasPermission`。
- 增加 `RequirePermission` 中间件。
- 给现有用户接口绑定权限码。
- 新增管理员接口：角色列表、权限列表、用户分配角色。
- 补齐 RBAC 中间件和 Service 测试。

产出：

- 可讲清楚“认证解决你是谁，授权解决你能做什么”。

### 第 3 轮：规范分层和错误响应

任务：

- 统一 Repository 命名和依赖方向。
- 补充 `CodePermissionDenied`、`CodeRefreshTokenInvalid`、`CodeRefreshTokenExpired` 等错误码。
- 统一 Handler 获取用户上下文的工具函数，减少重复代码。
- 对日志脱敏，避免输出 token、password、password_hash。

产出：

- 代码结构更适合简历展示和面试复盘。

### 第 4 轮：接入 Swagger 文档

任务：

- 接入 Swagger 依赖和生成命令。
- 给所有公开接口补注解。
- 注册 Swagger 路由。
- 在 README 补充文档访问方式。
- 将 `docs/http/test.http` 与 Swagger 字段保持一致。

产出：

- 前后端联调可以通过 Swagger 查看接口、字段和认证方式。

## 简历表述建议

完成上述迭代后，可以写成：

> 设计并实现基于 JWT Access/Refresh 双 Token 的无状态认证体系，支持 Refresh Token 轮换、服务端吊销和密码变更后的会话失效；基于 RBAC 五表模型实现角色、权限和用户授权关系，并通过 Gin 中间件完成接口级细粒度鉴权；项目采用 Handler、Service、Repository、Model 分层架构，统一封装错误模型、业务响应码和 Swagger 接口文档，提升前后端联调效率和可维护性。
