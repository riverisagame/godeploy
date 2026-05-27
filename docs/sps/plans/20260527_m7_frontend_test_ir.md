# Milestone 7: 前端测试体系（Vitest + Playwright + GitHub Actions CI）

## 背景
当前项目缺少前端测试覆盖，M1-M6 的后端 Go 测试体系完善，前端 Vue3 侧无任何自动化测试。本计划引入 Vitest（单元/组件渲染）与 Playwright（E2E 真实后端联调），并集成 GitHub Actions CI。

---

## 子任务分解（纳米级，每步 ≤20 行代码）

### 阶段 A：Vitest 单元测试体系

#### A1. 安装 Vitest 及依赖
- **文件**: `web/package.json`
- **动作**: 新增 devDependencies: `vitest`, `@vue/test-utils`, `@testing-library/vue`, `jsdom`, `@vitejs/plugin-vue`
- **行数**: ~6 行（仅依赖声明）

#### A2. 创建 `vitest.config.ts`
- **文件**: `web/vitest.config.ts` [NEW]
- **动作**: 配置 environment=jsdom, setupFiles, 别名解析
- **行数**: ~20 行

#### A3. 提取 Dashboard 工具函数到独立文件
- **文件**: `web/src/utils/deploy.ts` [NEW]
- **动作**: 从 Dashboard.vue 提取 `getStatusTagType`, `getStatusText`, `formatTime`, `buildWSUrl` 四个纯函数
- **行数**: ~20 行
- **影响**: Dashboard.vue 改为 import 调用，**不改变功能逻辑**

#### A4. 修改 Dashboard.vue 使用提取的工具函数
- **文件**: `web/src/views/Dashboard.vue`
- **动作**: 删除内联函数定义，改为 `import { ... } from '../utils/deploy'`
- **行数**: ~5 行变更（import + 删除原函数）

#### A5. 编写工具函数单元测试
- **文件**: `web/src/__tests__/deploy.utils.test.ts` [NEW]
- **动作**: 覆盖 `getStatusTagType`（4种状态）、`getStatusText`（4种状态）、`formatTime`、`buildWSUrl`（http/https 协议转换）
- **行数**: ~60 行

#### A6. 编写 Login 组件渲染测试
- **文件**: `web/src/__tests__/Login.spec.ts` [NEW]
- **动作**: 渲染 Login 组件，断言表单元素存在，测试表单 v-model 绑定
- **行数**: ~40 行

#### A7. 添加 test 脚本到 package.json
- **文件**: `web/package.json`
- **动作**: 新增 `"test": "vitest run"` 和 `"test:watch": "vitest"` 脚本
- **行数**: ~2 行

---

### 阶段 B：Playwright E2E 测试体系

#### B1. 安装 Playwright
- **文件**: `web/package.json`
- **动作**: 新增 `@playwright/test` devDependency
- **行数**: ~1 行

#### B2. 创建 E2E 测试配置
- **文件**: `web/playwright.config.ts` [NEW]
- **动作**: 配置 baseURL=http://localhost:9090, webServer.command 启动 Go 后端, timeout 设置
- **行数**: ~30 行

#### B3. 创建 E2E 测试用 Go 配置文件
- **文件**: `e2e_config.yaml` [NEW] （项目根目录）
- **动作**: 定义测试专用配置（端口9090、内存DB、测试密钥），测试结束后不留任何物理痕迹
- **行数**: ~20 行

#### B4. 编写登录流程 E2E 测试
- **文件**: `web/e2e/auth.spec.ts` [NEW]
- **动作**: 访问根路径→验证重定向到登录页→填写 admin/admin→提交→验证跳转到 Dashboard
- **行数**: ~35 行

#### B5. 编写部署触发 E2E 测试（含 WS 日志验证）
- **文件**: `web/e2e/deploy.spec.ts` [NEW]
- **动作**: 登录后选择项目→触发部署→验证日志弹窗打开→验证 WebSocket 连接建立（intercept）
- **行数**: ~50 行

---

### 阶段 C：GitHub Actions CI 集成

#### C1. 创建 CI 工作流文件
- **文件**: `.github/workflows/ci.yml` [NEW]
- **动作**: 定义三个 Job：
  1. `go-test`：运行 `go test ./...`
  2. `frontend-build`：`npm ci && npm run build`
  3. `e2e`：编译 Go 二进制 → 安装 Playwright → 运行 `npx playwright test`
- **行数**: ~70 行

---

## 文件清单
| 操作 | 文件路径 | 备注 |
|---|---|---|
| NEW | `web/vitest.config.ts` | Vitest 配置 |
| NEW | `web/src/utils/deploy.ts` | 提取的工具函数 |
| MODIFY | `web/src/views/Dashboard.vue` | import 工具函数（~5行变更）|
| NEW | `web/src/__tests__/deploy.utils.test.ts` | 单元测试 |
| NEW | `web/src/__tests__/Login.spec.ts` | 组件渲染测试 |
| MODIFY | `web/package.json` | 新增依赖和脚本 |
| NEW | `web/playwright.config.ts` | Playwright 配置 |
| NEW | `e2e_config.yaml` | E2E 专用 Go 配置 |
| NEW | `web/e2e/auth.spec.ts` | E2E 登录测试 |
| NEW | `web/e2e/deploy.spec.ts` | E2E 部署+WS 测试 |
| NEW | `.github/workflows/ci.yml` | GitHub Actions |

## 验收准则
1. `cd web && npm test` → 全部 Vitest 测试通过
2. `cd web && npx playwright test` → 全部 E2E 测试通过（后端自动启动）
3. Push 到 GitHub 后 Actions 自动触发，3个 Job 均显示绿色✅
4. 物理零污染：E2E 测试使用内存 DB，`e2e_config.yaml` 中禁止任何文件写入真实数据
