# 验收与验证报告 - 增量/全量双模式 Diff 及快照归档

## 1. 测试用例与通过情况

### 1.1 后端 Go 测试
- **测试路径**：`godeployer/api_enhance_test.go`
- **新增用例**：`TestAPI_DualDiff_PersistenceAndFallback`
- **验证细节**：
  1. 验证在全量上线（`target_type = "branch"` / `"tag"`）产生的快照中，若前端请求 `diff_type = live`，后端安全降级并返回友好说明：“提示：全量部署任务，未归档与线上对比快照。请在右上方切换为「本地变更(Git Log Diff)」查看文件修改。”，不返回报错。
  2. 验证前端请求 `diff_type = git_log` 时，后端精准返回 `git_log_diff` 归档块的内容。
- **运行命令**：
  ```bash
  go test -v -run TestAPI_DualDiff_PersistenceAndFallback ./godeployer
  ```
- **测试结果**：**PASS**

### 1.2 前端 Vitest 单元测试
- **验证细节**：检查并确保新增的前端双单选按钮状态及参数在懒加载时全部运转健康。
- **运行命令**：
  ```bash
  cd web && npx vitest run
  ```
- **测试结果**：**34 tests passed**

---

## 2. 编译与打包确认
- 运行 `go build .` 将 `web/dist/` 彻底嵌入后端生产二进制中，构建成功，无任何 TypeScript/Golang 编译警报。

## 3. 部署快照策略落地明细
- **Commit（增量上线）**：同时归档 `diff` (Live Diff) 与 `git_log_diff` (Git Log Diff)，历史详情支持双向切换。
- **Branch / Tag（全量上线）**：上线前支持双按钮比对，上线后**仅归档** `git_log_diff`，并在前端置灰禁用“与线上对比”按钮，提示用户，有效防止大范围发布导致快照文件过大进而耗尽服务器磁盘空间。
