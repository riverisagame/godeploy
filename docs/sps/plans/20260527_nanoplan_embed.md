# TASK-007: GoDeployer 资源嵌入与集成纳米计划

本计划旨在实现 GoDeployer 最终单二进制编译打包。在 Go 层面通过 `go:embed` 内嵌前端静态页面，实现 SPA 静态资源路由的正确映射及 Fallback，并编写命令行入口主程序。

---

## 1. 拟修改/创建文件清单

- **[NEW]** `godeployer/main.go` - 主程序入口，包含 CLI 参数解析、服务器启动及静态资源挂载

---

## 2. 纳米级任务细分 (代码量限制在 10-20 行/步)

### 2.1 任务 1: 前端静态资源 embed 声明
*文件路径*: `godeployer/main.go`

- **[x] [Sub-Task 1.1]** 声明 `//go:embed all:dist` 嵌入前端编译产物。
- **[x] [Sub-Task 1.2]** 编写基于 `http.FS` 的 Gin 静态资源映射处理器，代理 `/assets`、`favicon.svg` 等物理文件。

### 2.2 任务 2: SPA 路由回退处理 (Fallback)
*文件路径*: `godeployer/main.go`

- **[x] [Sub-Task 2.1]** 编写 SPA 中间件或 NotFound 路由处理器。如果客户端访问的非 API 路径在 embed 目录中不存在，则强制返回内嵌的 `index.html`，把路由权交还 Vue 路由器。

### 2.3 任务 3: CLI 命令行参数解析与环境校验
*文件路径*: `godeployer/main.go`

- **[x] [Sub-Task 3.1]** 引入 `flag` 库，处理 `--config` 参数（指定主 yaml 路径），并对参数文件存在性进行校验。
- **[x] [Sub-Task 3.2]** 编写启动逻辑：加载配置、初始化 SQLite、启动事件通知消费者 Goroutine。

### 2.4 任务 4: 服务监听与端口绑定
*文件路径*: `godeployer/main.go`

- **[x] [Sub-Task 4.1]** Gin 路由与 Web API 模块整合，并启动 `r.Run` 监听配置文件中的 `global.server_port`，完成部署系统的冷启动。

---

## 3. 验证与验证确认规划

- **集成测试与编译**：在根目录执行 `go build -o godeployer.exe ./godeployer`，保证编译成功无依赖错误。
- **功能对齐验证**：启动 `./godeployer.exe`：
  - 确认是否能在不依赖任何外部文件的情况下，单独访问 `http://localhost:8080` 进入 Vue 界面；
  - 静态资源如 JS/CSS 是否加载正常；
  - API 与前端能否顺畅通信。
