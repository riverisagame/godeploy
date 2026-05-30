# 架构约束基准 (Code Size & Diff Limits)

## 背景
用户询问如何合理限制单次上线的总 Diff 以及单个文件的 Diff 大小。之前的方案（总 Diff ≤ 200,000 行，单文件 ≤ 20,000 行）被认定为架构反模式。

## 业界标准与强制约束基准 (Hard Limits)

经过架构评估，针对 GoDeploy 这样要求绝对可靠性和高并发安全的部署系统，建议强制执行以下限制参数：

### 1. 单个文件行数限制 (File Size Limit)
- **Soft Limit (警告)**: `500 行`。超过此行数说明文件开始承担过多职责。
- **Hard Limit (阻断)**: `1,000 行`。超过 1000 行必须强制通过接口 (interface) 和多文件拆分进行重构。
- **例外**: 纯自动生成代码（如 Protobuf、Mock 代码、前端打包产物 dist），但需在 Lint 中加白名单。

### 2. 单个函数/方法行数限制 (Function Size Limit)
- **Soft Limit (警告)**: `50 行`。
- **Hard Limit (阻断)**: `80 行`。圈复杂度 (Cyclomatic Complexity) 必须 ≤ 10。

### 3. 单次提交/Pull Request 限制 (PR Diff Limit)
- **推荐值 (Golden Rule)**: `200 - 400 行`。这是人工 Code Review 能够保持注意力和找出逻辑漏洞的最佳范围。
- **Hard Limit (阻断)**: `800 行` (不含自动生成代码、依赖锁文件如 `package-lock.json` 或 `go.sum`)。如果需求巨大，必须拆分为多个前置 PR（如先上数据结构，再上核心逻辑，最后上 API 路由）。

### 4. 单次上线/发布限制 (Release Size Limit)
- **推荐范围**: `1,000 - 3,000 行` (跨越多个已合并的小 PR)。
- **Hard Limit**: `5,000 行`。超出此范围属于“大爆炸发布 (Big Bang Release)”，发布失败率和回滚难度将呈指数级上升。现代微服务架构建议每天多次发布 (CI/CD)，而非攒大招。

## 历史债务豁免协议 (Legacy Grandfather Policy)

针对已经客观存在的历史庞大文件（例如原有的历史遗留代码），可以采取**有条件绕过**策略，但必须严格遵循以下硬核约束：

1. **白名单隔离 (Grandfathering)**
   在 CI/CD 和 Lint 配置中，将这些具体的历史文件显式加入排除名单（如 `.eslintignore` 或 `golangci-lint` 的 `skip-files`）。绝对**禁止**关闭全局的限制规则。
   
2. **童子军规则 (The Boy Scout Rule)**
   如果你必须在这个历史文件中修改或新增逻辑：
   - **禁止继续堆砌**：严禁在这个遗留文件里写长篇大论的新逻辑。新逻辑必须写在全新的、符合规范的独立文件中，在遗留文件里仅仅进行函数调用。
   - **只减不增原则**：每次修改历史文件时，必须顺手将其中的一块腐化逻辑抽离出去。遗留文件的体积只能缩小，不能膨胀。

3. **变更增量熔断 (Diff Circuit Breaker)**
   即使历史文件拥有总行数的“免死金牌”，但针对该文件的**单次修改变动量 (PR Diff)** 依然必须绝对遵守 `≤ 800 行` 的红线。如果你的一次重构引发了超过 800 行的变更，说明步子迈得太大，极易引发回归 Bug，必须强行拆分为多个小 PR 进行。

## 防治 Diff 崩溃的技术方案 (Diff Crash Prevention)

当不得不处理超大文件或庞大历史代码时，极易导致 Git 客户端、GitHub/GitLab 的 Code Review 页面崩溃卡死。为彻底解决“Diff 崩”的问题，强制采取以下组合方案：

### 1. 物理屏蔽：配置 `.gitattributes`
对于前端编译产物 (dist)、锁定文件 (`package-lock.json`)、生成的 Mock 数据或实在无法拆分的几万行遗留文件，**必须**在根目录 `.gitattributes` 中将其标记为生成的或非 Diff 文件：
```text
# 告诉 Git 平台这是生成代码，默认折叠，防止拉爆浏览器内存
web/dist/* linguist-generated=true
*-lock.json linguist-generated=true

# 针对绝对无法拆分的史前巨兽文件，直接在终端和 PR 页面中跳过 diff 计算
legacy/god_object.go -diff
```

### 2. 格式化隔离与 Blame 屏蔽
如果是因为引入代码格式化工具 (Prettier / gofmt) 导致全文件大面积 Diff：
- **绝对禁止**将格式化变更与业务逻辑变更混在一个 Commit 或 PR 中。
- 格式化导致的巨大 Commit 必须单独提交，并在项目根目录建立 `.git-blame-ignore-revs` 文件，将该 Commit Hash 存入。这不仅能防止 Diff 崩溃，还能保证 `git blame` 不会丢失历史上下文。

### 3. 架构解药：绞杀者无花果模式 (Strangler Fig Pattern)
从根本上防止修改大文件导致 Diff 崩溃的唯一正确做法，是**停止修改大文件**。
当收到针对巨型文件的新需求时，在它旁边新建一个优雅的小文件。将新逻辑写在新文件里，并建立一个 Adapter 供老系统调用。随着时间推移，不断将老文件中的功能迁移到新文件，最终“绞杀”并删除原有的巨石文件。

## 后续动作 (Next Steps)
若用户确认此基准，我们可以：
1. **配置屏蔽**: 立即编写 `.gitattributes` 和 `.git-blame-ignore-revs`，将现有的高风险大文件进行物理隔离。
2. **后端 (Go)**: 引入 `golangci-lint`，配置 `lll` (单行长度), `funlen` (函数长度), `maintidx` (可维护性指数) 等拦截器。
3. **前端 (Vue/TS)**: 在 `ESLint` 中配置 `max-lines` 和 `max-lines-per-function`。
4. **CI/CD Pipeline**: 在 GitHub Actions 或本地 Git Hooks 中添加脚本，拒绝有效业务逻辑的单次 PR Diff 超过 800 行。
