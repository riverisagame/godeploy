#!/bin/bash
# =============================================================
#  GoDeployer Demo 一键启动与极速本地预览脚本
#  // @Ref: docs/sps/plans/20260530_demo_script_optimization_plan.md | @Date: 2026-05-30
#
#  用途：
#    一键本地部署、快速预览、开发验证。默认在本地秒级创建符合
#    系统 Diff、分支、Tag 和部署功能验证的 Mock Git 仓库。
#    100% 真实通过 API 接口模拟多用户部署流程，免除伪造数据库数据。
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

# ---- 配置路径 ----
BASE_URL="http://localhost:8080"
DB_PATH="$REPO_ROOT/demo_deployer.db"
LOG_DIR="$REPO_ROOT/demo_logs"
WORKSPACE="$REPO_ROOT/demo_workspace"
GITEE_WORKSPACE="$WORKSPACE/gitee_demo"
CONFIG="demo_config.yaml"
PID_FILE="/tmp/godeployer_demo.pid"

USE_MOCK=true

# ============================================================
# 依赖检查与安装指导
# ============================================================
check_deps() {
  info "检查必要依赖..."
  local missing=()
  for cmd in go git sqlite3 jq curl rsync; do
    if ! command -v "$cmd" &>/dev/null; then
      missing+=("$cmd")
    fi
  done
  
  if [ ${#missing[@]} -gt 0 ]; then
    echo -e "${RED}[ERR] 缺少以下系统依赖: ${missing[*]}${NC}"
    echo ""
    echo -e "${YELLOW}💡 请在 WSL Debian / Ubuntu 环境下运行以下命令进行安装：${NC}"
    echo -e "   ${GREEN}sudo apt-get update && sudo apt-get install -y git sqlite3 jq curl rsync golang-go${NC}"
    echo ""
    exit 1
  fi
  success "所有系统依赖检查通过"
}

# ============================================================
# 本地极速创建 Mock 演示仓库 (含 Branch, Tag, 多次 Commit 演进)
# ============================================================
create_mock_repos() {
  info "正在秒级生成本地 Mock Git 演示仓库..."
  mkdir -p "$GITEE_WORKSPACE"

  _generate_mock_repo() {
    local name="$1"
    local repo_dir="$GITEE_WORKSPACE/$name"
    
    if [ -d "$repo_dir/.git" ]; then
      info "$name Mock 仓库已存在，跳过生成"
      return
    fi
    
    info "生成 Mock 仓库: $name..."
    mkdir -p "$repo_dir"
    cd "$repo_dir"
    git init -q
    
    git config user.name "Demo Robot"
    git config user.email "robot@godeploy.demo"
    
    # 1. 提交 master 初始版本 (Tag: v1.0.0)
    echo -e "<?php\n// $name 初始版本\necho 'Hello $name v1.0.0';" > index.php
    git add index.php
    git commit -q -m "feat: init $name project structure"
    git tag v1.0.0
    
    # 2. 提交 master 升级版本 (Tag: v1.0.1)
    echo -e "<?php\n// $name 初始版本\necho 'Hello $name v1.0.1';\nfunction core() { return 'core_ok'; }" > index.php
    git commit -q -a -m "fix: resolve core path check issue"
    git tag v1.0.1

    # 3. 提交新特性 (Tag: v1.1.0-beta)
    echo -e "<?php\n// $name v1.1.0-beta\necho 'Hello $name v1.1.0';\nfunction core() { return 'core_ok'; }\nfunction newFeature() { return 'new_feature_ok'; }" > index.php
    git commit -q -a -m "feat: add new API endpoints for v1.1.0"
    git tag v1.1.0-beta
    
    # 4. 提交优化
    echo -e "<?php\n// $name v1.1.0\necho 'Hello $name v1.1.0';\n// Added security middleware\nfunction core() { return 'core_ok'; }" > index.php
    git commit -q -a -m "perf: optimize middleware query performance"
    
    # 5. 提交最终版
    echo -e "<?php\n// $name v1.1.0 Final\necho 'Hello $name v1.1.0';\nfunction core() { return 'core_ok'; }\n// End" > index.php
    git commit -q -a -m "docs: update API usage comments"
    
    # 拉出 develop 分支以拥有相同的提交历史并确保 refs 被 bare cache 全量克隆
    git branch -f develop master
    
    # 回到 master 默认分支
    git checkout -q master
    cd "$REPO_ROOT"
  }

  _generate_mock_repo "think"
  _generate_mock_repo "webman"
  _generate_mock_repo "CRMEB"
  
  # 动态修改配置文件中指向的本地 repo 绝对路径
  for name in think webman CRMEB; do
    local file_name=""
    [ "$name" = "think" ] && file_name="thinkphp.yaml"
    [ "$name" = "webman" ] && file_name="webman.yaml"
    [ "$name" = "CRMEB" ] && file_name="crmeb.yaml"
    sed -i "s|repo:.*|repo: \"./demo_workspace/gitee_demo/$name\"|g" "$REPO_ROOT/demo_projects.d/$file_name"
  done
  
  success "所有本地 Mock 演示仓库已与项目配置文件关联"
}

# ============================================================
# 克隆 Gitee 真实仓库 (高级选项)
# ============================================================
clone_real_repos() {
  info "正在从 Gitee 克隆真实演示仓库 (耗时较长，请确保网络畅通)..."
  mkdir -p "$GITEE_WORKSPACE"

  _clone_one() {
    local name="$1" url="$2" depth="$3"
    if [ ! -d "$GITEE_WORKSPACE/$name/.git" ]; then
      info "克隆 $name..."
      if ! git clone --depth="$depth" "$url" "$GITEE_WORKSPACE/$name" 2>&1 | tail -3; then
        warn "$name 克隆失败，将切换为本地 Mock 生成"
        USE_MOCK=true
        create_mock_repos
        return
      fi
    else
      info "$name 仓库已存在"
    fi
  }

  _clone_one "think"  "https://gitee.com/top-think/think.git" 100
  _clone_one "webman" "https://gitee.com/walkor/webman.git" 100
  _clone_one "CRMEB"  "https://gitee.com/ZhongBangKeJi/CRMEB.git" 20
  
  # 动态修改配置文件中指向的本地 repo 绝对路径
  for name in think webman CRMEB; do
    local file_name=""
    [ "$name" = "think" ] && file_name="thinkphp.yaml"
    [ "$name" = "webman" ] && file_name="webman.yaml"
    [ "$name" = "CRMEB" ] && file_name="crmeb.yaml"
    sed -i "s|repo:.*|repo: \"./demo_workspace/gitee_demo/$name\"|g" "$REPO_ROOT/demo_projects.d/$file_name"
  done
  success "真实演示仓库拉取并关联就绪"
}

# ============================================================
# 模拟真实用户调 API 发起部署与用户注册
# ============================================================
seed_db() {
  info "正在通过 API 接口模拟真实多用户操作产生数据..."
  mkdir -p "$LOG_DIR"

  LOCK_FILE="/tmp/godeployer_demo_seed.lock"
  exec 200>"$LOCK_FILE"
  if ! flock -n 200; then
    error "另一个 seed 进程正在运行，请稍后重试"
  fi
  trap "flock -u 200 2>/dev/null; rm -f $LOCK_FILE" EXIT

  # 清空旧任务数据库与本地缓存
  info "清空旧的任务部署记录、本地日志与 Git 缓存..."
  rm -f "$LOG_DIR"/task_*.log 2>/dev/null || true
  rm -rf "$WORKSPACE"/.cache 2>/dev/null || true

  if [ -f "$DB_PATH" ]; then
    sqlite3 "$DB_PATH" "DELETE FROM deploy_tasks;"
  fi

  # 1. 确保服务在线以接收 API 请求
  _wait_backend

  # 2. 获取管理员 Token 并初始化其他演示账号
  local ADMIN_TOKEN
  ADMIN_TOKEN=$(curl -s -X POST "$BASE_URL/api/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' | jq -r '.token // empty')

  if [ -z "$ADMIN_TOKEN" ]; then
    error "管理员账号登录失败，后端未就绪！"
  fi

  _ensure_user() {
    local uname="$1" pwd="$2" role="$3" projects="$4"
    curl -s -X POST "$BASE_URL/api/users" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $ADMIN_TOKEN" \
      -d "{\"username\":\"$uname\",\"password\":\"$pwd\",\"role\":\"$role\",\"permitted_projects\":\"$projects\"}" > /dev/null
  }

  _ensure_user "deployer" "deploy123" "deployer" "thinkphp-web,webman-api"
  _ensure_user "viewer"   "view123"   "viewer"   "thinkphp-web"

  # 3. 获取其他用户的 Token
  local DEPLOYER_TOKEN
  DEPLOYER_TOKEN=$(curl -s -X POST "$BASE_URL/api/login" \
    -H "Content-Type: application/json" \
    -d '{"username":"deployer","password":"deploy123"}' | jq -r '.token // empty')

  # 4. 获取 Git Mock 仓库的真实哈希
  _get_commits() {
    local name="$1"
    git -C "$GITEE_WORKSPACE/$name" log --all --format="%H" | head -n 5
  }
  THINK_COMMITS=($(_get_commits "think"))
  WEBMAN_COMMITS=($(_get_commits "webman"))
  CRMEB_COMMITS=($(_get_commits "CRMEB"))

  # 5. 用 API 发起部署的执行辅助函数
  deploy_task_via_api() {
    local token="$1"
    local project_id="$2"
    local env_id="$3"
    local commit_id="$4"
    local username="$5"
    
    local task_resp
    task_resp=$(curl -s -X POST "$BASE_URL/api/tasks" \
      -H "Content-Type: application/json" \
      -H "Authorization: Bearer $token" \
      -d "{\"project_id\":\"$project_id\",\"env_id\":\"$env_id\",\"commit_id\":\"$commit_id\"}")
    
    local task_id
    task_id=$(echo "$task_resp" | jq -r '.id // empty')
    
    if [ -z "$task_id" ]; then
      echo -e "  [API拒绝] 用户 $username 部署 $project_id ($env_id) $commit_id - 权限不足 (预期 403)"
      return
    fi
    
    echo -n "  [API发起] 任务 $task_id: 用户 $username 部署 $project_id ($env_id)..."
    # 轮询状态直到部署引擎运行完毕
    while true; do
      local status
      status=$(curl -s -H "Authorization: Bearer $token" "$BASE_URL/api/tasks/$task_id" | jq -r '.status // empty')
      if [ "$status" != "pending" ] && [ "$status" != "deploying" ]; then
        if [ "$status" = "success" ]; then
          echo -e " [${GREEN}成功${NC}]"
        else
          echo -e " [${RED}失败: $status${NC}]"
        fi
        break
      fi
      sleep 0.3
    done
  }

  # 6. 交错执行多用户真实部署 API
  info "开始按步骤模拟用户触发真实部署流水线："
  
  # ThinkPHP (管理员部署生产环境，部署员部署测试环境)
  deploy_task_via_api "$ADMIN_TOKEN" "thinkphp-web" "production" "${THINK_COMMITS[0]}" "admin"
  deploy_task_via_api "$DEPLOYER_TOKEN" "thinkphp-web" "staging" "${THINK_COMMITS[1]}" "deployer"
  deploy_task_via_api "$ADMIN_TOKEN" "thinkphp-web" "production" "${THINK_COMMITS[2]}" "admin"

  # Webman (部署员部署生产环境 - 由于 /root 权限不足而真实失败，管理员修复后再次部署成功)
  deploy_task_via_api "$DEPLOYER_TOKEN" "webman-api" "production" "${WEBMAN_COMMITS[0]}" "deployer" # 预期失败
  deploy_task_via_api "$ADMIN_TOKEN" "webman-api" "staging" "${WEBMAN_COMMITS[1]}" "admin"
  deploy_task_via_api "$ADMIN_TOKEN" "webman-api" "production" "${WEBMAN_COMMITS[2]}" "admin" # 若上面路径错误，这里也是失败或成功的真实反馈

  # CRMEB (模拟越权操作 - 部署员无权部署 CRMEB，返回 403 拒绝；管理员部署生产环境 - 由于端口 2223 无法连通而真实失败)
  deploy_task_via_api "$DEPLOYER_TOKEN" "crmeb-shop" "production" "${CRMEB_COMMITS[0]}" "deployer" # 预期 403 拒绝
  deploy_task_via_api "$ADMIN_TOKEN" "crmeb-shop" "production" "${CRMEB_COMMITS[1]}" "admin" # 预期由于2223端口失败
  deploy_task_via_api "$ADMIN_TOKEN" "crmeb-shop" "test" "${CRMEB_COMMITS[2]}" "admin" # 预期成功

  # 为演示产生更多丰富的历史记录 (增加 30+ 条)
  info "正在生成更多真实部署记录以丰富历史列表..."
  for i in {1..7}; do
    deploy_task_via_api "$ADMIN_TOKEN" "thinkphp-web" "staging" "${THINK_COMMITS[$((i % 3))]}" "admin"
    deploy_task_via_api "$DEPLOYER_TOKEN" "thinkphp-web" "production" "${THINK_COMMITS[$(((i+1) % 3))]}" "deployer"
    deploy_task_via_api "$ADMIN_TOKEN" "webman-api" "staging" "${WEBMAN_COMMITS[$((i % 3))]}" "admin"
    deploy_task_via_api "$ADMIN_TOKEN" "crmeb-shop" "test" "${CRMEB_COMMITS[$((i % 3))]}" "admin"
  done

  success "通过 API 触发的 100% 真实部署任务已执行完毕！"
}

# ============================================================
# 启动后端程序并自动构建/嵌入前端
# ============================================================
start_backend() {
  pkill -f "godeployer_demo_bin" 2>/dev/null || true
  pkill -f "go run main.go" 2>/dev/null || true
  
  if lsof -i :8080 -t &>/dev/null; then
    warn "检测到 8080 端口已被占用，正在清理冲突的旧服务..."
    local conflicts
    conflicts=$(lsof -i :8080 -t)
    kill -9 $conflicts 2>/dev/null || true
    sleep 1
  fi

  # ---- 探测可用的构建方式 ----
  local win_root="" win_cmd=""
  if [ -n "$WSL_DISTRO_NAME" ]; then
    win_root=$(wslpath -w "$REPO_ROOT" 2>/dev/null)
    # WSL interop 需要用完整路径调用 Windows 侧 cmd.exe
    WIN_CMD="/mnt/c/Windows/System32/cmd.exe"
    [ -f "/mnt/c/Windows/SysWOW64/cmd.exe" ] && WIN_CMD="/mnt/c/Windows/SysWOW64/cmd.exe"
    [ -f "/mnt/c/Windows/System32/cmd.exe" ] && WIN_CMD="/mnt/c/Windows/System32/cmd.exe"
  fi

  # 检测 Windows 侧工具 (比 WSL 内编译快 100+ 倍)
  local use_win_npm=false use_win_go=false
  if [ -n "$win_root" ] && [ -x "$WIN_CMD" ]; then
    $WIN_CMD /c "npm.cmd --version" >/dev/null 2>&1 && use_win_npm=true
    $WIN_CMD /c "go version" >/dev/null 2>&1 && use_win_go=true
  fi

  # ---- 前端构建 ----
  info "正在构建前端产物..."
  if $use_win_npm; then
    info "通过 Windows 原生 npm 构建前端 (极速模式)..."
    # Windows 原生 npm 构建，无 WSL 文件系统开销
    $WIN_CMD /c "cd /d $win_root\\web && npm.cmd run build"
    # copy 回来确保 godeployer/dist 有最新产物
    rm -rf godeployer/dist
    cp -r web/dist godeployer/dist
  elif [ -n "$WSL_DISTRO_NAME" ]; then
    info "检测到 WSL 环境，前端使用原生 /tmp 文件系统加速构建..."
    rm -rf /tmp/godeployer_web_build
    mkdir -p /tmp/godeployer_web_build
    rsync -a --exclude=node_modules --exclude=dist web/ /tmp/godeployer_web_build/
    cd /tmp/godeployer_web_build
    export npm_config_cache="/tmp/npm_cache_demo"
    npm install >/dev/null 2>&1
    npm run build >/dev/null 2>&1
    cd "$REPO_ROOT"
    rm -rf web/dist godeployer/dist
    cp -r /tmp/godeployer_web_build/dist web/dist
    cp -r /tmp/godeployer_web_build/dist godeployer/dist
  else
    if [ ! -d "web/dist" ] || [ ! -d "godeployer/dist" ]; then
      warn "⚠️ 未检测到前端产物 dist，正在本地构建..."
      cd web && npm install && npm run build && cd ..
      rm -rf godeployer/dist
      cp -r web/dist godeployer/dist
    fi
  fi

  # ---- Go 编译 ----
  info "正在构建并启动 GoDeployer 后端服务..."
  cd "$REPO_ROOT"
  rm -f seed_demo.go test_hash.go

  if $use_win_go; then
    info "通过 Windows 原生 Go 交叉编译 Linux 二进制 (极速模式)..."
    # CGO_ENABLED=0 因为 SQLite 使用纯 Go 实现
    $WIN_CMD /c "cd /d $win_root && set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build -o godeployer_demo_bin ."
  elif [ -n "$WSL_DISTRO_NAME" ]; then
    info "检测到 WSL 环境，后端使用 /tmp 依赖缓存加速构建..."
    export GOMODCACHE="/tmp/gopath_demo/pkg/mod"
    export GOCACHE="/tmp/gopath_demo/cache"
    export GOPROXY="https://goproxy.cn,direct"
    mkdir -p "$GOMODCACHE" "$GOCACHE"
    go build -o godeployer_demo_bin .
  else
    go build -o godeployer_demo_bin .
  fi

  nohup ./godeployer_demo_bin --config="$CONFIG" > godeployer_demo.log 2>&1 &
  echo $! > "$PID_FILE"
  
  _wait_backend
  success "后端服务启动成功 → $BASE_URL"
}

_wait_backend() {
  local n=0
  while ! curl -s "$BASE_URL/api/login" > /dev/null 2>&1; do
    sleep 1
    n=$((n+1))
    if [ $n -ge 15 ]; then
      error "后端启动响应超时，请排查日志: /tmp/godeployer_demo.log"
    fi
  done
}

# ============================================================
# 停止后端程序
# ============================================================
stop_backend() {
  if [ -f "$PID_FILE" ]; then
    kill "$(cat $PID_FILE)" 2>/dev/null || true
    rm -f "$PID_FILE"
  fi
  pkill -f "godeployer_demo_bin" 2>/dev/null || true
  pkill -f "go run main.go" 2>/dev/null || true
  rm -f /tmp/godeployer_demo_bin
  success "后端服务已停止"
}

# ============================================================
# 状态与信息展示
# ============================================================
show_status() {
  echo ""
  echo -e "${BLUE}=== GoDeployer 一键演示状态 ===${NC}"
  if curl -s "$BASE_URL/api/login" > /dev/null 2>&1; then
    success "后端核心运行中: $BASE_URL"
  else
    warn "后端核心未运行"
  fi

  if [ -f "$DB_PATH" ]; then
    local cnt
    cnt=$(sqlite3 "$DB_PATH" "SELECT COUNT(*) FROM deploy_tasks" 2>/dev/null || echo 0)
    success "演示数据库存在，当前由 API 真实产生的任务数: $cnt 条"
  else
    warn "演示数据库尚未初始化"
  fi

  echo ""
  echo -e "${YELLOW}🔑 演示系统登录账号：${NC}"
  echo "  1. 管理员 (Admin)   →  用户名: admin    密码: admin123  (拥有一切权限)"
  echo "  2. 运维人员 (Deploy) →  用户名: deployer 密码: deploy123 (仅拥有 ThinkPHP & Webman 的发布权限)"
  echo "  3. 观察员 (Viewer)  →  用户名: viewer   密码: view123   (仅拥有 ThinkPHP 的只读查看权限)"
  echo ""
  echo -e "${YELLOW}🌐 访问方式：${NC}"
  if [ -d "web/dist" ]; then
    echo "  - 已成功内嵌前端！请直接访问：http://localhost:8080"
  else
    echo "  - 前端未内嵌。请手动开启前端端口："
    echo "    cd web && npm run dev"
    echo "    开启后访问：http://localhost:5173"
  fi
  echo ""
}

# ============================================================
# 主入口参数解析
# ============================================================
case "${1:-all}" in
  --real)
    USE_MOCK=false
    check_deps
    clone_real_repos
    start_backend
    seed_db
    show_status
    ;;
  all)
    check_deps
    if [ "$USE_MOCK" = true ]; then
      create_mock_repos
    else
      clone_real_repos
    fi
    start_backend
    seed_db
    show_status
    ;;
  seed)
    seed_db
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
    bash "$REPO_ROOT/scripts/verify_demo.sh"
    bash "$REPO_ROOT/scripts/verify_diff.sh"
    ;;
  clone)
    if [ "$USE_MOCK" = true ]; then
      create_mock_repos
    else
      clone_real_repos
    fi
    ;;
  *)
    echo "用法: bash scripts/demo.sh [all|--real|seed|start|stop|status|verify|clone]"
    echo ""
    echo "  all      默认一键初始化 (使用秒级 Mock 本地仓库, 零网络依赖, 100% API 真实触发)"
    echo "  --real   一键初始化 (从远程 Gitee 克隆真实大型项目)"
    echo "  start    仅启动后端服务"
    echo "  stop     停止后端服务"
    echo "  status   查看服务状态与演示账户"
    echo "  seed     仅通过 API 触发部署重置演示数据"
    echo "  verify   对生成的演示链路与 API 进行完好性校验"
    echo "  clone    仅生成/拉取 Git 演示仓库"
    ;;
esac
