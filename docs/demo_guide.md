# GoDeploy Demo 环境部署指南

## 一、概述

`scripts/demo.sh` 是一键本地部署、快速预览、开发验证的自动化脚本。
通过 API 接口模拟多用户真实部署流程，100% 免伪造数据库数据。

核心能力：
- 秒级创建 Mock Git 仓库 (含分支、Tag、多次 Commit 演进)
- 可选从 Gitee 拉取真实开源仓库 (ThinkPHP、Webman、CRMEB)
- 自动编译前后端，启动服务
- 通过 API 调用模拟 admin/deployer/viewer 三类用户真实操作
- 产生包含成功、失败、越权拒绝、回滚的完整演示数据

---

## 二、部署原理

### 整体架构

```
浏览器 (Vue3 + Element Plus)
    │
    ▼  前端内嵌于 Go 二进制 (//go:embed dist)
    │
┌───────────────┐     ┌──────────────────────┐
│  Gin Web 服务  │────▶│      SQLite 数据库    │
│   (端口 8080)  │     │  (users/deploy_tasks) │
└───────┬───────┘     └──────────────────────┘
        │
        ▼
┌───────────────────────────────────────────────┐
│              DeployEngine (协程池, 并发上限 3)    │
│                                                │
│  ┌──────────┐  ┌──────────┐  ┌──────────────┐ │
│  │ Clone 仓库 │─▶│ 构建钩子  │─▶│ rsync 同步   │ │
│  │ (bare cache)│  │ (可选脚本) │  │ (Phase1)     │ │
│  └──────────┘  └──────────┘  └──────┬───────┘ │
│                                      │         │
│                          ┌───────────▼───────┐ │
│                          │ 原子切换 Symlink   │ │
│                          │ (Phase2, 零停机)   │ │
│                          └───────────────────┘ │
└───────────────────────────────────────────────┘
        │
        ▼
┌──────────────────────────────────────┐
│           目标服务器 (本地 SSH)        │
│                                      │
│  /tmp/demo_deploy/<project>/<env>/   │
│    ├── releases/                     │
│    │    ├── 20260530210001/          │
│    │    ├── 20260530210002/          │
│    │    └── 20260530210003/          │
│    ├── current -> releases/202602... │  ← 软链接, 原子切换
│    └── shared/  (持久化目录)          │
└──────────────────────────────────────┘
```

### 原子化部署流程

每次部署按固定 5 步执行：

| 步骤 | 名称 | 操作 |
|------|------|------|
| Step 1 | 克隆 | 从 bare cache 克隆仓库到 releases/<timestamp> |
| Step 2 | 检出 | checkout 目标 commit/branch |
| Step 3 | 构建 | 执行本地构建钩子 (可选 Shell 脚本) |
| Step 4 | Phase1 | rsync 同步文件到远程服务器 release 目录 |
| Step 5 | Phase2 | `ln -sfn` 原子切换 current 软链接 |

Phase1/Phase2 双阶段设计保证部署原子性——要么全部完成，要么不影响线上。
即使 Phase1 过程中发生中断，线上 `current` 指向的仍是旧版本。

### 回滚

回滚操作向 `/api/tasks/:id/rollback` 发送请求，
将 `current` 软链接指向上一个 release 目录。

### SSH 连接池

后端维护 SSH 连接池 (`ssh_pool.go`)，复用连接减少握手开销。
`users` 表中的每个用户可绑定 Git 提交人 (commit author)，开启限制后只有白名单作者提交的代码可被部署。

---

## 三、环境要求

| 工具 | 用途 | 备注 |
|------|------|------|
| Go 1.25+ | 编译后端 | Windows 安装或 WSL 内安装 |
| Node.js 20+ | 编译前端 (Vite + Vue3) | Windows 安装推荐 |
| Git | Mock 仓库生成 | WSL/Debian |
| SQLite3 | 查询演示数据 | WSL/Debian |
| jq | 解析 API JSON | WSL/Debian |
| curl | 调用 API | WSL/Debian |
| rsync | 部署文件同步 | WSL/Debian |
| SSH Server | 本地模拟目标服务器 | WSL/Debian (`sudo apt install openssh-server`) |

首次 WSL 安装依赖：

```bash
sudo apt-get update
sudo apt-get install -y git sqlite3 jq curl rsync golang-go openssh-server
sudo service ssh start
```

---

## 四、编译模式

### 极速模式 (默认, WSL 环境)

demo.sh 自动检测 Windows 侧工具，在 WSL 中通过 `/mnt/c/Windows/System32/cmd.exe` 调用：

```
Windows 主机
├── npm.cmd run build          → 前端 Vite 构建 (1.7s)
└── go build (GOOS=linux)      → 交叉编译 Linux 二进制 (1s)
                                    │
                                    ▼
                              WSL Debian
                              直接执行 Linux 二进制
```

**为什么快？** Windows 原生文件系统 I/O 比 WSL 跨文件系统至少快 100 倍。
CGO_ENABLED=0 因为 SQLite 使用纯 Go 实现 (`modernc.org/sqlite`)。

### 降级模式

当 Windows 侧工具不可用时，回退到 WSL 内构建：

