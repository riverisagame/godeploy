# 纳米级执行计划：前端 UI 组件全覆盖测试
**日期**: 2026-05-29
**计划ID**: PLAN-20260529-UI-TESTS

## 目标
利用 `@vue/test-utils` 和 `vitest` 为现有前端 Vue 3 核心页面补充自动化单元点击测试，确保所有按钮可交互、不断言错误。

## 涉及文件及改动

### 1. `web/src/__tests__/Login.spec.ts` (新建/覆盖)
- **目标逻辑**:
  - `mount(Login)` 并 Mock `useRouter` 与 `axios.post`。
  - 查找表单按钮 `.el-button--primary` (即"登录"按钮) 并 `trigger('click')`。
  - **断言**: 在不输入账号密码时表单的错误拦截是否生效；在输入正确信息点击后，`axios.post` 是否被调用且本地 `localStorage.setItem` 是否被执行，`router.push` 是否跳转到了 `/`。
- **行数**: 新增 ~35 行测试代码。

### 2. `web/src/__tests__/Dashboard.spec.ts` (新建/覆盖)
- **目标逻辑**:
  - `mount(Dashboard)` 并注入 mock 数据 (Mock API)。
  - 查找部署按钮并 `trigger('click')`。
  - **断言**: 点击前拦截是否存在（如未选环境）；点击后 `ElMessageBox.confirm` 是否被触发；在确认部署后 `axios.post('/api/tasks')` 是否被调用。
  - 查找环境标签切换按钮，`trigger('click')`，**断言**: 当前选中环境是否正确改变。
- **行数**: 新增 ~45 行测试代码。

### 3. `web/src/__tests__/UserManagement.spec.ts` (新建/覆盖)
- **目标逻辑**:
  - `mount(UserManagement)`。
  - 查找 "新建用户" 按钮，触发点击。
  - **断言**: 绑定的弹窗变量 `dialogVisible.value` 是否正确切换为 `true`。
  - 查找表格中的 "解绑" 按钮，触发点击。
  - **断言**: 是否正确向后端发起了解绑的 API 请求。
- **行数**: 新增 ~40 行测试代码。

## 污染控制
- 所有测试文件存放在 `__tests__` 目录下。
- 使用 `vi.mock('axios')` 与 `vi.mock('vue-router')` 彻底屏蔽真实网络调用，实现“零侵入”测试，保护现有业务逻辑。
