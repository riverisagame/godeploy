# TASK-006: GoDeployer 前端 UI 设计与开发纳米计划

本计划旨在开发 Vue 3 + Element Plus 暗色系前端管理界面，连接 Go 后端 API，实现只读配置展示、审计部署任务、一键回滚及 Git 差异对比功能。

---

## 1. 拟修改/创建文件清单

- **[NEW]** `web/src/main.ts` - 引入 Element Plus 暗色样式及主程序挂载
- **[NEW]** `web/src/router.ts` - 路由定义（登录、工作台、权限拦截）
- **[NEW]** `web/src/views/Login.vue` - 暗色登录界面
- **[NEW]** `web/src/views/Dashboard.vue` - 核心部署控制台页面
- **[NEW]** `web/src/App.vue` - 主应用入口

---

## 2. 纳米级任务细分 (代码量限制在 10-20 行/步)

### 2.1 任务 1: 前端路由与 Element Plus 暗色系集成
*文件路径*: `web/src/main.ts`, `web/src/router.ts`, `web/src/App.vue`

- **[x] [Sub-Task 1.1]** 编写 `router.ts`，定义路由并加入全局路由守卫（若无 JWT Token 统一重定向到 `/login`）。
- **[x] [Sub-Task 1.2]** 修改 `main.ts`，导入 Element Plus 及暗色样式包（`element-plus/theme-chalk/dark/css-vars.css`）。

### 2.2 任务 2: 登录页面开发
*文件路径*: `web/src/views/Login.vue`

- **[x] [Sub-Task 2.1]** 制作暗色渐变背景的 Login Box 界面。
- **[x] [Sub-Task 2.2]** 编写 Axios 提交登录逻辑，成功后将 token、username 存入 localStorage 并跳转至首页。

### 2.3 任务 3: 主工作台与配置只读展示
*文件路径*: `web/src/views/Dashboard.vue`

- **[x] [Sub-Task 3.1]** 实现左右分栏 Layout：左侧为项目卡片列表，右侧为当前项目环境与操作区。
- **[x] [Sub-Task 3.2]** 编写只读配置解析，使用 Element Plus 的 Descriptions 渲染环境服务器配置信息，禁止提供任何编辑表单。

### 2.4 任务 4: 部署触发、回滚与 Diff 展示
*文件路径*: `web/src/views/Dashboard.vue`

- **[x] [Sub-Task 4.1]** 部署操作区：加入分支、Commit 输入框及“一键部署”按钮，并展示部署历史记录表格。
- **[x] [Sub-Task 4.2]** 绑定“一键回滚” API，并在回滚成功后刷新部署历史。
- **[x] [Sub-Task 4.3]** 对比展示：在历史表格行加入“对比”按钮，弹出 Dialog，使用 `pre` 或专用视图展示 Git 差异 Diff 细节。

---

## 3. 验证与验证确认规划

- **编译与静态分析**：在 `web` 目录运行 `npm run build`，确保没有 TypeScript 和 Vite 打包错误。
- **功能对齐验证**：运行 `npm run dev`，由开发人员通过浏览器配合 Chrome DevTools 确认：
  - 界面展现是否呈现高级暗色现代质感；
  - JWT 是否成功用于鉴权拦截；
  - 部署日志能否通过 EventSource/WebSocket 正常流式呈现；
  - 配置是否符合只读约束。
