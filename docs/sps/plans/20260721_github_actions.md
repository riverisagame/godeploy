# 任务计划：新增 GitHub Actions 工作流 (Go 1.26.5)

## 任务背景
在 `.github/workflows/` 目录下物理创建两个受控文件，建立工业级 CI/CD 管道，全局统一使用 Go `1.26.5` 版本。

## 原子化执行拆解 (Nano-Plan)

### 步骤 1：同步基础配置
- **文件路径**：`go.mod`
- **改动范围**：将版本从 `go 1.26.4` 提升为 `go 1.26.5`。
- **改动行数**：~1行代码。
- **验证手段**：执行 `go mod tidy`。

### 步骤 2：创建 CI (持续集成) 工作流
- **文件路径**：`.github/workflows/ci.yml` (新建)
- **动作细节**：
  1. 定义 `on: push (main), pull_request (main)`。
  2. 声明 `runs-on: ubuntu-latest`。
  3. 配置 `actions/checkout@v4`。
  4. 配置 `actions/setup-go@v5` (指定 `go-version: 1.26.5`)。
  5. 注入 `golangci/golangci-lint-action@v6` 节点进行代码检查。
  6. 注入 `go test -race -v ./...` 节点验证代码。
- **改动行数**：~20行配置代码。

### 步骤 3：创建 Release (持续发布) 工作流
- **文件路径**：`.github/workflows/release.yml` (新建)
- **动作细节**：
  1. 定义 `on: push: tags: - 'v*'`。
  2. 声明权限 `permissions: contents: write` 以便发版。
  3. 配置 Matrix 并行构建组合：
     - linux: amd64, arm64
     - darwin: amd64, arm64
     - windows: amd64 (带 .exe 后缀)
  4. 执行 `go build -o ...` 编译。
  5. 依赖 `softprops/action-gh-release@v2` 将二进制产物挂载至新建的 Release。
- **改动行数**：~35行配置代码。

## 出口准则
计划内所有的物理文件必须按照指定路径创建并提交，由于工作流只在 GitHub 服务器端运行生效，本地验证将重点检查 YAML 语法和 `go.mod` 同步状态。
