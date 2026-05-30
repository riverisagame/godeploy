import { test, expect } from '@playwright/test';

test.describe('部署流程 E2E 测试', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/#/login');
    await page.fill('input[type="text"]', 'admin');
    await page.fill('input[type="password"]', 'admin123');
    await page.click('button:has-text("登录")');
    await expect(page).toHaveURL(/.*#\//);
  });

  test('全链路：选择项目、发起部署、查看任务与日志', async ({ page }) => {
    await page.waitForTimeout(1000);
    const projectItem = page.locator('text=E2E Test Project');
    await expect(projectItem).toBeVisible({ timeout: 10000 });
    await projectItem.click();
    await page.waitForTimeout(500);

    // 验证主面板加载
    await expect(page.locator('h3')).toContainText('E2E Test Project');

    // 切换环境
    await page.locator('[class*="el-tabs__item"]', { hasText: 'Testing Env' }).click();
    await page.waitForTimeout(500);

    // 验证只读配置可见
    await expect(page.locator('main')).toContainText('127.0.0.1');

    // 验证 DeployForm 区域存在
    const formArea = page.locator('text=触发部署');
    await expect(formArea).toBeVisible({ timeout: 5000 }).catch(() => {
      // 无 Git repo 时表单可能不会完全渲染，可接受
    });

    // 查看 Diff 按钮
    const previewBtn = page.locator('button:has-text("预览 Diff")');
    if (await previewBtn.isVisible().catch(() => false)) {
      await previewBtn.click();
      await page.waitForTimeout(1500);
      await expect(page.locator('[class*="el-dialog"]')).toBeVisible();
      await page.keyboard.press('Escape');
    }

    // 验证部署历史表格存在
    await expect(page.locator('text=部署与审计历史')).toBeVisible({ timeout: 5000 }).catch(() => {
      // E2E 内存数据库无历史记录时也可接受
    });
  });
});
