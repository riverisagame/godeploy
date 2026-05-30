# 架构门禁实施计划 (Code Size Limits IR Plan)

## 目标
根据架构审计 (`ARCH-011`) 落实“Diff 防御”与“代码体积硬约束”。

## 子任务拆解 (纳米级执行计划)

### 任务 1: 物理隔离 Diff 崩溃源 (原子化)
- **目标文件**: `d:\claudeprj\deploy\.gitattributes` (不存在则创建)
- **执行动作**: 写入以下配置以告诉 Git 服务端停止计算大规模产物的差异。
  ```text
  web/dist/* linguist-generated=true -diff
  *lock.json linguist-generated=true -diff
  *.db -diff
  *.exe -diff
  godeployer_linux -diff
  ```

### 任务 2: 初始化 Blame 屏蔽体系 (原子化)
- **目标文件**: `d:\claudeprj\deploy\.git-blame-ignore-revs`
- **执行动作**: 创建该文件并添加注释，为未来全量格式化操作预留屏蔽机制。
  ```text
  # 记录全量重构或代码格式化产生的巨大无意义 Commit Hash
  # 执行：git config blame.ignoreRevsFile .git-blame-ignore-revs
  ```

### 任务 3: 后端硬性检查规则 (原子化)
- **目标文件**: `d:\claudeprj\deploy\.golangci.yaml`
- **执行动作**: 新建基础配置，开启 `funlen` 和 `lll` (长文件限制)。
  ```yaml
  linters:
    enable:
      - funlen
      - lll
  linters-settings:
    funlen:
      lines: 80
      statements: 40
    lll:
      line-length: 150
  ```

### 任务 4: 前端文件大小约束
- **目标文件**: `d:\claudeprj\deploy\web\.eslintrc.cjs` (或类似 lint 配置)
- **执行动作**: 由于前台可能未配置严格的 ESLint，暂在计划中声明后续逐步在前端加入 `max-lines: ["error", 1000]` 的约束。本次仅执行 `[任务 1-3]` 保证基础防线。
