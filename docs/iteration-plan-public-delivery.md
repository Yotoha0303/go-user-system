# 公开交付候选版本计划

## 目标

把原有后端仓库整理为可供 GitHub 用户克隆、验证和自托管的全栈 `v1.0.0-rc.1`，同时保留清晰的后续路线图。

## 范围与状态

| 项目 | 状态 | 验收证据 |
| --- | --- | --- |
| 前端纳入主仓库 | 已完成 | `frontend/`、独立 Dockerfile、单元测试 |
| 依赖安全升级 | 已完成 | `govulncheck` 无可达漏洞，npm audit 0 漏洞 |
| 管理员安全初始化 | 已完成 | `bootstrap-admin`、普通注册仅 `user`、注册开关 |
| 本地完整运行 | 已完成 | Compose 自动 migration、前后端健康检查 |
| 浏览器端到端验证 | 已完成 | `frontend/e2e/auth.spec.ts` 与 CI e2e job |
| Kubernetes 可部署清单 | 已完成 | 固定 GHCR 镜像、Migration Job、正确 Ingress 路由 |
| GitHub 质量与发布 | 已完成 | CI、CodeQL、Dependabot、Release workflow |
| 社区与维护文档 | 已完成 | License、Security、Contributing、模板、Roadmap |
| 远程候选版本 | 已完成 | PR #3 合并为 `e532bab`，已推送并发布 `v1.0.0-rc.1` |
| 低危依赖安全补丁 | 已完成 | `edwards25519` 升级到 1.1.1，部署默认值同步到 `v1.0.0-rc.2` |

## 明确不纳入候选版本

- 登录设备列表和会话踢出，保留在 `iteration-plan-redis-device-token.md`。
- 邮箱验证、找回密码和多因素认证。
- 云厂商托管数据库、监控平台和生产 TLS 证书的具体配置。

这些项目不影响候选版本作为学习、自托管验证和二次开发基线，但在面向真实用户的生产环境前应按 `ROADMAP.md` 评估。

## 交付门槛

- Go lint、test、race、vet、build、govulncheck 全部通过。
- 前端 lint、8 个单元测试、TypeScript/Vite build、npm audit 全部通过。
- Goose、Compose、Kubernetes 和 GitHub Actions 配置校验通过。
- Compose 完整栈和 Playwright 核心认证流程通过。
- README 与 `docs/` 不再包含失效的管理员、迁移、镜像或路径说明。
- 功能分支推送、主分支合并、远程 tag 与候选发布均可追溯。
