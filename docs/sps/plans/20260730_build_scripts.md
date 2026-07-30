# 纳米级执行计划：锁定全栈构建流水线 (2026-07-30)

## 需求背景
依据用户决定，采用“流水线责任制”（路线 1），将前端打包与 Docker 打包逻辑隔离。为了避免人为疏忽导致的打包遗漏，需要建立固化的打包脚本。

## 变更文件
1. **[NEW] `Makefile`**
   - 编写 `build-ui` 目标：进入 `web` 目录执行 `npm install` 与 `npm run build`。
   - 编写 `build-image` 目标：执行 `docker build -t godeploy:latest .`。
   - 编写 `deploy` 目标（组合）：按顺序依赖 `build-ui` -> `build-image`。

2. **[NEW] `build.ps1`** (考虑到您的 Windows 宿主机环境)
   - 检查前端目录 `web`。
   - 自动调用 `npm install` 和 `npm run build`。
   - 判断执行状态并抛出硬核报错（如果失败就终止）。
   - 执行 `docker build -t godeploy:latest .`。
   - 成功后输出绿色成功信息。

## 最小化原则审计
完全不在 Go 代码和 `Dockerfile` 中做任何入侵，100% 维持目前的并发与轻量化状态。仅在根目录提供辅助运维的壳文件。
