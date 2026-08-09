# 补齐计划：Redis 存储、Token 注销增强、登录设备管理

## 背景

上一轮鉴权与 RBAC 迭代（见 `iteration-plan-auth-rbac-swagger.md`、`operation-record-auth-rbac-swagger.md`）已完成 JWT 双 Token、登出/轮换吊销、RBAC 五表与接口级鉴权。

对照目标清单，本文记录立项时的缺口，并作为分阶段实现与验收依据。

> 状态更新（2026-08-09）：Redis 认证基础设施、Access JTI 吊销、登录限流和 Refresh Token Family 重放检测已在 `feat/production-auth-hardening` 完成，详见 `iteration-plan-production-auth-hardening.md` 与 `operation-record-production-auth-hardening.md`。登录设备字段、会话 API 和前端设备管理仍未实现，继续作为后续迭代范围。

## 立项时现状对照（历史）

| 目标 | 状态 | 当前证据 | 结论 |
| --- | --- | --- | --- |
| Refresh Token | ✅ 已实现 | `internal/auth/token_manager.go`、`internal/service/auth.go`、`migrations/00003_create_refresh_tokens.sql`、`POST /api/v1/auth/refresh` | 无需重做，可复用并扩展字段 |
| Redis 存储 | ❌ 未实现 | `go.mod` 无 Redis 依赖；Token 与会话均落在 MySQL / 进程内存 | **本轮补齐** |
| Token 注销 | ⚠️ 部分实现 | Refresh：MySQL `revoked_at` + 轮换；Access：进程内 `map` 黑名单（`TokenManager.RevokeAccessToken`），注释写明生产应用 Redis | **本轮增强** |
| 登录设备管理 | ❌ 未实现 | `refresh_tokens` 无设备/IP/UA 字段；无设备列表/踢下线 API | **本轮补齐** |
| RBAC 权限 | ✅ 已实现 | 五表 + `RequirePermission` + `/admin/*` 与前端权限页 | 无需本轮改动 |

## 主要缺口（立项时记录）

### 1. 无 Redis，无法支撑跨实例会话与 Access 吊销

问题：

- Access Token 黑名单使用进程内 `map`，多副本部署时不共享，进程重启后失效。
- 文档与代码注释已将 Redis 列为后续方案，但尚未接入客户端、配置与仓库层。

目标：

- 引入 Redis 作为 **Access Token 吊销名单**（及可选会话缓存）的权威存储。
- 保持 Refresh Token 权威数据仍在 MySQL（便于审计、设备列表与事务轮换）；Redis 侧重低延迟、TTL 与多实例一致。

### 2. Token 注销能力不完整

已有：

- `POST /auth/logout`：吊销当前 Refresh Token，清除 Cookie。
- Refresh 轮换：旧 token 写 `revoked_at` / `replaced_by_jti`。
- 改密：`RevokeAllByUserID` + 当前 Access 进内存黑名单。

缺口：

- Access 黑名单非 Redis，生产多实例不可靠。
- 缺少「按用户吊销全部 Access」的统一入口（改密后其他实例上的 Access 仍可能有效直至过期）。
- 未与设备维度联动（踢设备时应同时使对应会话不可刷新，并尽量让 Access 立即失效）。

### 3. 无登录设备 / 会话管理

问题：

- `refresh_tokens` 仅有 `user_id`、`jti`、`token_hash`、过期与吊销字段，无法区分浏览器、手机、IP、User-Agent。
- 用户无法查看「当前在哪些设备登录」，也无法踢掉某一台设备或「除当前外全部下线」。

目标：

- 登录/刷新时记录设备与客户端元数据。
- 提供查询与吊销接口；前端可做「登录设备」页。

---

## 设计方案

### A. Redis 接入

#### 依赖与配置

- 客户端建议：`github.com/redis/go-redis/v9`（与 Go 1.2x 生态一致）。
- 配置（`config.yml` + 环境变量覆盖）：

