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

  it('2. 在选择项目和环境后，点击触发上线应当弹出 Diff 确认框而不直接调用 /api/tasks', async () => {
    const mockProjects = [{
      id: 'test-proj',
      name: 'Test',
      environments: [{ id: 'prod', name: 'Prod', default_branch: 'main' }]
    }];
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') return Promise.resolve({ data: mockProjects });
      if (url.includes('/preview_diff')) return Promise.resolve({ data: { diff: 'mock diff', files: ['src/main.go'] } });
      return Promise.resolve({ data: [] });
    });
    (axios.post as any).mockResolvedValue({ data: { id: 99 } });

    const wrapper = mount(Dashboard, {
      global: { 
        plugins: [ElementPlus],
        stubs: { 
          'el-table-column': true,
          'el-dialog': {
            template: '<div class="mock-el-dialog"><slot></slot></div>'
          }
        }
      }
    });
    await new Promise(r => setTimeout(r, 100)); // wait for API render

    // 确保已选择项目
    const projItems = wrapper.findAll('.project-item');
    if (projItems.length > 0) {
      await projItems[0].trigger('click');
    }
    await new Promise(r => setTimeout(r, 0));

    // 点击部署按钮
    const deployBtn = wrapper.findAll('.trigger-deploy-btn').find(b => b.text().includes('触发上线'));
    expect(deployBtn).toBeTruthy();
    if (deployBtn) {
      await deployBtn.trigger('click');
      await new Promise(r => setTimeout(r, 50)); // 等待 get preview_diff resolve

      // 验证并未直接调用 /api/tasks
      expect(axios.post).not.toHaveBeenCalledWith('/api/tasks', expect.any(Object));

      // 在 Diff 弹窗内查找“确认并部署”按钮
      const confirmBtn = wrapper.findAll('.diff-actions button').find(b => b.text().includes('确认并部署'));
      expect(confirmBtn).toBeTruthy();
      
      // 点击“确认并部署”
      if (confirmBtn) {
        console.log('Found confirm button');
        const btnDom = confirmBtn.element as HTMLElement;
        btnDom.click();
        
        await new Promise(r => setTimeout(r, 50));
        
        console.log('Axios calls:', (axios.post as any).mock.calls);

        // 验证真正发起了 post 请求
        expect(axios.post).toHaveBeenCalledWith('/api/tasks', expect.any(Object));
      }
    }
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

  it('6. 顶栏: [系统配置] openSettings 与 saveSettings', async () => {
    (axios.get as any).mockImplementation((url: string) => {
      if (url.includes('/git_binding')) {
        return Promise.resolve({ data: { restrict_git_authors: true, bound_git_authors: 'test@example.com' } });
      }
      return Promise.resolve({ data: [] });
    });
    (axios.put as any).mockResolvedValue({ data: {} });
    localStorage.setItem('role', 'admin');

    const wrapper = mount(Dashboard, {
      global: {
        plugins: [ElementPlus],
        mocks: { $router: { push: mockRouterPush } },
        stubs: {
          'el-table-column': true,
          'el-dialog': {
            template: '<div class="mock-dialog"><slot></slot><slot name="footer"></slot></div>'
          }
        }
      }
    });
    await new Promise(r => setTimeout(r, 50));

    const configBtn = wrapper.findAll('.el-button').find(b => b.text().includes('账号配置'));
    expect(configBtn).toBeTruthy();
    await configBtn?.trigger('click');
    await new Promise(r => setTimeout(r, 50));

    expect(axios.get).toHaveBeenCalledWith('/api/users/Admin/git_binding');

    const saveBtn = wrapper.findAll('.mock-dialog button').find(b => b.text().includes('保存'));
    expect(saveBtn).toBeTruthy();
    await saveBtn?.trigger('click');
    await new Promise(r => setTimeout(r, 50));

    expect(axios.put).toHaveBeenCalledWith('/api/users/Admin/git_binding', {
      restrict_git_authors: true,
      bound_git_authors: 'test@example.com'
    });
  });

  it('7. 侧边栏: [切换项目] selectProject 触发项目获取和详情加载', async () => {
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

    // Clear initial calls from onMounted selectProject(projects[0])
    vi.clearAllMocks();

    const projItems = wrapper.findAll('.project-item');
    expect(projItems.length).toBe(2);

    // Click the second project
    await projItems[1].trigger('click');
    await new Promise(r => setTimeout(r, 50));

    // Assert fetchHistory, fetchRefs, fetchCommits are called for proj-2
    expect(axios.get).toHaveBeenCalledWith('/api/tasks', expect.any(Object));
    expect(axios.get).toHaveBeenCalledWith('/api/projects/proj-2/refs');
    expect(axios.get).toHaveBeenCalledWith('/api/projects/proj-2/commits', expect.any(Object));
  });

  it('8. 历史记录: [查看日志]、[回滚] 动作断言', async () => {
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') {
        return Promise.resolve({ data: [{ id: 'p1', name: 'P1', environments: [{ id: 'e1', name: 'E1' }] }] });
      }
      if (url.includes('/log')) {
        return Promise.resolve({ data: { log: 'fake log' } });
      }
      return Promise.resolve({ data: [] });
    });
    (axios.post as any).mockResolvedValue({ data: {} });

    const wrapper = mount(Dashboard, {
      global: {
        plugins: [ElementPlus],
        mocks: { $router: { push: mockRouterPush } },
        stubs: {
          'el-table-column': {
            template: '<div class="mock-table-column"><slot :row="row"></slot></div>',
            data() {
              return {
                row: { id: 3, release_name: '20260527101500', commit_id: 'abcdef123', username: 'admin', status: 'success', created_at: new Date().toISOString() }
              };
            }
          }
        }
      }
    });
    await new Promise(r => setTimeout(r, 100));

    // Trigger Rollback
    const rollbackBtn = wrapper.findAll('.mock-table-column button').find(b => b.text().includes('回滚'));
    expect(rollbackBtn).toBeTruthy();
    await rollbackBtn?.trigger('click');
    await new Promise(r => setTimeout(r, 50));
    expect(axios.post).toHaveBeenCalledWith('/api/tasks/3/rollback');

    // Trigger Log (showLog)
    const logBtn = wrapper.findAll('.mock-table-column button').find(b => b.text().includes('日志'));
    expect(logBtn).toBeTruthy();
    await logBtn?.trigger('click');
    await new Promise(r => setTimeout(r, 50));
    expect(axios.get).toHaveBeenCalledWith('/api/tasks/3/log');
  });

  it('9. WebSocket 降级: 断开 mockWs 后轮询', async () => {
    const mockSend = vi.fn();
    const mockClose = vi.fn();
    let mockWsInstance: any = null;

    class MockWebSocket {
      url: string;
      onopen: (() => void) | null = null;
      onmessage: ((event: any) => void) | null = null;
      onerror: ((error: any) => void) | null = null;
      onclose: (() => void) | null = null;
      send = mockSend;
      close = mockClose;

      constructor(url: string) {
        this.url = url;
        mockWsInstance = this;
      }
    }
    vi.stubGlobal('WebSocket', MockWebSocket);

    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') {
        return Promise.resolve({ data: [{ id: 'p1', name: 'P1', environments: [{ id: 'e1', name: 'E1' }] }] });
      }
      if (url.includes('/api/tasks')) {
        return Promise.resolve({ data: [{ id: 3, status: 'pending', commit_id: 'abcdef123', release_name: '20260529' }] });
      }
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: {
        plugins: [ElementPlus],
        mocks: { $router: { push: mockRouterPush } },
        stubs: {
          'el-table-column': {
            template: '<div class="mock-table-column"><slot :row="row"></slot></div>',
            data() {
              return {
                row: { id: 3, release_name: '20260529', commit_id: 'abcdef123', username: 'admin', status: 'pending', created_at: new Date().toISOString() }
              };
            }
          }
        }
      }
    });
    // Wait for mounting to complete with real timers
    await new Promise(r => setTimeout(r, 100));

    // Turn on fake timers after mount/async actions complete
    vi.useFakeTimers();

    const logBtn = wrapper.findAll('.mock-table-column button').find(b => b.text().includes('日志'));
    expect(logBtn).toBeTruthy();
    await logBtn?.trigger('click');
    
    // Allow any microtasks inside the click handler to run
    await vi.advanceTimersByTimeAsync(10);

    expect(mockWsInstance).toBeTruthy();
    if (mockWsInstance) {
      mockWsInstance.onopen();
      expect(mockSend).toHaveBeenCalled();

      // Trigger closing the WebSocket (to simulate disconnect)
      mockWsInstance.onclose();

      // Clear axios calls to easily count new polling calls
      vi.clearAllMocks();

      // Fast-forward time by 1500ms to trigger the fallback polling
      await vi.advanceTimersByTimeAsync(1500);

      // Verify HTTP polling requests were made
      expect(axios.get).toHaveBeenCalledWith('/api/tasks/3/log');
      expect(axios.get).toHaveBeenCalledWith('/api/tasks/3');
    }

    vi.useRealTimers();
  });

  it('10. Diff 性能优化: 确保大文本被限制在 100KB 且不启用 lines 匹配防卡死', async () => {
    const hugeDiff = 'a'.repeat(250 * 1024); // 250KB 大文本
    const mockProjects = [{
      id: 'perf-proj',
      name: 'Perf Test',
      environments: [{ id: 'dev', name: 'Dev', default_branch: 'main' }]
    }];
    
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') return Promise.resolve({ data: mockProjects });
      if (url.includes('/preview_diff')) return Promise.resolve({ data: { diff: hugeDiff, files: ['src/large.go'] } });
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: { 
        plugins: [ElementPlus],
        stubs: { 
          'el-table-column': true,
          'el-dialog': {
            template: '<div class="mock-el-dialog"><slot></slot></div>'
          }
        }
      }
    });

    await new Promise(r => setTimeout(r, 80));

    // 选择项目
    const projItems = wrapper.findAll('.project-item');
    if (projItems.length > 0) {
      await projItems[0].trigger('click');
    }
    await new Promise(r => setTimeout(r, 50));

    // 触发预览
    const previewBtn = wrapper.findAll('.el-button').find(b => b.text().includes('预览 Diff'));
    expect(previewBtn).toBeTruthy();
    if (previewBtn) {
      await previewBtn.trigger('click');
      await new Promise(r => setTimeout(r, 50));

      // 模拟点击左侧文件列表行，触发单文件懒加载高亮渲染
      const vm = wrapper.vm as any;
      await vm.handleFileTreeNodeClick({ path: 'src/large.go' });
      await new Promise(r => setTimeout(r, 100));

      // 验证渲染时参数是否合理
      expect(mockHtml).toHaveBeenCalled();
      const lastCall = (mockHtml as any).mock.calls[(mockHtml as any).mock.calls.length - 1];
      
      // 断言：传入参数不能包含 lines 匹配，防死锁
      expect(lastCall[1].matching).not.toBe('lines');
      expect(lastCall[1].matching).not.toBe('words');

      // 断言：传入 diff2html 的文本长度应被截断限制，小于或等于 100KB 安全值 (100 * 1024) plus 截断提示的长度
      const passedText = lastCall[0];
      expect(passedText.length).toBeLessThan(120 * 1024); 
    }
  });

  it('11. 单文件懒加载 Diff 机制: 点击文件时应异步拉取该文件的单独差异且初次加载时不获取大 diff', async () => {
    const mockProjects = [{
      id: 'perf-proj',
      name: 'Perf Test',
      environments: [{ id: 'dev', name: 'Dev', default_branch: 'main' }]
    }];
    
    const requestParams: any[] = [];
    (axios.get as any).mockImplementation((url: string, config: any) => {
      if (url === '/api/projects') return Promise.resolve({ data: mockProjects });
      if (url.includes('/preview_diff')) {
        const params = config?.params || {};
        requestParams.push(params);
        if (params.file) {
          return Promise.resolve({ data: { diff: 'single file diff text', files: [] } });
        }
        // 初次加载不返回 diff
        return Promise.resolve({ data: { diff: '', files: ['src/large.go'] } });
      }
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: { 
        plugins: [ElementPlus],
        stubs: { 
          'el-table-column': true,
          'el-dialog': {
            template: '<div class="mock-el-dialog"><slot></slot></div>'
          }
        }
      }
    });

    await new Promise(r => setTimeout(r, 80));

    // 选择项目
    const projItems = wrapper.findAll('.project-item');
    if (projItems.length > 0) {
      await projItems[0].trigger('click');
    }
    await new Promise(r => setTimeout(r, 50));

    // 触发预览
    const previewBtn = wrapper.findAll('.el-button').find(b => b.text().includes('预览 Diff'));
    expect(previewBtn).toBeTruthy();
    if (previewBtn) {
      await previewBtn.trigger('click');
      await new Promise(r => setTimeout(r, 100)); // 等待初次加载及自动高亮第一个文件的异步请求完成

      // 验证初次加载时拉取了两次：第一次无 file 参数获取列表，第二次自动获取第一个文件 diff
      expect(requestParams.length).toBeGreaterThanOrEqual(1);
      expect(requestParams[0].file).toBeUndefined(); // 第一次无 file，说明性能好，只拉树

      const vm = wrapper.vm as any;
      expect(typeof vm.handleFileTreeNodeClick).toBe('function');
      
      // 清理历史，模拟手动点击行
      requestParams.length = 0;
      await vm.handleFileTreeNodeClick({ path: 'src/large.go' });
      await new Promise(r => setTimeout(r, 100));

      // 断言：点击行之后应该去拉取了带 file 参数的单个 diff
      expect(requestParams.length).toBe(1);
      expect(requestParams[0].file).toBe('src/large.go');
    }
  });

  it('12. 文件过滤联动: 确保切换环境或分支不会自动请求，需手动触发预览', async () => {
    const mockProjects = [{
      id: 'proj-a',
      name: 'Project A',
      environments: [
        { id: 'dev', name: 'Dev', default_branch: 'main' },
        { id: 'prod', name: 'Prod', default_branch: 'main' }
      ]
    }];
    
    let previewDiffCallCount = 0;
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') return Promise.resolve({ data: mockProjects });
      if (url.includes('/preview_diff')) {
        previewDiffCallCount++;
        return Promise.resolve({ data: { diff: '', files: ['src/a.go'] } });
      }
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: {
        plugins: [ElementPlus],
        stubs: { 'el-table-column': true }
      }
    });

    await new Promise(r => setTimeout(r, 80));

    // 切换到 prod 环境，不再自动触发 preview_diff
    const vm = wrapper.vm as any;
    vm.activeEnvTab = 'prod';
    
    await new Promise(r => setTimeout(r, 80));

    // 切换环境后不应有自动请求
    expect(previewDiffCallCount).toBe(0);
  });

  it('13. 弹窗文件树渲染: 确保 previewDeployDiff 和 showDiff 正确填充 fileTreeData', async () => {
    const mockProjects = [{
      id: 'proj-a',
      name: 'Project A',
      environments: [{ id: 'dev', name: 'Dev', default_branch: 'main' }]
    }];
    
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') return Promise.resolve({ data: mockProjects });
      if (url.includes('/preview_diff')) {
        return Promise.resolve({ data: { diff: '', files: ['src/a.go', 'src/b/c.go'] } });
      }
      if (url.includes('/api/tasks/')) {
        return Promise.resolve({ data: { diff: '', files: 'M\tsrc/a.go\nA\tsrc/d.go' } });
      }
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: {
        plugins: [ElementPlus],
        stubs: { 
          'el-table-column': true,
          'el-dialog': {
            template: '<div class="mock-dialog"><slot></slot></div>'
          }
        }
      }
    });

    await new Promise(r => setTimeout(r, 80));
    const vm = wrapper.vm as any;

    // 1. 模拟预览触发
    await vm.previewDeployDiff({ id: 'dev', name: 'Dev' });
    await new Promise(r => setTimeout(r, 50));
    
    // 断言：previewDeployDiff 应该把数据填充进 fileTreeData
    expect(vm.fileTreeData.length).toBeGreaterThan(0);
    expect(vm.fileTreeData[0].label).toBe('src');

    // 清空 fileTreeData 方便下一步测试
    vm.fileTreeData = [];

    // 2. 模拟历史 Diff 触发
    await vm.showDiff({ id: 100, commit_id: '12345678', release_name: 'test-release' });
    await new Promise(r => setTimeout(r, 50));

    // 断言：showDiff 应该把数据填充进 fileTreeData
    expect(vm.fileTreeData.length).toBeGreaterThan(0);
    expect(vm.fileTreeData[0].label).toBe('src');
  });

  it('14. Live Diff 禁用状态 Tooltip 提示: 针对非 commit 的历史任务，悬浮在 Live Diff 按钮上时应正确展示 Tooltip 说明', async () => {
    const mockProjects = [{
      id: 'tooltip-proj',
      name: 'Tooltip Project',
      environments: [{ id: 'dev', name: 'Dev', default_branch: 'main' }]
    }];
    
    (axios.get as any).mockImplementation((url: string) => {
      if (url === '/api/projects') return Promise.resolve({ data: mockProjects });
      if (url.includes('/preview_diff')) {
        return Promise.resolve({ data: { diff: '', files: ['src/a.go'] } });
      }
      return Promise.resolve({ data: [] });
    });

    const wrapper = mount(Dashboard, {
      global: { 
        plugins: [ElementPlus],
        stubs: { 
          'el-table-column': true,
          'el-dialog': {
            template: '<div class="mock-el-dialog"><slot></slot></div>'
          }
        }
      }
    });

    await new Promise(r => setTimeout(r, 80));

    const vm = wrapper.vm as any;
    // 模拟查看非 commit (比如 branch) 的历史部署任务
    vm.activeTask = {
      id: 999,
      project_id: 'tooltip-proj',
      env_id: 'dev',
      target_type: 'branch', // 非 commit，会触发禁用
      status: 'success'
    };
    vm.isPreDeploying = false; // 处于历史记录查看阶段
    vm.diffVisible = true; // 显示比对弹窗

    await wrapper.vm.$nextTick();
    await new Promise(r => setTimeout(r, 50));

    // 查找 tooltip 组件
    const liveDiffRadio = wrapper.findAllComponents({ name: 'ElRadioButton' }).find(r => r.props('value') === 'live');
    expect(liveDiffRadio).toBeTruthy();
    if (liveDiffRadio) {
      const tooltip = liveDiffRadio.findComponent({ name: 'ElTooltip' });
      expect(tooltip.exists()).toBe(true);
      expect(tooltip.props('content')).toContain('无 Live Diff 归档');
      expect(tooltip.props('disabled')).toBe(false); // 此时应该启用 tooltip 说明
    }
  });
});



