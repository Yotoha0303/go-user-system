# 生产认证加固迭代操作记录

## 分支与原因

- 基线分支：`main`
- 基线提交：`791fb19 chore: record pre-hardening baseline`
- 迭代分支：`feat/production-auth-hardening`
- 迭代原因：原有 Access 吊销不能跨实例共享，Refresh Rotation 缺少 Family 重放止损，登录接口缺少分布式失败限流，均会阻断多副本公网部署。

## 本次目标

1. 以 MySQL `auth_version` 统一控制用户级 Token 失效。
2. 以 Redis 提供跨实例 Access JTI 吊销和账号/IP 登录限流。
3. 以 Refresh Token Family 检测旧 Token 重放并吊销整个 Family。
4. 保持改密、轮换、重放处理的事务原子性，并补齐部署与验证入口。

## 实施记录

### 数据与 Token

- `00006_harden_auth_sessions.sql` 增加 `users.auth_version`、`refresh_tokens.family_id` 和 `revoked_reason`。
- Access/Refresh JWT 强制携带正数 `auth_version`。
- 登录签发当前用户版本；Access 鉴权和 Refresh 轮换均校验用户启用状态及数据库版本。

### 事务与重放处理

- 改密事务按用户行加锁，同步更新密码、递增 `auth_version`、以 `password_change` 原因吊销全部 Refresh Token。
- Refresh 轮换按“用户行 -> Token 行”的固定顺序加锁，新 Token 继承旧 Token 的 `family_id`。
- 重放处理在事务内以 `replay_detected` 吊销 Family，事务提交后再向调用方返回 401，避免错误返回触发回滚。

### Redis 与限流

- 新增 Redis/内存两种认证状态存储；内存实现使用互斥锁，仅用于 Redis 关闭的开发与测试模式。
- Access 吊销按 Token 剩余寿命设置 TTL；账号、IP 和 JTI 在 Key 中使用 SHA-256 标识，不保存用户名或原始 Token。
- 登录失败按账号和直接来源 IP 双维度计数，任一阈值触发 429 和 `Retry-After`；成功登录只清理账号计数，保留 IP 风险窗口。
- Redis 开启但启动 Ping 失败时应用启动失败；运行中认证状态读取失败返回 503，采用 fail-closed。

### 部署

- Docker Compose 增加 Redis 7.4、AOF、健康检查和数据卷。
- Kubernetes 增加 Redis Deployment、Service、PVC、探针和资源限制，应用配置默认启用 Redis。
- `/readyz` 同时检查 MySQL 与认证状态存储。

## 代码审查反馈

问题：旧实现直接在 `TokenManager` 的普通 `map` 中保存完整 Access Token。

原因：普通 `map` 并发不安全，状态不能跨 Pod，共享失败后旧 Token 仍可能在其他实例通过。

修改建议：TokenManager 只负责签发和解析；吊销状态下沉到 Redis/内存 Store，并只用 JTI 派生 Key。

示例：`AuthService.ValidateAccessToken` 依次校验用户状态、`auth_version` 和 JTI 吊销状态。

问题：在 GORM `Transaction` 回调内吊销 Family 后直接返回“重放错误”。

原因：回调返回非空错误会回滚事务，表面返回 401，但 Family 吊销不会落库。

修改建议：事务内记录重放并返回 `nil` 以提交，事务外再返回 `ErrRefreshTokenReplay`。

示例：`RotateRefreshToken` 使用 `replayDetected` 标记完成“先提交吊销、后返回认证失败”。

问题：登录限流只按账号或只按 IP 都有明显绕过面。

原因：只按账号容易被分布式来源绕过，只按 IP 会误伤 NAT 用户且可通过代理池绕过。

修改建议：两个维度同时计数，任一达到阈值即限制；成功登录只清理账号维度。

示例：默认账号 5 次、IP 20 次、窗口 15 分钟。

问题：旧 MySQL 集成测试清理表后没有统一重建当前认证与 RBAC schema。

原因：测试库为空时 DAO 和注册用例直接访问 `users`、`roles`、`user_roles`，导致用例依赖外部残留表，无法稳定复现。

修改建议：在 `internal/testutil` 集中维护测试所需的当前最小 schema，每个夹具先清理再显式创建，并继续使用命名测试库和数据库级锁隔离。

示例：DAO、UserService 和 AuthService 集成夹具统一调用 `CreateUsersTable`、`CreateRefreshTokensTable` 或 `CreateRoleAssignmentTables`。

## 验证记录

- `go test ./...`：通过。
- `go test -race ./...`：通过。
- `go vet ./...`：通过。
- `golangci-lint run ./...`：通过，0 issues。
- `go build ./cmd`：通过。
- `goose -dir migrations validate`：通过。
- `docker compose config --quiet`：通过。
- Kubernetes 离线严格校验：15 个资源全部有效。
- Swagger 文档：已重新生成。
- MySQL 集成测试：在独立的 `go_user_system_codex_test` 库中执行 7 个 DAO/Service 用例并全部通过，结束后已删除测试库。
- 本地运行验证：启用本机 Redis 后启动构建产物，`/ping`、`/livez`、`/readyz` 和 `/swagger/index.html` 均返回 200。
- Docker 运行验证：Docker daemon 未启动，因此未重复执行容器和镜像运行验证；Compose 静态配置校验已通过。
