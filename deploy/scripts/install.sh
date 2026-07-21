#!/usr/bin/env bash
set -eo pipefail

echo ">>> 开始部署 GoDeploy 服务..."

# 1. 检查并创建专属低权限用户
if ! id "godeploy" &>/dev/null; then
    echo ">>> 创建专属用户 godeploy..."
    useradd -r -s /bin/false godeploy
else
    echo ">>> 用户 godeploy 已存在，跳过创建。"
fi

# 2. 检查基础环境
if [ ! -f "godeploy" ]; then
    echo "!!! 错误：当前目录未找到 godeploy 二进制文件！请先执行构建。"
    exit 1
fi

# 3. 创建标准化目录结构
echo ">>> 创建目录结构 /var/lib/godeploy (数据) 和 /etc/godeploy (配置)..."
mkdir -p /var/lib/godeploy
mkdir -p /etc/godeploy
mkdir -p /var/log/godeploy

# 4. 初始化默认配置文件 (幂等：如果已存在则跳过)
if [ ! -f "/etc/godeploy/config.conf" ]; then
    echo ">>> 初始化默认配置文件 /etc/godeploy/config.conf..."
    if [ -f "configs/config.conf.example" ]; then
        cp configs/config.conf.example /etc/godeploy/config.conf
    else
        echo "!!! 警告：未找到 configs/config.conf.example，创建空配置文件。"
        touch /etc/godeploy/config.conf
    fi
else
    echo ">>> 配置文件已存在，跳过初始化。"
fi

# 5. 转移二进制并授权
echo ">>> 安装二进制文件到 /usr/local/bin..."
cp godeploy /usr/local/bin/godeploy
chmod +x /usr/local/bin/godeploy

echo ">>> 赋予目录及文件所有权..."
chown -R godeploy:godeploy /var/lib/godeploy /etc/godeploy /var/log/godeploy

# 6. 配置 Systemd
echo ">>> 安装 Systemd 服务文件..."
if [ -f "deploy/systemd/godeploy.service" ]; then
    cp deploy/systemd/godeploy.service /etc/systemd/system/
else
    echo "!!! 错误：未找到 deploy/systemd/godeploy.service，请确保您在项目根目录执行此脚本！"
    exit 1
fi

echo ">>> 重载 Systemd 守护进程..."
systemctl daemon-reload

echo ">>> 启动 GoDeploy 并设置开机自启..."
systemctl enable --now godeploy

echo ">>> 部署成功！"
echo ">>> 您可以使用 'systemctl status godeploy' 检查运行状态。"
