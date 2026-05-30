# ARCH-010 / TASK-010: 动态过滤、双视角快照与部署备注高级特性设计 (DEPLOY_ENHANCEMENTS)

## 1. 需求深挖与对齐 (SCAN)

用户提出了三个核心的部署流体验和审计加强需求，我们逐一进行细化和对齐：

### 需求 A: 上线文件可视化动态过滤排除
- **现状**：项目配置中的 `exclude` 字段是全局硬编码定义的，无法根据当次部署进行灵活调整。
- **优化与可视化文件树**：
  - **接口升级**：升级 `viewerGrp.GET("/projects/:id/preview_diff")`。当用户在上线界面选择好要上线的 Commit 后，前端自动或手动触发“获取发布预览”。后端除返回 diff 文本外，额外执行 `git diff --name-only <from> <to>`，将本次变更的文件相对路径列表（例如 `["src/App.vue", "src/components/Tree.vue", "package.json"]`）作为 `files` 字段一并返回。
  - **前端可视化树**：前端在“触发上线”对话框中，将扁平的变更路径列表在内存中自动构建为 **带有 checkbox 的文件/文件夹树**（使用 `el-tree` 渲染）。
    - 树中**只展示本次变动过的文件**（而不是项目的全部数万个文件），这不仅极度聚焦于本次上线的具体修改，而且规避了加载巨型文件树导致的浏览器卡死。
    - 节点默认全选。用户可直接取消勾选某个文件或文件夹。
  - **排除数据投递**：在触发上线时，前端通过 API 获取当前未被勾选的所有文件节点路径，将其作为 `extra_exclude` 字符串数组（或逗号分隔字符串）传给后端。
  - **后端 rsync 整合**：DeployEngine 在执行文件发布（rsync）时，除了加载项目默认的 `exclude` 规则，还会将前端投递的 `extra_exclude` 文件列表动态转换为 rsync 的 `--exclude='...'` 命令行参数传入，从而保证未勾选的文件绝对不会同步到远程生产服务器上。


### 需求 B: 部署文件列表 (File List) 与代码差异 (Code Diff) 双视角切换
- **现状**：现有的对比窗口只显示代码 diff 文本，计算慢，且用户无法快速感知本次上线“一共改变了哪些文件”。
- **优化**：
  - 后端：我们对 `/api/tasks/:id/diff` 进行升级，或引入新接口 `/api/tasks/:id/changes`。出于性能考量，我们可以让 `HandleGetTaskDiff` 直接返回包含两个属性的 JSON：
    ```json
    {
      "files": "M src/App.vue\nA src/components/Skeleton.vue\nD old.js",
      "diff": "diff --git ..."
    }
    ```
    - 变更文件列表由 `git diff --name-status <prev> <current>` 快速计算。
    - 持久化缓存：将文件列表一同缓存在磁盘年月子目录中，或单独缓存为 `task_<id>_files.log`。为了保持高内聚，我们创建 `task_<id>_changes.json` 代替原来的 `_diff.log` 纯文本，里面结构化存储 `{"files": "...", "diff": "..."}`。若历史快照文件是纯文本，则降级进行平滑兼容。
  - 前端：对比对话框增加 `el-tabs` 切换：
    - **Tab 1: 变更文件列表 (名称与修改状态)**
    - **Tab 2: 代码差异比对 (现有的高亮 diff 视图)**

### 需求 C: 部署原因/备注审计功能
- **现状**：历史表格只有版本、ID 和时间，没有备注，审计员无法了解某次部署是出于什么目的（如“修复线上Bug”、“添加自愈功能”）。
- **优化**：
  - 后端：在数据库 `deploy_tasks` 表中新增 `description TEXT DEFAULT ''`（部署备注）及 `extra_exclude TEXT DEFAULT ''` 字段。
  - 后端：`HandleCreateTask`（创建部署任务）时读取前端传参 `description` 写入数据库中。
  - 前端：在“触发上线”时提供多行文本域「部署备注（选填）」。
  - 前端：在“部署与审计历史”表格中，新增一列「部署注释/备注」，清晰直观展示每次上线意图。

---

## 2. 物理零污染与 DDL 绝对禁绝对冲审计
- **表结构变更保障**：
  - SQLite 本身不支持 `ALTER TABLE ADD COLUMN IF NOT EXISTS` 复合语法，但支持 `ALTER TABLE deploy_tasks ADD COLUMN description TEXT DEFAULT ""`。
  - 我们将在 `InitDB` 时，利用 `PRAGMA table_info(deploy_tasks)` 提前检测列是否存在，若不存在则动态调用 `ALTER TABLE`，避开 `DROP` 或 `TRUNCATE`，100% 保证现有数据毫发无损。
- **文件迁移平滑兼容**：
  - 遇到升级前的旧 `_diff.log` 文件（非 JSON），后端在读取时采用降级解析，把文件内容完整填入 `diff` 字段，且 `files` 字段置为 `"历史备份，请重新计算"`。不损坏任何历史缓存。

---

## 3. 触达文件与影响范围
1. `godeployer/db.go`：新增安全 schema 迁移脚本。
2. `godeployer/config.go`：支持在任务创建入参解析。
3. `godeployer/api.go`：
   - 升级 `HandleCreateTask` 以保存备注和临时排除规则。
   - 升级 `HandleGetTaskDiff` 以同步输出并缓存 `changes.json`（文件列表 + diff 文本）。
4. `godeployer/engine.go`：在执行 rsync 或 git diff 阶段，动态接收并拼装 `extra_exclude` 参数。
5. `web/src/views/Dashboard.vue`：
   - 触发上线弹窗增加“部署备注”输入框与“临时屏蔽文件”配置项。
   - 历史记录表格加一列“部署备注”。
   - 对比弹窗增加变更文件列表 Tab 和 Diff 文本 Tab。
