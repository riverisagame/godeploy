#!/bin/bash
# =============================================
# GoDeployer Demo 数据生成脚本
# 使用 Gitee 真实 PHP 项目 commit 数据
# =============================================
set -e

BASE="http://localhost:8080"
DB="/mnt/d/claudeprj/deploy/demo_deployer.db"

echo "========================================"
echo "  GoDeployer PHP 项目 Demo 数据生成"
echo "========================================"

# 检查后端是否运行
if ! curl -s "$BASE/api/login" > /dev/null 2>&1; then
  echo "错误：后端服务未运行，请先启动服务"
  exit 1
fi

# ---- 获取 admin token ----
echo ""
echo "[步骤 1/5] 获取管理员 Token..."
LOGIN_RESP=$(curl -s -X POST "$BASE/api/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')
TOKEN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
echo "  Token 获取成功: ${TOKEN:0:30}..."

# ---- 创建测试用户 ----
echo ""
echo "[步骤 2/5] 创建演示用户..."

# deployer 用户：可访问 thinkphp 和 webman，不能访问 CRMEB
curl -s -X POST "$BASE/api/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"username":"deployer","password":"deploy123","role":"deployer","permitted_projects":"thinkphp-web,webman-api"}' | python3 -c "import sys,json; r=json.load(sys.stdin); print(f'  deployer 用户: {r}')"

# viewer 用户：只读，仅能看 thinkphp
curl -s -X POST "$BASE/api/users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"username":"viewer","password":"view123","role":"viewer","permitted_projects":"thinkphp-web"}' | python3 -c "import sys,json; r=json.load(sys.stdin); print(f'  viewer  用户: {r}')"

echo "  用户创建完成"

# ---- 插入历史部署任务（使用真实 commit hash）----
echo ""
echo "[步骤 3/5] 插入历史部署记录..."

NOW=$(date +%s)

# 构建 config snapshots
TP_SNAPSHOT='{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}'
WM_SNAPSHOT='{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}'
CR_SNAPSHOT='{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}'

sqlite3 "$DB" <<'ENDSQL'
-- ============================================================
-- 清理旧的 demo 任务（保留真实数据）
-- ============================================================
DELETE FROM deploy_logs WHERE task_id IN (
  SELECT id FROM deploy_tasks WHERE project_id IN ('backend-api','frontend-web')
);
DELETE FROM deploy_tasks WHERE project_id IN ('backend-api','frontend-web');

-- ============================================================
-- ThinkPHP 项目部署历史（5条记录 - 使用真实 Gitee commits）
-- ============================================================

-- Task 1: ThinkPHP 首次上线 production (30天前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'thinkphp-web', 'production',
  '43983ad1d0b25506c6792a7444ae6f22863359fd',
  'success', '20260428100000', 1, 'admin',
  '{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',
  datetime('now', '-30 days')
);

-- Task 2: ThinkPHP 更新 production (20天前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'thinkphp-web', 'production',
  'c7c11f62f10258b9d9ad2aea1c2a62eda8b2531f',
  'success', '20260508100000', 2, 'deployer',
  '{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',
  datetime('now', '-20 days')
);

-- Task 3: ThinkPHP staging 测试 (10天前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'thinkphp-web', 'staging',
  '7cc4119dcaab2f72606d46eafd16582e887b5d3e',
  'success', '20260518090000', 2, 'deployer',
  '{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',
  datetime('now', '-10 days')
);

-- Task 4: ThinkPHP 升级filesystem (5天前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'thinkphp-web', 'production',
  '98d8c5e09712042a51a2a79622bbce422b48c0ea',
  'success', '20260523143000', 1, 'admin',
  '{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',
  datetime('now', '-5 days')
);

-- Task 5: ThinkPHP 最新配置参数 (1天前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'thinkphp-web', 'production',
  '49917ae7c0de3c7c12de367b99b671321c3c304c',
  'success', '20260527113000', 2, 'deployer',
  '{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',
  datetime('now', '-1 days')
);

-- ============================================================
-- Webman 项目部署历史（4条记录）
-- ============================================================

-- Task 6: webman 首次部署 production (15天前) - 失败（SSH超时）
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'webman-api', 'production',
  'fbd7377964d6a2f9b8fd6b4bbd543a30c28d13af',
  'failed', '20260513160000', 1, 'admin',
  '{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',
  datetime('now', '-15 days')
);

