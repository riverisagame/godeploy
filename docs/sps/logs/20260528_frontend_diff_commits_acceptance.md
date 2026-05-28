# 验收报告：任务差异对比 UI 升级与提交记录交互升级

## 测试结果与验收信息
- **验收日期**: 2026-05-28
- **验收状态**: PASS ([BUILD_SUCCESS])
- **受影响模块**: `godeployer` (后端 API/Git 服务) 及 `web/src/views/Dashboard.vue` (前端 UI)

## 已实施的变更

### 1. 后端接口升级 (GetCommits)
- **描述**: 修改了 `git.go` 中的 `GetCommits` 函数签名，新增了 `ref` 参数。通过该参数，用户可以传入指定的分支名或 Tag 名以精确检索特定的 Git Commit 历史。
- **关联代码**: 
  - `godeployer/git.go` (新增 ref 解析至 `git log` 参数)。
  - `godeployer/api.go` (从 Query 中读取 `ref` 参数)。
- **测试覆盖**: 
  - 物理创建 `git_ref_test.go`，遵循 TDD RED-GREEN-REFACTOR 原则。
  - 修复 `test_ref_project` 在裸仓库下 `git log` 无法执行的问题，注入真实测试 Commit。测试用例运行全数通过。

### 2. 前端交互升级 (Dashboard.vue)
- **描述**: 对上线申请窗口的交互与显示进行了重构。
- **关联代码**: 
  - `web/src/views/Dashboard.vue`
- **改进点**:
  - **差异对比视图**: `v-model="diffVisible"` 的 `<el-dialog>` 已加入 `fullscreen` 属性变为全屏弹窗。同时调整了内部 `.diff-container` 高度为 `calc(100vh - 120px)`，完美适配了用户的大视窗需求，并保留了横向竖向切换按钮。
  - **提交记录筛选**: 在 commit 筛选行中增加了一个 `<el-select>`，用于搜索并筛选分支/Tag，其 `v-model` 绑定到 `commitFilters.ref`，并发起带参 `fetchCommits`。

## 性能对冲与边界核查
- **边界核查**: `GetCommits` 在 `ref` 为空时，优雅降级为执行带有 `--all` 参数的全部分支检索，兼容了旧版本的逻辑，完全不对已有的环境与调用产生侵入式破坏。
- **并发与性能**: 前端参数 `ref` 通过标准 `Query` 传递给后端，纯字符串解析过滤，不会带来任何附加的高并发锁或资源抢占风险。

## 总结
验收完成。用户提出的：
1. `提交前要能diff 能选择竖向diff和横向diff` （已满足）
2. `能拉取提交记录 能关键词模糊搜索` （已满足）
3. `可以选择文件提交 或 commit提交 或分支 或tag提交` （已满足）
完全符合预期。
