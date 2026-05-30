# WSL Debian 环境发布异常与路径越界修复方案

## 1. 目标
解决 WSL 环境中执行 `wsl debian demo` 导致的 `fatal: unable to read tree` Git 克隆故障，以及 Windows 与 WSL 双重挂载下硬编码绝对路径导致的寻找目录失败。

## 2. 变更说明

### 2.1 修改 `godeployer/engine.go`
#### [MODIFY] 引擎部署克隆参数
**修改行**：约在 334 行
**原代码**：
```go
	if _, statErr := os.Stat(cacheDir); statErr == nil {
		writeLog("Step 1: Cloning repository locally from cache %s into %s...", cacheDir, buildPath)
		cloneCmd = exec.Command("git", "clone", cacheDir, buildPath)
	} else {
```
**新代码**：
```go
	if _, statErr := os.Stat(cacheDir); statErr == nil {
		writeLog("Step 1: Cloning repository locally from cache %s into %s...", cacheDir, buildPath)
		cloneCmd = exec.Command("git", "clone", "--no-hardlinks", cacheDir, buildPath)
	} else {
```
**动机**：强制 Git 复制对象库而绝不使用硬链接（`--no-hardlinks`），防止 Windows/WSL 在 `/mnt/d/` 等挂载目录下的 inode 连接失败。

### 2.2 修改 `godeployer/git.go`
#### [MODIFY] 缓存生成参数
**修改行**：约在 51 行
**原代码**：
```go
		cmd := exec.CommandContext(ctx, "git", "clone", "--bare", repoURL, cacheDir)
```
**新代码**：
```go
		cmd := exec.CommandContext(ctx, "git", "clone", "--no-hardlinks", "--bare", repoURL, cacheDir)
```
**动机**：在第一次将外部代码缓存为本地 bare repo 时也排除硬链接引用，避免裸库污染。

### 2.3 修改 `scripts/demo.sh`
#### [MODIFY] 屏蔽宿主机绝对路径渗透
**修改行**：约在 123 行与 161 行
**原代码**：
```bash
    sed -i "s|repo:.*|repo: \"$GITEE_WORKSPACE/$name\"|g" "$REPO_ROOT/demo_projects.d/$file_name"
```
**新代码**：
```bash
    # 使用相对路径，确保在 WSL 与 Win 混用时不导致挂载前缀解析报错
    sed -i "s|repo:.*|repo: \"demo_workspace/gitee_demo/$name\"|g" "$REPO_ROOT/demo_projects.d/$file_name"
```
**动机**：相对路径让 `godeployer` 执行时从其运行时工作目录（`$REPO_ROOT`）动态展开路径，免疫跨平台绝对路径隔离。

## 3. 测试覆盖
无需修改现有业务测试。因为这属于进程底层的启动挂载与文件操作兼容性问题。我们将在后续流程中直接运行 WSL 全量系统集成测试。
