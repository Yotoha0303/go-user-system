# 认证交付加固操作记录

## 提交节点

| 项 | 值 |
| --- | --- |
| 基线分支 | `main` |
| 基线提交 | `eae81f1b2f1b89f41a97d8290f0f232f59ba507b` |
| 实施分支 | `agent/auth-delivery-hardening` |
| 实施提交 | `8e31f0a` (`fix: harden browser authentication delivery`) |
| 合并 PR | [#12](https://github.com/Yotoha0303/go-user-system/pull/12) |
| 目标版本 | `v1.0.0-rc.3` |
| 实施日期 | `2026-08-10` |

本次工作从已发布的 `v1.0.0-rc.2` 建立独立分支，先完成问题复现和代码证据映射，再修改代码、测试、部署清单和文档。合并 PR、最终主分支提交和版本标签以 GitHub PR 与 Release 记录为最终依据。

## 问题、原因、修改与证据

| 问题 | 原因 | 本次修改 | 验证证据 |
| --- | --- | --- | --- |
| 登出没有吊销当前 Access | logout 使用无认证 Axios 实例 | 显式携带内存中的 Access Token，后端成功吊销后再清 Cookie | API 单测与 E2E 均验证原当前 Token 返回 401 |
| 多标签页可能同时轮换 Refresh | Promise 只能协调单个 JS 上下文 | 使用同源 Web Locks 串行化不同标签页的 Refresh | 双协调器单测；Playwright 两标签页同时恢复成功 |
| 代理后 IP 限流共享代理地址 | Handler 只读取 `RemoteAddr` | Gin 仅信任配置 CIDR，并按可信代理链解析 XFF | 覆盖可信、不可信和伪造前缀的 Handler 测试 |
| Secure Cookie 依赖代理协议推断 | 多层 TLS 终止可能覆盖 `X-Forwarded-Proto` | 新增显式 `COOKIE_SECURE`，生产模式强制为 true | 配置测试、Cookie 属性测试、Kubernetes 生产配置 |
| 配置宣称 RS256 但实现固定 HS256 | 算法配置没有进入 TokenManager | 配置只接受真实支持的 HS256 | RS256 拒绝测试与 README 配置说明 |
| 生产可误用进程内认证状态 | Redis 默认关闭，单进程内存状态不能跨副本 | `APP_ENV=production` 强制 Redis 与 Secure Cookie | 生产配置启动校验测试；Compose/Kubernetes 均显式配置 |
| 密码策略弱且禁用用户响应可枚举 | 最短 6 位，禁用状态使用独立 403 | 提升到 12 字符/72 字节；缺失、错误、禁用统一凭据错误并统一计入限流 | Service 边界测试、Swagger 和前端校验同步 |

## 验收记录

静态与自动化门禁：

```text
go test ./...
go test -race ./...
go vet ./...
golangci-lint run ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.6.0 ./...
npm --prefix frontend run check
npm --prefix frontend audit --audit-level=high --registry=https://registry.npmjs.org
goose -dir migrations validate
docker compose --env-file .env.example config --quiet
go run github.com/yannh/kubeconform/cmd/kubeconform@v0.8.0 -strict -summary -ignore-missing-schemas k8s
```

结果：Go test/race/vet/lint 全部通过；govulncheck 可达漏洞为 0；前端 7 个测试文件、10 个测试、lint 和 build 全部通过；npm audit 为 0；Goose 校验通过；Kubernetes 13 个文件中的 17 个资源全部有效。

运行与浏览器验收：

- 使用隔离项目 `go-user-system-rc3-check` 构建并启动 Compose 全栈。
- 默认端口和 `18082` 被本机已有进程占用，最终使用前端 `28080`、后端 `28082`，不覆盖已有服务。
- MySQL、Redis、后端和前端均达到 healthy；migration 正常完成；`/readyz` 返回 `status=ready`。
- Playwright Desktop Chrome 与 Pixel 7 项目均通过注册、登录、双标签页并发恢复、资料读取、登出和 Access 吊销，共 2 项通过。
- 浏览器截图确认登录页面非空、布局正常；应用流程除预期的匿名 Refresh 401 外无 console/page error。
- 验收后删除隔离容器、网络和数据卷。

## 边界与后续

- Web Locks 是跨标签页协调的浏览器能力；不支持该 API 的浏览器仍只有页内 Promise 合并。稳定版需要明确浏览器基线，或增加短窗口、不可换取 Token 的后端轮换冲突恢复协议。
- 登出吊销请求中携带的当前 Access JTI，不等价于全设备登出；Access TTL 仍是泄露窗口上限，全设备失效应继续使用改密或后续专用接口。
- Kubernetes 清单中的私网可信代理 CIDR 是自托管示例，部署者必须按实际 Ingress Controller 网络收窄。
- `APP_ENV=production` 是配置安全门槛，不替代真实 TLS、备份恢复、监控告警和故障演练。
