#!/bin/bash
# =============================================
# GoDeployer PHP Demo - 数据库 Seed（仅任务表）
# =============================================

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
DB="$REPO_ROOT/demo_deployer.db"

echo "[步骤 3/5] 插入历史部署记录..."

sqlite3 "$DB" <<'ENDSQL'
-- 清理旧 demo 任务
DELETE FROM deploy_tasks WHERE project_id IN ('thinkphp-web','webman-api','crmeb-shop');

-- ============================================================
-- ThinkPHP 项目部署历史（5条 - 真实 Gitee commits）
-- ============================================================
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('thinkphp-web','production','43983ad1d0b25506c6792a7444ae6f22863359fd','success','20260428100000',1,'admin','{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',datetime('now','-30 days'));

INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('thinkphp-web','production','c7c11f62f10258b9d9ad2aea1c2a62eda8b2531f','success','20260508100000',2,'deployer','{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',datetime('now','-20 days'));

INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('thinkphp-web','staging','7cc4119dcaab2f72606d46eafd16582e887b5d3e','success','20260518090000',2,'deployer','{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',datetime('now','-10 days'));

INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('thinkphp-web','production','98d8c5e09712042a51a2a79622bbce422b48c0ea','success','20260523143000',1,'admin','{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',datetime('now','-5 days'));

INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('thinkphp-web','production','49917ae7c0de3c7c12de367b99b671321c3c304c','success','20260527113000',2,'deployer','{"id":"thinkphp-web","name":"ThinkPHP 后端框架","repo":"https://gitee.com/top-think/think.git"}',datetime('now','-1 days'));

-- ============================================================
-- Webman 项目部署历史（4条）
-- ============================================================
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('webman-api','production','fbd7377964d6a2f9b8fd6b4bbd543a30c28d13af','failed','20260513160000',1,'admin','{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',datetime('now','-15 days'));

INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('webman-api','production','e8cedb15979e7804d9b7bf72128b15ebdb538d75','success','20260516100000',1,'admin','{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',datetime('now','-12 days'));

INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('webman-api','production','9216f9c05ba6e6cc54aeed6476d46b0c12419295','success','20260520143000',2,'deployer','{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',datetime('now','-8 days'));

INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('webman-api','staging','99c2aafc555521c6be37edb03f1d4704ca9c2818','success','20260525091500',2,'deployer','{"id":"webman-api","name":"Webman 微服务接口","repo":"https://gitee.com/walkor/webman.git"}',datetime('now','-3 days'));

-- ============================================================
-- CRMEB 商城系统部署历史（5条）
-- ============================================================
INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('crmeb-shop','test','779733627341fdb56e2c2291f695c18d3de52eaf','success','20260520150000',1,'admin','{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',datetime('now','-8 days'));

INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('crmeb-shop','production','1953370289f0c6bd5a7e07ddf52bd57a6dd233ac','failed','20260522193000',1,'admin','{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',datetime('now','-6 days'));

INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('crmeb-shop','test','50ac735ac375be46ef9eabfca9c46eaae16779d0','success','20260525103000',1,'admin','{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',datetime('now','-3 days'));

INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('crmeb-shop','production','60b594745f50fe3e9a4ffc08ca8ea1f02001742a','success','20260526143000',1,'admin','{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',datetime('now','-2 days'));

INSERT INTO deploy_tasks (project_id, env_id, commit_id, status, release_name, user_id, username, config_snapshot, created_at) VALUES
('crmeb-shop','production','af0de843746289693f01ac7fe08a3bacdf862137','success','20260528211000',1,'admin','{"id":"crmeb-shop","name":"CRMEB 商城系统","repo":"https://gitee.com/ZhongBangKeJi/CRMEB.git"}',datetime('now','-1 hours'));
ENDSQL

echo "  14 条历史任务写入完成"

echo ""
echo "[验证] 当前任务统计："
sqlite3 "$DB" "
SELECT project_id as 项目, env_id as 环境, status as 状态, COUNT(*) as 次数
FROM deploy_tasks
WHERE project_id IN ('thinkphp-web','webman-api','crmeb-shop')
GROUP BY project_id, env_id, status
ORDER BY project_id, env_id;
"
