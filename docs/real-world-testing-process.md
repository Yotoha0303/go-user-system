# 真实业务场景测试流程与记录规范

| 字段 | 内容 |
| --- | --- |
| Owner | 项目维护者；生产验收时由发布负责人和验证人共同确认 |
| Status | Active for local/acceptance; Draft for production |
| Last reviewed | 2026-08-30 |
| Applies to | `main@2b6a459`、`v1.0.0-rc.3` |
| Scope | 注册登录、Token 会话、个人资料、密码修改、RBAC、依赖故障、发布与回滚 |

## 1. 目标和验证边界

本流程用于验证系统在真实用户、管理员和运维场景下是否满足业务契约，而不只是验证函数能否返回预期值。测试结论必须区分以下层级：

1. 单元测试通过：只证明被测函数、组件和模拟依赖下的行为。
2. 集成测试通过：证明专用测试库上的 SQL、事务、约束和认证状态行为。
3. 完整栈测试通过：证明 MySQL、migration、Redis、后端、前端和浏览器能够共同工作。
4. 类生产验收通过：证明真实 TLS、反向代理、多副本、监控、备份恢复和回滚行为。
5. 生产验证通过：只由目标生产环境的实际证据支持，不能由本地或 CI 结果代替。

本文定义测试方法和执行记录格式，不代表文中用例已经在当前环境执行。每次执行都必须新建测试记录并附证据。

## 2. 安全边界

- 只允许连接专用测试数据库；数据库名必须包含 `test`。
- 禁止使用生产用户数据、生产密码、生产 Token、生产 Cookie 或生产连接串。
- 执行故障注入前确认环境是本地或独立验收环境，并通知同一环境的使用者。
- 默认使用 `docker compose down` 保留数据卷。`docker compose down -v` 会删除 MySQL 和 Redis 数据卷，只能在确认数据可丢弃后执行。
- 日志、截图、Trace 和测试报告不得包含密码、JWT、Refresh Cookie、Token Hash、私钥或未脱敏用户数据。
- 真实生产环境不执行破坏性故障注入；生产只进行经审批的冒烟、观测和预案演练。

## 3. 测试角色和数据

每轮完整业务测试至少创建以下数据，用户名增加时间戳或随机后缀，防止并发冲突：

| 角色/对象 | 用途 |
| --- | --- |
| `anonymous` | 未登录访问、注册和登录失败场景 |
| `user_a` | 正常注册、资料、改密、双会话和登出 |
| `user_b` | 用户间数据和权限隔离 |
| `disabled_user` | 禁用账户登录和会话校验 |
| `admin` | 一次性管理员初始化、角色和权限管理 |
| Browser A / Browser B | 同一用户跨浏览器或隔离上下文会话 |
| Tab A / Tab B | 同一 Cookie 会话下的并发 Refresh |

推荐密码边界数据：11 个字符、12 个字符、72 个 UTF-8 字节、73 个 UTF-8 字节，以及多字节 Unicode 输入。当前行为以代码和 Swagger 契约为准；策略变化时必须同步修改前端、后端、测试和文档。

## 4. 测试方法

| 方法 | 适用范围 | 关键做法 |
| --- | --- | --- |
| 等价类和边界值 | 用户名、密码、昵称、ID、JSON | 覆盖合法、空值、临界值、超限和非法格式 |
| 状态迁移 | Refresh、登出、改密、禁用账户 | 同时检查操作前后 HTTP、MySQL、Redis 和客户端状态 |
| 权限矩阵 | 匿名、普通用户、管理员 | 不依赖前端隐藏；直接请求后端接口验证 401/403/成功路径 |
| 并发测试 | Refresh Rotation、登录限流 | 使用并发请求、双标签页和 `go test -race` 验证竞态与唯一成功者 |
| 故障注入 | MySQL、Redis、网络、进程退出 | 一次只改变一个依赖，记录故障前、故障中和恢复后的信号 |
| 黑盒契约测试 | API、浏览器、Cookie | 以 Swagger、HTTP 状态、响应结构和用户可见结果为准 |
| 白盒不变量检查 | Token Family、`auth_version`、RBAC | 检查数据库/Redis 状态，但不把实现细节当成唯一业务断言 |
| 探索性测试 | 浏览器兼容、响应式、异常操作顺序 | 记录操作路径、观察结果、截图和可复现步骤 |

关键业务用例统一采用 Given/When/Then，并至少检查：用户可见结果、HTTP 契约、持久化状态、认证状态、日志与指标。

登录限流会修改共享 Redis 的账号/IP 计数。自动化套件必须为它使用独立 Redis，或把它放在全部正常登录场景之后；重复运行时不得复用仍处于限流窗口内的 Redis 状态。