-- Task 7: webman 修复后重新部署 (12天前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'webman-api', 'production',
  'e8cedb15979e7804d9b7bf72128b15ebdb538d75',
  'success', '20260516100000', 1, 'admin',
  '{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',
  datetime('now', '-12 days')
);

-- Task 8: webman 更新控制器 (8天前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'webman-api', 'production',
  '9216f9c05ba6e6cc54aeed6476d46b0c12419295',
  'success', '20260520143000', 2, 'deployer',
  '{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',
  datetime('now', '-8 days')
);

-- Task 9: webman Docker支持 staging (3天前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'webman-api', 'staging',
  '99c2aafc555521c6be37edb03f1d4704ca9c2818',
  'success', '20260525091500', 2, 'deployer',
  '{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',
  datetime('now', '-3 days')
);

-- ============================================================
-- CRMEB 商城系统部署历史（5条记录）
-- ============================================================

-- Task 10: CRMEB 修复队列问题 test (8天前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'crmeb-shop', 'test',
  '779733627341fdb56e2c2291f695c18d3de52eaf',
  'success', '20260520150000', 1, 'admin',
  '{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',
  datetime('now', '-8 days')
);

-- Task 11: CRMEB 紧急上生产 (6天前) - 失败（权限问题）
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'crmeb-shop', 'production',
  '1953370289f0c6bd5a7e07ddf52bd57a6dd233ac',
  'failed', '20260522193000', 1, 'admin',
  '{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',
  datetime('now', '-6 days')
);

-- Task 12: CRMEB 重构 adminapi test (3天前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'crmeb-shop', 'test',
  '50ac735ac375be46ef9eabfca9c46eaae16779d0',
  'success', '20260525103000', 1, 'admin',
  '{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',
  datetime('now', '-3 days')
);

-- Task 13: CRMEB Merge master production (2天前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'crmeb-shop', 'production',
  '60b594745f50fe3e9a4ffc08ca8ea1f02001742a',
  'success', '20260526143000', 1, 'admin',
  '{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',
  datetime('now', '-2 days')
);

-- Task 14: CRMEB 最新优化配置框宽度 production (1小时前) - 成功
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at)
VALUES (
  'crmeb-shop', 'production',
  'af0de843746289693f01ac7fe08a3bacdf862137',
  'success', '20260528211000', 1, 'admin',
  '{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',
  datetime('now', '-1 hours')
);

-- ============================================================
-- 为每个任务插入真实部署日志
-- ============================================================

-- Task 1 日志（ThinkPHP 首次成功）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into thinkphp-web/20260428100000...
remote: Enumerating objects: 127, done.
remote: Counting objects: 100% (127/127), done.
remote: Total 127 (delta 0), reused 127 (delta 0)
Receiving objects: 100%, done.', 'success', datetime('now', '-30 days') FROM deploy_tasks WHERE release_name='20260428100000';

INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at 43983ad1 增加 auto_detect_browser配置参数', 'success', datetime('now', '-30 days', '+3 seconds') FROM deploy_tasks WHERE release_name='20260428100000';

INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sending incremental file list
./
app/
app/BaseController.php
app/controller/
app/controller/Index.php
sent 8,452 bytes  received 87 bytes  17,078.00 bytes/sec
total size is 42,318  speedup is 4.96', 'success', datetime('now', '-30 days', '+8 seconds') FROM deploy_tasks WHERE release_name='20260428100000';

INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Switching symlink: current -> releases/20260428100000
Deployment completed successfully!', 'success', datetime('now', '-30 days', '+10 seconds') FROM deploy_tasks WHERE release_name='20260428100000';

