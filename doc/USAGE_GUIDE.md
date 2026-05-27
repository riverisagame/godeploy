# Deployer 详细使用说明书 (Detailed Usage Guide)

## 1. 核心语法手册
Deployer 使用流畅的 PHP DSL (Domain Specific Language) 来定义部署逻辑。

### 1.1 主机定义 (Host)
使用 `host()` 函数定义目标服务器。
```php
host('prod.example.com')
    ->set('remote_user', 'deploy')
    ->set('deploy_path', '/var/www/app')
    ->set('identity_file', '~/.ssh/id_rsa');
```

### 1.2 任务定义 (Task)
使用 `task()` 定义具体的操作。
```php
task('my_task', function () {
    run('ls -al {{release_path}}'); // 在远程执行
    writeln('任务执行完成！'); // 在本地输出
});
```

### 1.3 变量管理 (Set/Get)
- **设置变量**: `set('branch', 'master');`
- **获取变量**: `get('branch');`
- **在字符串中使用**: `"git checkout {{branch}}"`

## 2. 常用内置函数
| 函数 | 描述 |
| :--- | :--- |
| `run(string $cmd)` | 在当前主机执行远程 Shell 命令。 |
| `runLocally(string $cmd)` | 在本地机器执行 Shell 命令。 |
| `cd(string $path)` | 切换远程工作目录。 |
| `upload(string $src, string $dest)` | 上传文件或目录到远程服务器。 |
| `download(string $src, string $dest)` | 从远程服务器下载文件。 |
| `ask(string $question)` | 在命令行交互式提问。 |

## 3. 部署生命周期 (Lifecycle)
一个标准的部署流程（如引入 `recipe/common.php`）包含：
1. `deploy:prepare`: 检查环境、创建目录。
2. `deploy:lock`: 锁定主机，防止冲突。
3. `deploy:release`: 创建新的 release 目录。
4. `deploy:update_code`: 从 Git 拉取代码。
5. `deploy:shared`: 挂载共享文件（如 `.env`）。
6. `deploy:writable`: 设置目录写权限。
7. `deploy:vendors`: 安装 Composer 依赖。
8. `deploy:clear_paths`: 清理不需要的路径。
9. `deploy:publish`: **关键步骤**，原子级更新 `current` 软链。
10. `deploy:unlock`: 释放锁定。

## 4. 命令行高级技巧
- **并发执行**: `dep deploy -p 10` (同时部署 10 台主机)。
- **详细日志**: `dep deploy -vvv` (查看所有 SSH 交互)。
- **列出任务**: `dep list`。
- **选择主机**: `dep deploy production`。

## 5. 技术审计引用
本说明书涉及的核心实现：
- `Deployer\Deployer`, `Deployer\Host\Host`, `Deployer\Task\Task`, `Deployer\Collection\Collection`
