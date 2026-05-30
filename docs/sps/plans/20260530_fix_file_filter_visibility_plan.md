# 纳米级执行计划: 修复上线文件选择过滤 visibility 缺陷

## 1. 拟修改文件清单

### [MODIFY] [web/src/views/Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue)
- 优化用于获取可选变更文件树的 watch 侦听器。

---

## 2. 细化子任务与修改逻辑

### 任务 2.1: 重写 `watch(() => deployForm.branch, ...)`
- **目标路径**：`web/src/views/Dashboard.vue` 第 506-524 行。
- **改动详情**：
  - 将单一侦听源改为联合侦听源数组：`[() => deployForm.branch, () => selectedProject.value?.id, () => activeEnvTab.value]`。
  - 在回调函数中，采用联合侦听的新入参，如果项目、分支或环境其中之一为空，将文件相关变量清空返回。
  - 增加 `{ immediate: true, deep: true }` 选项。
- **预估代码改动量**：约 20 行。

---

## 3. 验证方案

### 自动化测试
- 我们将更新已有的单元测试用例，增加对 watch 联合侦听和 immediate 首次加载触发渲染的验证断言。
- 运行 `cd web && npm run test` 确保无异常。

### 手动验证
- 启动项目，首次载入页面，确认“上线文件过滤”勾选列表已经能立即正确渲染显示出来。
- 切换不同项目、切换不同部署环境，确认文件勾选树依然可以联动刷新。