-- Task 2 日志（ThinkPHP 更新）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into thinkphp-web/20260508100000...
remote: Total 138 (delta 0), reused 138 (delta 0)
Receiving objects: 100%, done.', 'success', datetime('now', '-20 days') FROM deploy_tasks WHERE release_name='20260508100000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at c7c11f62 Merge branch 8.x', 'success', datetime('now', '-20 days', '+2 seconds') FROM deploy_tasks WHERE release_name='20260508100000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sent 9,124 bytes  received 91 bytes  18,430.00 bytes/sec', 'success', datetime('now', '-20 days', '+6 seconds') FROM deploy_tasks WHERE release_name='20260508100000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Switching symlink: current -> releases/20260508100000
Deployment completed successfully!', 'success', datetime('now', '-20 days', '+8 seconds') FROM deploy_tasks WHERE release_name='20260508100000';

-- Task 3 日志（ThinkPHP staging）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into thinkphp-web/20260518090000...
Receiving objects: 100%, done.', 'success', datetime('now', '-10 days') FROM deploy_tasks WHERE release_name='20260518090000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at 7cc4119d readem调整', 'success', datetime('now', '-10 days', '+2 seconds') FROM deploy_tasks WHERE release_name='20260518090000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sent 7,890 bytes  received 78 bytes  15,936.00 bytes/sec', 'success', datetime('now', '-10 days', '+5 seconds') FROM deploy_tasks WHERE release_name='20260518090000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Deployment completed successfully!', 'success', datetime('now', '-10 days', '+7 seconds') FROM deploy_tasks WHERE release_name='20260518090000';

-- Task 4 日志（ThinkPHP filesystem）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into thinkphp-web/20260523143000...
Receiving objects: 100%, done.', 'success', datetime('now', '-5 days') FROM deploy_tasks WHERE release_name='20260523143000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at 98d8c5e0 增加 topthink/think-filesystem:3.0', 'success', datetime('now', '-5 days', '+3 seconds') FROM deploy_tasks WHERE release_name='20260523143000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sent 10,234 bytes  received 102 bytes  20,672.00 bytes/sec', 'success', datetime('now', '-5 days', '+8 seconds') FROM deploy_tasks WHERE release_name='20260523143000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Deployment completed successfully!', 'success', datetime('now', '-5 days', '+10 seconds') FROM deploy_tasks WHERE release_name='20260523143000';

-- Task 5 日志（ThinkPHP 最新）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into thinkphp-web/20260527113000...
Receiving objects: 100%, done.', 'success', datetime('now', '-1 days') FROM deploy_tasks WHERE release_name='20260527113000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at 49917ae7 增加配置参数 修改依赖', 'success', datetime('now', '-1 days', '+2 seconds') FROM deploy_tasks WHERE release_name='20260527113000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sent 11,456 bytes  received 112 bytes  22,136.00 bytes/sec', 'success', datetime('now', '-1 days', '+7 seconds') FROM deploy_tasks WHERE release_name='20260527113000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Deployment completed successfully!', 'success', datetime('now', '-1 days', '+9 seconds') FROM deploy_tasks WHERE release_name='20260527113000';

-- Task 6 日志（webman 失败）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into webman-api/20260513160000...
Receiving objects: 100%, done.', 'success', datetime('now', '-15 days') FROM deploy_tasks WHERE release_name='20260513160000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at fbd73779 wellcome', 'success', datetime('now', '-15 days', '+2 seconds') FROM deploy_tasks WHERE release_name='20260513160000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'Error: Rsync failed on 127.0.0.1:
ssh: connect to host 127.0.0.1 port 22: Connection refused
rsync: [sender] write error: Broken pipe (32)
rsync error: error in rsync protocol data stream (code 12)', 'failed', datetime('now', '-15 days', '+5 seconds') FROM deploy_tasks WHERE release_name='20260513160000';

