# ==========================================
# 阶段 1：构建阶段 (Build)
# ==========================================
FROM golang:1.26.5-alpine AS builder

# 安装构建必需的基础包
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

# 预先缓存依赖，利用 Docker 缓存层
COPY go.mod go.sum ./
RUN go mod download

# 拷贝全量源代码
COPY . .

# 编译无 CGO 依赖的二进制文件
ENV CGO_ENABLED=0
ENV GOOS=linux
RUN go build -ldflags="-w -s" -o godeploy ./cmd/pdeploy

# ==========================================
# 阶段 2：运行阶段 (Runtime)
# ==========================================
FROM alpine:latest

# 导入时区与根证书
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 创建专用低权限用户以提升安全性 (大厂零信任标准)
RUN addgroup -g 10000 godeploy && \
    adduser -D -u 10000 -G godeploy godeploy

WORKDIR /app

# 从构建阶段拷贝编译好的二进制
COPY --from=builder /app/godeploy .

# 创建必要目录并赋予权限
RUN mkdir -p /var/lib/godeploy /etc/godeploy && \
    chown -R godeploy:godeploy /app /var/lib/godeploy /etc/godeploy

# 切换为低权限用户
USER godeploy:godeploy

# 声明挂载点 (重要：外部必须挂载此目录以保证 SQLite 状态留存)
VOLUME ["/var/lib/godeploy", "/etc/godeploy"]

# 暴露服务端口
EXPOSE 8080

# 启动服务
CMD ["./godeploy", "-config", "/etc/godeploy/config.conf"]
