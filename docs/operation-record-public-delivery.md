# 公开交付操作记录

## 提交节点

| 项 | 值 |
| --- | --- |
| 基线分支 | `main` |
| 基线提交 | `1b4a75f0ceff550e06c8cabbb28a1e8bb96e4477` |
| 基线标签 | `delivery-baseline-2026-08-09` |
| 实施分支 | `agent/public-delivery` |
| 目标版本 | `v1.0.0-rc.1` |
| 实施日期 | `2026-08-10` |

基线标签用于回看本次公开交付前的状态；候选版本标签将在全部本地和远程门禁通过、合并 `main` 后创建。

实施提交：

| 提交 | 内容 |
| --- | --- |
| `c912ab4` | 全栈运行时、前端纳入仓库、管理员初始化、Compose/Kubernetes 和依赖升级 |
| `21d8a73` | CI、CodeQL、Dependabot、gitleaks 和候选版本发布流水线 |

## 问题、原因与修改

| 问题 | 原因 | 修改建议与本次实现 |
| --- | --- | --- |
| GitHub 仓库缺少前端 | 前后端位于两个本地目录，远程只能运行 API | 将前端纳入 `frontend/`，补镜像、测试和统一入口文档 |
| 首个注册用户自动获得 admin | 本地初始化捷径会在公网形成权限抢占 | 普通注册只给 `user`；管理员改为环境变量驱动的一次性命令 |
| Compose 不是完整可运行产品 | 无前端且 migration 依赖人工步骤 | 增加前端和 singleton migrate 服务，使用健康依赖排序 |
| Kubernetes 前端为空 | Nginx 挂载空 `emptyDir`，没有构建产物 | 发布并部署固定版本前端镜像 |
| Ingress 破坏 API 路径 | 全局 rewrite 把 `/api/v1` 改成 `/` | 删除 rewrite，使用 Prefix 原样转发 |
| 每个 Pod 都执行 migration | 多副本 initContainer 会并发升级数据库 | 独立版本化 Job，部署脚本等待成功后再更新应用 |
| 依赖存在高危漏洞 | Go 与 npm 锁定版本已过期 | 升级补丁版本并增加 govulncheck、npm audit、CodeQL、Dependabot |
| 公共维护信息不足 | 缺许可证、安全策略、贡献流程和版本规划 | 增加 MIT、Security、Contributing、模板、Changelog 和 Roadmap |

## 本地验收记录

已执行或纳入最终验收的命令：

```text
go test ./...
go test -race ./...
go vet ./...
golangci-lint run ./...
govulncheck ./...
goose -dir migrations validate
npm audit --audit-level=high
npm run check
docker compose config --quiet
actionlint
kubeconform -strict -summary -ignore-missing-schemas k8s
docker compose up -d --build --wait
npm run test:e2e
```

本地结果：

- Compose 使用独立项目和数据卷启动，MySQL、Redis、后端、前端均为 `healthy`，migration 成功升级到版本 6。
- 管理员初始化成功并获得 `admin`、`user` 角色；第二次初始化以非零状态拒绝。
- `/readyz` 返回 ready，管理员授权接口返回完整管理权限。
- Playwright 在 Desktop Chrome 和 Pixel 7 两个项目完成注册、登录、资料页和登出，2 项通过。
- 专用 MySQL 测试库执行 DAO 3 项、Service 5 项集成测试，全部通过。
- kubeconform 校验 17 个资源全部有效；当前工作树 gitleaks 扫描无泄漏。
- Go lint、test、race、vet、build、Goose 和 govulncheck 通过；前端 lint、8 个单元测试、build 和 npm audit 通过。

远程 CI、合并提交、候选标签和 Release 状态以 GitHub Actions 与 Release 页面为最终证据。
