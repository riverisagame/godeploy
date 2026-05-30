# ADR: 修复部署面板中“上线文件选择过滤”不显示的 Visibility Bug

## 问题背景
用户反馈在触发部署面板中“仍然没看到可选文件的界面”。

## 根本原因分析
触发部署表单中的“上线文件选择过滤”由 `<div v-if="rawFilesList.length > 0">` 控制是否显示。该变量的数据拉取仅依赖于以下单变量监听器：
```typescript
watch(() => deployForm.branch, async (newVal) => { ... })
```
这导致了三个致命的显示边界 Bug：
1. **未设置 `immediate: true`**：当页面初次加载，且通过 `selectProject` 赋了默认分支初始值（如 `main`）时，因为 `deployForm.branch` 的值并没有在 watch 挂载之后发生“改变”，导致该监听器不执行，初始化时面板直接被隐藏。
2. **忽略了项目与环境的切换**：当用户在左侧切换部署项目，或者在顶部切换部署环境时，如果选中的分支名恰好相同（例如均是默认的 `main` 或者是 `master`），由于监听的值没有改变，watch 同样不会触发。这导致新项目/新环境的变更文件列表无法获取，面板显示为空或显示上一项目的过期文件数据。
3. **缺少错误降级清空**：如果请求发生错误，应该将文件列表清空，而不是保留上一次的结果。

---

## 解决方案：多因子联合侦听器

重构该侦听器，监听 `deployForm.branch`、`selectedProject` 和 `activeEnvTab` 的三方联合变化，并指定 `{ immediate: true }`，确保首次加载、项目切换、环境切换时均能立刻更新并刷新可选文件树：
```typescript
watch(
  [() => deployForm.branch, () => selectedProject.value?.id, () => activeEnvTab.value],
  async ([newBranch, newProjId, newEnv]) => {
    // 自动刷新可选文件树
  },
  { immediate: true }
)
```

---

## 修改可能触达的现有功能
- 前端：`web/src/views/Dashboard.vue`。