```yaml
redis:
  enabled: true
  addr: "127.0.0.1:6379"
  password: ""
  db: 0
  dialTimeoutMs: 2000
  readTimeoutMs: 1000
  writeTimeoutMs: 1000
```

环境变量示例：`REDIS_ENABLED`、`REDIS_ADDR`、`REDIS_PASSWORD`、`REDIS_DB`。

- `compose.yaml` / k8s：增加 Redis 服务与健康检查；`configmap` / `secret` 注入地址与密码。
- 启动：`cmd/main.go` 初始化 Redis 客户端；`/readyz` 可在 `enabled=true` 时 ping Redis。

#### Key 设计

| 用途 | Key 模式 | Value | TTL |
| --- | --- | --- | --- |
| Access 单 token 吊销 | `auth:access:revoke:jti:{jti}` | `1` 或 `user_id` | 剩余 Access 有效期（与 JWT `exp` 对齐） |
| 用户级 Access 吊销水位 | `auth:access:user_revoke_before:{user_id}` | Unix 秒（签发时间早于此的 Access 一律无效） | 可设较长 TTL 或不过期，改密/全量下线时更新 |
| （可选）设备会话索引 | `auth:user:sessions:{user_id}` | Set of jti | 与最长 Refresh 有效期对齐 |

解析 Access 时顺序建议：

1. 验签与 `token_type`。
2. 查 `auth:access:revoke:jti:{jti}`。
3. 查 `auth:access:user_revoke_before:{user_id}`，若 `claims.iat < revoke_before` 则拒绝。

#### 分层建议

```text
pkg/redis/                  客户端初始化
internal/repository/        AccessRevocationRepository（Redis 实现）
internal/auth/              TokenManager 注入吊销检查（替换内存 map 或双写过渡）
```

本地/单测：`redis.enabled=false` 时回退内存实现或 no-op，并在测试中 mock 接口。

#### 验收标准

- [ ] 配置可加载、可被环境变量覆盖；Compose 可起 Redis。
- [ ] 多进程/多实例共享同一 Access 吊销结果。
- [ ] Redis 不可用时行为明确：fail-open 仅用于开发（默认建议 fail-closed 拒绝需校验吊销的请求，或降级日志 + 配置开关）。
- [ ] 单元/集成测试覆盖 set/get/TTL 过期。

---

### B. Token 注销增强

#### 行为矩阵

| 场景 | Refresh Token (MySQL) | Access Token (Redis) |
| --- | --- | --- |
| 登出当前会话 | 吊销当前 jti | 吊销当前 Access jti（若请求带 Bearer） |
| 刷新轮换 | 旧 jti revoked + replaced_by | 无需处理旧 Access（短 TTL 自然过期） |
| 修改密码 | `RevokeAllByUserID` | 更新 `user_revoke_before` = now |
| 踢掉某设备 | 吊销该设备对应 refresh jti | 更新用户水位 **或** 吊销该设备最近 Access jti（若有记录） |
| 踢掉全部设备（可保留当前） | 批量吊销 refresh | 更新 `user_revoke_before`；可选为当前会话重签一对 Token |

#### 接口调整（兼容优先）

保持现有：

```http
POST /api/v1/auth/logout
POST /api/v1/auth/refresh
PATCH /api/v1/users/me/update/password
```

增强点：

- `LogoutHandler`：在吊销 Refresh 后，若存在 `Authorization: Bearer`，将 Access 的 jti 写入 Redis 黑名单。
- 改密路径：除 `RevokeAllRefreshTokens` 外，调用 `RevokeAllAccessTokensForUser(userID)`（写用户水位）。
- 删除或收敛 `TokenManager` 内纯内存 blacklist 的生产路径；测试可用 fake Redis。

#### 验收标准

- [ ] 登出后：Refresh 不可再刷新；该 Access 立即 401。
- [ ] 改密后：任意旧 Access/Refresh 均不可用；需重新登录。
- [ ] 进程重启后 Access 吊销状态仍有效（在 TTL 内）。
- [ ] 现有 Cookie 刷新/登出契约不变；Swagger 与 `docs/http/test.http` 同步。

