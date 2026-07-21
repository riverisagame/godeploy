# 任务计划：生成全套大厂规范部署资产与脚本

## 任务背景
在项目空间内批量化输出针对不同基础设施的自动化部署配置文件、脚本以及使用指南，保证强幂等性和极低心智负担。

## 原子化执行拆解 (Nano-Plan)

### 步骤 1：构建配置环境
- 物理路径：`configs/config.conf.example` 和 `docs/DEPLOY_GUIDE_NANO.md`。
- 目的：输出标准化配置样本及“纳米级”白话指南，让普通用户立刻知晓如何启动。

### 步骤 2：生成容器化组件
- 物理路径：`build/package/Dockerfile` 和 `.dockerignore`。
- 规则约束：Multi-stage 构建，非 root 用户运行，暴漏 8080 端口和 `/var/lib/godeploy`。

### 步骤 3：生成集群编排清单
- 物理路径：`deploy/helm/godeploy/` 及其子目录 (`Chart.yaml`, `values.yaml`, `templates/statefulset.yaml`, `templates/service.yaml`, `templates/pvc.yaml`)。
- 规则约束：强制 `kind: StatefulSet` 避免 SQLite 并发写异常。

### 步骤 4：生成单机裸机组件
- 物理路径：`deploy/systemd/godeploy.service`, `deploy/scripts/install.sh`, `deploy/scripts/update.sh`。
- 规则约束：应用 Systemd 安全指令（沙盒防护）；脚本中必须嵌入自动化备份与提权验证逻辑。

## 出口准则
所有文件严格落实到指定的物理目录后，输出 `[PLAN_LOCKED]`。此环节完全绕过 `TDD` 单元测试周期，因为它们均属于静态配置文件和操作脚本。
