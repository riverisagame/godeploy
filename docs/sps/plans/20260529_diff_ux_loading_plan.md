# UI-002 / ARCH-009: 代码对比交互与自愈式快照清理计划 (DIFF_UX_LOADING)

## 1. 计划目标
- 完善前端 diff loading 骨架屏动画样式。
- 后端支持项目级按年月归档的 `task_<id>_diff.log` 快照缓存。
- 后端支持在 `config.yaml` 中全局配置大小限制（`diff_max_size_kb`、`disk_min_space_mb`、`task_retain_max`、`task_retain_days`）。
- 后端引入“先库后盘”最终一致性自愈机制，并提供 `/api/system/prune` 手动修复孤儿文件及老化记录的接口。
- 前端对 `admin` 管理员提供「系统清理与自愈」操作按钮。

---

## 2. 纳米级修改步骤

### [步骤一] 修改 `godeployer/config.go`
- 在 `GlobalConfig` 结构体中添加以下字段以支持全局限额配置：
```go
DiffMaxSizeKB  int `yaml:"diff_max_size_kb"`
DiskMinSpaceMB int `yaml:"disk_min_space_mb"`
TaskRetainMax  int `yaml:"task_retain_max"`
TaskRetainDays int `yaml:"task_retain_days"`
```

### [步骤二] 修改 `godeployer/api.go` 的初始化与默认值设定
- 在创建 `APIHandler` 或在系统启动校验配置时，若 `DiffMaxSizeKB == 0` 则默认为 `2048`；若 `DiskMinSpaceMB == 0` 则默认为 `500`。

### [步骤三] 修改 `godeployer/api.go` 中的 `HandleGetTaskDiff` 方法
- 解析任务的 `created_at` 并提取年月字符（格式如 `"202605"`）。
- 拼装物理路径：`filepath.Join(h.config.Global.LogPath, "diffs", "projects", projectID, createdYM, fmt.Sprintf("task_%d_diff.log", id))`。
- 逻辑调整：
  1. 优先读取快照文件。若文件存在，则读取其内容并直接以 5ms 级速度响应返回。
  2. 若文件不存在且未找到对比基准，则写入 `__EMPTY_DIFF__` 占位符缓存并返回。
  3. 执行 `git diff`，将输出限制在 `DiffMaxSizeKB` 字节以内。超限则进行截断，并在末尾附加超限提示。
  4. 检查系统剩余磁盘空间（通过 Windows/Linux 适配或简易容量安全包检测，如空闲空间大于 `DiskMinSpaceMB`）。若空闲，则创建对应的年月子目录，并将结果写入缓存文件。

### [步骤四] 实现 `HandleSystemPrune` API (`POST /api/system/prune`) 并挂载路由
- 挂载路由：仅允许 `admin` 角色访问 `POST /api/system/prune`。
- 实现 `HandleSystemPrune`：
  1. **主动老化清理**：查询 `deploy_tasks` 表，如果存在超过天数/超出总数的老旧成功或失败任务，在数据库中先执行删除。
  2. **脏数据/孤儿文件物理自愈**：遍历 `LogPath` 及 `LogPath/diffs/` 下的所有 `.log` 和 `_diff.log` 物理文件，解析文件名中的 `task_id`。如果该 `task_id` 在数据库中不存在，物理删除此磁盘文件并释放空间。
  3. 返回被清理文件的总数与释放的字节大小。

### [步骤五] 修改前端 `web/src/views/Dashboard.vue` 样式与操作入口
- 在 `<style>` 末尾补齐 CSS：
```css
.diff-loading-skeleton {
  padding: 10px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.skeleton-bar {
  height: 16px;
  background: rgba(255, 255, 255, 0.05);
  border-radius: 4px;
  animation: skeleton-blink 1.2s infinite ease-in-out;
}
.skeleton-bar.wide { width: 100%; }
.skeleton-bar.medium { width: 75%; }
.skeleton-bar.narrow { width: 40%; }
@keyframes skeleton-blink {
  0% { opacity: 0.3; }
  50% { opacity: 0.6; }
  100% { opacity: 0.3; }
}
```
- 在顶部栏用户名展示区旁，当角色为 `admin` 时，增加一个 `el-button`（图标为 `Refresh` 或 `Brush`），点击调用 `/api/system/prune` 接口，并展示清理通知卡片。

---

## 3. 验证计划

### 自动化测试
1. 编写集成测试，测试 `POST /api/system/prune` 的鉴权与自愈机制。
2. 验证大文件 diff 自动被限额截断。
3. 验证第二次请求相同的 task diff 时，耗时降低到 10ms 以内（即走了持久化缓存）。

### 手动验证
- 启动系统，在前端对比任意任务，观察骨架屏动效。
- 管理员点击「系统自愈清理」，确认自愈物理清理结果输出。
