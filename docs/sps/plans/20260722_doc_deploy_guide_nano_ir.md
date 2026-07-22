# 部署指南补充计划 (Nano-level Detailed Deployment Guide)

## 需求对齐 (Alignment)
当前 `docs/DEPLOY_GUIDE_NANO.md` 内容仅有基础命令，无法满足“纳米级”详细的要求。
需对 Docker、Systemd (install.sh / update.sh)、K8s / Helm 提供细化到参数级、挂载点级、回滚级、权限隔离级的极度详尽文档。

## 执行计划 (Execution Plan)
1. **[MODIFY]** `docs/DEPLOY_GUIDE_NANO.md`
   - **Docker 篇**: 补充环境变量注入细节、挂载卷数据结构解析（/var/lib/godeploy）、网络模式等。
   - **Systemd 篇**: 深度拆解 `install.sh` 的权限分配（创建 godeploy 用户）、`update.sh` 的 SQLite 自动备份防呆机制。
   - **K8s & Helm 篇**: 补充 `values.yaml` 参数对照表、PV/PVC 挂载注意事项、如何无损升级 Release 等。

请确认是否按此计划输出极尽详细（纳米级）的文档。

