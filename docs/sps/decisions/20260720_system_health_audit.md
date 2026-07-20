# ADR-012: 系统全面体检与优化方向决策

**日期**: 2026-07-20
**状态**: 待确认
**触发**: P0三大需求闭环后，对系统进行全面深度扫描

## 背景
三个P0级需求（回滚/历史、环境变量、部署前Diff）已全部落地并通过验收。通过3个并行分析子代理对系统进行了全面CT扫描，发现47个待修复项（5 P0 + 14 P1 + 18 P2 + 10 P3）。

## 关键发现
1. **致命BUG**: TriggerDeploy传了ProjectID而非env.ID
2. **竞态条件**: logLines切片被多goroutine无锁并发append
3. **零认证**: 所有API完全裸奔
4. **命令注入**: PreDeploy/PostDeploy用户输入直接拼接shell
5. **SSE日志丢失**: 部署开始后SSE连接前的早期日志必然丢失

## 详细报告
见 implementation_plan.md artifact