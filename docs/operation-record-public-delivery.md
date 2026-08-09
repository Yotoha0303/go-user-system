# 公开交付操作记录

## 提交节点

| 项 | 值 |
| --- | --- |
| 基线分支 | `main` |
| 基线提交 | `1b4a75f0ceff550e06c8cabbb28a1e8bb96e4477` |
| 基线标签 | `delivery-baseline-2026-08-09` |
| 实施分支 | `agent/public-delivery` |
| 合并 PR | [#3](https://github.com/Yotoha0303/go-user-system/pull/3) |
| 合并提交 | `e532bab584312fb867ac3167715d07b5c3ca5aa5` |
| 初始候选版本 | [`v1.0.0-rc.1`](https://github.com/Yotoha0303/go-user-system/releases/tag/v1.0.0-rc.1) |
| 安全补丁候选版本 | `v1.0.0-rc.2` |
| 实施日期 | `2026-08-10` |

基线标签用于回看本次公开交付前的状态；候选版本标签在本地验收、PR 门禁、合并后主分支 CI 和 CodeQL 全部通过后创建。

实施提交：

| 提交 | 内容 |
| --- | --- |
| `c912ab4` | 全栈运行时、前端纳入仓库、管理员初始化、Compose/Kubernetes 和依赖升级 |
| `21d8a73` | CI、CodeQL、Dependabot、gitleaks 和候选版本发布流水线 |
| `ee64107` | README、部署文档、交付记录、社区文件和版本说明 |
| `c527d62` | E2E 清理环境和 Go CodeQL 构建模式修复 |
| `bac9fe0` | 合并、CI、发布和制品校验结果的文档收尾记录 |

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
| `rc.1` 发布后出现低危告警 | GitHub Advisory Database 新报告 `edwards25519 < 1.1.1` 问题 | 升级到 1.1.1，并以 `v1.0.0-rc.2` 重新发布所有制品 |
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

## 远程验收记录

- PR #3 的后端、前端、部署清单、Playwright E2E、Go CodeQL 和 JavaScript CodeQL 全部通过后，以 merge commit 合并。
- 合并提交的 [主分支 CI](https://github.com/Yotoha0303/go-user-system/actions/runs/31326670292) 和 [CodeQL](https://github.com/Yotoha0303/go-user-system/actions/runs/31326670295) 均通过；E2E 为 2 项通过并成功清理 Compose 资源。
- [`v1.0.0-rc.1` 发布工作流](https://github.com/Yotoha0303/go-user-system/actions/runs/31327121410) 通过，GitHub prerelease 包含 Linux amd64、Linux arm64、Windows amd64、前端静态包和校验和文件。
- 逐项下载 4 个归档并与 `checksums.txt` 比对，SHA-256 全部匹配。
- `ghcr.io/yotoha0303/go-user-system-backend:v1.0.0-rc.1` 和 `ghcr.io/yotoha0303/go-user-system-frontend:v1.0.0-rc.1` 可匿名读取，均包含 `linux/amd64` 与 `linux/arm64`。
- `main` 已要求 PR、分支同步、6 项状态检查和讨论解决，并禁止强推和删除；漏洞告警、Dependabot 安全更新和私密漏洞报告已启用。
- `rc.1` 发布后发现的低危 Dependabot 告警已在补丁候选 `v1.0.0-rc.2` 中修复；该标签用于追溯包含补丁的最终合并提交。

## 已知后续项

- 旧 Git 历史曾包含 Kubernetes Secret 示例。当前分支已删除该文件并使用示例占位值，但公开历史不可视为秘密存储；任何曾使用过的同值凭据必须轮换。
- `govulncheck` 对代码路径和导入包报告 0 漏洞；依赖图中的 `golang.org/x/crypto/openpgp` 有无修复版本的模块级公告，但项目没有导入或调用该包，后续继续跟踪上游处理。
- 后端多架构镜像首次无缓存构建约 21 分钟，功能正确但发布效率需要通过原生构建平台交叉编译优化。
- Dependabot 首次扫描创建的其余依赖升级 PR 应逐项评估兼容性并通过现有门禁合并，不应直接批量升级主版本。
