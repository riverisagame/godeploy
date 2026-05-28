#!/bin/bash
# 从 Gitee 克隆 PHP 项目，提取真实 commit 信息
set -e

WORKSPACE="/mnt/d/claudeprj/deploy/demo_workspace/gitee_demo"
mkdir -p "$WORKSPACE"
cd "$WORKSPACE"

echo "========================================"
echo "克隆 PHP 项目中（shallow clone, depth=5）"
echo "========================================"

# 1. ThinkPHP 基础框架
if [ ! -d "think" ]; then
  echo "[1/3] 克隆 ThinkPHP..."
  git clone --depth 5 --quiet https://gitee.com/top-think/think.git think
fi

# 2. webman
if [ ! -d "webman" ]; then
  echo "[2/3] 克隆 webman..."
  git clone --depth 5 --quiet https://gitee.com/walkor/webman.git webman
fi

# 3. CRMEB
if [ ! -d "CRMEB" ]; then
  echo "[3/3] 克隆 CRMEB 商城..."
  git clone --depth 5 --quiet https://gitee.com/ZhongBangKeJi/CRMEB.git CRMEB
fi

echo ""
echo "========================================"
echo "提取真实 Commit 信息"
echo "========================================"

echo "--- ThinkPHP commits ---"
git -C think log --oneline -5 --format='%H|%an|%s|%ai'

echo "--- webman commits ---"
git -C webman log --oneline -5 --format='%H|%an|%s|%ai'

echo "--- CRMEB commits ---"
git -C CRMEB log --oneline -5 --format='%H|%an|%s|%ai'
