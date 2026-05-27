# M6: Frontend WebSocket & Animation Overhaul (SCAN)

## 1. 核心需求剖析
**目标**：将现有的前端轮询（Short-Polling）重构为 WebSocket 实时推送机制，并全面升级部署流水线的动效（Framer Motion / Anime.js）。

**触达范围**：
- **后端**：`api.go` (新增 WebSocket Handler)、`engine.go` (状态机必须与 WS 推送解耦，不能阻塞核心逻辑)。
- **前端**：`web/src/views/Deploy.vue` (或类似视图)、API 层服务。

## 2. 深度质疑与自我攻击 (Adversarial Critique)
- **并发风暴攻击**：如果有 1000 个客户端同时连接 WebSocket 并监听同一个部署任务的状态变更，后端 goroutine 是否会激增导致 OOM？我们需要针对每个任务做一个 pub/sub 还是全局广播？
- **断线重连（Resilience）**：WebSocket 存在网络波动断开的可能，如果断开并在 5 秒后重连，这期间丢失的状态推送如何弥补？是否需要提供类似 `last_message_id` 的补偿拉取接口？或者仅仅在重连后 fallback 到一次全量 HTTP 拉取？
- **性能对冲**：引入动效如果处理不当，可能导致浏览器重排/重绘严重。Vue 的动画应当约束在 opacity 和 transform 属性上。
- **物理零污染**：此更改是否会对 `deploy_tasks` 和其他表的已有数据产生任何影响？答案：纯前后端交互，不涉及 DDL/DML，但必须确保前端依然能正确解析原有的任务日志 JSON。

## 3. 待决策的问题 (Open Questions)
1. **WebSocket 握手鉴权**：在建立 WebSocket 时，无法在 HTTP Header 中轻易塞入 Bearer Token（除非通过 URL Query 或前端 Protocol 头传递），这需要调整目前的 JWT 中间件。如何优雅地鉴权？
2. **后备方案（Fallback）**：是否保留当前的 HTTP 轮询作为 WebSocket 失败（比如某些 Nginx 配置不支持 WS）时的回退策略？
3. **动效库选型**：由于前端使用的是 Vue 3 (从文件推断)，Framer Motion 主要用于 React，而 Anime.js 较底层。推荐直接使用 Vue 的原生 `<TransitionGroup>` 配合 CSS 动画，或者引入 `@vueuse/motion`。请确认是否必须强依赖外部大型动画库？

## 4. 测试与验证策略
- **Mock 测试**：编写单测验证 WebSocket 广播中心的注册、注销和内存泄漏。
- **并发压测**：使用脚本同时发起 500 个 WS 连接观察系统 CPU。
- **无损回退**：切断网络模拟 WS 掉线，验证系统能否正常恢复。
