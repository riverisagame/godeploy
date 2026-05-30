#!/bin/bash
# 启动 GoDeployer 后端并运行 demo seed 脚本

cd /mnt/d/claudeprj/deploy

# 停止旧进程
pkill -f "go run main.go" 2>/dev/null || true
sleep 1

# 启动后端
echo "启动后端服务..."
go run main.go --config=demo_config.yaml > /tmp/godeployer_demo.log 2>&1 &
sleep 3

# 检查是否启动成功
if curl -s http://localhost:8080/api/login > /dev/null 2>&1; then
  echo "后端启动成功"
else
  echo "等待后端启动..."
  sleep 3
fi

# 执行 seed
bash /mnt/d/claudeprj/deploy/scripts/seed_php_demo.sh
