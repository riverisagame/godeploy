#!/bin/bash
# =============================================================
#  GoDeployer Demo 一键启动脚本
#  用途：快速搭建含真实 Gitee PHP 项目数据的本地演示环境
#
#  用法：
#    bash scripts/demo.sh              # 完整初始化 + 启动
#    bash scripts/demo.sh seed         # 仅重置数据库演示数据
#    bash scripts/demo.sh start        # 仅启动后端（不重置数据）
#    bash scripts/demo.sh stop         # 停止后端
#    bash scripts/demo.sh status       # 查看服务状态
#    bash scripts/demo.sh verify       # 验证演示数据与接口
#
#  前置要求：
#    - WSL Debian 环境（或原生 Linux）
#    - Go 1.21+
#    - git, sqlite3, jq, curl
#    - Node.js 18+（前端）
# =============================================================

set -e
cd "$(dirname "$0")/.."
REPO_ROOT="$(pwd)"

# ---- 颜色输出 ----
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; RED='\033[0;31m'; BLUE='\033[0;34m'; NC='\033[0m'
info()    { echo -e "${BLUE}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC}   $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERR]${NC}  $*"; exit 1; }

# ---- 配置变量 ----
BASE_URL="http://localhost:8080"
DB_PATH="$REPO_ROOT/demo_deployer.db"
LOG_DIR="$REPO_ROOT/demo_logs"
WORKSPACE="$REPO_ROOT/demo_workspace"
GITEE_WORKSPACE="$WORKSPACE/gitee_demo"
CONFIG="demo_config.yaml"
PID_FILE="/tmp/godeployer_demo.pid"

# ============================================================
# 子命令：check_deps
# ============================================================
check_deps() {
  info "检查依赖..."
  for cmd in go git sqlite3 jq curl; do
    if ! command -v "$cmd" &>/dev/null; then
      error "缺少依赖: $cmd，请先安装"
    fi
  done
  success "依赖检查通过"
}

# ============================================================
# 子命令：clone_repos — 克隆/更新 Gitee PHP 演示仓库
# ============================================================
clone_repos() {
  info "准备 Gitee PHP 演示仓库..."
  mkdir -p "$GITEE_WORKSPACE"

  # ThinkPHP
  if [ ! -d "$GITEE_WORKSPACE/think/.git" ]; then
    info "克隆 ThinkPHP..."
    git clone --depth=100 https://gitee.com/top-think/think.git "$GITEE_WORKSPACE/think" 2>&1 | tail -3
  else
    info "ThinkPHP 仓库已存在，跳过克隆"
  fi

  # Webman
  if [ ! -d "$GITEE_WORKSPACE/webman/.git" ]; then
    info "克隆 Webman..."
    git clone --depth=100 https://gitee.com/walkor/webman.git "$GITEE_WORKSPACE/webman" 2>&1 | tail -3
  else
    info "Webman 仓库已存在，跳过克隆"
  fi

  # CRMEB（大型项目，depth=50 避免太慢）
  if [ ! -d "$GITEE_WORKSPACE/CRMEB/.git" ]; then
    info "克隆 CRMEB 商城系统（约 187MB，请稍候）..."
    git clone --depth=50 https://gitee.com/ZhongBangKeJi/CRMEB.git "$GITEE_WORKSPACE/CRMEB" 2>&1 | tail -3
  else
    info "CRMEB 仓库已存在，跳过克隆"
  fi

  success "所有演示仓库就绪"
}

