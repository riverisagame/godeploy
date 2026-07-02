# 更新日志

所有值得关注的项目变更都会记录在此文件中。

格式基于 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)，
版本号遵循 [语义化版本](https://semver.org/lang/zh-CN/)。

---

## [2.0.0-ddd] - 2026-05-31

### 重大变更
- **全面拥抱 DDD 架构**：拆分为 `domain` / `application` / `infrastructure` / `interfaces` 四层，完成战术级重构
- **数据库层升级**：GORM 全面支持 MySQL 和 PostgreSQL，不再局限于 SQLite
- 值对象、实体、领域服务、依赖反转全部落地，现有功能无损，所有测试绿灯

### 修复
- CI 后端测试：补充 dummy dist 目录和清理 test-app 残留
- 前端测试：修复 `triggerDeploy` 缺少 `description` 字段

### 杂项
- gitignore 补充测试工作目录规则，删除重复 `.omc/` 条目
- 清理临时 Python 脚本和过期日志
- 清理无用的临时文件

---

## [1.1.0] - 2026-05-30

### 新增
- **纯 Go SQLite 驱动迁移**：无需 CGO，跨平台编译零依赖
- **系统级性能门控**：带 Tooltip 提示的性能指标校验
- **可视化文件过滤**：部署时按需选择文件
- **部署备注**：每次部署可附带说明
- **双标签对比对话框（DiffDialog）**：支持并排对比两次部署差异
- **WebSocket 连接**：心跳 + 自动重连
- **全局 API 拦截器**：axios 实例 + 统一错误处理
- **PHP 项目 Demo 环境脚本**及黑色 Diff 主题

### 变更
- 从 Dashboard 中抽取独立组件：`DeployForm`、`DeployHistoryTable`、`LogTerminal`、`ProjectSidebar`、`UserSettingsDialog`
- Dashboard 与 UserManagement 统一暗色主题
- 消除 Mock 数据，单状态机替代多标志位，移除 watch 自动触发

### 修复
- E2E 测试选择器适配新组件结构，5 个全部通过
- DiffDialog 集成完善，补充 `handleDiffClose` 处理

---

## [1.0.8] 及更早

早期版本覆盖基础功能搭建、认证系统、SSH 部署引擎、WebSocket 实时推送等核心能力。详见 git 提交历史。

---

[2.0.0-ddd]: https://github.com/your-org/godeploy/compare/v1.1.0...v2.0.0-ddd
[1.1.0]: https://github.com/your-org/godeploy/compare/v1.0.8...v1.1.0
