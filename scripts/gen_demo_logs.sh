#!/bin/bash
# =============================================
# GoDeployer PHP Demo - 日志文件生成
# 为每个历史任务创建对应的 .log 文件
# =============================================

DB="/mnt/d/claudeprj/deploy/demo_deployer.db"
LOG_DIR="/mnt/d/claudeprj/deploy/demo_logs"

mkdir -p "$LOG_DIR"

echo "获取任务 ID 列表..."

# 从数据库获取各任务 ID
declare -A TASK_IDS

while IFS='|' read -r release_name task_id; do
  TASK_IDS["$release_name"]="$task_id"
done < <(sqlite3 "$DB" "SELECT release_name, id FROM deploy_tasks WHERE project_id IN ('thinkphp-web','webman-api','crmeb-shop') ORDER BY id;")

echo "找到以下任务："
for key in "${!TASK_IDS[@]}"; do
  echo "  release=$key id=${TASK_IDS[$key]}"
done

# ---- ThinkPHP 日志 ----

write_log() {
  local task_id="$1"
  local content="$2"
  echo "$content" > "$LOG_DIR/task_${task_id}.log"
  echo "  写入 task_${task_id}.log"
}

ID="${TASK_IDS[20260428100000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-04-28 10:00:00] Step 1: Cloning repository from https://gitee.com/top-think/think.git into thinkphp-web/20260428100000...
Cloning into 'thinkphp-web/20260428100000'...
remote: Enumerating objects: 127, done.
remote: Counting objects: 100% (127/127), done.
remote: Compressing objects: 100% (89/89), done.
remote: Total 127 (delta 0), reused 127 (delta 0), pack-reused 0 (from 0)
Receiving objects: 100% (127/127), 156.23 KiB | 1.23 MiB/s, done.
[2026-04-28 10:00:03] Step 2: Checking out target commit/branch: 43983ad1d0b25506c6792a7444ae6f22863359fd...
HEAD is now at 43983ad1 增加 auto_detect_browser配置参数
[2026-04-28 10:00:04] Step 3: Executing local build hooks...
(no build hooks configured)
[2026-04-28 10:00:04] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sending incremental file list
./
app/
app/BaseController.php
app/ExceptionHandle.php
app/controller/
app/controller/Index.php
config/
config/app.php
sent 8,452 bytes  received 87 bytes  17,078.00 bytes/sec
total size is 42,318  speedup is 4.96
[2026-04-28 10:00:12] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
Switching symlink: current -> releases/20260428100000
[2026-04-28 10:00:13] Deployment completed successfully!"
fi

ID="${TASK_IDS[20260508100000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-08 10:00:00] Step 1: Cloning repository from https://gitee.com/top-think/think.git into thinkphp-web/20260508100000...
remote: Total 138 (delta 0), reused 138 (delta 0)
Receiving objects: 100% (138/138), 162.45 KiB | 2.01 MiB/s, done.
[2026-05-08 10:00:02] Step 2: Checking out target commit/branch: c7c11f62f10258b9d9ad2aea1c2a62eda8b2531f...
HEAD is now at c7c11f62 Merge branch '8.x' of github.com:top-think/think into 8.x
[2026-05-08 10:00:03] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sent 9,124 bytes  received 91 bytes  18,430.00 bytes/sec
[2026-05-08 10:00:09] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
[2026-05-08 10:00:10] Deployment completed successfully!"
fi

ID="${TASK_IDS[20260518090000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-18 09:00:00] Step 1: Cloning repository from https://gitee.com/top-think/think.git into thinkphp-web/20260518090000...
Receiving objects: 100% (138/138), 162.45 KiB | 1.89 MiB/s, done.
[2026-05-18 09:00:02] Step 2: Checking out target commit/branch: 7cc4119dcaab2f72606d46eafd16582e887b5d3e...
HEAD is now at 7cc4119d readem调整
[2026-05-18 09:00:03] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sent 7,890 bytes  received 78 bytes  15,936.00 bytes/sec
[2026-05-18 09:00:08] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
[2026-05-18 09:00:09] Deployment completed successfully!"
fi

ID="${TASK_IDS[20260523143000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-23 14:30:00] Step 1: Cloning repository from https://gitee.com/top-think/think.git into thinkphp-web/20260523143000...
Receiving objects: 100% (145/145), 172.12 KiB | 2.34 MiB/s, done.
[2026-05-23 14:30:03] Step 2: Checking out target commit/branch: 98d8c5e09712042a51a2a79622bbce422b48c0ea...
HEAD is now at 98d8c5e0 增加 topthink/think-filesystem:3.0
[2026-05-23 14:30:04] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sent 10,234 bytes  received 102 bytes  20,672.00 bytes/sec
[2026-05-23 14:30:12] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
[2026-05-23 14:30:13] Deployment completed successfully!"
fi

