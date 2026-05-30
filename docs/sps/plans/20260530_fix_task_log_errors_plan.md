# 纳米级执行计划 - 修复任务部署 Git 错误与 Vue 组件导入缺失

## 1. 拟修改与新增文件清单

### [MODIFY] [godeployer/git.go](file:///d:/claudeprj/deploy/godeployer/git.go)
- 修改 `EnsureRepoCache` 函数，在 fetch 之前读取并校验裸仓库 `origin` URL 是否与当前的 `repoURL` 一致。如果不一致，清空缓存并重新克隆。

### [MODIFY] [scripts/demo.sh](file:///d:/claudeprj/deploy/scripts/demo.sh)
- 优化 `seed_db` 逻辑：无论数据库是否存在，都确保执行 `rm -rf "$WORKSPACE"/.cache`。
- 优化 `create_mock_repos` 逻辑：将所有 mock 提交均提交到 `master` 分支打上 tag，然后再通过 `git branch develop master` 或从特定 master commit 迁出 `develop` 分支，保持 refs 完整。

### [MODIFY] [web/src/views/Dashboard.vue](file:///d:/claudeprj/deploy/web/src/views/Dashboard.vue)
- 手动导入需要的局部子组件 `ProjectSidebar.vue`, `DeployForm.vue`, `DeployHistoryTable.vue`, `DiffDialog.vue`。

---

## 2. 细化子任务与修改逻辑

### 任务 2.1: 增强 `godeployer/git.go` 的缓存 remote 校验
在 `EnsureRepoCache` 中：
```go
	// 如果目录已存在，先校验其 remote origin 是否与当前请求 of repoURL 相同
	if _, err := os.Stat(cacheDir); err == nil {
		cmdCheck := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
		cmdCheck.Dir = cacheDir
		if out, err := cmdCheck.CombinedOutput(); err == nil {
			currentRemote := strings.TrimSpace(string(out))
			if currentRemote != repoURL {
				// 若不一致，直接删除本地缓存以重新 clone
				os.RemoveAll(cacheDir)
			}
		} else {
			// 获取失败说明本地可能不是正常的 bare 库，清空以重新 clone
			os.RemoveAll(cacheDir)
		}
	}
```

### 任务 2.2: 优化 `scripts/demo.sh` 清理及分支生成
在 `scripts/demo.sh` 的 `seed_db()` 中：
```bash
  # 将缓存和日志清理移出数据库判断
  info "清空旧的任务部署记录、本地日志与 Git 缓存..."
  rm -f "$LOG_DIR"/task_*.log 2>/dev/null || true
  rm -rf "$WORKSPACE"/.cache 2>/dev/null || true
  
  if [ -f "$DB_PATH" ]; then
    sqlite3 "$DB_PATH" "DELETE FROM deploy_tasks;"
  fi
```

在 `create_mock_repos()` 的 `_generate_mock_repo()` 中，确保所有 5 个 commit 按顺序提交至 `master`，打上 Tag。然后通过分支命令建立 `develop` 分支：
```bash
    # 1. feat: init project
    ...
    # 2. fix: resolve issue
    ...
    # 3. feat: add API endpoints
    ...
    # 4. perf: optimize performance
    ...
    # 5. docs: update comments
    ...
    # 线性提交打完 tag 以后，拉出 develop 分支以确保所有 commit 均能在任意克隆中被 checkout 到
    git branch -f develop master
```

### 任务 2.3: 修复 `web/src/views/Dashboard.vue` 组件导入
在 `web/src/views/Dashboard.vue` 的 `<script setup lang="ts">` 第一行，补充：
```typescript
import ProjectSidebar from '../components/ProjectSidebar.vue'
import DeployForm from '../components/DeployForm.vue'
import DeployHistoryTable from '../components/DeployHistoryTable.vue'
import DiffDialog from '../components/DiffDialog.vue'
```

---

## 3. 验证方案
- 运行 `cd web && npm run build` 确保前端构建正常，组件被正确编译并无打包报错。
- 运行 `bash scripts/demo.sh` 清理旧缓存、重新生成 Mock 仓库，编译并启动最新后端服务，触发真实 API 模拟部署。
- 查看 `demo_logs/` 下所有生成的 `task_*.log`。期望结果为：所有任务均部署成功（非权限受限或端口受限的预期失败外，所有 checkout 动作 100% 成功，无 `unable to read tree` 报错）。
