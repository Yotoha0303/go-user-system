# 前端认证、RBAC 与工程化改造操作记录

日期：2026-07-14

## 目标

- 对齐后端 JWT Access/Refresh 双 Token 契约。
- 建立可靠的会话恢复、并发刷新和权限路由。
- 完成个人资料、安全设置和 RBAC 管理页面。
- 补齐响应式、可访问性、自动化测试和项目文档。

## 操作记录

### 1. API 与认证状态重构

改动：

- 将 Axios 客户端分为 public/private 实例，并统一解析后端响应和错误。
- Access Token 仅保存在 Redux 内存状态。
- Axios 全局启用 `withCredentials`，Refresh Token 由后端 HttpOnly Cookie 管理。
- 增加单飞刷新协调器；并发 401 只发送一次刷新请求，成功后重放各自原请求。
- 403 不进入刷新逻辑；刷新失败统一清理会话。
- 增加启动恢复状态，避免页面刷新时先渲染匿名页面再跳转。
- 删除 `js-cookie`、`jwt-decode` 和旧的重复 Axios Hook/Service。

问题 - 原因 - 修改建议 - 示例：

- 问题：旧实现由多个 Hook 和 Service 分别管理 Token，容易出现竞态和重复刷新。
- 原因：HTTP 客户端、Redux 状态与组件生命周期之间没有唯一会话入口。
- 修改建议：集中到 API auth bridge 与单飞刷新协调器，组件只消费认证状态。
- 示例：两个请求同时收到 401 时，测试确认只有一个 `/api/v1/auth/refresh` 请求。

### 2. 权限与路由

改动：

- 登录和会话恢复后加载 `/api/v1/users/me/authorization`。
- 增加 `RequireAuth`、`RequireAnonymous` 和 `RequirePermission`。
- 导航菜单根据权限码展示管理入口。
- 增加 403、404 页面；移除无业务价值的演示页。
- 管理页支持角色列表、权限列表以及按用户 ID 分配角色，并对页面内操作继续做权限判断。

问题 - 原因 - 修改建议 - 示例：

- 问题：只隐藏菜单不能构成权限控制，用户仍可直接访问 URL。
- 原因：菜单展示与路由访问没有共享同一权限状态。
- 修改建议：菜单和路由都读取 Redux 权限码，后端中间件继续作为最终安全边界。
- 示例：缺少 `admin:roles:read` 时访问 `/admin/access` 会进入 `/forbidden`。

### 3. 业务页面与交互

改动：

- 重做登录、注册、个人资料、昵称修改和密码修改页面。
- 表单增加显式 label、autocomplete、提交 loading、字段错误与服务端错误。
- 改密成功后清理本地认证状态并要求重新登录。
- 登出请求失败时仍清理本地会话，避免界面停留在错误的已登录状态。
- 使用 Lucide 图标、明确的按钮层级、表格横向滚动和移动端自适应布局。
- 清理 Vite/React 样板资源、默认标题和强制打开浏览器配置。

### 4. 测试与质量门禁

新增测试：

- 单飞刷新协调器。
- Axios 并发 401 刷新与 403 不刷新。
- Redux 认证状态转换。
- 权限路由放行与拦截。
- 登录后加载资料与权限。
- 服务端登出失败时仍清理本地状态。

新增命令：

```bash
npm run lint
npm run test
npm run build
npm run check
```

## 关键文件

- `src/api/client.ts`：Axios、Access Token 注入、刷新和请求重放。
- `src/app/authSlice.ts`：会话、用户资料和授权状态。
- `src/components/auth/auth-bootstrap.tsx`：启动恢复。
- `src/components/auth/require-permission.tsx`：路由权限控制。
- `src/pages/admin/access-page.tsx`：RBAC 管理界面。
- `src/api/client.test.ts`：并发刷新行为测试。

## 验收结果

已执行：

```bash
npm run lint
npm test
npm run build
```

结果：

- ESLint 通过，0 warning。
- Vitest 共 6 个测试文件、8 个测试用例，全部通过。
- TypeScript 与 Vite 生产构建通过；主 JavaScript 产物 gzip 后约 112 KB。
- 后端 Compose 应用健康检查为 healthy，前端开发代理可正常访问后端 API。
- 使用 Edge DevTools 设备模拟完成真实浏览器注册、登录、资料加载和权限拒绝链路。
- 普通用户登录后进入 `/profile`；直接访问 `/admin/access` 被重定向到 `/forbidden`。
- 390x844 移动视口：`innerWidth=390`，`scrollWidth=390`。
- 1440x900 桌面视口：`innerWidth=1440`，`scrollWidth=1440`。
- 桌面和移动端资料页均无横向溢出、导航重叠或文本遮挡。

问题 - 原因 - 修改建议 - 示例：

- 问题：直接使用 Edge `--window-size=390,844` 截图时出现右侧裁切。
- 原因：Windows 无头窗口存在约 500px 的最小布局宽度，输出图片宽度与 CSS 布局视口不一致。
- 修改建议：通过 DevTools `Emulation.setDeviceMetricsOverride` 设置真实 390px CSS 视口，并用 `scrollWidth <= innerWidth` 做数值断言。
- 示例：最终移动端结果为 `390 <= 390`，确认页面没有实际横向溢出。

环境说明：

- 组合命令 `npm run check` 在受限沙箱中会因 esbuild 子进程被拒绝而报 `spawn EPERM`。
- 将同一组命令放到已授权环境分别执行后，lint、test、build 全部通过；该错误不是项目代码或测试失败。