```
WSL Debian
├── rsync web/ → /tmp/ (原生 ext4)
├── npm install + build (在 /tmp 内)
└── go build (WSL 内, GOPROXY 代理)
```

---

## 五、脚本流程详解

```bash
bash scripts/demo.sh [all|--real|start|stop|seed|status]
```

### 完整流程 (`all`, 默认)

```
check_deps()          检查系统依赖
    │
create_mock_repos()   生成 3 个 Mock Git 仓库
    │                  think → index.php Tag: v1.0.0, v1.0.1, v1.1.0-beta
    │                  webman → index.php (同上模式)
    │                  CRMEB  → index.php (同上模式)
    │                  每个仓库有 master + develop 分支, 5 次 commit
    ▼
start_backend()       编译并启动后端
    │                  ① 前端: npm build → web/dist
    │                  ② 复制: web/dist → godeployer/dist (Go embed 需要)
    │                  ③ 后端: go build -o godeployer_demo_bin .
    │                  ④ 启动: nohup ./godeployer_demo_bin &
    ▼
seed_db()             通过 API 模拟多用户操作
    │                  ① 登录 admin 获取 Token
    │                  ② 创建 deployer/viewer 账号
    │                  ③ 获取各仓库真实 commit hash
    │                  ④ 交叉执行部署请求:
    │                     - admin 部署 thinkphp-web/production
    │                     - deployer 部署 thinkphp-web/staging (预期成功)
    │                     - deployer 部署 webman-api/production (预期失败, 权限)
    │                     - deployer 部署 crmeb-shop (预期 403 越权拒绝)
    │                     - admin 部署 crmeb-shop/production (预期失败, 端口不通)
    │                  ⑤ 循环生成 30+ 条额外丰富历史
    ▼
show_status()         展示访问地址和演示账号
```

### `--real` 模式

与 `all` 区别：跳过 `create_mock_repos()`，改为从 Gitee 克隆真实仓库：

```bash
bash scripts/demo.sh --real
```

克隆仓库 (含完整 git 历史，diff 对比更真实)：
- ThinkPHP: `https://gitee.com/top-think/think.git` (depth=100)
- Webman: `https://gitee.com/walkor/webman.git` (depth=100)
- CRMEB: `https://gitee.com/ZhongBangKeJi/CRMEB.git` (depth=20)

### `seed` 模式

只重新生成演示数据，不重启服务：

```bash
bash scripts/demo.sh seed
```

操作：清空 `deploy_tasks` 表 → 重新通过 API 触发部署 → 等待轮询完成。

### `start` / `stop` / `status`

```bash
bash scripts/demo.sh start    # 仅启动后端
bash scripts/demo.sh stop     # 停止后端
bash scripts/demo.sh status   # 查看运行状态和账号信息
```

---

## 六、演示账号与权限

| 账号 | 密码 | 角色 | 权限范围 |
|------|------|------|---------|
| admin | admin123 | Admin | 全部项目、用户管理、系统清理 |
| deployer | deploy123 | Deployer | thinkphp-web + webman-api 的部署 |
| viewer | view123 | Viewer | thinkphp-web 只读查看 |

RBAC 在 `api.go` 中实现，通过中间件 `requireRole` 校验。

---

## 七、目录结构

```
deploy/
├── scripts/demo.sh            ← 本脚本
├── demo_config.yaml           ← Demo 专用配置
├── demo_projects.d/           ← 项目配置 (YAML)
│   ├── thinkphp.yaml
│   ├── webman.yaml
│   └── crmeb.yaml
├── demo_workspace/
│   ├── gitee_demo/            ← 源仓库 (Mock 或 Gitee 克隆)
│   │   ├── think/
│   │   ├── webman/
│   │   └── CRMEB/
│   ├── .cache/                ← bare repo 缓存 (部署加速)
│   └── thinkphp-web/          ← 部署工作目录
│       └── 20260530210001/    ← 每次部署的 release
├── demo_deployer.db           ← SQLite 数据库
├── demo_logs/                 ← 部署日志和 diff 缓存
└── godeployer_demo.log        ← 服务运行日志
```

---

## 八、开发测试

```bash
# 前端开发 (独立 Vite, 端口 5173, 代理 /api → :8080)
cd web && npm run dev

# 后端测试 (强制 -race 检测)
go test -v -race ./...

# 前端单元测试
cd web && npm run test

# E2E 测试 (自动启动前后端)
cd web && npm run test:e2e
```

---

## 九、常见问题

**Q: WSL 内报 "lsof: command not found"**
```bash
sudo apt-get install lsof
```

**Q: 部署失败 "git clone failed: repository does not exist"**
bare cache 尚未初始化，首次部署会自动创建。seed 脚本中的失败任务是预期行为（模拟真实错误场景）。

**Q: 8080 端口被占用**
```bash
bash scripts/demo.sh stop
lsof -i :8080 -t | xargs kill -9
bash scripts/demo.sh start
```

**Q: 前端修改不生效**
重新编译二进制或使用 Vite dev server：
```bash
# 方案一: 重新编译
bash scripts/demo.sh start

# 方案二: 开发模式
# Terminal 1: bash scripts/demo.sh start   (只启动后端)
# Terminal 2: cd web && npm run dev         (前端热更新)
# 浏览器访问 http://localhost:5173
```
