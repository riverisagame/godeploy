# 架构决策记录: 修复前端加载过大 Diff 时的卡顿冻结问题

## 1. 背景与现象
用户报告：“现在点触发上线 加载diff时 卡住”。
经过排查：
- 前端 `Dashboard.vue` 的 `triggerDeploy` 动作会调用 `previewDeployDiff`，向后端 `/api/projects/{id}/preview_diff` 发起请求获取代码变更。
- 后端在 `HandleGetProjectPreviewDiff` 中通过 `GetDiff` 完整拉取了所有的 `git diff` 字符串。
- 当差异巨大时，后端完整返回这段上百 MB 的文本，前端通过 computed 属性调用 `diff2html` 同步进行解析，导致浏览器主线程严重阻塞、直接假死。

## 2. 影响范围 (Blast Radius)
- **前端卡死**：用户体验极差，只能强制关闭浏览器标签页。
- **后端内存波动**：大量内存分配可能触发 OOM (尽管本地 Demo 无大碍，但在生产服务器上会耗尽资源)。

## 3. 根因分析与对冲策略
在早期的 SDD（如 `docs/sps/plans/20260529_code_limits_plan.md`）中，我们已确定：**一旦 Diff 大小超过 DiffMaxSizeKB (5MB)，必须执行物理截断。**
然而，在 `HandleGetProjectPreviewDiff` 的代码实现中，遗漏了对这个全局限制的调用。在查看历史部署记录时的 `HandleGetTaskDiff` 已正确实现此截断。

## 4. 解决方案 (最小化修改)
在 `godeployer/api.go` 中的 `HandleGetProjectPreviewDiff` 函数，增加与 `HandleGetTaskDiff` 一致的截断逻辑。

```go
	diffText, err := GetDiff(c.Request.Context(), projectID, fromCommit, toCommit)
	if err != nil {
		diffText = "无法获取差异对比文本，可能是基准 Commit 在本地不存在。"
	} else {
		limitBytes := h.config.Global.DiffMaxSizeKB * 1024
		if limitBytes > 0 && len(diffText) > limitBytes {
			diffText = diffText[:limitBytes] + "\n\n... [Diff 截断: 文件变更过大，超出系统物理隔离安全限制 (DiffMaxSizeKB)]"
		}
	}
```
**防御性编程 (对冲) 确认**：`len(diffText)` 获取的是字节长度，Golang 原生 string 截断操作 `diffText[:limitBytes]` 性能极高。虽然极端情况可能切断多字节字符，但由于此结果只做 diff 文本展示和阻断，因此即使末尾字符乱码也是可接受的边缘损失，浏览器不会因其崩溃。
