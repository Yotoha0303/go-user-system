# go-user-system-frontend

Go User System 的 React 前端，覆盖注册登录、会话恢复、个人资料、密码修改和 RBAC 管理操作。

## 技术栈

| 类型 | 技术 |
| --- | --- |
| UI | React 18、TypeScript、Tailwind CSS、Lucide |
| 路由 | React Router 7 |
| 状态 | Redux Toolkit |
| 表单 | React Hook Form、Yup |
| HTTP | Axios |
| 测试 | Vitest、Testing Library、jsdom、Playwright |
| 质量检查 | ESLint、TypeScript、Vite build |

## 界面结构

- 匿名页面使用完整背景视觉和聚焦的认证表单，桌面端与移动端共享同一套表单流程。
- 登录后的工作区在桌面端使用固定侧边栏，在移动端使用可关闭的抽屉导航。
- 资料、安全和权限页面复用统一的标题、表单、反馈和数据表格样式。
- 认证背景资源位于 `public/images/identity-access-background.webp`。

## 本地运行

前置条件：后端监听 `http://127.0.0.1:8082`，Node.js 22.22.2 或更高版本。

```bash
npm ci
npm run dev
```

访问 `http://127.0.0.1:8888/`。开发服务器默认把 `/api` 代理到 `http://localhost:8082`。

需要直连其他后端时设置：

```dotenv
VITE_BACKEND_URL=http://127.0.0.1:8082
```

跨域直连时，后端必须允许凭证请求并配置明确的前端 Origin；生产环境建议由同一站点反向代理前后端。

## 页面与权限

| 路径 | 功能 | 前端要求 |
| --- | --- | --- |
| `/auth/login` | 登录 | 仅匿名用户 |
| `/auth/signup` | 注册 | 仅匿名用户 |
| `/profile` | 个人资料 | 已登录 |
| `/profile/edit-nickname` | 修改昵称 | `profile:update` |
| `/security/password` | 修改密码 | `password:update` |
| `/admin/access` | 角色、权限与用户角色分配 | `admin:roles:read`，页面内继续按权限控制功能 |
| `/forbidden` | 无权限页面 | 无 |

前端权限判断只负责用户体验，后端 Gin 权限中间件仍负责最终鉴权。

## 认证流程

- Access Token 只保存在 Redux 内存状态，不写入 localStorage、sessionStorage 或可读 Cookie。
- Refresh Token 由后端写入 HttpOnly Cookie，Axios 使用 `withCredentials` 发送。
- 页面刷新时，`AuthBootstrap` 先尝试 Cookie 刷新，再并行加载用户资料和授权信息。
- 受保护请求返回 401 时，由单飞刷新协调器合并并发刷新，再重放原请求。
- 403 不触发刷新，直接进入权限错误处理。
- 登出即使遇到网络错误也会清理本地会话；改密后强制重新登录。

## 质量检查

```bash
npm run lint
npm run test
npm run build
npm run check
```

针对运行中的完整 Compose 栈执行浏览器测试：

```bash
npx playwright install chromium
npm run test:e2e
```

构建生产镜像：

```bash
docker build -t go-user-system-frontend:dev .
```

生产镜像监听 `8080`，支持 SPA fallback，并在 Compose 中把 `/api` 代理到后端。Kubernetes 入口会直接把 `/api` 路由到后端 Service。

认证与 RBAC 历史记录见 `docs/operation-record-frontend-auth-rbac.md`；本次界面改造见仓库根目录的 `docs/operation-record-frontend-modernization.md`；全栈公开交付记录见仓库根目录的 `docs/operation-record-public-delivery.md`。
