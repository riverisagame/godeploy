#!/bin/bash
# 验证 diff 接口修复效果

BASE="http://localhost:8080"

TOKEN=$(curl -s -X POST "$BASE/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

echo "=== Task 21 Diff（ThinkPHP: 43983ad → c7c11f6）==="
RESP=$(curl -s "$BASE/api/tasks/21/diff" -H "Authorization: Bearer $TOKEN")
DIFF=$(echo "$RESP" | jq -r '.diff')
LINES=$(echo "$DIFF" | wc -l)
echo "  diff 行数: $LINES"
echo "$DIFF" | head -8

echo ""
echo "=== Task 24 Diff（ThinkPHP: 98d8c5e → 49917ae）==="
RESP=$(curl -s "$BASE/api/tasks/24/diff" -H "Authorization: Bearer $TOKEN")
DIFF=$(echo "$RESP" | jq -r '.diff')
LINES=$(echo "$DIFF" | wc -l)
echo "  diff 行数: $LINES"
echo "$DIFF" | head -8

echo ""
echo "=== Task 29 Diff（CRMEB: 1953370 → 77973362）==="
RESP=$(curl -s "$BASE/api/tasks/29/diff" -H "Authorization: Bearer $TOKEN")
DIFF=$(echo "$RESP" | jq -r '.diff')
LINES=$(echo "$DIFF" | wc -l)
echo "  diff 行数: $LINES"
echo "$DIFF" | head -8

echo ""
echo "=== Task 20 Diff（ThinkPHP 首次部署，应返回无对比基准）==="
curl -s "$BASE/api/tasks/20/diff" -H "Authorization: Bearer $TOKEN" | jq -r '.diff'
