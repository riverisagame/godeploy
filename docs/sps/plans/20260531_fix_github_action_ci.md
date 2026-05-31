# GitHub Action CI 修复计划

## 1. 发现的问题

在 GitHub Action 中，`Frontend Unit Tests (Vitest)` 失败，具体报错如下：
```text
FAIL  src/__tests__/Dashboard.spec.ts > Dashboard.vue Component UI Test > 2. triggerDeploy 设置 deployState 为 confirming 并打开 Diff
AssertionError: expected 'idle' to be 'confirming' // Object.is equality
```

**原因分析**：
在 `Dashboard.vue` 的 `triggerDeploy` 逻辑中，如果 `deployForm.description` 为空，会直接返回并弹窗警告，导致状态未能变更为 `confirming`。
而在 `Dashboard.spec.ts` 对应的测试用例中，并未模拟填充 `description` 字段。

## 2. 修复方案 (纳米级改动)

### 文件：`web/src/__tests__/Dashboard.spec.ts`
- 定位：测试用例 `2. triggerDeploy 设置 deployState 为 confirming 并打开 Diff` (约第74行附近)。
- 动作：在调用 `vm.triggerDeploy(...)` 之前，设置 `vm.deployForm.description = 'test desc'`。

## 3. 测试与验证策略
- 局部验证：通过 `cd web && npm run test` 验证修复后的测试是否通过。
- 全局验证：推送到 GitHub 后观察 CI 流水线状态。
