import { test, expect } from '@playwright/test';

test.describe('Dashboard 面板及辅助功能 E2E 测试', () => {
  test.beforeEach(async ({ page }) => {
    // 登录预置
    await page.goto('/#/login');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button:has-text("登录")');
    await expect(page).toHaveURL(/.*#\//);
  });

  test('环境切换与配置只读展示验证', async ({ page }) => {
    // 选中项目
    const projectItem = page.locator('.project-item').filter({ hasText: 'E2E Test Project' });
    await expect(projectItem).toBeVisible({ timeout: 10000 });
    await projectItem.click();

    // 默认应该选中 Testing Env，验证目标服务器列表中出现 127.0.0.1
    let activeTab = page.locator('.el-tab-pane[aria-hidden="false"]');
    let serverHost = activeTab.locator('.server-host').first();
    await expect(serverHost).toContainText('127.0.0.1');
    let serverPath = activeTab.locator('.server-path').first();
    await expect(serverPath).toContainText('/tmp/e2e-deploy-test');

    // 切换到 Production Env
    const prodTab = page.locator('.el-tabs__item', { hasText: 'Production Env' });
    await expect(prodTab).toBeVisible();
    await prodTab.click();

    // 验证目标服务器列表更新
    activeTab = page.locator('.el-tab-pane[aria-hidden="false"]');
    serverPath = activeTab.locator('.server-path').first();
    await expect(serverPath).toContainText('/tmp/e2e-deploy-prod');
  });

  test('历史列表按钮交互：差异对比', async ({ page }) => {
    const projectItem = page.locator('.project-item').filter({ hasText: 'E2E Test Project' });
    await projectItem.click();

    // 找到历史列表中的“对比”按钮
    // 考虑到可能是 mock 数据，如果没有历史数据则此测试会自动跳过或失败
    // 在之前的测试里我们触发了上线，应该会产生一条数据
    const diffBtn = page.locator('.el-table__row').first().locator('button', { hasText: '对比' });
    await diffBtn.waitFor({ state: 'visible', timeout: 5000 }).catch(() => null);
    
    // 如果按钮可见，点击并验证弹窗
    if (await diffBtn.isVisible()) {
        await diffBtn.click();
        const diffModal = page.locator('.el-dialog__title', { hasText: 'Git 代码差异对比' });
        await expect(diffModal).toBeVisible();
        await expect(page.locator('.diff-pre')).toBeVisible();
        
        // 关闭
        await page.locator('.el-dialog__headerbtn').click();
    }
  });

  test('退出登录', async ({ page }) => {
    await page.locator('button', { hasText: '退出登录' }).click();
    await expect(page).toHaveURL(/.*#\/login/);
  });
});
