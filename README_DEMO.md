# GoDeployer Demo 环境搭建指南

本文档说明如何快速搭建含**真实 Gitee PHP 项目**历史数据的本地演示环境，用于功能验证与开发调试。

## 演示项目

| 项目 | 来源仓库 | 说明 |
|------|---------|------|
| **ThinkPHP 后端框架** | [gitee.com/top-think/think](https://gitee.com/top-think/think) | 5条历史部署，全部成功 |
| **Webman 微服务接口** | [gitee.com/walkor/webman](https://gitee.com/walkor/webman) | 4条历史部署，含1次SSH失败 |
| **CRMEB 商城系统** | [gitee.com/ZhongBangKeJi/CRMEB](https://gitee.com/ZhongBangKeJi/CRMEB) | 5条历史部署，含1次权限失败 |

## 演示账号

| 账号 | 密码 | 角色 | 可见项目 |
|------|------|------|---------|
| `admin` | `admin123` | 管理员 | 全部 3 个项目 |
| `deployer` | `deploy123` | 部署员 | ThinkPHP + Webman |
| `viewer` | `view123` | 只读 | 仅 ThinkPHP |

## 快速启动

### 前置要求

- WSL Debian 或原生 Linux 环境
- Go 1.21+
- git, sqlite3, jq, curl
- Node.js 18+（前端页面）

### 一键初始化（首次运行）

```bash
# 完整初始化：克隆 Gitee 仓库 + 启动后端 + 写入演示数据
bash scripts/demo.sh
```

### 启动前端

```bash
cd web && npm install && npm run dev
# 打开 http://localhost:5173
```

### 常用命令

```bash
bash scripts/demo.sh              # 完整初始化（首次）
bash scripts/demo.sh start        # 仅启动后端
bash scripts/demo.sh stop         # 停止后端
bash scripts/demo.sh seed         # 重置演示数据（不重新克隆）
bash scripts/demo.sh status       # 查看服务状态
bash scripts/demo.sh verify       # 验证数据与接口
bash scripts/demo.sh clone        # 仅克隆/更新 Gitee 仓库
```

## 可验证的功能点

| 功能 | 验证方法 |
|------|---------|
| **项目权限隔离** | 用 `deployer` 登录，左侧无 CRMEB；用 `viewer` 登录，只见 ThinkPHP |
| **历史部署列表** | 点击项目 → 切换环境 Tab → 查看历史任务表格 |
| **部署日志** | 点击任务 → 查看日志，含真实 git clone/rsync 输出 |
| **失败任务详情** | task_25（Webman SSH拒绝）/ task_30（CRMEB 权限错误）|
| **代码 Diff** | 点击「查看Diff」，黑色主题高对比度对比两次提交变更 |
| **用户管理 CRUD** | admin 顶栏 → 用户管理 → 增删改查 |
| **账号配置** | 点击「账号配置」设置 Git 作者白名单 |

## 目录结构说明

```
scripts/
├── demo.sh              # ⭐ 主入口：一键 Demo 环境管理
├── gen_demo_logs.sh     # 生成历史部署日志文件
├── seed_tasks_only.sh   # 仅写入 DB 任务数据
├── verify_demo.sh       # 接口 & 权限验证
├── verify_diff.sh       # Diff 接口验证
├── verify_api.sh        # 全量 API 验收
├── clone_gitee_php.sh   # 克隆 Gitee PHP 仓库
└── install.sh           # 生产部署安装脚本

demo_config.yaml         # Demo 专用配置（端口 8080）
demo_projects.d/         # Demo 项目 YAML 配置
  ├── thinkphp.yaml
  ├── webman.yaml
  └── crmeb.yaml

# 以下目录由 demo.sh 自动生成，不提交 git
demo_deployer.db         # 演示数据库
demo_logs/               # 任务日志文件 task_*.log
demo_workspace/          # Git 仓库工作区
  └── gitee_demo/
      ├── think/         # ThinkPHP 本地仓库
      ├── webman/        # Webman 本地仓库
      └── CRMEB/         # CRMEB 本地仓库（~187MB）
```

## Diff 功能说明

Demo 数据中的任务使用了真实 commit hash，`GET /api/tasks/:id/diff` 接口会：

1. 优先从部署构建目录执行 `git diff`
2. 若构建目录已被清理（Demo 场景），自动降级到 `demo_workspace/gitee_demo/` 中查找对应仓库
3. 使用 diff2html 渲染为黑色高对比度主题

## 开发调试

后端日志实时查看：

```bash
tail -f /tmp/godeployer_demo.log
```

重置数据库（清空重来）：

```bash
rm -f demo_deployer.db
bash scripts/demo.sh seed
```