## 5. 分层执行流程

### 5.1 执行前确认

在 `go-user-system-backend` 目录执行：

```powershell
git status -sb
git log -1 --oneline --decorate
go version
node --version
docker version
docker compose version
```

记录当前提交、未提交修改、工具版本、测试环境、执行人和开始时间。存在无关工作区修改时不得清理或覆盖。

### 5.2 快速门禁：每次提交

```powershell
go test -count=1 ./...
go test -race -count=1 ./...
go vet ./...
make lint
make security
make migrate-validate
make observability-validate

npm --prefix frontend ci
npm --prefix frontend run check
```

要求：

- 正式记录使用 `-count=1`，避免只命中 Go 测试缓存。
- 测试警告必须写入记录；命令退出 0 不表示警告可以忽略。
- `govulncheck`、`npm audit`、lint、test、build 和 migration validate 均为门禁。
- `go test ./...` 退出 0 不代表 MySQL 集成测试已执行，必须检查是否存在 `SKIP`。

### 5.3 真实 MySQL 集成测试：每次合并

使用独立 MySQL 8.4 测试实例，并设置数据库名包含 `test` 的 DSN：

```powershell
$env:TEST_DATABASE_DSN = "tester:<test-password>@tcp(127.0.0.1:3307)/go_user_system_test?charset=utf8mb4&parseTime=True&loc=Local"
go test -count=1 -run Integration -v ./internal/dao ./internal/service
```

通过标准：

- DAO 的创建、读取、更新、不存在记录和取消上下文场景实际执行并通过。
- Service 的注册登录、禁用账户、管理员初始化、并发 Refresh 和改密失效实际执行并通过。
- 输出中没有 `set TEST_DATABASE_DSN to run MySQL integration tests` 的跳过信息。
- 测试结束后没有连接到非测试库，也没有残留真实凭据。

执行完毕后清理当前 PowerShell 会话中的 DSN：

```powershell
Remove-Item Env:TEST_DATABASE_DSN -ErrorAction SilentlyContinue
```

### 5.4 完整 Compose 和浏览器测试：候选版本

先确认测试变量只包含本地测试凭据。共享开发机使用
`compose.test.yaml` 隔离项目名、镜像标签、数据卷和 Docker 子网，避免覆盖同机其他项目：

```powershell
$env:DB_ROOT_PASSWORD = "<local-test-root-password>"
$env:DB_PASSWORD = "<local-test-app-password>"
$env:JWT_SECRET = "<local-test-jwt-secret-at-least-32-characters>"
$env:FRONTEND_PORT = "18080"
$env:BACKEND_PORT = "18082"
$env:TEST_COMPOSE_SUBNET = "172.31.50.0/24"

$compose = @("compose", "-f", "compose.yaml", "-f", "compose.test.yaml", "-p", "go-user-system-test")
docker @compose config --quiet
docker @compose up -d --build --wait
docker @compose ps

Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8082/ping
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8082/livez
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:8082/readyz

$env:E2E_BASE_URL = "http://127.0.0.1:18080"
npm --prefix frontend run test:e2e
```

测试结束后收集服务状态和必要日志：

```powershell
docker @compose ps
docker @compose logs --since=30m app migrate mysql redis frontend
docker @compose down

Remove-Item Env:E2E_BASE_URL -ErrorAction SilentlyContinue
Remove-Item Env:TEST_COMPOSE_SUBNET -ErrorAction SilentlyContinue
```

现有 Playwright 主流程覆盖注册、登录、双标签页恢复和登出，且在 Desktop Chrome 与 Pixel 7 配置上运行。它不能替代下文的管理员、改密、限流、重放和故障用例。

### 5.5 类生产验收：发布前

在独立验收环境执行以下验证：

- HTTPS 登录响应的 Refresh Cookie 实际包含 `Secure`、`HttpOnly` 和 `SameSite=Lax`。
- `TRUSTED_PROXIES` 只信任真实入口；伪造 `X-Forwarded-For` 不能绕过 IP 限流。
- 多副本环境下刷新、吊销、限流和就绪状态一致。
- Migration 由单一 Job/服务执行，没有多副本并发升级数据库。
- Prometheus Target、readiness、错误率和延迟可观测；告警能够触发、送达并恢复。
- 使用脱敏测试数据完成备份恢复演练，核对业务行数、约束、migration 版本和登录链路。
- 使用上一个固定版本完成应用回滚，回滚后重新验证 `/readyz` 和核心认证流程。

具体部署边界同时遵循 `docs/deploy/production-checklist.md` 和同级 `go-user-system-ops` 运维知识库。

