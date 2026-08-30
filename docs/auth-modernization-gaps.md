# 认证现代化缺陷清单

## 文档状态

- 记录日期：2026-08-26
- 审查基线：`main`，提交 `0f30a4b`
- 当前版本：`v1.0.0-rc.3`
- 范围：登录凭据、Token 生命周期、浏览器会话、账户恢复和密钥管理
- 性质：缺陷与能力缺口记录，不表示已实施修复

## 总体结论

当前系统已经具备较现代的 Token 和会话安全基础，包括短期 Access Token、HttpOnly Refresh Cookie、Refresh Rotation、Token Family 重放检测、Redis JTI 吊销、登录失败限流、`auth_version` 和服务端 RBAC。

系统仍属于“现代化会话实现 + 传统用户名密码单因素登录”。以下项目是达到完整现代化登录体验和更高生产安全水平前需要处理的缺口。

## 缺陷清单

### AUTH-GAP-001：缺少抗钓鱼认证方式

- 优先级：P1
- 类型：认证能力缺口
- 状态：待处理
- 当前证据：代码中没有 MFA、WebAuthn、Passkey、TOTP 或恢复码实现；`docs/iteration-plan-production-auth-hardening.md` 将 MFA 和 Passkey 明确列为非目标。
- 影响：用户名和密码是唯一认证因子，无法抵御凭据钓鱼、密码复用和已泄露凭据被直接使用。
- 验收标准：
  - 支持至少一种 WebAuthn/Passkey 登录或二次认证流程。
  - 注册、认证、撤销和丢失恢复均有服务端验证及自动化测试。
  - 管理员敏感操作支持重新认证或 step-up authentication。
  - 认证器私钥、恢复码和敏感挑战值不以明文保存或记录到日志。

### AUTH-GAP-002：单因素密码策略未达到当前推荐基线

- 优先级：P0
- 类型：安全控制缺口
- 状态：待处理
- 当前证据：`internal/service/user.go` 要求密码至少 12 个字符、最多 72 个 UTF-8 字节；注册和改密路径没有常见、上下文相关或已泄露密码 blocklist。
- 影响：在没有 MFA 的情况下，12 字符下限低于当前 NIST 单因素密码建议；按字节限制还可能拒绝长度合理的多字节 Unicode 密码。
- 验收标准：
  - 单因素密码最短长度提高到至少 15 个 Unicode 字符；若账户启用 MFA，可按明确策略允许更短但不少于 8 个字符。
  - 支持至少 64 个 Unicode 字符，并明确采用 NFC 规范化策略。
  - 注册、改密和管理员重置密码时拒绝常见、已泄露及包含用户名派生形式的密码。
  - 保留密码管理器、自动填充和粘贴能力，不增加强制字符组合规则。
  - 前端、后端、Swagger、测试和部署文档保持一致。

### AUTH-GAP-003：密码哈希仍使用 bcrypt 默认成本

- 优先级：P1
- 类型：纵深防御改进
- 状态：待处理
- 当前证据：注册和改密使用 `bcrypt.GenerateFromPassword(..., bcrypt.DefaultCost)`。
- 影响：bcrypt 目前仍可接受，但受 72 字节输入上限约束，并非新系统首选的内存困难型密码哈希方案。
- 验收标准：
  - 根据部署资源基准测试并确定 Argon2id 参数及最大并发成本。
  - 新密码使用带版本和参数信息的 Argon2id 格式保存。
  - 旧 bcrypt 用户在成功登录后安全地渐进重哈希，不要求一次性重置全部密码。
  - 迁移失败不破坏原哈希，且不会把密码、派生值或哈希写入日志。

### AUTH-GAP-004：缺少用户可见的设备与会话管理

- 优先级：P0
- 类型：会话管理缺口
- 状态：待处理
- 当前证据：`refresh_tokens` 没有设备、IP、User-Agent 和最近活动字段；没有会话列表、吊销指定设备或退出其他设备接口。详细设计见 `docs/iteration-plan-redis-device-token.md`。
- 影响：用户无法识别异常登录，也不能在不改密码的情况下终止指定设备会话。
- 验收标准：
  - 记录最小必要的设备名称、IP、User-Agent、首次登录和最后活动时间，并定义保留期限。
  - 用户只能查看和吊销自己的会话，响应不得包含 Refresh Token 或 Token Hash。
  - 支持吊销指定会话和除当前会话外的全部会话。
  - 会话吊销后对应 Refresh 立即失效；Access 的最长残留窗口有明确策略和测试。

