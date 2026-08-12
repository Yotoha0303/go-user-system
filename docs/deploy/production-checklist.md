# 生产部署检查清单

`v1.0.0-rc.3` 进入生产前应逐项验证；无法满足的项目需要记录风险、负责人和补救期限。

## 密钥与权限

- [ ] `DB_ROOT_PASSWORD`、`DB_PASSWORD`、`JWT_SECRET` 均为独立强随机值。
- [ ] 应用使用最小权限数据库账号，不使用 MySQL root。
- [ ] `.env`、`k8s/secret.yaml`、管理员密码、Token 和用户数据未提交。
- [ ] GitHub secret scanning、push protection、Dependabot security updates 已启用。
- [ ] 已通过私有渠道处理历史泄漏，并轮换可能受影响的凭据。
- [ ] 管理员由 `bootstrap-admin` 创建，系统已有管理员后该命令会被拒绝。
- [ ] `REGISTRATION_ENABLED` 符合业务策略；不需要公开注册时设为 `false`。

## 构建与供应链

- [ ] CI 的 backend、frontend、manifests 和 e2e 作业全部通过。
- [ ] CodeQL 分析通过。
- [ ] `govulncheck ./...` 无可达漏洞。
- [ ] `npm audit --audit-level=high` 无阻断项。
- [ ] 部署固定版本或 digest，不使用 `latest`。
- [ ] GHCR 镜像包含 provenance 和 SBOM，发布归档校验和已核对。

## 数据与迁移

- [ ] 迁移由单一 Compose 服务或 Kubernetes Job 执行，不由每个应用副本并发执行。
- [ ] `goose validate` 通过，目标库 migration 版本已确认。
- [ ] 数据库和 Redis 持久卷容量、备份、恢复和保留策略已配置。
- [ ] 已在非生产环境执行一次备份恢复演练。
- [ ] 破坏性 schema 回滚前已评估数据兼容性。

## 网络与运行时

- [ ] `APP_ENV=production`，应用在 Redis 或 Secure Cookie 配置缺失时拒绝启动。
- [ ] 外部入口启用 TLS，代理传递正确的 `X-Forwarded-Proto`。
- [ ] `COOKIE_SECURE=true`，且登录响应中的 Refresh Cookie 实际包含 `Secure`。
- [ ] `TRUSTED_PROXIES` 只包含真实入口代理的 IP/CIDR；伪造 XFF 不会改变登录限流 IP。
- [ ] MySQL 和 Redis 不暴露公网。
- [ ] 容器使用非 root 用户、禁用提权并移除不需要的 capabilities。
- [ ] CPU、内存、重启策略和副本数符合目标负载。
- [ ] SIGTERM 优雅关闭已验证。
- [ ] Ingress 保留 `/api/v1/...` 原始路径，前端 SPA fallback 正常。

## 认证行为

- [ ] Refresh Cookie 在 HTTPS 下带 `Secure`、`HttpOnly` 和 `SameSite=Lax`。
- [ ] Redis 不可用时受保护请求拒绝访问并触发告警。
- [ ] 登出后当前 Access/Refresh 失效。
- [ ] 改密后全部旧 Access/Refresh 失效。
- [ ] Refresh 重放会吊销整个 Token Family。
- [ ] 登录限流返回 429 和正确的 `Retry-After`。
- [ ] 两个标签页并发触发 Refresh 时不会误报重放或吊销 Token Family。
- [ ] 注册和改密拒绝少于 12 个字符或超过 72 个 UTF-8 字节的密码。
- [ ] 普通注册用户只有 `user` 角色，无法访问管理员接口。

## 可观测与验收

- [ ] `/livez` 和 `/readyz` 纳入平台探针与告警。
- [ ] `/metrics` 仅允许监控系统访问；HTTP、readiness、Go Runtime 和 build info 指标可查询。
- [ ] Prometheus 配置和规则通过 `promtool`，TargetDown/NotReady 告警在非生产环境实际触发并恢复。
- [ ] Alertmanager 接收器、升级联系人和通知 Secret 已在部署平台配置；不能只依赖本地 Prometheus 页面。
- [ ] 日志采集支持按 `request_id` 检索，且不记录密码、密钥、Token 或密码哈希。
- [ ] 浏览器注册、登录、资料、登出流程通过。
- [ ] 管理员角色查询、权限查询和角色分配通过。
- [ ] 关键失败场景和恢复步骤已记录在运行手册。

## 回滚

- [ ] 保留上一个已验证的固定版本镜像。
- [ ] 记录应用镜像回滚和数据库恢复命令。
- [ ] 回滚后重新验证 migration 版本、`/readyz` 和核心认证流程。
- [ ] 发布负责人能够在目标恢复时间内完成演练。