## 6. 真实业务验收用例

### P0：认证和权限主链路

| ID | 场景 | Given / When | 预期结果与证据 |
| --- | --- | --- | --- |
| REG-01 | 正常注册 | 开启注册；匿名用户提交唯一合法账号 | 注册成功；数据库只有一个新用户；只绑定 `user` 角色；管理 API 返回 403 |
| REG-02 | 关闭注册 | `REGISTRATION_ENABLED=false` 后重建后端；访问页面并直接请求 API | 前端不提供有效注册路径；后端注册路由不可用；登录仍可使用 |
| REG-03 | 注册边界 | 提交重复用户名、空值、11/12 字符和 72/73 字节密码 | 状态和错误结构符合契约；数据库没有半成品用户；日志不含密码 |
| AUTH-01 | 正常与失败登录 | 正常、错误密码、不存在用户、禁用用户分别登录 | 正常登录创建会话；失败场景不泄露账户是否存在或禁用状态 |
| AUTH-02 | 账号/IP 限流 | 同账号多 IP、同 IP 多账号连续失败 | 达到阈值返回 429 和 `Retry-After`；可信代理之外的 XFF 不改变客户端 IP |
| SES-01 | 刷新轮换 | 登录后刷新 Access Token，再使用旧 Refresh | 新 Token 可用；旧 Refresh 不可再次正常使用；数据库只保存摘要 |
| SES-02 | 双标签页并发刷新 | Tab A、Tab B 同时遇到 401 | 只产生一次有效刷新；请求可重放；不会误吊销正常 Token Family |
| SES-03 | Refresh 重放 | 在轮换后故意再次提交旧 Refresh | 重放被拒绝；整个 Token Family 被吊销；后续 Refresh 继续失败 |
| SES-04 | 登出 | 保存当前 Access 后执行登出 | Refresh Cookie 被清除；当前 Refresh 和 Access JTI 失效；受保护接口返回 401 |
| PWD-01 | 改密使旧会话失效 | `user_a` 在 Browser A/B 登录；A 修改密码 | 两端旧 Access/Refresh 均失效；旧密码失败；新密码成功；`auth_version` 增加 |
| RBAC-01 | 普通用户越权 | 普通用户隐藏管理页面后直接调用管理 API | 服务端返回 403；没有角色、权限或其他用户数据泄漏 |
| RBAC-02 | 管理员授权 | 一次性初始化管理员；查询角色/权限并给 `user_b` 分配角色 | 管理员操作成功且幂等；非法用户/角色被拒绝；第二次初始化管理员失败 |

### P1：资料、可用性和运维

| ID | 场景 | Given / When | 预期结果与证据 |
| --- | --- | --- | --- |
| PROFILE-01 | 读取和修改本人资料 | `user_a` 修改昵称并刷新页面 | 新昵称持久化；不能通过 ID 或请求体修改 `user_b` |
| OPS-01 | MySQL 故障与恢复 | 在测试环境停止 MySQL，再恢复 | `/readyz` 失败；故障可从日志/指标定位；恢复后就绪和核心链路恢复 |
| OPS-02 | Redis 故障与恢复 | 在测试环境停止 Redis，再请求受保护接口 | 认证 fail-closed；请求不被错误放行；产生就绪异常和告警证据 |
| OPS-03 | 进程和可观测性 | 请求成功、401、403、429、500 和超时路径 | 日志带 `request_id`；指标使用有界路由标签；无密码、Token 或 Hash |
| REL-01 | Migration 和发布回滚 | 从空库升级、备份、部署候选版、回滚应用 | 单一 migration 成功；备份可恢复；回滚后数据兼容且冒烟测试通过 |
| UI-01 | 桌面与移动端 | 桌面和 Pixel 7 配置执行注册登录和导航 | 无页面错误；受保护路由正确；抽屉导航、表单和反馈可操作 |

## 7. 故障注入步骤

以下命令只允许在已经确认的本地/验收 Compose 环境执行，并沿用 5.4 节定义的 `$compose` 参数。

Redis 故障：

```powershell
docker @compose stop redis
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/readyz -SkipHttpErrorCheck
docker @compose logs --since=5m app redis
docker @compose start redis
docker @compose up -d --wait app
```

MySQL 故障：

```powershell
docker @compose stop mysql
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/livez -SkipHttpErrorCheck
Invoke-WebRequest -UseBasicParsing http://127.0.0.1:18082/readyz -SkipHttpErrorCheck
docker @compose logs --since=5m app mysql
docker @compose start mysql
docker @compose up -d --wait app
```

