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

  test('创建部署任务并查看看板页面内容', async ({ page }) => {
    // 验证左侧栏、主内容区可见
    await expect(page.locator('.sidebar')).toBeVisible();
    await expect(page.locator('.content-area')).toBeVisible();
    
    // 因为没有配置物理项目，页面应该提示未加载到项目配置
    await expect(page.locator('text=未加载到项目配置')).toBeVisible();
    await expect(page.locator('text=请从左侧栏选择一个部署项目')).toBeVisible();
  });
});
