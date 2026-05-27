# Deployer 使用手册 (User Manual)

## 1. 安装指南
Deployer 推荐通过 Composer 安装在项目中：

```bash
composer require deployer/deployer --dev
```

安装完成后，可以使用 `./vendor/bin/dep` 命令。

## 2. 快速初始化
在项目根目录运行：
```bash
npx dep init
```
按照提示选择你的框架（如 Laravel 或 Symfony），它将生成一个 `deploy.php` 文件。

## 3. 配置解析 (deploy.php)
一个典型的配置文件包含以下部分：

```php
namespace Deployer;

require 'recipe/laravel.php'; // 引入配方

// 1. 设置存储库
set('repository', 'git@github.com:org/repo.git');

// 2. 配置主机
host('prod')
    ->setHostname('1.2.3.4')
    ->setRemoteUser('deploy')
    ->setDeployPath('/var/www/my-app');

// 3. 自定义任务
task('build', function () {
    cd('{{release_path}}');
    run('npm run build');
});

// 4. 定义钩子
after('deploy:failed', 'deploy:unlock');
```

## 4. 核心命令
- **部署**: `dep deploy [host]` - 执行完整的部署流程。
- **回滚**: `dep rollback` - 撤销本次部署。
- **解锁**: `dep deploy:unlock` - 如果部署意外中断，手动清除锁定。
- **远程终端**: `dep ssh [host]` - 直接进入远程服务器目录。

## 5. 常见任务说明
- `deploy:prepare`: 创建发布目录结构。
- `deploy:vendors`: 安装依赖。
- `deploy:publish`: 更新符号链接，正式发布。


## 7. 技术底座
本手册所描述的逻辑由以下核心组件驱动：
- 执行引擎：`Deployer\Deployer`
- 集合管理：`Deployer\Collection\Collection`
- 主机管理：`Deployer\Host\Host`
- 任务实体：`Deployer\Task\Task`
