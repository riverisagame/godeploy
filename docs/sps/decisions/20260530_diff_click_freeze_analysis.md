# ADR: 解决 Diff 界面点击文件行导致浏览器卡死问题

## 问题背景
当用户点击“变更文件列表”中的某一行文件时，系统会执行 `handleDiffRowClick`。
该操作会执行以下两步：
1. 将 `activeDiffTab` 切换为 `'diff'`。
2. 在 `nextTick` 回调中，通过 `document.querySelectorAll('.d2h-file-header')` 查询并滚动到指定文件。

然而，在切换到 `'diff'` 标签时，浏览器需要渲染大体积的代码差异（300KB 截断）。在此渲染过程中，发生了严重的卡死（甚至导致浏览器崩溃）。

## 根本原因分析
1. **`diff2html` 的匹配性能问题**：
   在计算属性 `highlightedDiff` 中，`diff2html` 配置了 `matching: 'lines'`。
   当 diff 文本达到 100KB ~ 300KB 时，`diff2html` 的行间匹配算法（Levenshtein 相似度动态规划）会消耗极其庞大的 CPU 算力。这会使主线程完全阻塞数十秒。
2. **大尺寸 DOM 渲染瓶颈**：
   300KB 的差异文本包含上千行代码，会被解析为上万个 DOM 节点。通过 `v-html` 强行一次性插入到 DOM 树中，会引发浏览器超负荷的重绘与重排。
3. **`nextTick` 内的强同步 DOM 查询与滚动**：
   切换 Tab 时，Vue 刚刚将庞大的 DOM 标记为需要更新。但在 `nextTick` 中，系统立刻调用了 `document.querySelectorAll` 和 `el.scrollIntoView`。这会强制浏览器进行“同步重排（Forced Synchronous Layout）”，使得原本已经阻塞的渲染雪上加霜，导致界面瞬间死锁。

## 方案选型与技术对比

| 优化维度 | 方案 A (高风险) | 方案 B (推荐，最小化、最稳妥) |
| :--- | :--- | :--- |
| **`diff2html` 匹配模式** | 保持 `matching: 'lines'` | 将 `matching` 降低为 `'none'`，只做基本渲染。对于大 diff 足够，计算开销从二次方级降为 O(N)。 |
| **前端保护阈值** | 前端继续允许 300KB | 进一步限制前端安全渲染上限至 100KB，避免海量 DOM 节点压垮非虚拟滾动的页面。 |
| **Tab 切换与滚动时机** | 在 `nextTick` 里立刻执行同步 DOM 检索和滚动 | 引入 `requestAnimationFrame` 或短期 `setTimeout`（如 50ms），确保 Vue 的 `v-html` 完全被浏览器排版渲染完毕后，再进行轻量级的检索和定位，避开强同步重排。 |

## 修改可能触达的现有功能
- 前端项目部署及历史对比的 Diff 渲染界面 (`web/src/views/Dashboard.vue`)。
- 不影响后端 Git 提取逻辑，不影响其他业务逻辑。
