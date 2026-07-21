# 纳米级部署极简指南 (Nano Deploy Guide)

## 🎯 方案一：Docker 极速体验 (推荐尝鲜)
无需安装环境，一键拉起容器：
```bash
docker run -d \
  --name godeploy \
  -p 8080:8080 \
  -v /my/local/data:/var/lib/godeploy \
  -v /my/local/config.conf:/etc/godeploy/config.conf \
  riverisagame/godeploy:latest
```

## 🛠️ 方案二：Systemd 裸机单机部署 (推荐中小型生产)
直接在宿主机（如 Ubuntu/CentOS）上运行进程：
```bash
# 赋予脚本执行权限并一键安装
chmod +x deploy/scripts/install.sh
sudo ./deploy/scripts/install.sh

# 查看服务状态
systemctl status godeploy
```

## ☸️ 方案三：Kubernetes/Helm 集群部署 (推荐大型高可用)
适用于 K8s 集群，基于 StatefulSet 保障数据安全：
```bash
# 进入 helm 目录
cd deploy/helm/

# 一键安装 release
helm install godeploy ./godeploy --namespace godeploy-system --create-namespace

# 查看 Pod 和状态
kubectl get pods -n godeploy-system
```
