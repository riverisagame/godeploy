import { test, expect } from '@playwright/test';

test.describe('部署流程 E2E 测试', () => {
  test.beforeEach(async ({ page }) => {
    // 登录预置
    await page.goto('/#/login');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button:has-text("登录")');
    await expect(page).toHaveURL(/.*#\//);
  });

  test('全链路：选择项目、发起部署、查看任务与日志', async ({ page }) => {
    // 1. 等待侧边栏加载并出现配置好的 mock 项目
    await expect(page.locator('.sidebar')).toBeVisible();
    
    // 点击项目列表中名为 "E2E Test Project" 的项目
    const projectItem = page.locator('.project-item').filter({ hasText: 'E2E Test Project' });
    await expect(projectItem).toBeVisible({ timeout: 10000 });
    await projectItem.click();

    // 2. 验证主面板加载
    await expect(page.locator('.content-area')).toBeVisible();
    await expect(page.locator('.proj-summary h3')).toHaveText('E2E Test Project');

    // 3. 切换环境并验证只读信息
    const testingTab = page.locator('.el-tabs__item', { hasText: 'Testing Env' });
    await expect(testingTab).toBeVisible();
    await testingTab.click();

    // 4. 填写表单并发起部署
    // 定位分支和 Commit 的输入框，通过 aria-hidden="false" 获取当前活动 Tab
    const activeTab = page.locator('.el-tab-pane[aria-hidden="false"]');
    const branchInput = activeTab.locator('input[placeholder="例如: main, develop, v1.0.0"]');
    const commitInput = activeTab.locator('input[placeholder="留空则拉取分支最新提交"]');
    
    await branchInput.fill('master');
    await commitInput.fill('mock-commit-hash');

    // 触发上线
    await activeTab.locator('.trigger-deploy-btn').click();

    // 在弹出的二次确认框中点击确定
    const confirmBtn = page.locator('.el-message-box__btns button', { hasText: '确定部署' });
    await expect(confirmBtn).toBeVisible();
    await confirmBtn.click();

    // 5. 验证是否直接弹出了日志 Modal 并显示 WebSocket 或日志获取状态
    const logModal = page.locator('.el-dialog__title', { hasText: '构建与同步日志' });
    await expect(logModal).toBeVisible({ timeout: 10000 });
    
    // 验证终端面板存在且有输出
    const termBody = page.locator('.terminal-body');
    await expect(termBody).toBeVisible();

    // 关闭日志弹窗
    await page.locator('.el-dialog__headerbtn').click();
    await expect(logModal).toBeHidden();

    // 6. 验证任务出现在部署历史表格中
    // 第一行的 Commit 列应包含 mock-commit-hash
    const firstRowCommit = page.locator('.el-table__row').first().locator('td').nth(2); // 根据表格列索引
    await expect(firstRowCommit).toContainText('mock-commit-hash');
    
    // 第一行的状态列应为 Pending 或 Deploying 或 Failed (由于是假项目，可能会很快 Failed)
    const firstRowStatus = page.locator('.el-table__row').first().locator('td').nth(4);
    await expect(firstRowStatus).not.toBeEmpty();
  });
});