---

### C. 登录设备管理

#### 数据模型扩展

新增 migration（建议 `00007_add_refresh_token_device_fields.sql`；`00006` 已用于生产认证加固）：

```sql
-- +goose Up
ALTER TABLE refresh_tokens
    ADD COLUMN device_id VARCHAR(64) NULL DEFAULT NULL COMMENT '客户端生成的稳定设备标识' AFTER token_hash,
    ADD COLUMN device_name VARCHAR(128) NULL DEFAULT NULL COMMENT '展示名，如 Chrome on Windows' AFTER device_id,
    ADD COLUMN user_agent VARCHAR(512) NULL DEFAULT NULL AFTER device_name,
    ADD COLUMN ip_address VARCHAR(45) NULL DEFAULT NULL AFTER user_agent,
    ADD COLUMN last_seen_at DATETIME(3) NULL DEFAULT NULL AFTER ip_address,
    ADD KEY idx_refresh_tokens_user_device (user_id, device_id);

-- +goose Down
ALTER TABLE refresh_tokens
    DROP KEY idx_refresh_tokens_user_device,
    DROP COLUMN last_seen_at,
    DROP COLUMN ip_address,
    DROP COLUMN user_agent,
    DROP COLUMN device_name,
    DROP COLUMN device_id;
```

`model.RefreshToken` 同步字段；`StoreRefreshToken` / `RotateRefreshToken` 写入并更新 `last_seen_at`。

登录请求可选扩展（向后兼容）：

```json
{
  "username": "...",
  "password": "...",
  "device_id": "uuid-from-client",
  "device_name": "Chrome on Windows"
}
```

服务端补充：`User-Agent`、`ClientIP`（Gin）。同一 `user_id + device_id` 策略建议：

- **推荐**：同设备再次登录时吊销该设备旧 Refresh，再写入新会话（单设备单有效 Refresh）。
- 无 `device_id` 时：每次登录新建一条会话记录（兼容旧客户端）。

#### 新增 API

```http
GET    /api/v1/users/me/sessions
DELETE /api/v1/users/me/sessions/:jti
POST   /api/v1/users/me/sessions/revoke-others
```

说明：

| 方法 | 路径 | 权限码（建议） | 行为 |
| --- | --- | --- | --- |
| GET | `/users/me/sessions` | `profile:read` 或新码 `session:list` | 列出当前用户未过期且未吊销的 Refresh 会话（脱敏，不返回 token_hash） |
| DELETE | `/users/me/sessions/:jti` | `session:revoke` | 吊销指定会话；若为当前会话则等同登出 |
| POST | `/users/me/sessions/revoke-others` | `session:revoke` | 吊销除当前 Cookie/Refresh 对应 jti 外的全部会话 |

列表响应示例：

```json
{
  "code": 0,
  "data": {
    "sessions": [
      {
        "jti": "…",
        "device_id": "…",
        "device_name": "Chrome on Windows",
        "user_agent": "Mozilla/5.0 …",
        "ip_address": "203.0.113.10",
        "created_at": "…",
        "last_seen_at": "…",
        "expires_at": "…",
        "is_current": true
      }
    ]
  }
}
```

RBAC：在 `00004` 后续 seed migration 或独立 migration 中插入权限码，并赋予 `user` / `admin` 角色。

#### 前端（`frontend/`）

- API：`src/api` 增加 sessions 相关方法。
- 页面：安全中心增加「登录设备」列表、踢下线、下线其他设备。
- 登录：localStorage/cookie 持久化 `device_id`（非密钥，仅标识），登录请求带上 `device_id` / `device_name`。

#### 验收标准

- [ ] 登录后库表可见设备元数据；刷新更新 `last_seen_at`。
- [ ] 列表仅返回本人会话；无 hash、无完整 refresh token。
- [ ] 踢设备后该设备 Refresh 刷新失败；可选 Access 立即失效。
- [ ] `revoke-others` 后其他设备需重新登录，当前会话仍可用。
- [ ] 后端单测 + 前端关键路径测试；Swagger 与 `test.http` 更新。

