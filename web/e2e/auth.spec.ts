import { test, expect } from '@playwright/test';

test('用户登录流程 E2E 测试', async ({ page }) => {
  // 访问主页会自动被未登录拦截重定向到登录页
  await page.goto('/#/login');
  await expect(page).toHaveURL(/.*#\/login/);

  // 填写账号密码
  await page.fill('input[type="text"]', 'admin');
  await page.fill('input[type="password"]', 'admin123');

  // 点击登录
  await page.click('button:has-text("登录")');

  // 验证登录成功跳转到 Dashboard
  await expect(page).toHaveURL(/.*#\//);
  
  // 验证页面包含欢迎信息或部署字样
  await expect(page.locator('text=GoDeployer 控制台')).toBeVisible();
});
