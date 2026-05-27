# 纳米级执行计划 - 20260511_doc_gen_plan

## 任务概况
- **任务 ID**: DOC-GEN-001
- **目标**: 为 Deployer 项目生成完整的产品、手册和开发文档。
- **参考**: Deployer 8.x 源码与官方文档。

## 执行链条

### 1. 撰写产品说明书 [PRODUCT_SPEC.md]
- [ ] **[DOC-01-01]**: 创建 `doc/PRODUCT_SPEC.md`。
- [ ] **[DOC-01-02]**: 撰写“核心价值”章节（PHP 原生、原子部署、回滚机制）。
- [ ] **[DOC-01-03]**: 撰写“功能特性矩阵”（多阶段、并行执行、跨主机支持）。
- [ ] **[DOC-01-04]**: 撰写“竞品对比”（对比 Ansible, Capistrano）。

### 2. 撰写使用手册 [MANUAL.md]
- [ ] **[DOC-02-01]**: 创建 `doc/MANUAL.md`。
- [ ] **[DOC-02-02]**: 撰写“安装与配置”（基于 Phar 或 Composer）。
- [ ] **[DOC-02-03]**: 撰写“核心概念”（Inventory, Tasks, Hooks）。
- [ ] **[DOC-02-04]**: 撰写“实战：Laravel/Symfony 部署示例”。
- [ ] **[DOC-02-05]**: 撰写“常用命令详解”（dep deploy, dep ssh, dep logs）。

### 3. 撰写开发文档 [DEVELOPMENT.md]
- [ ] **[DOC-03-01]**: 创建 `doc/DEVELOPMENT.md`。
- [ ] **[DOC-03-02]**: 分析 `src/Deployer.php` 核心入口逻辑并绘图/说明。
- [ ] **[DOC-03-03]**: 解析任务调度机制（`Task/Collection.php`, `Host/Storage.php`）。
- [ ] **[DOC-03-04]**: 说明 `recipe/common.php` 中的核心逻辑流。
- [ ] **[DOC-03-05]**: 撰写“贡献指南与测试流程”（基于 PHPUnit）。

## 物理审计点 (Audit Points)
- **源码一致性**: 检查 `doc/` 中引用的所有函数/类是否在 `deployer/src/` 中存在。
- **路径准确性**: 所有的文件引用必须是相对于项目根目录的正确路径。
- **版本对齐**: 确保所有文档反映的是 PHP 8.3 + Symfony 7/8 的技术规范。


### 4. 架构级增强 [ARCHITECTURE ENHANCEMENT]
- [ ] **[DOC-04-01]**: 绘制并说明“任务执行拓扑生命周期”。
- [ ] **[DOC-04-02]**: 编写“分布式环境一致性设计”章节。
- [ ] **[DOC-04-03]**: 深度解析“连接器 (Connector) 抽象层”。
- [ ] **[DOC-04-04]**: 增加“架构约束与反模式 (Anti-Patterns)”说明。
