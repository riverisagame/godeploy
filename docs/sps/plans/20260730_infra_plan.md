# 基础设施部署配置计划 (2026-07-30)

## 目标
创建 Dockerfile、基础 K8s 配置文件以及完整的 Helm Chart，以便于将 godeploy 部署到现代云原生环境，同时遵守 SQLite 单机锁约束。

## 任务拆解 (纳米级)

### Task 1: 创建 Dockerfile
- **文件路径**: `Dockerfile`
- **逻辑改动**:
  - `FROM golang:1.26-alpine AS builder`
  - 复制 go.mod, go.sum，执行 `go mod download`
  - 编译 `CGO_ENABLED=0 GOOS=linux go build -o pdeploy ./cmd/pdeploy`
  - `FROM alpine:latest`
  - `apk add --no-cache git rsync openssh-client`
  - 复制产物，创建数据目录 `/app/data` 和 `/app/workspace`
  - 设定 `ENTRYPOINT ["/app/pdeploy"]`

### Task 2: 创建独立 K8s 配置文件 (Raw Manifests)
- **文件路径**: `k8s/configmap.yaml`, `k8s/pvc.yaml`, `k8s/deployment.yaml`, `k8s/service.yaml`
- **逻辑改动**:
  - `configmap.yaml`: 定义 `PORT: "8080"`, `DB_PATH: "/app/data/pdeploy.db"`, `WORKSPACE_DIR: "/app/workspace"`
  - `pvc.yaml`: 请求 5Gi 存储空间用于 `/app/data` 和 `/app/workspace` (或拆分为两个)
  - `deployment.yaml`: `replicas: 1`，挂载 pvc
  - `service.yaml`: 暴露 8080 端口，`type: ClusterIP`

### Task 3: 创建 Helm Chart (骨架与 Values)
- **文件路径**: `helm/godeploy/Chart.yaml`, `helm/godeploy/values.yaml`
- **逻辑改动**:
  - `Chart.yaml`: `apiVersion: v2`, `name: godeploy`, `version: 0.1.0`
  - `values.yaml`: 定义 `replicaCount: 1`, `image.repository`, `persistence.data.enabled`, `persistence.workspace.enabled`

### Task 4: 创建 Helm Chart (Templates)
- **文件路径**: `helm/godeploy/templates/deployment.yaml`, `helm/godeploy/templates/service.yaml`, `helm/godeploy/templates/pvc.yaml`, `helm/godeploy/templates/configmap.yaml`
- **逻辑改动**: 
  - 根据 `values.yaml` 渲染对应的 K8s 对象，逻辑同 Task 2，但参数化。
