# 生产部署前检查清单

本清单用于防止把本地开发配置直接带到生产环境。

## 配置与密钥

- [ ] `JWT_SECRET` 已替换为 32 位以上强随机字符串。
- [ ] `DB_PASSWORD` 已替换为生产数据库密码。
- [ ] 生产环境不使用 MySQL `root` 用户连接业务库。
- [ ] `.env`、`.env.*`、`config.local.yml` 未提交到 Git。
- [ ] `APP_PORT`、`DB_HOST`、`DB_PORT`、`DB_USER`、`DB_NAME` 与部署环境一致。

## 构建与质量门禁

- [ ] `go test ./...` 通过。
- [ ] `go vet ./...` 通过。
- [ ] `go build -o go-user-system ./cmd` 通过。
- [ ] `docker build -t go-user-system:ci .` 通过。

## 数据库

- [ ] 目标数据库已创建。
- [ ] 应用启动后已自动执行 `migrations/` 下的 `.up.sql` 脚本。
- [ ] `schema_migrations` 表中能看到已执行的 migration 版本。
- [ ] 数据库字符集使用 `utf8mb4`。
- [ ] 已确认数据库备份和恢复方式。

## 服务验证

- [ ] `GET /livez` 返回 200。
- [ ] `GET /readyz` 返回 200，并能证明数据库可访问。
- [ ] 注册、登录、鉴权、修改昵称核心链路验证通过。
- [ ] 应用日志中没有数据库连接失败或 JWT 配置缺失错误。

## 回滚准备

- [ ] 保留上一个可用镜像版本。
- [ ] 数据库变更有对应 down migration 或备份恢复方案。
- [ ] 回滚后重新验证 `/readyz` 和核心接口。
