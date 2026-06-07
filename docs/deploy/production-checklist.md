# 生产部署前检查清单

该清单用于防止把本地开发配置直接带到生产环境，并确保服务具备基本的可运行、可观测、可回滚能力。

## 1. 配置与密钥

- [ ] `JWT_SECRET` 已替换为 32 位以上强随机字符串。
- [ ] `DB_PASSWORD` 已替换为生产数据库密码。
- [ ] 生产环境不使用 MySQL `root` 用户连接业务库。
- [ ] `.env`、`.env.*`、`config.local.yml` 未提交到 Git。
- [ ] `APP_PORT`、`DB_HOST`、`DB_PORT`、`DB_USER`、`DB_NAME` 与部署环境一致。
- [ ] 日志中不会打印密码、JWT secret、access token、password hash。

## 2. 构建与质量门禁

- [ ] `go test ./...` 通过。
- [ ] `go vet ./...` 通过。
- [ ] `go build -o bin/go-user-system ./cmd` 通过。
- [ ] `docker build -t go-user-system:ci .` 通过。
- [ ] GitHub Actions CI 通过。

## 3. 数据库

- [ ] 目标数据库已创建。
- [ ] 数据库字符集使用 `utf8mb4`。
- [ ] 应用启动后已自动执行 `migrations/*.up.sql`。
- [ ] `schema_migrations` 表中可以看到已执行的 migration 版本。
- [ ] 数据库变更具备 down migration 或备份恢复方案。
- [ ] 已确认数据库备份和恢复流程。

## 4. 容器与运行时

- [ ] 镜像使用非 root 用户运行应用。
- [ ] 容器包含 `/readyz` healthcheck。
- [ ] 部署平台会向容器发送 `SIGTERM` 进行停止。
- [ ] 服务收到 `SIGTERM` 后能优雅关闭 HTTP server 并释放数据库连接。
- [ ] 已配置合理的重启策略。

## 5. 服务验证

- [ ] `GET /livez` 返回 200。
- [ ] `GET /readyz` 返回 200，且能证明数据库可访问。
- [ ] 注册、登录、鉴权、当前用户查询、昵称修改链路验证通过。
- [ ] 应用日志中没有数据库连接失败或 JWT 配置缺失错误。

## 6. 回滚准备

- [ ] 保留上一个可用镜像版本。
- [ ] 数据库变更有 down migration 或备份恢复方案。
- [ ] 回滚后重新验证 `/readyz` 和核心接口。
- [ ] 回滚操作命令已提前记录并演练。
