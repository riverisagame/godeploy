#!/bin/bash
pkill -f "go run main.go" 2>/dev/null || true
sleep 1
cd /mnt/d/claudeprj/deploy
go run main.go --config=demo_config.yaml > /tmp/demo_srv.log 2>&1 &
sleep 2
echo "后端已启动 PID=$!"
