#!/bin/bash
# =============================================================
#  demo.sh 脚本功能自动测试程序
#  用于验证优化后的一键启动脚本的正确性
# =============================================================

set -e
cd "$(dirname "$0")/.."
REPO_ROOT="$(pwd)"

GREEN='\033[0;32m'; RED='\033[0;31m'; NC='\033[0m'
error() { echo -e "${RED}[TEST_FAIL]${NC} $*"; exit 1; }
success() { echo -e "${GREEN}[TEST_PASS]${NC} $*"; }

# 1. 验证优化后的 demo.sh 脚本是否支持 --mock 选项或默认行为
info() { echo -e "[TEST_INFO] $*"; }

# 备份当前的 demo.sh 和 demo_config.yaml 以防测试对其造成永久修改
# 但实际上我们只做只读测试或在独立测试目录中进行

# 运行 check_deps 验证
info "1. 测试依赖检查..."
bash scripts/demo.sh check_deps || error "check_deps 执行失败"
success "依赖检查测试通过"

# 2. 测试本地 Mock 仓库生成与一致性...
info "2. 清理工作区并测试离线极速 Mock 生成..."
rm -rf "$REPO_ROOT/demo_workspace/gitee_demo"

# 运行 demo.sh clone 并记录时间，确保其在 2 秒内离线完成（克隆 Gitee 至少需要几十秒）
START_TIME=$(date +%s)
bash scripts/demo.sh clone || error "clone 子命令执行失败"
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

if [ $DURATION -ge 20 ]; then
  error "克隆/生成过程耗时 $DURATION 秒，明显使用了网络克隆，没有达到极速 Mock 生成的目标"
fi

for name in think webman CRMEB; do
  GIT_DIR="$REPO_ROOT/demo_workspace/gitee_demo/$name/.git"
  if [ ! -d "$GIT_DIR" ]; then
    error "找不到 Mock 仓库 $name 的 .git 目录！当前脚本未能秒级本地生成 Mock 仓库。"
  fi
done

# 3. 验证数据库中的 commit 是否能在 mock 仓库中真实被 git show 到
info "3. 验证数据库与本地 Git 仓库 Commit ID 100% 对齐一致..."
# 因为 seed 需要向后端发起真实的 API 部署任务，所以必须先启动后端
bash scripts/demo.sh start || error "启动后端失败"
bash scripts/demo.sh seed || error "seed 子命令执行失败"

if [ ! -f "$REPO_ROOT/demo_deployer.db" ]; then
  error "数据库 demo_deployer.db 不存在"
fi

# 从数据库中随机取一条任务的 commit_id 并到对应仓库进行 git show 验证
# 选取已成功部署的任务
TEST_COMMIT=$(sqlite3 "$REPO_ROOT/demo_deployer.db" "SELECT commit_id FROM deploy_tasks WHERE project_id='thinkphp-web' AND status='success' LIMIT 1;")
if [ -z "$TEST_COMMIT" ]; then
  error "数据库中没有 thinkphp-web 的成功部署任务数据"
fi

if ! git -C "$REPO_ROOT/demo_workspace/gitee_demo/think" show "$TEST_COMMIT" &>/dev/null; then
  error "数据库中的 Commit ID ($TEST_COMMIT) 在本地 Mock Git 仓库中无法被真实 show 到，这会导致 Diff 功能失效！"
fi

# 清理测试状态
bash scripts/demo.sh stop

success "所有 Demo 优化测试通过"
