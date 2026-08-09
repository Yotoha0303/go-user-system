# 生产认证加固迭代计划

## 迭代背景

当前项目已经具备注册、登录、Access/Refresh 双 Token、Refresh Rotation、登出、改密、RBAC、结构化日志、测试和容器部署能力，但仍存在三类阻断公网生产部署的问题：

1. Access Token 吊销依赖进程内 `map`，无法跨 Pod 共享，并存在并发读写风险。
2. Refresh Token 虽然完成轮换，但旧 Token 重放只会被拒绝，不会吊销攻击者可能持有的后继 Token。
3. 登录接口没有分布式失败次数限制，无法有效抵御暴力破解、撞库和密码喷洒。

这些问题影响的是认证状态的一致性和攻击发生后的止损能力，因此优先级高于设备管理、MFA 和界面扩展。

## 迭代目标

### 1. 用户级认证版本

- 在 `users` 表增加单调递增的 `auth_version`。
- Access/Refresh Token 携带签发时的 `auth_version`。
- 鉴权和刷新时校验用户仍为启用状态，且 Token 版本等于数据库版本。
- 修改密码时，在同一数据库事务中更新密码、递增 `auth_version`、吊销全部 Refresh Token。

这使改密和禁用用户的行为以 MySQL 为一致性来源，不依赖 Redis 是否可用，也不受多副本影响。

### 2. Redis 认证基础设施

- 增加 Redis 配置、客户端初始化、健康检查和 Compose 服务。
- 使用 Redis 保存按 JTI 撤销的 Access Token，并按 Token 剩余寿命设置 TTL。
- 使用 Redis 实现按账号和 IP 维度的登录失败限流。
- Redis 启用但不可用时启动失败；运行中鉴权检查失败时采用 fail-closed。
- Redis 关闭仅作为本地开发和单元测试模式，使用线程安全的内存实现。

### 3. Refresh Token Family 重放检测

- Refresh Token 增加 `family_id`，轮换后的 Token 继承原 Family。
- 已吊销的旧 Refresh Token 再次出现时，认定为重放。
- 在同一事务中吊销该 Family 的全部有效 Refresh Token，再返回认证失败。
- 记录 `revoked_reason`，便于审计和问题定位。

## 非目标

- 登录设备列表和前端设备管理页面。
- MFA、Passkey、邮箱验证和密码找回。
- JWT 非对称密钥与 `kid` 轮换。
- 完整风控平台、验证码和异地登录判断。

这些能力有价值，但不应与本轮认证一致性修复混在一个变更中。

## 关键行为

| 场景 | 预期结果 |
| --- | --- |
| 修改密码 | 密码、`auth_version`、全部 Refresh 吊销原子提交；所有旧 Token 在全部实例失效 |
| 用户禁用 | 旧 Access 不能继续访问，旧 Refresh 不能换取新 Token |
| 当前会话登出 | Refresh 在 MySQL 吊销；当前 Access JTI 在 Redis 吊销 |
| Refresh 正常轮换 | 旧 Token 标记 replaced，新 Token 继承 Family |
| Refresh 重放 | 整个 Family 被吊销，攻击者与合法客户端都必须重新登录 |
| 登录连续失败 | 账号和 IP 任一维度达到阈值后返回 `429` 与 `Retry-After` |
| Redis 故障 | 已启用 Redis 的生产配置启动失败或鉴权失败关闭，不静默放行 |

## 验收标准

- [x] 同一个旧 Refresh Token 并发刷新时最多一个请求成功。
- [x] 已轮换 Refresh Token 重放后，其后继 Token 也无法继续刷新。
- [x] 修改密码后，任意实例都拒绝改密前签发的 Access/Refresh Token。
- [x] 被禁用用户不能访问普通接口、管理接口或刷新 Token。
- [x] 当前会话登出后，当前 Access Token 立即返回 401。
- [x] 登录失败达到阈值后返回 429，成功登录会清理账号维度失败计数。
- [x] Redis Key 不保存原始 Access Token、Refresh Token、密码或用户名明文。
- [x] `go test ./...`、`go test -race ./...`、`go vet ./...` 通过。
- [x] Migration 校验、配置测试和 Docker Compose 配置校验通过。

## 实施顺序

1. 数据库 migration、模型和 JWT Claim。
2. 用户认证状态校验与改密事务。
3. Refresh Family 与重放处理。
4. Redis 客户端、Access JTI 撤销和健康检查。
5. 登录失败限流、测试、部署配置和操作记录。

## 分支与基线

- 基线分支：`main`
- 基线提交：`791fb19 chore: record pre-hardening baseline`
- 迭代分支：`feat/production-auth-hardening`

## 实施状态

- 完成时间：2026-08-09
- 自动化结果：单元测试、真实 MySQL 集成测试、race、vet、golangci-lint、构建、Goose migration、Compose 配置和 Kubernetes 离线 schema 校验通过。
- 并发证据：锁语义单元测试和真实 MySQL 并发轮换、重放吊销、改密失效集成测试均已通过；集成测试仍由 `TEST_DATABASE_DSN` 控制并拒绝非测试库。
- 运行证据：启用本机 Redis 后启动后端二进制，`/ping`、`/livez`、`/readyz` 和 `/swagger/index.html` 均返回 200。
- 环境限制：Docker daemon 未运行，因此没有重复执行容器和镜像运行验证；Compose 静态配置已通过。
