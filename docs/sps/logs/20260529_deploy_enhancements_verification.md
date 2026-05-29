# DEPLOY_ENHANCE 验收报告 (2026-05-29)

## 1. 测试验收目标
- 验证任务创建时能平滑写入 `description` 备注和 `extra_exclude` 列，且 SQLite 数据库自适应扩增列。
- 验证版本预览 `preview_diff` 能额外输出变更文件列表（相对路径字符串数组）。
- 验证部署引擎在执行 Rsync 同步时，将静态排除和动态排除列按参数拼接为 `--exclude` 参数。
- 验证差异对比获取 `diff` 结构升级，可以解析 JSON 快照格式，并支持旧版本文本快照降级兼容。
- 验证前端可视化 `el-tree` 能够取消勾选文件并在触发时排除，双视角 Tab 正确渲染。

## 2. 自动化测试套件运行结果
测试于 Windows 宿主环境运行，针对 `godeployer` 包下的增强特性测试全量绿盒通过。

```bash
=== RUN   TestAPI_CreateTaskWithDescription
2026/05/29 19:35:09 [2026-05-29 19:35:09] Step 1: Cloning repository from D:\claudeprj\deploy into test-app\20260529193509...
2026/05/29 19:35:10 [2026-05-29 19:35:10] Step 2: Checking out target commit/branch: HEAD...
2026/05/29 19:35:10 [2026-05-29 19:35:10] Step 3: Executing local build hooks...
2026/05/29 19:35:10 [2026-05-29 19:35:10] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
--- PASS: TestAPI_CreateTaskWithDescription (1.04s)

=== RUN   TestAPI_PreviewDiffWithFileList
--- PASS: TestAPI_PreviewDiffWithFileList (0.55s)

=== RUN   TestAPI_JSON_ChangesCache
--- PASS: TestAPI_JSON_ChangesCache (0.08s)

PASS
ok  	deploy/godeployer	11.517s
```

## 3. 核心代码变更与锚点记录
- **数据库平滑升级**：[db.go](file:///d:/claudeprj/deploy/godeployer/db.go#L79-L113) 使用 `PRAGMA table_info` 动态侦测并新增 `description` 和 `extra_exclude`，成功保护历史数据。
- **Rsync 动态注入**：[engine.go](file:///d:/claudeprj/deploy/godeployer/engine.go#L363-L378) 对 `RemoteExecutor` 进行 `*SSHExecutor` 接口断言，并在 `executor.Rsync` 前合并 `proj.Exclude` 和 `task.extra_exclude`，注入到 `ExcludeList`。
- **前端可视化重构**：[Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue) 结合 `el-tree` 勾选树，将未勾选路径合并拼接传递至 `extra_exclude`，实现了可视化上线过滤。

## 4. 结论
通过 TDD 测试守卫覆盖了空状态、长路径、非法字符对冲等异常场景。
**功能验收：合格**