### AUTH-GAP-005：Token Family 没有绝对会话寿命上限

- 优先级：P0
- 类型：会话生命周期缺口
- 状态：待处理
- 当前证据：每次 Refresh 都从当前时间重新签发完整 Refresh TTL；数据模型没有 Family 首次认证时间或绝对过期时间。
- 影响：只要持续刷新，同一登录会话理论上可以无限延续，缺少强制重新认证边界。
- 验收标准：
  - 同时定义服务端空闲超时和绝对会话超时。
  - Refresh Rotation 不得延长 Token Family 的绝对过期时间。
  - 达到任一超时后，服务端拒绝刷新并吊销剩余 Family，客户端清理本地认证状态。
  - 超时策略可配置、有安全默认值，并覆盖时钟边界和并发刷新测试。

### AUTH-GAP-006：账户验证和安全恢复流程缺失

- 优先级：P1
- 类型：账户生命周期缺口
- 状态：待处理
- 当前证据：系统没有邮箱验证、忘记密码、受控密码重置、恢复码或认证器丢失恢复流程；`ROADMAP.md` 将邮箱验证和密码重置列为后续能力。
- 影响：用户遗忘密码或丢失认证器后没有安全自助恢复路径，容易转而依赖高权限人工或数据库操作。
- 验收标准：
  - 密码重置 Token 高熵、单次使用、短时有效，并仅保存不可逆摘要。
  - 重置成功后递增 `auth_version`，吊销全部旧 Access/Refresh 会话。
  - 请求重置接口不泄露账户是否存在，并受账号/IP 双维度限流。
  - 邮件、日志和接口响应不包含密码、现有 Token 或其他长期凭据。

### AUTH-GAP-007：JWT 缺少受众约束和签名密钥轮换

- 优先级：P2
- 类型：多服务与密钥管理缺口
- 状态：待处理
- 当前证据：当前仅支持 HS256 单一共享密钥；Claims 没有明确 `aud`，也没有 `kid`、JWKS 或签名密钥轮换。`ROADMAP.md` 已将 RS256/JWKS 列为后续能力。
- 影响：单体部署尚可使用，但拆分多个资源服务后，共享签名密钥会扩大泄露影响面，也难以平滑轮换和限制 Token 使用范围。
- 验收标准：
  - 明确 Access Token 的 issuer、subject、audience 和 token type 契约，并由资源服务强制校验。
  - 支持带 `kid` 的非对称签名和 JWKS 发布，私钥不进入应用镜像或仓库。
  - 支持新旧公钥重叠验证的无中断轮换，并提供过期密钥清理规则。
  - 单体部署继续使用 HS256 时，明确其适用边界和密钥轮换操作步骤。

## 建议实施顺序

1. `AUTH-GAP-002`：密码长度、Unicode 和 blocklist。
2. `AUTH-GAP-005`：空闲与绝对会话超时。
3. `AUTH-GAP-004`：设备元数据和用户会话管理。
4. `AUTH-GAP-003`：Argon2id 渐进迁移。
5. `AUTH-GAP-001`：WebAuthn/Passkey 和敏感操作 step-up。
6. `AUTH-GAP-006`：邮箱验证与安全恢复。
7. `AUTH-GAP-007`：受众约束、非对称签名和密钥轮换。

## 验证边界

本记录基于当前文档、代码、迁移和本地自动化测试。它不证明真实公网 TLS、Kubernetes Ingress、多副本故障、邮件送达、浏览器兼容性或生产密钥轮换已经通过验证。每项缺陷关闭时必须补充对应的单元、集成、浏览器和部署验证证据。

## 参考基线

- [NIST SP 800-63B：Authenticators](https://pages.nist.gov/800-63-4/sp800-63b/authenticators/)
- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [OWASP Session Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)
- [RFC 9700：Best Current Practice for OAuth 2.0 Security](https://www.rfc-editor.org/rfc/rfc9700.html)
