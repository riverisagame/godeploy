#!/usr/bin/env bash
set -eo pipefail

echo ">>> 开始更新 GoDeploy 服务..."

# 1. 检查二进制
if [ ! -f "godeploy" ]; then
    echo "!!! 错误：当前目录未找到新的 godeploy 二进制文件！"
    exit 1
fi

# 2. 停机
echo ">>> 停止运行中的服务..."
systemctl stop godeploy || true

# 3. 数据备份 (防丢失机制)
DB_PATH="/var/lib/godeploy/godeploy.db"
if [ -f "$DB_PATH" ]; then
    BACKUP_NAME="godeploy_$(date +%Y%m%d_%H%M%S).db.bak"
    echo ">>> 正在备份 SQLite 数据库至 /var/lib/godeploy/$BACKUP_NAME ..."
    cp "$DB_PATH" "/var/lib/godeploy/$BACKUP_NAME"
    chown godeploy:godeploy "/var/lib/godeploy/$BACKUP_NAME"
fi

# 4. 替换二进制
echo ">>> 替换二进制文件..."
cp godeploy /usr/local/bin/godeploy
chmod +x /usr/local/bin/godeploy

# 5. 重启
echo ">>> 重启服务..."
systemctl start godeploy

echo ">>> 更新完成！已成功恢复运行。"
systemctl status godeploy --no-pager | head -n 10