-- Task 7 日志（webman 修复后成功）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into webman-api/20260516100000...
Receiving objects: 100%, done.', 'success', datetime('now', '-12 days') FROM deploy_tasks WHERE release_name='20260516100000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at e8cedb15 Update IndexController.php', 'success', datetime('now', '-12 days', '+2 seconds') FROM deploy_tasks WHERE release_name='20260516100000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sent 15,234 bytes  received 152 bytes  30,772.00 bytes/sec
total size is 98,432  speedup is 6.43', 'success', datetime('now', '-12 days', '+9 seconds') FROM deploy_tasks WHERE release_name='20260516100000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Deployment completed successfully!', 'success', datetime('now', '-12 days', '+11 seconds') FROM deploy_tasks WHERE release_name='20260516100000';

-- Task 8 日志（webman 控制器更新）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into webman-api/20260520143000...
Receiving objects: 100%, done.', 'success', datetime('now', '-8 days') FROM deploy_tasks WHERE release_name='20260520143000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at 9216f9c0 Update IndexController.php', 'success', datetime('now', '-8 days', '+2 seconds') FROM deploy_tasks WHERE release_name='20260520143000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sent 16,789 bytes  received 168 bytes  33,914.00 bytes/sec', 'success', datetime('now', '-8 days', '+7 seconds') FROM deploy_tasks WHERE release_name='20260520143000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Deployment completed successfully!', 'success', datetime('now', '-8 days', '+9 seconds') FROM deploy_tasks WHERE release_name='20260520143000';

-- Task 9 日志（webman Docker staging）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into webman-api/20260525091500...
Receiving objects: 100%, done.', 'success', datetime('now', '-3 days') FROM deploy_tasks WHERE release_name='20260525091500';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at 99c2aafc Add Docker support for webman deployment', 'success', datetime('now', '-3 days', '+2 seconds') FROM deploy_tasks WHERE release_name='20260525091500';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sent 23,456 bytes  received 234 bytes  47,380.00 bytes/sec', 'success', datetime('now', '-3 days', '+8 seconds') FROM deploy_tasks WHERE release_name='20260525091500';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Deployment completed successfully!', 'success', datetime('now', '-3 days', '+10 seconds') FROM deploy_tasks WHERE release_name='20260525091500';

-- Task 10 日志（CRMEB 修复队列）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into crmeb-shop/20260520150000...
remote: Enumerating objects: 52847, done.
remote: Total 52847 (delta 0)
Receiving objects: 100%, done.', 'success', datetime('now', '-8 days') FROM deploy_tasks WHERE release_name='20260520150000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at 77973362 fix(queue): adjust queue service check logic', 'success', datetime('now', '-8 days', '+8 seconds') FROM deploy_tasks WHERE release_name='20260520150000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sent 1,234,567 bytes  received 12,345 bytes
total size is 187,432,456  speedup is 149.68', 'success', datetime('now', '-8 days', '+45 seconds') FROM deploy_tasks WHERE release_name='20260520150000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Deployment completed successfully!', 'success', datetime('now', '-8 days', '+47 seconds') FROM deploy_tasks WHERE release_name='20260520150000';

-- Task 11 日志（CRMEB 失败-权限）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into crmeb-shop/20260522193000...
remote: Total 52848 (delta 0)
Receiving objects: 100%, done.', 'success', datetime('now', '-6 days') FROM deploy_tasks WHERE release_name='20260522193000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at 19533702 更新', 'success', datetime('now', '-6 days', '+5 seconds') FROM deploy_tasks WHERE release_name='20260522193000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'Error: Rsync failed on 127.0.0.1:
rsync: [Receiver] mkstemp "/tmp/demo_deploy/crmeb/production/releases/20260522193000/public/.index.html.XXXXXX" failed: Permission denied (13)
rsync error: some files/attrs were not transferred (see previous errors) (code 23)', 'failed', datetime('now', '-6 days', '+30 seconds') FROM deploy_tasks WHERE release_name='20260522193000';

