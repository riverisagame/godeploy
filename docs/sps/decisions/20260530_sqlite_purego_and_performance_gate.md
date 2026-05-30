# ADR: SQLite Pure Go Migration and System-Wide Performance Gate

## 背景
为提升系统的分发健壮性，我们需要摆脱对 CGO 的编译环境依赖；同时，通过限制外部 CLI 的并发和资源占用，防止大文件和并发部署卡死。

## 方案设计

### 1. SQLite 纯 Go 升级 (CGO_ENABLED=0)
- **依赖替换**：移去原先基于 C 编译的 `github.com/mattn/go-sqlite3`，替换为由 C 翻译生成的纯 Go 驱动 `modernc.org/sqlite`。
- **配置变更**：
  - 在 `godeployer/db.go` 中：
    - 导入驱动由 `_ "github.com/mattn/go-sqlite3"` 改为 `_ "modernc.org/sqlite"`；
    - 初始化连接由 `sql.Open("sqlite3", dsn)` 改为 `sql.Open("sqlite", dsn)`。
- **编译对冲**：此后，所有的构建和测试步骤可以使用 `CGO_ENABLED=0 go test ./...` 和 `CGO_ENABLED=0 go build` 快速、安全地通过，实现无动态库链接的静态二进制文件发布。

### 2. 全局进程限流（Semaphore）与硬超时控制
- 为限制外部 `git` 和 `rsync` 进程引起的 CPU 满载：
  - 引入一个全局并发限制限流器（例如利用 Buffered Channel 实现最大 3 并发部署和最大 5 并发 Diff）；
  - 封装带超时和缓冲区截断保护的外部进程执行管道，避免 CLI 进程产生僵尸进程或内存泄漏。
