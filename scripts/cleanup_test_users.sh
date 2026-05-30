#!/bin/bash
# 验收后清理测试用户（恢复原始数据状态）
BASE=http://localhost:8080

TOKEN=$(curl -s -X POST "$BASE/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | \
  grep -o '"token":"[^"]*"' | cut -d'"' -f4)

echo "删除 viewer 用户..."
curl -s -X DELETE "$BASE/api/users/viewer" \
  -H "Authorization: Bearer $TOKEN"
echo ""

echo "删除 deployer 用户..."
curl -s -X DELETE "$BASE/api/users/deployer" \
  -H "Authorization: Bearer $TOKEN"
echo ""

echo "当前用户列表（应只剩 admin）:"
curl -s -X GET "$BASE/api/users" \
  -H "Authorization: Bearer $TOKEN"
echo ""