-- Task 12 日志（CRMEB refactor test）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into crmeb-shop/20260525103000...
Receiving objects: 100%, done.', 'success', datetime('now', '-3 days') FROM deploy_tasks WHERE release_name='20260525103000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at 50ac735a refactor(adminapi/setting): 重构系统配置相关接口逻辑', 'success', datetime('now', '-3 days', '+5 seconds') FROM deploy_tasks WHERE release_name='20260525103000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sent 2,345,678 bytes  received 23,456 bytes
total size is 187,543,210  speedup is 78.32', 'success', datetime('now', '-3 days', '+55 seconds') FROM deploy_tasks WHERE release_name='20260525103000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Deployment completed successfully!', 'success', datetime('now', '-3 days', '+57 seconds') FROM deploy_tasks WHERE release_name='20260525103000';

-- Task 13 日志（CRMEB Merge production）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into crmeb-shop/20260526143000...
Receiving objects: 100%, done.', 'success', datetime('now', '-2 days') FROM deploy_tasks WHERE release_name='20260526143000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at 60b59474 Merge branch master of https://gitee.com/ZhongBangKeJi/CRMEB', 'success', datetime('now', '-2 days', '+5 seconds') FROM deploy_tasks WHERE release_name='20260526143000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sent 3,123,456 bytes  received 31,234 bytes
total size is 187,601,234  speedup is 58.94', 'success', datetime('now', '-2 days', '+62 seconds') FROM deploy_tasks WHERE release_name='20260526143000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Deployment completed successfully!', 'success', datetime('now', '-2 days', '+64 seconds') FROM deploy_tasks WHERE release_name='20260526143000';

-- Task 14 日志（CRMEB 最新 1小时前）
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Git Clone', 'Cloning into crmeb-shop/20260528211000...
Receiving objects: 100%, done.', 'success', datetime('now', '-1 hours') FROM deploy_tasks WHERE release_name='20260528211000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Checkout', 'HEAD is now at af0de843 优化自定义配置数字输入框宽度', 'success', datetime('now', '-1 hours', '+5 seconds') FROM deploy_tasks WHERE release_name='20260528211000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Rsync', 'sent 1,890,234 bytes  received 18,902 bytes
total size is 187,612,345  speedup is 98.10', 'success', datetime('now', '-1 hours', '+48 seconds') FROM deploy_tasks WHERE release_name='20260528211000';
INSERT INTO deploy_logs (task_id, step, output, status, created_at) SELECT id, 'Symlink', 'Deployment completed successfully!', 'success', datetime('now', '-1 hours', '+50 seconds') FROM deploy_tasks WHERE release_name='20260528211000';

ENDSQL

echo "  14 条历史任务 + 对应日志写入完成"

# ---- 查询验证 ----
echo ""
echo "[步骤 4/5] 验证数据..."
echo "  当前用户列表："
curl -s "$BASE/api/users" \
  -H "Authorization: Bearer $TOKEN" | \
  python3 -c "
import sys, json
users = json.load(sys.stdin)
for u in users:
    print(f'    [{u[\"role\"]:10s}] {u[\"username\"]:12s} 可访问: {u[\"permitted_projects\"]}')
"

echo ""
echo "  部署任务统计："
sqlite3 "$DB" "
SELECT project_id, env_id, status, COUNT(*) as cnt
FROM deploy_tasks
WHERE project_id IN ('thinkphp-web','webman-api','crmeb-shop')
GROUP BY project_id, env_id, status
ORDER BY project_id, env_id;
"

echo ""
echo "[步骤 5/5] 生成完成摘要"
echo ""
echo "========================================"
echo "  演示账号"
echo "========================================"
echo "  admin    / admin123  -> 可访问: 所有项目（管理员）"
echo "  deployer / deploy123 -> 可访问: thinkphp-web, webman-api"
echo "  viewer   / view123   -> 可访问: thinkphp-web（只读）"
echo ""
echo "========================================"
echo "  演示项目 & 历史数据"
echo "========================================"
echo "  ThinkPHP 后端框架  - 5次部署（全部成功）"
echo "  Webman 微服务接口  - 4次部署（1次失败SSH超时，3次成功）"
echo "  CRMEB 商城系统     - 5次部署（1次失败权限问题，4次成功）"
echo ""
echo "  访问地址: http://localhost:5173"
echo "========================================"