ID="${TASK_IDS[20260527113000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-27 11:30:00] Step 1: Cloning repository from https://gitee.com/top-think/think.git into thinkphp-web/20260527113000...
Receiving objects: 100% (147/147), 175.89 KiB | 3.12 MiB/s, done.
[2026-05-27 11:30:02] Step 2: Checking out target commit/branch: 49917ae7c0de3c7c12de367b99b671321c3c304c...
HEAD is now at 49917ae7 增加配置参数 修改依赖
[2026-05-27 11:30:03] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sent 11,456 bytes  received 112 bytes  22,136.00 bytes/sec
[2026-05-27 11:30:10] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
[2026-05-27 11:30:11] Deployment completed successfully!"
fi

# ---- Webman 日志 ----

ID="${TASK_IDS[20260513160000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-13 16:00:00] Step 1: Cloning repository from https://gitee.com/walkor/webman.git into webman-api/20260513160000...
Receiving objects: 100% (89/89), 98.34 KiB | 1.45 MiB/s, done.
[2026-05-13 16:00:02] Step 2: Checking out target commit/branch: fbd7377964d6a2f9b8fd6b4bbd543a30c28d13af...
HEAD is now at fbd73779 wellcome
[2026-05-13 16:00:03] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
[2026-05-13 16:00:08] Error: Rsync failed on 127.0.0.1:
ssh: connect to host 127.0.0.1 port 22: Connection refused
rsync: [sender] write error: Broken pipe (32)
rsync error: error in rsync protocol data stream (code 12) at io.c(228) [sender=3.2.7]
[2026-05-13 16:00:08] Error: Phase 1 Rsync failed on one or more nodes. Halting deployment."
fi

ID="${TASK_IDS[20260516100000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-16 10:00:00] Step 1: Cloning repository from https://gitee.com/walkor/webman.git into webman-api/20260516100000...
Receiving objects: 100% (89/89), 98.34 KiB | 2.12 MiB/s, done.
[2026-05-16 10:00:02] Step 2: Checking out target commit/branch: e8cedb15979e7804d9b7bf72128b15ebdb538d75...
HEAD is now at e8cedb15 Update IndexController.php
[2026-05-16 10:00:03] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sending incremental file list
app/controller/IndexController.php
sent 15,234 bytes  received 152 bytes  30,772.00 bytes/sec
total size is 98,432  speedup is 6.43
[2026-05-16 10:00:11] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
[2026-05-16 10:00:12] Deployment completed successfully!"
fi

ID="${TASK_IDS[20260520143000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-20 14:30:00] Step 1: Cloning repository from https://gitee.com/walkor/webman.git into webman-api/20260520143000...
Receiving objects: 100% (89/89), 98.34 KiB | 2.56 MiB/s, done.
[2026-05-20 14:30:02] Step 2: Checking out target commit/branch: 9216f9c05ba6e6cc54aeed6476d46b0c12419295...
HEAD is now at 9216f9c0 Update IndexController.php
[2026-05-20 14:30:03] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sent 16,789 bytes  received 168 bytes  33,914.00 bytes/sec
[2026-05-20 14:30:10] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
[2026-05-20 14:30:11] Deployment completed successfully!"
fi

ID="${TASK_IDS[20260525091500]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-25 09:15:00] Step 1: Cloning repository from https://gitee.com/walkor/webman.git into webman-api/20260525091500...
Receiving objects: 100% (92/92), 101.23 KiB | 2.89 MiB/s, done.
[2026-05-25 09:15:02] Step 2: Checking out target commit/branch: 99c2aafc555521c6be37edb03f1d4704ca9c2818...
HEAD is now at 99c2aafc Add Docker support for webman deployment
[2026-05-25 09:15:03] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sending incremental file list
Dockerfile
docker-compose.yml
.dockerignore
sent 23,456 bytes  received 234 bytes  47,380.00 bytes/sec
[2026-05-25 09:15:11] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
[2026-05-25 09:15:12] Deployment completed successfully!"
fi

# ---- CRMEB 日志 ----

ID="${TASK_IDS[20260520150000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-20 15:00:00] Step 1: Cloning repository from https://gitee.com/ZhongBangKeJi/CRMEB.git into crmeb-shop/20260520150000...
remote: Enumerating objects: 52847, done.
remote: Counting objects: 100% (52847/52847), done.
remote: Total 52847 (delta 0), reused 52847 (delta 0)
Receiving objects: 100% (52847/52847), 187.43 MiB | 8.92 MiB/s, done.
[2026-05-20 15:00:38] Step 2: Checking out target commit/branch: 779733627341fdb56e2c2291f695c18d3de52eaf...
HEAD is now at 77973362 fix(queue): adjust queue service check logic
[2026-05-20 15:00:39] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sent 1,234,567 bytes  received 12,345 bytes  498,364.80 bytes/sec
total size is 187,432,456  speedup is 149.68
[2026-05-20 15:01:24] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
[2026-05-20 15:01:25] Deployment completed successfully!"
fi

