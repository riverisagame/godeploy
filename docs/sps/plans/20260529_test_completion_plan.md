# 纳米级测试执行计划 (IR) - 全功能测试补齐 (修订版)

## 目标
根据 `[SCAN]` 阶段审计的后端防线与前端脆弱点，生成针对每个文件的原子化测试补充计划。
所有测试必须使用内存数据/Mocks，遵守零物理污染原则。特别强化：**确保前端所有可点击的交互组件均受到功能闭环的测试约束。**

## 阶段 1: 修复前端测试环境与架构 (Frontend Env)
### Task 1.1: 修复 Node Modules 依赖
- **动作**: 创建并执行 `scripts/fix_test_env.sh`
- **逻辑**: 执行 `cd web && rm -rf node_modules package-lock.json && npm install`
- **出口**: `vitest run --coverage` 不再报 `binding` 错误。

## 阶段 2: 后端并发与防御护城河补齐 (Backend Defenses)
### Task 2.1: `godeployer/engine_test.go` - 测试 Shell 注入防御与排除合并
- **逻辑**: 新增 `TestDeployEngine_ExcludeInjection`。传入 `ExtraExclude: []string{"; rm -rf /", "*/sensitive"}`，断言传入 `rsync` 的切片中分号不被执行。
### Task 2.2: `godeployer/engine_test.go` - 测试并发锁与排队
- **逻辑**: 使用 `sync.WaitGroup` 发起两个具有相同 `ProjectID` 的并发 `RunDeploy`，断言引擎只会执行一个。
### Task 2.3: `godeployer/db_test.go` - 测试断电自愈
- **逻辑**: 写入状态为 `RUNNING` 的 task，调用 `RepairStalledTasks()`，断言其回退为 `FAILED`。

## 阶段 3: 前端降级测试与 "100% 点击功能正确性" 组件覆盖 (Frontend UI)
### Task 3.1: `Dashboard.spec.ts` - 核心看板点击组件覆盖
- **文件路径**: `web/src/__tests__/Dashboard.spec.ts`
- **逻辑**: 编写套件拦截所有 `axios` 和 `vue-router` 调用，使用 `@vue/test-utils` 的 `wrapper.find().trigger('click')` 逐个断言以下功能正确执行并发送了正确的 Payload 或路由变更：
  1. **顶栏**: [用户管理跳转] `router.push('/users')`
  2. **顶栏**: [系统清理] `handleSystemPrune` 发送 POST `/api/system/prune`
  3. **顶栏**: [系统配置] `openSettings` 与 `saveSettings` 的弹窗状态和 API 断言
  4. **顶栏**: [登出] `handleLogout` 清理 localStorage
  5. **侧边栏**: [切换项目] `selectProject` 触发项目获取和详情加载
  6. **环境块**: [触发上线] 弹窗展现、输入框 `setValue`、树组件半选勾选断言
  7. **环境块**: [Diff 对比弹窗] `previewDeployDiff` 接口请求与弹窗渲染
  8. **历史记录**: [查看日志]、[查看文件变动]、[回滚] 动作断言。
  9. **WebSocket 降级**: 断开 `mockWs` 后，确认系统使用 `setInterval` 轮询发起 GET 请求。

### Task 3.2: `UserManagement.spec.ts` - 用户模块点击组件覆盖
- **文件路径**: `web/src/__tests__/UserManagement.spec.ts`
- **逻辑**:
  1. [返回控制台] `router.push('/')`
  2. [新增用户] 触发新建弹窗，[保存] 触发 POST `/api/users`
  3. [编辑] 触发表单数据回显，[保存] 触发 PUT 请求
  4. [删除] 断言确认弹窗与 DELETE 请求发出，断言 'admin' 用户按钮为 `disabled`。

### Task 3.3: `Login.spec.ts` - 登录模块
- **文件路径**: `web/src/__tests__/Login.spec.ts`
- **逻辑**:
  1. 输入账号密码后，[登录按钮] 的防抖与 API 请求。
