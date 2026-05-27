# Deployer 实战案例集 (Use Case Examples)

## 案例 1: 基础静态项目部署
适用于纯 HTML 或无需后端编译的简单 PHP 项目。

```php
<?php
namespace Deployer;

require 'recipe/common.php';

set('repository', 'git@github.com:example/static-site.git');

host('my-static-site.com')
    ->set('deploy_path', '/var/www/static-site');

task('deploy', [
    'deploy:prepare',
    'deploy:publish',
]);
```

## 案例 2: Laravel 企业级部署
涵盖 Composer 依赖、数据库迁移、缓存清理及 Horizon/Queue 重启。

```php
<?php
namespace Deployer;

require 'recipe/laravel.php';

// 1. 基本配置
set('repository', 'git@github.com:company/laravel-app.git');
set('php_binary_path', '/usr/bin/php8.3');

// 2. 环境配置
host('prod')
    ->setHostname('10.0.0.1')
    ->setRemoteUser('deploy')
    ->set('deploy_path', '/var/www/laravel');

// 3. 自定义构建任务（如 npm）
task('build:assets', function () {
    cd('{{release_path}}');
    run('npm install && npm run build');
});

// 4. 定义流程钩子
after('deploy:vendors', 'build:assets');
after('deploy:failed', 'deploy:unlock');
```

## 案例 3: 多阶段环境 (Staging & Production)
展示如何在一份文件中管理不同环境的主机。

```php
<?php
namespace Deployer;
require 'recipe/common.php';

// 预发布环境
host('staging')
    ->setHostname('staging.example.com')
    ->set('branch', 'develop')
    ->set('deploy_path', '/var/www/staging');

// 生产环境
host('production')
    ->setHostname('example.com')
    ->set('branch', 'master')
    ->set('deploy_path', '/var/www/production');

// 执行命令示例:
// dep deploy staging
// dep deploy production
```

## 案例 4: 并行部署到 Web 服务器集群
展示如何利用并行特性快速部署到多个节点。

```php
<?php
namespace Deployer;
require 'recipe/common.php';

$webServers = ['web1.com', 'web2.com', 'web3.com', 'web4.com'];

foreach ($webServers as $hostname) {
    host($hostname)
        ->setRemoteUser('deploy')
        ->set('deploy_path', '/var/www/app');
}

// 执行命令示例（10个并发进程）:
// dep deploy -p 10
```

## 案例 5: 技术架构审计引用
以上案例均依赖以下核心组件的正确协同：
- `Deployer\Deployer`, `Deployer\Host\Host`, `Deployer\Task\Task`, `Deployer\Collection\Collection`
