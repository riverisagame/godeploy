import { test, expect } from '@playwright/test';

test.describe('Dashboard 面板及辅助功能 E2E 测试', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/#/login');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button:has-text("登录")');
    await expect(page).toHaveURL(/.*#\//);
  });

  test('环境切换与配置只读展示验证', async ({ page }) => {
    // 选中项目 - 使用文本查找
    const projectItem = page.locator('text=E2E Test Project');
    await expect(projectItem).toBeVisible({ timeout: 10000 });
    await projectItem.click();
    await page.waitForTimeout(500);

    // 验证目标服务器列表中出现 127.0.0.1
    const pageContent = page.locator('main');
    await expect(pageContent).toContainText('127.0.0.1');
    await expect(pageContent).toContainText('/tmp/e2e-deploy-test');

    // 切换到 Production Env
    await page.locator('[class*="el-tabs__item"]', { hasText: 'Production Env' }).click();
    await page.waitForTimeout(300);

    await expect(page.locator('main')).toContainText('/tmp/e2e-deploy-prod');
  });

  test('历史列表按钮交互：差异对比', async ({ page }) => {
    await page.locator('text=E2E Test Project').click();
    await page.waitForTimeout(500);

    // 找到历史列表中的"对比"按钮
    const diffBtn = page.locator('button:has-text("对比")').first();
    const visible = await diffBtn.isVisible().catch(() => false);
    if (visible) {
      await diffBtn.click();
      await expect(page.locator('[class*="el-dialog__title"]')).toBeVisible();
      await page.keyboard.press('Escape');
    }
  });

  test('退出登录', async ({ page }) => {
    await page.locator('button:has-text("退出登录")').click();
    await expect(page).toHaveURL(/.*#\/login/);
  });
});