# ============================================================
# 子命令：seed_db — 初始化数据库并写入历史部署数据
# ============================================================
seed_db() {
  info "初始化演示数据库..."
  mkdir -p "$LOG_DIR"

  # 启动后端初始化表结构（如未存在则先启动1秒再杀掉）
  if [ ! -f "$DB_PATH" ]; then
    info "首次运行，初始化数据库表结构..."
    cd "$REPO_ROOT"
    go run main.go --config="$CONFIG" &
    TMP_PID=$!
    sleep 3
    kill $TMP_PID 2>/dev/null || true
    sleep 1
  fi

  info "写入演示任务数据..."
  sqlite3 "$DB_PATH" <<'ENDSQL'
-- 清理旧 demo 任务
DELETE FROM deploy_tasks WHERE project_id IN ('backend-api','frontend-web');

-- ============================================================
-- ThinkPHP 后端框架（5条 - 真实 Gitee commits）
-- ============================================================
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('thinkphp-web','production','43983ad1d0b25506c6792a7444ae6f22863359fd','success','20260428100000',1,'admin','{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',datetime('now','-30 days'));
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('thinkphp-web','production','c7c11f62f10258b9d9ad2aea1c2a62eda8b2531f','success','20260508100000',2,'deployer','{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',datetime('now','-20 days'));
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('thinkphp-web','staging','7cc4119dcaab2f72606d46eafd16582e887b5d3e','success','20260518090000',2,'deployer','{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',datetime('now','-10 days'));
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('thinkphp-web','production','98d8c5e09712042a51a2a79622bbce422b48c0ea','success','20260523143000',1,'admin','{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',datetime('now','-5 days'));
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('thinkphp-web','production','49917ae7c0de3c7c12de367b99b671321c3c304c','success','20260527113000',2,'deployer','{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',datetime('now','-1 days'));

-- ============================================================
-- Webman 微服务接口（4条）
-- ============================================================
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('webman-api','production','fbd7377964d6a2f9b8fd6b4bbd543a30c28d13af','failed','20260513160000',1,'admin','{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',datetime('now','-15 days'));
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('webman-api','production','e8cedb15979e7804d9b7bf72128b15ebdb538d75','success','20260516100000',1,'admin','{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',datetime('now','-12 days'));
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('webman-api','production','9216f9c05ba6e6cc54aeed6476d46b0c12419295','success','20260520143000',2,'deployer','{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',datetime('now','-8 days'));
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('webman-api','staging','99c2aafc555521c6be37edb03f1d4704ca9c2818','success','20260525091500',2,'deployer','{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',datetime('now','-3 days'));

-- ============================================================
-- CRMEB 商城系统（5条）
-- ============================================================
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('crmeb-shop','test','779733627341fdb56e2c2291f695c18d3de52eaf','success','20260520150000',1,'admin','{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',datetime('now','-8 days'));
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('crmeb-shop','production','1953370289f0c6bd5a7e07ddf52bd57a6dd233ac','failed','20260522193000',1,'admin','{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',datetime('now','-6 days'));
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('crmeb-shop','test','50ac735ac375be46ef9eabfca9c46eaae16779d0','success','20260525103000',1,'admin','{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',datetime('now','-3 days'));
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('crmeb-shop','production','60b594745f50fe3e9a4ffc08ca8ea1f02001742a','success','20260526143000',1,'admin','{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',datetime('now','-2 days'));
INSERT OR IGNORE INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('crmeb-shop','production','af0de843746289693f01ac7fe08a3bacdf862137','success','20260528211000',1,'admin','{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',datetime('now','-1 hours'));
ENDSQL

  success "演示任务数据写入完成（14 条）"

  # 创建演示用户
  info "创建演示用户..."
  _wait_backend
  local TOKEN
  TOKEN=$(curl -s -X POST "$BASE_URL/api/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' | jq -r '.token // empty')

  if [ -n "$TOKEN" ]; then
    curl -s -X POST "$BASE_URL/api/users" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN" \
      -d '{"username":"deployer","password":"deploy123","role":"deployer","permitted_projects":"thinkphp-web,webman-api"}' > /dev/null
    curl -s -X POST "$BASE_URL/api/users" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $TOKEN" \
      -d '{"username":"viewer","password":"view123","role":"viewer","permitted_projects":"thinkphp-web"}' > /dev/null
    success "演示用户创建完成 (deployer / viewer)"
  else
    warn "后端未就绪，跳过用户创建（可稍后运行 bash scripts/demo.sh seed）"
  fi

  # 生成日志文件
  info "生成历史部署日志文件..."
  bash "$REPO_ROOT/scripts/gen_demo_logs.sh"
}

# ============================================================
# 子命令：start — 启动后端
# ============================================================
start_backend() {
  if curl -s "$BASE_URL/api/login" > /dev/null 2>&1; then
    warn "后端已在运行，跳过启动"
    return
  fi
  info "启动 GoDeployer 后端..."
  cd "$REPO_ROOT"
  nohup go run main.go --config="$CONFIG" > /tmp/godeployer_demo.log 2>&1 &
  echo $! > "$PID_FILE"
  _wait_backend
  success "后端已启动 → $BASE_URL"
}

_wait_backend() {
  local n=0
  while ! curl -s "$BASE_URL/api/login" > /dev/null 2>&1; do
    sleep 1
    n=$((n+1))
    if [ $n -ge 15 ]; then
      warn "后端启动超时，请检查 /tmp/godeployer_demo.log"
      return
    fi
  done
}

# ============================================================
# 子命令：stop — 停止后端
# ============================================================
stop_backend() {
  if [ -f "$PID_FILE" ]; then
    kill "$(cat $PID_FILE)" 2>/dev/null || true
    rm -f "$PID_FILE"
  fi
  pkill -f "go run main.go" 2>/dev/null || true
  success "后端已停止"
}

# ============================================================
# 子命令：status — 状态检查
# ============================================================
show_status() {
  echo ""
  echo -e "${BLUE}=== GoDeployer Demo 状态 ===${NC}"
  if curl -s "$BASE_URL/api/login" > /dev/null 2>&1; then
    success "后端运行中 → $BASE_URL"
  else
    warn "后端未运行"
  fi

  if [ -f "$DB_PATH" ]; then
    local cnt
    cnt=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM deploy_tasks WHERE project_id IN ('thinkphp-web','webman-api','crmeb-shop')" 2>/dev/null || echo 0)
    success "数据库存在，演示任务: $cnt 条"
  else
    warn "数据库不存在"
  fi

  if [ -d "$GITEE_WORKSPACE/think" ]; then
    success "Git 仓库已克隆 (think / webman / CRMEB)"
  else
    warn "Git 仓库尚未克隆"
  fi

  echo ""
  echo "  演示账号："
  echo "    admin    / admin123  → 全部项目"
  echo "    deployer / deploy123 → ThinkPHP + Webman"
  echo "    viewer   / view123   → ThinkPHP（只读）"
  echo ""
  echo "  前端：http://localhost:5173  (cd web && npm run dev)"
  echo ""
}

# ============================================================
# 子命令：verify — 验证演示数据与接口
# ============================================================
verify() {
  bash "$REPO_ROOT/scripts/verify_demo.sh"
  bash "$REPO_ROOT/scripts/verify_diff.sh"
}

# ============================================================
# 主流程
# ============================================================
CMD="${1:-all}"

case "$CMD" in
  all)
    echo ""
    echo -e "${BLUE}╔══════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║   GoDeployer Demo 完整初始化             ║${NC}"
    echo -e "${BLUE}╚══════════════════════════════════════════╝${NC}"
    echo ""
    check_deps
    clone_repos
    start_backend
    seed_db
    show_status
    echo -e "${GREEN}✅ Demo 环境就绪！打开 http://localhost:5173 开始验证${NC}"
    ;;
  seed)
    start_backend
    seed_db
    success "演示数据重置完成"
    ;;
  start)
    start_backend
    show_status
    ;;
  stop)
    stop_backend
    ;;
  status)
    show_status
    ;;
  verify)
    verify
    ;;
  clone)
    clone_repos
    ;;
  *)
    echo "用法: bash scripts/demo.sh [all|seed|start|stop|status|verify|clone]"
    echo ""
    echo "  all     完整初始化：克隆仓库 + 启动后端 + 写入演示数据"
    echo "  seed    仅重置数据库演示数据（保留 git 仓库）"
    echo "  start   仅启动后端服务"
    echo "  stop    停止后端服务"
    echo "  status  查看当前服务状态"
    echo "  verify  验证接口与演示数据完整性"
    echo "  clone   仅克隆/更新 Gitee 演示仓库"
    ;;
esac
