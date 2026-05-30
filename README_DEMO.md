# GoDeployer Demo 极速部署与启动指南

本文档介绍如何在 **WSL Debian/Ubuntu** 或原生 Linux 环境下一键极速搭建 GoDeployer 本地演示环境。

> 💡 **核心改进**：默认使用本地秒级生成的 Mock Git 仓库，**零网络依赖，秒级启动**。同时支持自动编译并内嵌前端资源，实现**单端口（8080）极速完整预览**，避免了克隆百兆仓库和单独启动前端服务的繁琐步骤。

---

## 🚀 极速一键部署 (Recommended)

只需一行命令，即可自动完成：依赖检查、本地 Mock 仓库生成、前端自动构建打包、后端编译以及服务启动。

```bash
# 默认离线极速一键初始化并启动 (极其推荐，适合别的 AI 或开发者快速预览)
bash scripts/demo.sh
```

**启动成功后，直接打开浏览器访问：[http://localhost:8080](http://localhost:8080) 即可开始体验！**

---

## 🔑 演示系统登录账号

| 账号 | 密码 | 角色 | 权限说明 |
| :--- | :--- | :--- | :--- |
| **`admin`** | `admin123` | 管理员 (Admin) | 拥有全部 3 个项目的可见性与操作权限，支持用户管理 CRUD |
| **`deployer`** | `deploy123` | 运维人员 (Maintainer) | 仅对 ThinkPHP 和 Webman 项目可见，且有部署权限 |
| **`viewer`** | `view123` | 观察员 (Viewer) | 仅对 ThinkPHP 可见，且只有只读权限 |

---

## ⚙️ 常用指令

```bash
bash scripts/demo.sh          # 默认：极速本地一键部署与启动 (离线)
bash scripts/demo.sh --real   # 选项：从 Gitee 克隆真实大型仓库并启动 (需要网络)
bash scripts/demo.sh start    # 仅启动/恢复后端服务
bash scripts/demo.sh stop     # 停止后端服务
bash scripts/demo.sh status   # 检查当前服务运行状态与演示账户
bash scripts/demo.sh seed     # 重置演示数据库中的所有任务与数据
bash scripts/demo.sh verify   # 自动对演示链路与 API 进行完好性校验
```

---

## 🛠️ 本地 Mock Git 仓库优势
优化后的 Demo 脚本在 `demo_workspace/gitee_demo/` 下自动创建轻量级、结构合法的本地 Git 仓库，其中：
1. **多分支与 Tag 支持**：自动包含 `master`、`develop` 分支以及 `v1.0.0`、`v1.0.1`、`v1.1.0-beta` 等 tags。
2. **Commit 与 Diff 完备**：数据库中填充的历史部署数据已与生成的本地 Git Commit 历史进行 100% 动态对齐。您可以在页面上完美验证**基于分支/Tag/Commit 的部署**、**多版本代码 Diff 对比** 等核心功能。
3. **真实构建与部署**：当您在页面触发“部署”时，部署引擎会真实拉取本地 Mock 仓库、执行打包编译逻辑，并通过 SSH/rsync 真正传输到目标路径，实现完整的、无损的端到端真实部署链条验证。

---

## 📂 调试与日志

- **查看后端服务日志**：
  ```bash
  tail -f /tmp/godeployer_demo.log
  ```
- **重置演示环境**：
  ```bash
  bash scripts/demo.sh stop
  rm -f demo_deployer.db
  rm -rf demo_workspace
  bash scripts/demo.sh
  ```
