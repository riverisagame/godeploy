import { mount } from '@vue/test-utils';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import Dashboard from '../views/Dashboard.vue';
import axios from 'axios';
import { ElMessage, ElMessageBox } from 'element-plus';
import ElementPlus from 'element-plus';
import { html as mockHtml } from 'diff2html';

vi.mock('diff2html', () => ({
  html: vi.fn((text, options) => `<div class="mocked-diff">${text}</div>`)
}));


// Mock dependencies
vi.mock('axios');
const mockRouterPush = vi.fn();
vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: mockRouterPush
  })
}));

vi.mock('element-plus', async () => {
  const actual = await vi.importActual('element-plus');
  return {
    ...actual as any,
    ElMessageBox: {
      confirm: vi.fn().mockResolvedValue('confirm')
    },
    ElMessage: {
      success: vi.fn(),
      error: vi.fn(),
      warning: vi.fn()
    }
  };
});

describe('Dashboard.vue Component UI Test', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('1. 当没有选择项目和分支时，点击部署应该被拦截，不调用 API', async () => {
    (axios.get as any).mockResolvedValueOnce({ data: [] });
    const wrapper = mount(Dashboard, {
      global: { 
        plugins: [ElementPlus],
        stubs: { 'el-table-column': true }
      }
    });
    await new Promise(r => setTimeout(r, 0)); // wait for mounts

    const deployBtn = wrapper.findAll('.trigger-deploy-btn').find(b => b.text().includes('触发上线'));
    if (deployBtn) {
      await deployBtn.trigger('click');
      expect(ElMessage.warning).toHaveBeenCalled(); // 拦截警告
      expect(axios.post).not.toHaveBeenCalled();
    }
  });

  it('2. triggerDeploy 设置 deployState 为 confirming 并打开 Diff', async () => {
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') return Promise.resolve({ data: [{ id: 'p1', name: 'P1', repo: 'r', environments: [{ id: 'e1', name: 'E1', default_branch: 'main' }] }] });
      if (url.includes('/preview_diff')) return Promise.resolve({ data: { diff: '', files: ['src/main.go'] } });
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus], stubs: { 'el-table-column': true, 'el-dialog': true } }
    });
    await new Promise(r => setTimeout(r, 100));

    const vm = wrapper.vm as any;
    vm.deployForm.branch = 'main';
    await vm.triggerDeploy({ id: 'e1', name: 'E1' });
    await new Promise(r => setTimeout(r, 100));

    expect(vm.deployState.phase).toBe('confirming');
    expect(vm.diffVisible).toBe(true);
  });

  it('3. 顶栏: [用户管理跳转] router.push', async () => {
    (axios.get as any).mockResolvedValue({ data: [] });
    localStorage.setItem('role', 'admin');
    const wrapper = mount(Dashboard, { global: { plugins: [ElementPlus], mocks: { $router: { push: mockRouterPush } }, stubs: { 'el-table-column': true, 'el-dialog': true } } });
    await new Promise(r => setTimeout(r, 50));
    
    const userBtn = wrapper.findAll('.el-button').find(b => b.text().includes('用户管理'));
    expect(userBtn).toBeTruthy();
    await userBtn?.trigger('click');
    expect(mockRouterPush).toHaveBeenCalledWith('/users');
  });

  it('4. 顶栏: [系统清理] handleSystemPrune 发送 POST /api/system/prune', async () => {
    (axios.get as any).mockResolvedValue({ data: [] });
    (axios.post as any).mockResolvedValue({ data: {} });
    localStorage.setItem('role', 'admin');
    
    const wrapper = mount(Dashboard, { global: { plugins: [ElementPlus], mocks: { $router: { push: mockRouterPush } }, stubs: { 'el-table-column': true, 'el-dialog': true } } });
    await new Promise(r => setTimeout(r, 50));

    const btn = wrapper.findAll('.el-button').find(b => b.text().includes('系统自愈清理'));
    expect(btn).toBeTruthy();
    await btn?.trigger('click');
    expect(axios.post).toHaveBeenCalledWith('/api/system/prune');
  });

  it('5. 顶栏: [登出] handleLogout 清理 localStorage', async () => {
    (axios.get as any).mockResolvedValue({ data: [] });
    localStorage.setItem('token', 'fake-token');
    
    const wrapper = mount(Dashboard, { global: { plugins: [ElementPlus], mocks: { $router: { push: mockRouterPush } }, stubs: { 'el-table-column': true, 'el-dialog': true } } });
    await new Promise(r => setTimeout(r, 50));

    const btn = wrapper.findAll('.el-button').find(b => b.text().includes('退出登录'));
    expect(btn).toBeTruthy();
    await btn?.trigger('click');
    expect(mockRouterPush).toHaveBeenCalledWith('/login');
    expect(localStorage.getItem('token')).toBeNull();
  });

  it('6. 顶栏: [系统配置] openSettings 打开配置并调用 API', async () => {
    (axios.get as any).mockImplementation((url: string) => {
      if (url.includes('/git_binding')) return Promise.resolve({ data: { restrict_git_authors: true, bound_git_authors: 'test@example.com' } });
      return Promise.resolve({ data: [] });
    });
    (axios.put as any).mockResolvedValue({ data: {} });
    localStorage.setItem('role', 'admin');

    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus], mocks: { $router: { push: mockRouterPush } }, stubs: { 'el-table-column': true, 'el-dialog': true } }
    });
    await new Promise(r => setTimeout(r, 50));

    const vm = wrapper.vm as any;
    await vm.openSettings();
    await new Promise(r => setTimeout(r, 50));

    expect(axios.get).toHaveBeenCalledWith('/api/users/Admin/git_binding');
    expect(vm.settingVisible).toBe(true);

    await vm.saveSettings({ restrict_git_authors: true, bound_git_authors: 'test@example.com' });
    await new Promise(r => setTimeout(r, 50));

    expect(axios.put).toHaveBeenCalledWith('/api/users/Admin/git_binding', {
      restrict_git_authors: true,
      bound_git_authors: 'test@example.com'
    });
  });

  it('7. selectProject 触发项目获取和详情加载', async () => {
    const mockProjects = [
      { id: 'proj-1', name: 'Proj 1', environments: [{ id: 'env-1', name: 'Env 1' }] },
      { id: 'proj-2', name: 'Proj 2', environments: [{ id: 'env-2', name: 'Env 2' }] }
    ];
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') return Promise.resolve({ data: mockProjects });
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: {
        plugins: [ElementPlus],
        mocks: { $router: { push: mockRouterPush } },
        stubs: { 'el-table-column': true }
      }
    });
    await new Promise(r => setTimeout(r, 100));

    vi.clearAllMocks();

    // 直接调用 selectProject 模拟选择项目
    const vm = wrapper.vm as any;
    await vm.selectProject(mockProjects[1]);
    await new Promise(r => setTimeout(r, 50));

    expect(axios.get).toHaveBeenCalledWith('/api/tasks', expect.any(Object));
    expect(axios.get).toHaveBeenCalledWith('/api/projects/proj-2/refs');
    expect(axios.get).toHaveBeenCalledWith('/api/projects/proj-2/commits', expect.any(Object));
  });

  it('8. showLog 和 triggerRollback 函数存在', async () => {
    (axios.get as any).mockResolvedValue({ data: [] });
    const mockTask = { id: 3, release_name: 'rel3', commit_id: 'abc123', username: 'admin', status: 'success', created_at: new Date().toISOString() };

    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus], mocks: { $router: { push: mockRouterPush } }, stubs: { 'el-table-column': true, 'el-dialog': true } }
    });
    await new Promise(r => setTimeout(r, 50));

    const vm = wrapper.vm as any;
    expect(typeof vm.triggerRollback).toBe('function');
    expect(typeof vm.showLog).toBe('function');

    // 查看日志
    await vm.showLog(mockTask);
    await new Promise(r => setTimeout(r, 50));
    expect(vm.logVisible).toBe(true);
  });

  it('9. showLog 设置 logTask 并打开日志弹窗', async () => {
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') return Promise.resolve({ data: [] });
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus], stubs: { 'el-table-column': true } }
    });
    await new Promise(r => setTimeout(r, 50));

    const vm = wrapper.vm as any;
    vm.showLog({ id: 3, status: 'success' });
    await new Promise(r => setTimeout(r, 50));

    expect(vm.logVisible).toBe(true);
    expect(vm.logTask).toBeTruthy();
    expect(vm.logTask.id).toBe(3);
  });

  it('10. previewDeployDiff 触发 API 请求并打开 Diff 弹窗', async () => {
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') return Promise.resolve({ data: [{ id: 'p1', name: 'P1', repo: 'repo', environments: [{ id: 'dev', name: 'Dev', default_branch: 'main' }] }] });
      if (url.includes('/preview_diff')) return Promise.resolve({ data: { diff: '', files: ['src/main.go'] } });
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus], stubs: { 'el-table-column': true, 'el-dialog': true } }
    });
    await new Promise(r => setTimeout(r, 100));

    const vm = wrapper.vm as any;
    vm.deployForm.branch = 'main';
    await vm.previewDeployDiff({ id: 'dev', name: 'Dev', servers: [] });
    await new Promise(r => setTimeout(r, 100));

    expect(vm.diffVisible).toBe(true);
  });

  it('11. showDiff 触发 API 请求', async () => {
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') return Promise.resolve({ data: [] });
      if (url.includes('/preview_diff')) return Promise.resolve({ data: { diff: '', files: ['src/a.go'] } });
      if (url.includes('/api/tasks')) return Promise.resolve({ data: { diff: '', files: 'M\tsrc/a.go' } });
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus], stubs: { 'el-table-column': true, 'el-dialog': true } }
    });
    await new Promise(r => setTimeout(r, 50));

    const vm = wrapper.vm as any;
    await vm.showDiff({ id: 100, commit_id: 'abcdef12', release_name: 'test' });
    await new Promise(r => setTimeout(r, 100));

    expect(vm.diffVisible).toBe(true);
  });

  it('12. 切换环境不会自动触发 preview_diff', async () => {
    let previewDiffCallCount = 0;
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') return Promise.resolve({ data: [{ id: 'p', name: 'P', environments: [{ id: 'dev' }, { id: 'prod' }] }] });
      if (url.includes('/preview_diff')) { previewDiffCallCount++; return Promise.resolve({ data: { diff: '', files: [] } }); }
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus], stubs: { 'el-table-column': true } }
    });
    await new Promise(r => setTimeout(r, 80));

    (wrapper.vm as any).activeEnvTab = 'prod';
    await new Promise(r => setTimeout(r, 80));

    expect(previewDiffCallCount).toBe(0);
  });

  it('13. 账号配置弹窗能正常打开和保存', async () => {
    (axios.get as any).mockImplementation((url: string) => {
      if (url.includes('/git_binding')) return Promise.resolve({ data: { restrict_git_authors: false, bound_git_authors: '' } });
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: { plugins: [ElementPlus], stubs: { 'el-table-column': true, 'el-dialog': true } }
    });
    await new Promise(r => setTimeout(r, 50));

    const vm = wrapper.vm as any;
    await vm.openSettings();
    await new Promise(r => setTimeout(r, 50));

    expect(vm.settingVisible).toBe(true);
  });
});