每次故障注入记录：开始时间、假设、唯一变更、用户影响、健康端点、HTTP 结果、日志、指标、恢复动作和恢复时间。不得同时停止两个依赖，否则无法判断单一原因。

## 8. 进入和退出标准

### 进入完整栈测试

- 目标提交和变更范围已经确认。
- 快速门禁全部通过或例外已记录并批准。
- 专用测试数据和凭据已准备。
- Compose 端口无冲突，环境中没有需要保留的同名测试数据。

### 候选版本通过

- P0 用例全部通过，无跳过、无未解释警告。
- MySQL 集成测试实际执行，不是因为缺少 DSN 而跳过。
- Playwright 桌面和移动端项目通过，Trace/日志已保存。
- 没有权限扩大、会话绕过、敏感信息泄漏或不可恢复的数据错误。
- 已知 P1 问题有缺陷编号、影响、Owner 和处理期限。

### 生产发布阻断条件

- 任一 P0 用例失败。
- 真实 TLS Cookie、可信代理或多副本认证状态未验证。
- Migration、备份恢复或应用回滚不可验证。
- Redis 故障时出现 fail-open。
- 日志或测试制品泄漏凭据、Token 或用户敏感数据。
- 运行版本、镜像 digest、配置或 migration 版本无法确认。

## 9. 缺陷严重级别

| 级别 | 定义 | 示例 |
| --- | --- | --- |
| P0 | 阻断发布；权限扩大、认证绕过、数据破坏或核心功能不可用 | 普通用户访问管理接口、登出后 Token 仍可用、Redis 故障时放行 |
| P1 | 重要业务错误或显著可靠性退化，发布前通常必须处理 | 并发 Refresh 误吊销、改密未使全部旧会话失效 |
| P2 | 有替代路径的功能、兼容性或可观测性问题 | 移动端局部交互异常、非核心指标缺失 |
| P3 | 低风险体验或文档问题 | 提示文案不一致、记录字段缺失 |

缺陷至少记录：环境、提交、用例 ID、前置条件、复现步骤、实际结果、期望结果、证据、影响、严重级别和验收标准。

## 10. 测试执行记录模板

每轮测试复制以下模板到新的记录文件。建议路径：`docs/test-records/YYYY-MM-DD-<version>-<environment>.md`。

```markdown
# 测试执行记录：<version> / <environment>

| 字段 | 内容 |
| --- | --- |
| 执行时间 | |
| 执行人 | |
| 环境 | local / acceptance / staging / production smoke |
| 分支与提交 | |
| 前端/后端镜像 digest | |
| Migration 版本 | |
| 测试数据前缀 | |
| 结果 | PASS / FAIL / BLOCKED |

## 变更和风险

- 本次变更：
- 不在范围：
- 已知风险：

## 自动化结果

| 门禁 | 结果 | 证据位置 | 跳过/警告 |
| --- | --- | --- | --- |
| Go unit | | | |
| Go race | | | |
| MySQL integration | | | |
| Frontend check | | | |
| Compose E2E | | | |
| Security scan | | | |

## 业务用例

| 用例 ID | 结果 | 实际结果 | 证据位置 | 缺陷 ID |
| --- | --- | --- | --- | --- |
| REG-01 | | | | |
| AUTH-01 | | | | |
| SES-02 | | | | |
| PWD-01 | | | | |
| RBAC-01 | | | | |

## 故障与恢复

| 时间 | 假设 | 动作 | 结果 | 恢复证据 |
| --- | --- | --- | --- | --- |

## 最终结论

- 已验证：
- 未验证：
- 残留风险：
- 发布决定及批准人：
- 下一步、Owner、期限：
```

## 11. 当前自动化缺口和补齐顺序

当前 Playwright 只有注册、登录、双标签页恢复和登出主流程。建议依次补充：

1. `PWD-01`：改密后 Browser A/B 的旧会话全部失效。
2. `RBAC-01`、`RBAC-02`：普通用户直接越权和管理员授权。
3. `AUTH-02`：账号/IP 双维度限流及 `Retry-After`。
4. `SES-03`：旧 Refresh 重放导致 Token Family 吊销。
5. `REG-02`：关闭注册后的 UI 与 API 行为。
6. `OPS-01`、`OPS-02`：MySQL/Redis 故障与恢复。
7. HTTPS Cookie、可信代理、多副本、备份恢复和回滚验收。

CI 应显式提供隔离的 MySQL 测试服务和 `TEST_DATABASE_DSN`，并把“集成测试被跳过”视为门禁失败。覆盖率只用于发现未测关键路径，不以单一百分比替代 P0 业务场景和状态不变量验证。
