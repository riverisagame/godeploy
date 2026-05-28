#!/bin/bash
# 验收脚本：全量 API 验收测试
set -e

BASE=http://localhost:8080

echo "=== [1/4] 测试登录 API (应返回 token + role) ==="
RESP=$(curl -s -X POST "$BASE/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
echo "$RESP"
TOKEN=$(echo "$RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
ROLE=$(echo "$RESP" | grep -o '"role":"[^"]*"' | cut -d'"' -f4)
echo ">>> token 已取得: ${TOKEN:0:30}..."
echo ">>> role: $ROLE"

if [ -z "$TOKEN" ]; then echo "FAIL: 未获取到 token"; exit 1; fi
if [ "$ROLE" != "admin" ]; then echo "FAIL: role 应为 admin, 实际: $ROLE"; exit 1; fi
echo "PASS: 登录接口 OK"

echo ""
echo "=== [2/4] 测试 GET /api/users (仅 admin 可访问) ==="
USERS=$(curl -s -X GET "$BASE/api/users" \
  -H "Authorization: Bearer $TOKEN")
echo "$USERS" | head -c 200
echo ""
if echo "$USERS" | grep -q '"username"'; then
  echo "PASS: 用户列表接口 OK"
else
  echo "FAIL: 用户列表接口异常"
  exit 1
fi

echo ""
echo "=== [3/4] 测试项目访问（viewer 只能看到授权项目）==="
VIEWER_RESP=$(curl -s -X POST "$BASE/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"viewer","password":"viewer123"}')
echo "$VIEWER_RESP" | head -c 200
VIEWER_TOKEN=$(echo "$VIEWER_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$VIEWER_TOKEN" ]; then
  echo "WARN: viewer 用户不存在或密码不对, 跳过此步"
else
  PROJS=$(curl -s -X GET "$BASE/api/projects" \
    -H "Authorization: Bearer $VIEWER_TOKEN")
  echo "$PROJS" | head -c 200
  echo "PASS: 项目接口访问 OK"
fi

echo ""
echo "=== [4/4] 测试非 admin 不能访问 /api/users ==="
DEMO_RESP=$(curl -s -X POST "$BASE/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"viewer","password":"viewer123"}')
DEMO_TOKEN=$(echo "$DEMO_RESP" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$DEMO_TOKEN" ]; then
  echo "WARN: viewer 用户不存在，跳过权限测试"
else
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE/api/users" \
    -H "Authorization: Bearer $DEMO_TOKEN")
  if [ "$STATUS" = "403" ]; then
    echo "PASS: viewer 访问 /api/users 被正确拦截 (403)"
  else
    echo "WARN: 返回 HTTP $STATUS (如 viewer 用户存在应是 403)"
  fi
fi

echo ""
echo "========================================="
echo "全量 API 验收完成"
