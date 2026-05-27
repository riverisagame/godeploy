# ADR-003: GoDeployer 核心部署引擎与发布流深度架构设计

## 1. 背景与目标
在多项目、多环境配置的基础上，为确保系统的高可用性、部署的高效性以及容灾的高可靠性，从软件架构设计和分布式系统发布模式出发，对核心部署引擎、目标机交互、同步效率、滚动升级以及容灾机制进行深度设计和评审。

---

## 2. 深度架构设计与建议

### 2.1 基于 Rsync 硬链接的极速增量发布 (Sync Performance)
- **现状缺陷**：传统的发布方式（如直接复制或普通的 rsync）每次部署都将整个项目同步到新宿主机 release 目录。对于包含大量静态资源（图片、Vue 编译包）或依赖包（vendor、node_modules）的项目，每次传输可能消耗几分钟，极大浪费网络带宽与 I/O。
- **架构建议（Rsync Link-Dest）**：
  - 使用 Linux Rsync 的 `--link-dest` 特性。
  - 核心逻辑：在目标机执行同步时，将 `--link-dest` 指向上一个成功的 Release 目录。
  - **命令模式**：
    ```bash
    rsync -az --delete --link-dest=../<last_release_name>/ <local_path>/ <user>@<host>:<deploy_path>/releases/<new_release_name>/
    ```
  - **优势**：
    - 只有发生改变的文件才会在网络中传输。
    - 未改变的文件在新宿主机目录中被创建为指向旧文件的**硬链接 (Hard Link)**。
    - 节省磁盘空间：10 个 Release 可能只占 1.1 个 Release 的实际磁盘空间。
    - 极速发布：对于大项目，同步过程可以从分钟级降至 **1-3 秒**。

### 2.2 无空窗期原子软链接切换 (Zero Downtime Atomic Symlink)
- **现状缺陷**：直接删除软链接再重新创建（`rm -f current && ln -s ... current`）并不是原子操作。在删除与新建的微秒甚至毫秒级空档期，访问该路径的 HTTP 请求或进程会报错，造成系统抖动。
- **架构建议（原子替换）**：
  - 在目标服务器上使用 **`ln` + `mv` 组合** 来实现真正的原子切换。
  - **执行步骤**：
    1. 在根目录下创建一个临时的软链接指向新的 Release：
       ```bash
       ln -sfn <deploy_path>/releases/<new_release_name> <deploy_path>/current_temp
       ```
    2. 通过重命名原子覆盖原有的 `current`：
       ```bash
       mv -Tf <deploy_path>/current_temp <deploy_path>/current
       ```
  - **优势**：
    - Linux 内核保证了 `mv` 重命名目录/链接的操作在系统调用级别是原子（Atomic）的。
    - 即使正在承受高并发流量，切换期间也不会产生任何 HTTP 502/404。

### 2.3 滚动升级策略（Rolling Update）
- **现状缺陷**：当一个环境（如 Production）有多台部署目标服务器时，若采用全部并发同步和切换，若新版本代码存在致命 Bug（如配置文件解析 Panic），会导致集群中所有服务同时崩溃，造成严重事故。
- **架构建议**：
  - 在环境配置中引入 `rolling_update` 参数：
    ```yaml
    environments:
      - id: "production"
        rolling_update:
          batch_size: 1  # 每次仅部署 1 台服务器，成功后再部署下一台
          stop_on_failure: true  # 如果单台部署或健康检查失败，立即终止后续部署
    ```
  - 部署引擎采用“分批并发”模式。第一批部署成功且通过基本健康检查后，再对剩余的节点进行部署。

### 2.4 安全沙箱与最小权限限制 (Security Sandbox)
- **现状缺陷**：宿主机部署系统需要持有目标机的 SSH 私钥。如果宿主机遭遇攻击，攻击者可直接控制所有目标服务器的 root 权限。
- **架构建议**：
  - **SSH 用户隔离**：禁止使用 `root` 用户进行部署，目标机需创建专用的 `deploy` 用户。
  - **Sudo 权限限制**：限制 `deploy` 用户的 sudo 权限。如确实需要 reload nginx 或 php-fpm，只能在 `/etc/sudoers` 中开放特定命令的免密 sudo：
    ```sudoers
    deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload php-fpm, /usr/bin/systemctl reload nginx
    ```
  - **目录写权限收敛**：`deploy` 用户只对 `<deploy_path>` 拥有写权限，其余系统目录保持只读。

---

## 3. 设计权衡 (Tradeoffs)

| 方案建议 | 带来优势 | 引入成本 / 限制 |
| :--- | :--- | :--- |
| **Rsync Link-Dest** | 节省大量磁盘，传输速度提升 90% | 要求目标服务器支持 Linux 硬链接，对于不支持的特殊存储挂载目录（如 NFS/Samba 挂载盘）会退化为全量同步。 |
| **滚动升级 (Rolling)** | 防止故障扩大化，提供金丝雀发布效果 | 增长了总的部署耗时。且需要系统支持多版本代码混跑的兼容性（例如数据库 Schema 必须向前兼容）。 |
| **原子软链接切换** | 保证零停机时间 (Zero Downtime) | 必须保证宿主机在 Linux 兼容的 POSIX 文件系统上运行。在极少数非标准环境下可能需要定制重命名命令。 |
