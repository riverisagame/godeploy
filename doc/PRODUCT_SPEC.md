# Deployer 产品说明书 (Product Specification)

## 1. 产品定位
Deployer 是一款为 PHP 开发者量身定制的**原生 PHP 部署工具**。它结合了 Capistrano 的思想与 PHP 的灵活性，旨在提供一种简单、快速且可靠的自动化部署方案。

## 2. 核心价值
- **PHP 原生驱动**：无需学习 YAML 或 Python，直接使用熟悉的 PHP 语法编写部署逻辑。
- **零停机部署 (Zero Downtime)**：通过符号链接 (Symlink) 机制，实现秒级切换，确保用户访问不中断。
- **原子性保证**：部署过程中的每一步都经过精心设计，若中间失败，不会破坏当前线上环境。
- **内置配方 (Recipes)**：开箱即用，支持 Laravel, Symfony, Magento, WordPress 等主流 PHP 框架。

## 3. 功能矩阵
| 功能特性 | 描述 |
| :--- | :--- |
| **并行执行** | 支持同时向数百台服务器推送代码。 |
| **原子部署** | 采用 `releases` 目录管理，通过 `current` 软链接切换。 |
| **回滚机制** | `dep rollback` 命令一键恢复到上一个稳定版本。 |
| **任务挂钩 (Hooks)** | 支持 `before` 和 `after` 任务，实现精细化流程控制。 |
| **主机管理** | 灵活的 Inventory 系统，支持别名、多环境（Prod/Staging）配置。 |

## 4. 竞品对比
| 维度 | Deployer | Ansible | Capistrano |
| :--- | :--- | :--- | :--- |
| **语言** | PHP | Python / YAML | Ruby |
| **易用性** | 极高 (PHP 开发者) | 中 (需学习 YAML) | 中 (需 Ruby 环境) |
| **速度** | 极快 (原生 SSH) | 快 | 中 |
| **适用范围** | PHP 项目首选 | 全平台通用 | Ruby/通用 |

## 5. 核心技术指标
- **最低环境要求**: PHP 8.3+。
- **核心依赖**: Symfony Console, Process, Yaml 组件。
- **核心类**: `Deployer\Deployer`, `Deployer\Host\Host`, `Deployer\Task\Task`, `Deployer\Collection\Collection`
