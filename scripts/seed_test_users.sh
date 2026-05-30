#!/bin/bash
# 创建测试用 viewer 用户
BASE=http://localhost:8080

TOKEN=$(curl -s -X POST "$BASE/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | \
  grep -o '"token":"[^"]*"' | cut -d'"' -f4)

echo "Admin token: ${TOKEN:0:30}..."

echo "创建 viewer 用户..."
curl -s -X POST "$BASE/api/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"username":"viewer","password":"viewer123","role":"viewer","permitted_projects":"proj1"}'
echo ""

echo "创建 deployer 用户..."
curl -s -X POST "$BASE/api/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"username":"deployer","password":"deployer123","role":"deployer","permitted_projects":"*"}'
echo ""

echo "当前用户列表:"
curl -s -X GET "$BASE/api/users" \
  -H "Authorization: Bearer $TOKEN"
echo ""