---

## 建议实施顺序

```text
PR1  Redis 基础设施
     config / compose / pkg/redis / 健康检查 / 接口抽象

PR2  Access 吊销迁 Redis
     替换内存 blacklist；登出/改密写入 jti 与 user_revoke_before

PR3  设备字段与登录落库
     migration + model + login/refresh 写入元数据

PR4  会话管理 API + RBAC 权限码
     list / revoke / revoke-others

PR5  前端设备管理页 + device_id 上报
     安全中心 UI 与 API 对接
```

依赖关系：PR2 依赖 PR1；PR4 依赖 PR3；PR5 依赖 PR4。PR3 可与 PR1/PR2 并行。

## 非目标（本轮不做）

- 将 Refresh Token 主存储从 MySQL 迁到 Redis（可保留 MySQL 为 source of truth）。
- 强制 RS256 / 密钥轮换全量切换（既有文档中的 JWT 改进可另开迭代）。
- IP 强制绑定（可先记录 IP，绑定策略单独评估误杀风险）。
- 复杂风控（异地登录告警、设备指纹 SDK 等）。

## 风险与注意点

| 风险 | 缓解 |
| --- | --- |
| Redis 宕机导致全站鉴权失败 | 明确 `enabled` 与 fail 策略；监控与本地降级文档 |
| 用户水位吊销过猛导致全端掉线 | 改密/踢全部时预期行为；产品文案提示 |
| device_id 可被伪造 | 仅作会话分组标识，不作为信任根；敏感操作仍靠密码/2FA（若后续加） |
| 隐私（存 IP/UA） | 最小化字段；列表仅本人可见；文档说明保留期限可按 `expires_at` 清理 |
| 旧客户端无 device_id | 字段可空；会话仍可按 jti 管理 |

## 验证清单

已完成的生产认证加固：

```bash
make migrate-validate
make test
make race-test
make vet
docker compose config --quiet
```

后续设备管理迭代完成后还需验证：

```text
登录 -> 刷新 -> 查看会话列表 -> 踢指定设备 -> 被踢设备刷新返回 401
```

## 相关文件（实施时重点改动）

| 区域 | 路径 |
| --- | --- |
| 配置 | `config/config.go`、`config.yml`、`.env.example`、`k8s/configmap.yaml` |
| 编排 | `compose.yaml`、`docs/deploy/*` |
| Auth | `internal/auth/token_manager.go`、`internal/service/auth.go`、`internal/handler/user_handler.go` |
| 数据 | `migrations/`、`internal/model/refresh_token.go`、`internal/repository/refresh_token.go` |
| 路由 | `router/router.go`、`docs/http/test.http`、Swagger 注解 |
| 前端 | `frontend/src/api/*`、`frontend/src/pages/security/*`、登录页 device 上报 |

## 实施状态

| 项 | 状态 | 备注 |
| --- | --- | --- |
| 文档立项 | 已完成 | 本文档 |
| Redis 基础设施 | 已完成 | Redis/内存 Store、配置、健康检查、Compose、Kubernetes |
| Access 吊销迁 Redis | 已完成 | JTI 派生 Key + Token 剩余寿命 TTL；运行故障 fail-closed |
| 登录失败限流 | 已完成 | 账号/IP 双维度，429 + `Retry-After` |
| Refresh Family 重放检测 | 已完成 | `family_id`、`revoked_reason`、Family 级吊销 |
| 设备字段与会话 API | 待实现 | |
| 前端设备管理 | 待实现 | |
| 已完成范围操作记录 | 已完成 | `operation-record-production-auth-hardening.md` |
| 设备管理操作记录 | 待写 | 设备迭代完成后新增独立操作记录 |

---

**记录说明**：本计划最初由 2026-08-07 的现状核对得出。2026-08-09 已完成 Redis、Access 吊销和 Refresh Family 加固；设备管理仍按本文 C 部分另行迭代。
