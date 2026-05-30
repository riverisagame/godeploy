#!/bin/bash
echo "=== 后端状态 ==="
RESP=$(curl -s -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo "$RESP" | jq -r '.token // empty')
if [ -n "$TOKEN" ]; then
  echo "✅ 后端就绪"
  echo ""
  echo "=== 项目列表 ==="
  curl -s http://localhost:8080/api/projects \
    -H "Authorization: Bearer $TOKEN" | jq -r '.[] | "  [\(.id)] \(.name)"'
  echo ""
  echo "=== 最新部署任务（前5条）==="
  curl -s "http://localhost:8080/api/tasks?limit=5" \
    -H "Authorization: Bearer $TOKEN" | \
    jq -r '.[] | "  [\(.status)] \(.project_id) | \(.commit_id[0:8]) | \(.env_id)"'
  echo ""
  echo "=== 用户列表 ==="
  curl -s http://localhost:8080/api/users \
    -H "Authorization: Bearer $TOKEN" | \
    jq -r '.[] | "  \(.username) [\(.role)] -> \(.permitted_projects)"'
else
  echo "❌ 后端未就绪: $RESP"
fi
