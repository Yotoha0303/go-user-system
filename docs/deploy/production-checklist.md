# 生产部署前检查清单

这份清单用于避免把本地开发配置直接带到生产环境，并确保服务具备基本的可运行、可观测、可回滚能力。

## 1. 配置与密钥

- [ ] `JWT_SECRET` 已替换为 32 位以上强随机字符串。
- [ ] `DB_PASSWORD` 已替换为生产数据库密码。
- [ ] 生产环境不使用 MySQL `root` 用户连接业务库。
- [ ] `.env`、`.env.*`、`config.local.yml` 未提交到 Git。
- [ ] `APP_PORT`、`DB_HOST`、`DB_PORT`、`DB_USER`、`DB_NAME`、`JWT_EXPIRE_HOURS` 与部署环境一致。
- [ ] `APP_PORT` 和 `JWT_EXPIRE_HOURS` 是合法数字，避免启动时配置解析失败。
- [ ] 日志中不会打印密码、JWT secret、access token、password hash。

## 2. 构建与质量门禁

- [ ] 本地 `make lint` 通过。
- [ ] 本地 `make test` 通过。
- [ ] 本地 `make race-test` 通过。
- [ ] 本地 `make vet` 通过。
- [ ] 本地 `make build` 通过。
- [ ] `make migrate-validate` 通过。
- [ ] GitHub Actions CI 通过。
- [ ] 本地 golangci-lint 使用 v2，和 `.golangci.yml` 配置版本一致。

## 3. 数据库

- [ ] 目标数据库已创建。
- [ ] 数据库字符集使用 `utf8mb4`。
- [ ] 数据库账号只授予应用需要的最小权限。
- [ ] 已执行 `migrations/*.sql`。
- [ ] `goose_db_version` 表中可以看到已执行的 migration 版本。
- [ ] 数据库连接池参数已确认：`maxOpenConns`、`maxIdleConns`、`connMaxLifeTime`、`connMaxIdleTime`。
- [ ] 数据库变更具备 down migration 或备份恢复方案。
- [ ] 已确认数据库备份和恢复流程。

## 4. 容器与运行时

- [ ] 镜像使用非 root 用户运行应用。
- [ ] 容器包含 `/readyz` healthcheck。
- [ ] 部署平台会向容器发送 `SIGTERM` 进行停止。
- [ ] 服务收到 `SIGTERM` 后能优雅关闭 HTTP server 并释放数据库连接。
- [ ] 已配置合理的重启策略。
- [ ] 已确认 `config.yml` 随镜像复制，敏感信息通过环境变量注入。

## 5. 超时与稳定性

- [ ] HTTP server 已配置 `readTimeout`、`writeTimeout`、`idleTimeout`、`readHeaderTimeout`。
- [ ] 请求级 `timeout` 已配置为合理值。
- [ ] 数据库启动 ping 使用 `pingTimeout`。
- [ ] 已理解当前请求 timeout 是 context deadline，不会自动中断不检查 context 的 handler，也不会自动返回 504。
- [ ] handler -> service -> dao 链路持续传递 `context.Context`。

## 6. 日志与可观测性

- [ ] Access log 是结构化日志，包含 `request_id`、`method`、`path`、`router`、`status`、`latency`、`client_ip`、`body_size`。
- [ ] Panic recovery 会记录 `request_id`、`method`、`path`、`panic`、`stack`。
- [ ] 业务错误日志包含 `request_id`、`path`、`method`、业务错误码和 cause。
- [ ] 日志采集系统能按 `request_id` 检索一次请求的相关日志。

## 7. 服务验证

- [ ] `GET /ping` 返回 200。
- [ ] `GET /livez` 返回 200。
- [ ] `GET /readyz` 返回 200，且能证明数据库可访问。
- [ ] 注册、登录、鉴权、当前用户查询、昵称修改、密码修改链路验证通过。
- [ ] 应用日志中没有数据库连接失败或 JWT 配置缺失错误。

## 8. 回滚准备

- [ ] 保留上一个可用镜像版本。
- [ ] 数据库变更有 down migration 或备份恢复方案。
- [ ] 回滚命令已提前记录并演练。
- [ ] 回滚后重新验证 `/readyz` 和核心接口。