ID="${TASK_IDS[20260522193000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-22 19:30:00] Step 1: Cloning repository from https://gitee.com/ZhongBangKeJi/CRMEB.git into crmeb-shop/20260522193000...
Receiving objects: 100% (52848/52848), 187.44 MiB | 7.23 MiB/s, done.
[2026-05-22 19:30:41] Step 2: Checking out target commit/branch: 1953370289f0c6bd5a7e07ddf52bd57a6dd233ac...
HEAD is now at 19533702 更新
[2026-05-22 19:30:42] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
[2026-05-22 19:31:12] Error: Rsync failed on 127.0.0.1:
rsync: [Receiver] mkstemp \"/tmp/demo_deploy/crmeb/production/releases/20260522193000/public/.index.html.XXXXXX\" failed: Permission denied (13)
rsync: [Receiver] mkstemp \"/tmp/demo_deploy/crmeb/production/releases/20260522193000/storage/.app.XXXXXX\" failed: Permission denied (13)
rsync error: some files/attrs were not transferred (see previous errors) (code 23) at main.c(1338) [sender=3.2.7]
[2026-05-22 19:31:12] Error: Phase 1 Rsync failed on one or more nodes. Halting deployment."
fi

ID="${TASK_IDS[20260525103000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-25 10:30:00] Step 1: Cloning repository from https://gitee.com/ZhongBangKeJi/CRMEB.git into crmeb-shop/20260525103000...
Receiving objects: 100% (52856/52856), 187.54 MiB | 9.12 MiB/s, done.
[2026-05-25 10:30:38] Step 2: Checking out target commit/branch: 50ac735ac375be46ef9eabfca9c46eaae16779d0...
HEAD is now at 50ac735a refactor(adminapi/setting): 重构系统配置相关接口逻辑
[2026-05-25 10:30:39] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sent 2,345,678 bytes  received 23,456 bytes  756,172.80 bytes/sec
total size is 187,543,210  speedup is 78.32
[2026-05-25 10:31:34] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
[2026-05-25 10:31:35] Deployment completed successfully!"
fi

ID="${TASK_IDS[20260526143000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-26 14:30:00] Step 1: Cloning repository from https://gitee.com/ZhongBangKeJi/CRMEB.git into crmeb-shop/20260526143000...
Receiving objects: 100% (52861/52861), 187.60 MiB | 10.34 MiB/s, done.
[2026-05-26 14:30:36] Step 2: Checking out target commit/branch: 60b594745f50fe3e9a4ffc08ca8ea1f02001742a...
HEAD is now at 60b59474 Merge branch 'master' of https://gitee.com/ZhongBangKeJi/CRMEB
[2026-05-26 14:30:37] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sent 3,123,456 bytes  received 31,234 bytes  986,918.40 bytes/sec
total size is 187,601,234  speedup is 58.94
[2026-05-26 14:31:39] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
[2026-05-26 14:31:40] Deployment completed successfully!"
fi

ID="${TASK_IDS[20260528211000]}"
if [ -n "$ID" ]; then
write_log "$ID" "[2026-05-28 21:10:00] Step 1: Cloning repository from https://gitee.com/ZhongBangKeJi/CRMEB.git into crmeb-shop/20260528211000...
Receiving objects: 100% (52863/52863), 187.61 MiB | 11.23 MiB/s, done.
[2026-05-28 21:10:35] Step 2: Checking out target commit/branch: af0de843746289693f01ac7fe08a3bacdf862137...
HEAD is now at af0de843 优化自定义配置数字输入框宽度
[2026-05-28 21:10:36] Step 4 [Phase1]: Synchronizing files to remote server 127.0.0.1:22...
sending incremental file list
app/adminapi/controller/setting/
app/adminapi/controller/setting/SystemConfigController.php
resources/admin/src/views/setting/
resources/admin/src/views/setting/systemConfig.vue
sent 1,890,234 bytes  received 18,902 bytes  607,732.08 bytes/sec
total size is 187,612,345  speedup is 98.10
[2026-05-28 21:11:18] Step 5 [Phase2]: Switching active symlink on 127.0.0.1:22...
[2026-05-28 21:11:19] Deployment completed successfully!"
fi

echo ""
echo "所有任务日志文件已生成："
ls -la "$LOG_DIR"/task_*.log 2>/dev/null | awk '{print "  "$NF, $5"B"}'
echo ""
echo "完成！"
