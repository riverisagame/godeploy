# 配置终极指南 (Configuration Guide)

## 什么是部署器配置？

部署器 (GoDeployer) 的核心驱动力是配置文件（通常为 `config.yaml`）。它定义了项目的所有环境、服务器节点、以及在各个阶段需要执行的钩子函数。通过灵活的配置，您可以实现从最简单的单机部署到复杂的多环境、多集群滚动发布。

## 配置结构总览

一个完整的配置文件由三个主要部分构成：
1. **GlobalConfig** (全局配置)：定义了应用的端口、日志路径等。
2. **ProjectConfig** (项目配置)：支持多项目，每个项目包含独立的 Git 仓库地址。
3. **EnvironmentConfig** (环境配置)：定义了每个项目的发布环境（例如 `test`, `prod`），以及该环境下的服务器列表和执行钩子。

---

## 详细字段解析

### GlobalConfig (全局)
| 字段名 | 作用 | 示例值 |
|:---|:---|:---|
| `Port` | Web API 启动监听端口 | `8080` |
| `DBPath` | SQLite 数据库文件存放路径 | `./godeployer.db` |
| `Workspace` | 项目拉取和构建的工作空间目录 | `/tmp/godeployer_workspace` |

### ProjectConfig (项目级别)
| 字段名 | 作用 | 示例值 |
|:---|:---|:---|
| `Name` | 项目名称（显示在控制台） | `thinkphp-demo` |
| `Repo` | Git 仓库地址 (支持 SSH 和 HTTP) | `git@github.com:fex-team/fex-team.github.io.git` |
| `Environments` | 包含此项目下的多个环境配置（Map 结构） | - |

### EnvironmentConfig (环境级别)
这是整个部署过程中最重要的部分。

| 字段名 | 作用 | 示例值 |
|:---|:---|:---|
| `DeployPath` | 目标服务器上的部署基础目录 | `/var/www/myproject` |
| `Servers` | 该环境包含的目标服务器列表 | - |
| `BuildConfig` | 构建阶段配置（在本地或专门构建机运行） | - |
| `BeforeSync` | **代码同步前**的远端钩子。在 rsync 发生之前在目标机器上执行。 | `echo "Ready to receive sync..."` |
| `BeforeSymlink` | **软链切换前**的远端钩子。通常用于执行 `.env` 拷贝、数据库迁移、`npm run build`。此时新代码已经同步到 `releases/xxx` 目录。 | `composer install --no-dev` |
| `AfterSymlink` | **软链切换后**的远端钩子。通常用于重启 PHP-FPM、Nginx 或清理缓存。 | `systemctl reload nginx` |

### ServerConfig (服务器节点)
| 字段名 | 作用 | 示例值 |
|:---|:---|:---|
| `Host` | 服务器 IP 或域名 | `192.168.1.100` |
| `Port` | SSH 端口 | `22` |
| `User` | SSH 登录用户 | `root` |
| `SSHKeyPath` | SSH 私钥路径 | `/home/user/.ssh/id_rsa` 或 `C:\Users\Admin\.ssh\id_rsa` |
| `Password` | SSH 密码 (若不使用密钥) | `password123` |

### BuildConfig (构建指令)
| 字段名 | 作用 | 示例值 |
|:---|:---|:---|
| `Command` | 部署前在本地执行的脚本命令 | `npm install && npm run build` |

---

## 钩子函数流转示意图

一次发布过程的完整生命周期：
1. **Pull Code**: 在本地拉取最新的代码。
2. **Local Build (BuildConfig.Command)**: 执行前端编译或依赖下载。
3. **Remote BeforeSync**: 远端机器在接收代码前，做预备工作。
4. **Rsync**: 核心文件同步到目标机器的新 Release 目录 (`deployPath/releases/timestamp`)。
5. **Remote BeforeSymlink**: 在 Release 目录中安装依赖、做数据库 Migrations 等操作。
6. **Symlink Change**: 切换 `deployPath/current` 软链接至最新 Release。
7. **Remote AfterSymlink**: 清理工作、平滑重启服务等。

## 实战范例

以下是一个完整的多环境部署范例配置：
```yaml
global:
  port: "8080"
  db_path: "./godeployer.db"
  workspace: "/tmp/godeployer_workspace"

projects:
  my-web:
    name: "My Enterprise Web"
    repo: "https://gitee.com/liu21st/thinkphp.git"
    environments:
      prod:
        deploy_path: "/www/wwwroot/prod_site"
        before_symlink: |
          cd {{release_path}}
          composer install --no-dev
          php artisan migrate --force
        after_symlink: |
          systemctl reload php-fpm
        servers:
          - host: "10.0.0.1"
            port: "22"
            user: "root"
            ssh_key_path: "/root/.ssh/id_rsa"
          - host: "10.0.0.2"
            port: "22"
            user: "root"
            ssh_key_path: "/root/.ssh/id_rsa"
```
