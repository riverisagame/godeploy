# PDeploy (GoDeploy) 纳米级部署极详指南

本文档提供本系统在 Docker、Systemd (裸机) 以及 Kubernetes (Helm) 三种环境下的纳米级底层实现拆解及极详部署说明。所有部署方案均严格遵守大厂 **“零信任安全”** 与 **“数据防丢机制”** 标准。

---

## 🐋 1. Docker 容器化部署 (推荐快速尝鲜/云服务器)

本系统采用两阶段构建 (Multi-stage build) 和 Alpine 底座，最终产物极其轻量（约 30MB）且完全摆脱 CGO 依赖。

### 1.1 纳米级实现解析 (基于 `Dockerfile`)
1. **零信任降权 (Security)**：容器内不使用 `root` 运行，强制创建 `godeploy:godeploy` (UID: 10000) 专属低权限用户。如果发生容器逃逸，攻击者获得的也是极低权限。
2. **状态分离 (Stateless/Stateful)**：系统自身是无状态的，但元数据依赖 SQLite。容器强制声明了两个 `VOLUME` 挂载点：
   - `/var/lib/godeploy`：用于持久化 SQLite 数据库 (`pdeploy.db`) 和 Workspace 裸仓库。
   - `/etc/godeploy`：用于存放配置文件 `config.conf`。

### 1.2 极详部署步骤

首先在宿主机创建数据目录，并**务必调整权限**，使容器内的 UID 10000 拥有读写权：

```bash
mkdir -p /data/pdeploy/lib /data/pdeploy/etc
# 非常关键：将宿主机目录归属权移交给容器内的 godeploy 用户(UID 10000)
sudo chown -R 10000:10000 /data/pdeploy/lib /data/pdeploy/etc
```

在 `/data/pdeploy/etc` 放入您的 `config.conf`。然后执行运行：

```bash
docker run -d \
  --name pdeploy-server \
  --restart unless-stopped \
  -p 8080:8080 \
  -e PORT="8080" \
  -e JWT_SECRET="请务必替换为您生成的一个至少16位以上的强密码_!@#" \
  -v /data/pdeploy/lib:/var/lib/godeploy \
  -v /data/pdeploy/etc:/etc/godeploy \
  riverisagame/godeploy:latest
```

---

## 🛠️ 2. Systemd 裸机单机部署 (推荐中小型物理机/VM)

针对不希望引入容器虚拟化开销的传统 VM 或物理服务器，我们提供了一键化的防御性安装脚本 `install.sh` 和安全更新脚本 `update.sh`。

### 2.1 `install.sh` 纳米级动作拆解
- **自动降权创建用户**：执行 `useradd -r -s /bin/false godeploy` 创建禁止 Shell 登录的系统级影子用户，避免服务被攻破后引发横向越权。
- **强制目录隔离**：
  - 数据：`/var/lib/godeploy`
  - 配置：`/etc/godeploy`
  - 日志：`/var/log/godeploy` (如未启用标准输出)
- **安全赋权**：所有工作目录严格执行 `chown -R godeploy:godeploy`。
- **平滑自启注册**：拉取 `godeploy.service`，注册至 systemd 并执行 `systemctl daemon-reload` 和 `enable --now`。

#### **安装流程：**
```bash
# 必须在项目源码根目录执行，且必须具有构建好的 godeploy 二进制文件
chmod +x deploy/scripts/install.sh
sudo ./deploy/scripts/install.sh
```

### 2.2 `update.sh` 纳米级动作拆解 (防抖设计)
生产环境升级最怕数据覆写，`update.sh` 对此做了严格防御：
1. **预检查**：检查当前目录是否存在新版 `godeploy` 二进制，防误触。
2. **平滑停机**：执行 `systemctl stop godeploy` 拒绝新请求，等待旧连接排空。
3. **强制自动快照 (防呆)**：在替换前，脚本会自动将 `/var/lib/godeploy/pdeploy.db` 复制为 `godeploy_YYYYMMDD_HHMMSS.db.bak`，保留升级前的数据底座。
4. **覆盖并拉起**：替换 `/usr/local/bin/godeploy`，再执行 `start` 恢复服务。

#### **升级流程：**
```bash
# 获取新版本二进制后，在根目录执行
chmod +x deploy/scripts/update.sh
sudo ./deploy/scripts/update.sh
```

---

## ☸️ 3. Kubernetes / Helm 集群部署 (推荐云厂商托管 K8s)

针对大型企业级高可用基础设施，本系统提供了完整的 Helm Chart，位于 `deploy/helm/godeploy/`。

### 3.1 架构设计解析
**为什么是 StatefulSet 而不是 Deployment？**
- 尽管我们提倡云原生无状态，但本系统使用 SQLite 作为数据队列和元数据中心（摒弃了繁重的 Redis/MySQL 集群以保极简）。
- 在多副本的 K8s 环境下，SQLite 无法支持多个 Pod 同时进行跨节点的并发写。因此，Helm Chart 默认设计为单副本挂载独立的 `PersistentVolumeClaim (PVC)`，从而兼顾高容灾与极简架构。

### 3.2 极详部署步骤

#### 步骤一：定制 `values.yaml` 参数
编辑 `deploy/helm/godeploy/values.yaml`，注意以下核心参数：
- `image.repository` & `image.tag`: 修改为您推送的私有仓库地址。
- `persistence.enabled`: **务必保持为 true**。
- `persistence.size`: 建议 `10Gi` 起步。
- `persistence.storageClass`: 如果您的集群 (如 AWS EKS) 支持默认存储类，可不填；否则指定 `gp2` / `alicloud-disk` 等。
- `env.JWT_SECRET`: 务必替换为长串安全秘钥。

#### 步骤二：安装 Release
```bash
cd deploy/helm/

# --create-namespace 会自动创建隔离的命名空间
helm install pdeploy ./godeploy \
    --namespace pdeploy-system \
    --create-namespace \
    --set env.JWT_SECRET="ProductionSafeSecretKey123!@#"
```

#### 步骤三：日常无损升级
当发布了新版本（例如修改了 image tag）：
```bash
# 采用 Helm Upgrade 直接热更，StatefulSet 会平滑卸载旧 Pod，将现存 PVC 挂载到新 Pod 上，保证数据绝对无损。
helm upgrade pdeploy ./godeploy \
    --namespace pdeploy-system \
    --set image.tag="v3.1.0"
```

### 3.3 卸载提示
由于 K8s 的保护机制，`helm uninstall pdeploy -n pdeploy-system` **不会自动删除 PVC**。
如需彻底清理数据，必须手工删除 PVC：
```bash
kubectl delete pvc data-pdeploy-0 -n pdeploy-system
```
