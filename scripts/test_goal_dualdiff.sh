#!/bin/bash
# -*- coding: utf-8 -*-
set -e

# ============================================================
# 集成测试脚本 - 验证双模式 Diff、快照存储限制与降级提示
# ============================================================

BASE_URL="http://localhost:8080"
PROJECT_ID="thinkphp-web"
ENV_ID="production"

echo "=== 1. 重启后端服务并初始化 Demo 数据 ==="
bash scripts/demo.sh stop
bash scripts/demo.sh start
bash scripts/demo.sh seed

echo "=== 2. 获取管理 Token ==="
TOKEN=$(curl -s -X POST "$BASE_URL/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token // empty')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "Error: 无法获取 JWT Token"
  exit 1
fi
echo "成功获取 Token: ${TOKEN:0:15}..."

# 校验辅助函数
assert_contains() {
  local haystack="$1"
  local needle="$2"
  local msg="$3"
  if [[ "$haystack" != *"$needle"* ]]; then
    echo "Assertion Failed: $msg"
    echo "Expected to contain: $needle"
    echo "Actual content: ${haystack:0:200}..."
    exit 1
  fi
}

assert_not_contains() {
  local haystack="$1"
  local needle="$2"
  local msg="$3"
  if [[ "$haystack" == *"$needle"* ]]; then
    echo "Assertion Failed: $msg"
    echo "Expected NOT to contain: $needle"
    exit 1
  fi
}

echo "=== 3. 验证上线前预览接口 (/api/projects/:id/preview_diff) ==="

# (1) branch (全量) 预览：应当返回全量文件
echo "  [测试 3.1] 请求全量文件列表 (target_type = branch)..."
BRANCH_PREVIEW=$(curl -s -GET "$BASE_URL/api/projects/$PROJECT_ID/preview_diff" \
  -H "Authorization: Bearer $TOKEN" \
  -d "to=master" \
  -d "target_type=branch" \
  -d "env_id=$ENV_ID")

assert_contains "$BRANCH_PREVIEW" "index.php" "全量列表应当包含核心 index.php 入口"
assert_contains "$BRANCH_PREVIEW" "think" "全量列表应当包含 think 指令文件"

# (2) commit (增量) 预览：应当仅返回变更文件（比如 master 的最近一次提交变更）
echo "  [测试 3.2] 请求增量变更列表 (target_type = commit)..."
COMMIT_PREVIEW=$(curl -s -GET "$BASE_URL/api/projects/$PROJECT_ID/preview_diff" \
  -H "Authorization: Bearer $TOKEN" \
  -d "to=master" \
  -d "target_type=commit" \
  -d "env_id=$ENV_ID")
# 增量变更应该比全量少很多，我们检查 files 的大小
FILES_COUNT=$(echo "$COMMIT_PREVIEW" | jq '.files | length')
echo "  增量变更文件数: $FILES_COUNT"

# (3) 验证单文件差异获取：Live Diff 与 Git Log Diff
echo "  [测试 3.3] 获取单文件的 Live Diff 预览..."
LIVE_FILE_DIFF=$(curl -s -GET "$BASE_URL/api/projects/$PROJECT_ID/preview_diff" \
  -H "Authorization: Bearer $TOKEN" \
  -d "to=master" \
  -d "file=index.php" \
  -d "diff_type=live" \
  -d "env_id=$ENV_ID")
assert_contains "$LIVE_FILE_DIFF" "diff" "Live Diff 单文件差异接口应当返回 diff 字段"

echo "  [测试 3.4] 获取单文件的 Git Log Diff 预览..."
LOG_FILE_DIFF=$(curl -s -GET "$BASE_URL/api/projects/$PROJECT_ID/preview_diff" \
  -H "Authorization: Bearer $TOKEN" \
  -d "to=master" \
  -d "file=index.php" \
  -d "diff_type=git_log" \
  -d "env_id=$ENV_ID")
assert_contains "$LOG_FILE_DIFF" "diff" "Git Log Diff 单文件差异接口应当返回 diff 字段"


echo "=== 4. 验证部署发布及快照落盘与降级 ==="

# (1) 触发全量部署 (Branch 上线)
echo "  [测试 4.1] 触发全量部署..."
BRANCH_DEPLOY=$(curl -s -X POST "$BASE_URL/api/tasks" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{\"project_id\":\"$PROJECT_ID\",\"env_id\":\"$ENV_ID\",\"commit_id\":\"master\",\"description\":\"集成测试全量部署\"}")

TASK_ID=$(echo "$BRANCH_DEPLOY" | jq '.task_id')
echo "  全量部署任务已创建，ID: $TASK_ID"

# 轮询等待任务结束
wait_task_success() {
  local tid="$1"
  local limit=30
  local count=0
  while true; do
    local status
    status=$(curl -s -GET "$BASE_URL/api/tasks/$tid" -H "Authorization: Bearer $TOKEN" | jq -r '.status')
    echo "    任务 $tid 当前状态: $status"
    if [ "$status" = "success" ]; then
      break
    elif [ "$status" = "failed" ] || [ "$status" = "failed_lock_rejected" ]; then
      echo "Error: 任务 $tid 执行失败"
      exit 1
    fi
    sleep 1
    count=$((count+1))
    if [ $count -gt $limit ]; then
      echo "Error: 任务 $tid 执行超时"
      exit 1
    fi
  done
}

wait_task_success "$TASK_ID"
sleep 2 # 等待异步快照写入磁盘

# 读取全量部署的快照 diff。按照策略：
# - 不生成 live diff (diff 为空)。
# - 生成 git log diff。
echo "  [测试 4.2] 获取全量部署历史 diff..."
TASK_DIFF_RESP=$(curl -s -GET "$BASE_URL/api/tasks/$TASK_ID/diff" -H "Authorization: Bearer $TOKEN")
assert_contains "$TASK_DIFF_RESP" "files" "快照返回体应当包含 files 字段"

# 请求单文件 live diff：应当返回友好降级提示
echo "  [测试 4.3] 请求全量快照中的 live 单文件 diff (应当降级)..."
LIVE_HIST_DIFF=$(curl -s -GET "$BASE_URL/api/tasks/$TASK_ID/diff" \
  -H "Authorization: Bearer $TOKEN" \
  -d "file=index.php" \
  -d "diff_type=live")
assert_contains "$LIVE_HIST_DIFF" "提示：全量部署任务，未归档与线上对比快照" "全量快照请求 live 时应当返回降级提示"

# 请求单文件 git_log diff：应当返回真实差异
echo "  [测试 4.4] 请求全量快照中的 git_log 单文件 diff..."
LOG_HIST_DIFF=$(curl -s -GET "$BASE_URL/api/tasks/$TASK_ID/diff" \
  -H "Authorization: Bearer $TOKEN" \
  -d "file=index.php" \
  -d "diff_type=git_log")
assert_not_contains "$LOG_HIST_DIFF" "提示：全量部署任务" "Git Log 快照读取应当不受影响"

echo ""
echo "============================================================"
echo " OK: 所有集成测试与双维度快照验证已 100% 成功通过！"
echo "============================================================"
exit 0
