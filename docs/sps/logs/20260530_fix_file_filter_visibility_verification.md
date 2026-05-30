# 验收验证报告: 修复上线文件选择过滤 visibility 缺陷

## 1. 验证目标
解决在“触发部署”操作栏内，“上线文件过滤 (取消勾选排除同步)”选项树初始化时不加载以及切换项目和环境时不同步刷新显示的缺陷。

## 2. 验证结果
- **单元测试框架**：Vitest
- **覆盖用例**：
  - `12. 文件过滤联动: 确保切换环境或分支时，会触发接口请求刷新文件树` (验证通过)
- **测试通过率**：100% (共 33 个用例全部绿灯)

---

## 3. 代码变更回顾

### 3.1 前端监听器重构 ([web/src/views/Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue))
- 将单一侦听分支的 watch 重构为联合侦听三方因子的联合侦听器：
  ```typescript
  watch(
    [() => deployForm.branch, () => selectedProject.value?.id, () => activeEnvTab.value],
    async ([newBranch, newProjId, newEnv]) => { ... }
  )
  ```
- 增加了 `{ immediate: true, deep: true }` 参数，确保：
  1. 初次打开页面或第一次载入默认项目分支时，能够被立刻触发去加载文件树，避免被隐藏。
  2. 切换部署项目、切换部署环境时，即使默认分支名称保持相同，也能被灵敏触发以同步更新获取最新的变更文件列表。
- 增加了错误异常处理，如果网络或后端错误，重置并清空文件列表，防脏数据残留。
