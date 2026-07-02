# GoDeploy 开发工具参考

## 文件操作

| 工具 | 命令/触发方式 | 用途 |
|------|-------------|------|
| `read` | `read <路径>` | 读取文件、目录、URL、SQLite 数据库。支持行范围选择器 `:N-M` |
| `write` | `write <路径> <内容>` | 创建或覆写文件 |
| `edit` | `edit <路径>` + 行号 | 精确行级编辑：替换(`replace N..M`)、删除(`delete N..M`)、插入(`insert before/after N`) |
| `find` | `find <glob>` | 文件名/目录名匹配搜索，遵守 `.gitignore` |
| `search` | `search <正则> <路径>` | 正则表达式内容搜索 |

## 代码智能（依赖 gopls）

| 操作 | LSP 命令 | 说明 |
|------|---------|------|
| 跳转定义 | `lsp definition <文件> <行>` | 查看符号定义位置 |
| 查找引用 | `lsp references <文件> <行>` | 列出所有调用/引用处 |
| 类型信息 | `lsp hover <文件> <行>` | 悬停查看类型签名和文档 |
| 安全重命名 | `lsp rename <文件> <行> <新名称>` | 跨文件重命名（理解调用图） |
| 快速修复 | `lsp code_actions <文件> <行>` | 导入、重构、修复建议 |
| 诊断 | `lsp diagnostics <文件或*>` | 获取编译错误/警告 |

## AST 语法树

| 操作 | 命令 | 说明 |
|------|------|------|
| 结构搜索 | `ast_grep <模式> <路径>` | 按语法结构匹配，不是文本匹配 |
| 批量改写 | `ast_edit <模式> → <替换> <路径>` | 基于语法树的批量代码转换 |

模式变量：`$A` 匹配单节点、`$$$A` 匹配零或多个节点。同名变量约束一致。

## 浏览器

| 操作 | 说明 |
|------|------|
| `browser open <名称> <URL>` | 打开 Chromium 标签页 |
| `browser run <名称> <JS代码>` | 在标签页中执行 Puppeteer API 脚本 |

## 调试器

```bash
# 启动调试
debug launch --program ./my_app

# 设断点
debug set_breakpoint --file main.go --line 42

# 执行控制
debug continue / step_over / step_in / step_out

# 检查变量
debug evaluate --expression "task.Status"
```

## GitNexus 代码关系图谱

项目已索引为 **godeploy**（2879 symbols, 5302 relationships, 90 flows）。

| 操作 | 用途 |
|------|------|
| 爆炸半径 | `gitnexus_impact(target="符号名", direction="upstream")` |
| 全貌关系 | `gitnexus_context(name="符号名")` |
| 概念搜索 | `gitnexus_query(query="概念")` |
| 提交检测 | `gitnexus_detect_changes()` |
| 图谱重命名 | `gitnexus_rename(symbol_name="旧名", new_name="新名")` |
| API 映射 | `gitnexus_route_map()` |
| 响应形状 | `gitnexus_shape_check()` |

> 修改任何导出符号前，**必须先执行** `gitnexus_impact` 评估影响范围。

## 外部资源

| 操作 | 用途 |
|------|------|
| 库文档 | `mcp__context_query_docs` — 查阅最新框架 API 文档 |
| 代码实例 | `mcp__gh_grep_searchgithub` — 搜索 GitHub 公开代码 |
| Web 搜索 | `web_search` |

## 开发工作流

### 需求到代码的标准流程

1. **头脑风暴** → 需求深挖、方案对比 → 输出设计规格 `.md`
2. **制定计划** → 纳米级任务拆解（每个任务 10-20 行改动）
3. **TDD 红绿重构** → 先写测试（预期失败）→ 最小实现 → 重构
4. **代码审查** → 派发 reviewer 子代理检查
5. **验证** → 运行全量测试 + 编译 + go vet

### 常用命令

```bash
# 全量测试（强制 -race，CI 要求）
go test -v -race ./...

# 单包测试
go test -v ./godeployer/application/

# 编译
go build .

# 静态分析
go vet ./godeployer/...

# 前端
cd web && npm run test
cd web && npm run test:e2e
```

## 约束规则

- **修改前先 impact**：任何导出符号修改前运行爆炸半径分析
- **LSP 优先于盲搜**：重命名、查引用、跳转定义必须用 LSP
- **AST 优先于文本替换**：避免 sed/regex 导致的误匹配
- **验证后才声明完成**：测试通过 + 编译通过 + vet 零告警
- **godeployer 已拆为 DDD 四层**：domain → application → infrastructure → interfaces
- **状态字符串仅允许在 `domain/value_objects.go`**：其他包统一用 `domain.StatusXxx` 常量
