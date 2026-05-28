#!/bin/bash
# 简洁版验收脚本，使用 jq 解析 JSON
BASE="http://localhost:8080"

echo "======================================"
echo "  GoDeployer PHP Demo 数据验收报告"
echo "======================================"

# admin 登录
TOKEN=$(curl -s -X POST "$BASE/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "错误：后端未就绪，请稍后重试"
  exit 1
fi
echo "✅ admin 登录成功"

echo ""
echo "--- 项目列表（admin 视角：应看到3个项目）---"
curl -s "$BASE/api/projects" -H "Authorization: Bearer $TOKEN" | \
  jq -r '.[] | "  [\(.id)] \(.name)  repo: \(.repo)"'

echo ""
echo "--- 最新 5 条部署任务 ---"
curl -s "$BASE/api/tasks?limit=5" -H "Authorization: Bearer $TOKEN" | \
  jq -r '.[] | "  [\(.status | ascii_upcase | .[0:7])] \(.project_id) | \(.commit_id[0:8]) | env=\(.env_id) | by=\(.username)"'

echo ""
echo "--- deployer 登录，项目可见性（应只看到 thinkphp + webman）---"
DTOKEN=$(curl -s -X POST "$BASE/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"deployer","password":"deploy123"}' | jq -r '.token')
curl -s "$BASE/api/projects" -H "Authorization: Bearer $DTOKEN" | \
  jq -r '.[] | "  [\(.id)]"'

echo ""
echo "--- viewer 登录，项目可见性（应只看到 thinkphp）---"
VTOKEN=$(curl -s -X POST "$BASE/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"viewer","password":"view123"}' | jq -r '.token')
curl -s "$BASE/api/projects" -H "Authorization: Bearer $VTOKEN" | \
  jq -r '.[] | "  [\(.id)]"'

echo ""
echo "--- task_29 日志节选（CRMEB 成功部署，187MB项目）---"
curl -s "$BASE/api/tasks/29/log" -H "Authorization: Bearer $TOKEN" | \
  jq -r '.log' | head -6

echo ""
echo "--- task_30 日志节选（CRMEB 失败部署，权限错误）---"
curl -s "$BASE/api/tasks/30/log" -H "Authorization: Bearer $TOKEN" | \
  jq -r '.log' | tail -5

echo ""
echo "--- 任务汇总统计 ---"
curl -s "$BASE/api/tasks?limit=50" -H "Authorization: Bearer $TOKEN" | \
  jq -r 'group_by(.project_id)[] | 
    "\(.[0].project_id): 共\(length)次部署, 成功\([.[] | select(.status=="success")] | length)次, 失败\([.[] | select(.status=="failed")] | length)次"'

echo ""
echo "======================================"
echo "✅ Demo 数据验收完毕"
echo ""
echo "  现在请打开浏览器验证："
echo "  http://localhost:5173"
echo ""
echo "  演示账号："
echo "  admin    / admin123  → 可见全部 3 个 PHP 项目"
echo "  deployer / deploy123 → 只能看到 ThinkPHP + Webman"
echo "  viewer   / view123   → 只能看到 ThinkPHP（只读）"
echo "======================================"
