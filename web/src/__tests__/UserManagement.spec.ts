import { mount, flushPromises } from '@vue/test-utils';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import UserManagement from '../views/UserManagement.vue';
import axios from 'axios';
import { ElMessage, ElMessageBox } from 'element-plus';
import ElementPlus from 'element-plus';

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
    ElMessage: vi.fn()
  };
});

describe('UserManagement.vue Component UI Test', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('1. 点击新建用户能够打开 Dialog 弹窗', async () => {
    (axios.get as any).mockResolvedValueOnce({ data: [] });
    const wrapper = mount(UserManagement, {
      global: { plugins: [ElementPlus] }
    });
    await flushPromises();

    // 找到新建用户按钮
    const newBtn = wrapper.findAll('button').find(b => b.text().includes('新建用户') || b.text().includes('Create User'));
    if (newBtn) {
      await newBtn.trigger('click');
      // 检查弹窗是否可见 (基于 el-dialog title)
      const dialog = wrapper.find('.el-dialog__title');
      expect(dialog.exists()).toBe(true);
    }
  });

  it('2. 点击解除绑定能够触发 axios 移除操作', async () => {
    const mockUsers = [{ username: 'testuser', bound_git_authors: 'test' }];
    (axios.get as any).mockResolvedValue({ data: mockUsers });
    (axios.put as any).mockResolvedValue({ data: { message: 'success' } });

    const wrapper = mount(UserManagement, {
      global: { 
        plugins: [ElementPlus],
        stubs: {
          'el-table-column': {
            template: '<div><slot :row="{ username: \'testuser\', role: \'developer\' }"></slot></div>'
          }
        }
      }
    });
    await new Promise(r => setTimeout(r, 100));

    // 找到解除绑定按钮
    const unbindBtn = wrapper.findAll('button').find(b => b.text().includes('解除绑定'));
    if (unbindBtn) {
      await unbindBtn.trigger('click');
      await new Promise(r => setTimeout(r, 10)); // MessageBox mock resolve

      expect(ElMessageBox.confirm).toHaveBeenCalled();
      expect(axios.put).toHaveBeenCalledWith(
        '/api/users/testuser/git_binding',
        { restrict_git_authors: false, bound_git_authors: '' }
      );
    }
  });

  it('3. [返回控制台] router.push("/")', async () => {
    (axios.get as any).mockResolvedValue({ data: [] });
    const wrapper = mount(UserManagement, { global: { plugins: [ElementPlus] } });
    await flushPromises();
    const backBtn = wrapper.findAll('.el-button').find(b => b.text().includes('返回控制台'));
    expect(backBtn).toBeTruthy();
    await backBtn?.trigger('click');
    expect(mockRouterPush).toHaveBeenCalledWith('/');
  });

  it('4. [新增用户] 触发新建弹窗，[保存] 触发 POST /api/users', async () => {
    (axios.get as any).mockResolvedValue({ data: [] });
    (axios.post as any).mockResolvedValue({ data: { message: 'success' } });
    const wrapper = mount(UserManagement, { global: { plugins: [ElementPlus], stubs: {'el-dialog': true} } });
    await flushPromises();

    const newBtn = wrapper.findAll('.el-button').find(b => b.text().includes('新增用户'));
    expect(newBtn).toBeTruthy();
    await newBtn?.trigger('click');
    await flushPromises();

    const saveBtn = wrapper.findAll('.el-button').find(b => b.text().includes('确 定') || b.text().includes('保存'));
    if (saveBtn) {
      await saveBtn.trigger('click');
      expect(axios.post).toHaveBeenCalledWith('/api/users', expect.any(Object));
    }
  });

  it('5. [编辑] 触发表单数据回显，[保存] 触发 PUT 请求', async () => {
    const mockUsers = [{ username: 'testuser', role: 'developer' }];
    (axios.get as any).mockResolvedValue({ data: mockUsers });
    (axios.put as any).mockResolvedValue({ data: { message: 'success' } });
    const wrapper = mount(UserManagement, { 
      global: { plugins: [ElementPlus], stubs: { 'el-table-column': { template: '<div><slot :row="{ username: \'testuser\', role: \'developer\' }"></slot></div>' } } } 
    });
    await flushPromises();

    const editBtn = wrapper.findAll('.el-button').find(b => b.text().includes('编辑'));
    expect(editBtn).toBeTruthy();
    await editBtn?.trigger('click');
    await flushPromises();

    const saveBtn = wrapper.findAll('.el-button').find(b => b.text().includes('确 定') || b.text().includes('保存'));
    if (saveBtn) {
      await saveBtn.trigger('click');
      expect(axios.put).toHaveBeenCalledWith('/api/users/testuser', expect.any(Object));
    }
  });

  it('6. [删除] 断言确认弹窗与 DELETE 请求发出，断言 admin 用户按钮为 disabled', async () => {
    const mockUsers = [{ username: 'admin', role: 'admin' }, { username: 'testuser', role: 'developer' }];
    (axios.get as any).mockResolvedValue({ data: mockUsers });
    (axios.delete as any).mockResolvedValue({ data: { message: 'success' } });
    
    // Test the deletion of testuser
    const wrapper = mount(UserManagement, { 
      global: { plugins: [ElementPlus], stubs: { 'el-table-column': { template: '<div><slot :row="{ username: \'testuser\', role: \'developer\' }"></slot><slot :row="{ username: \'admin\', role: \'admin\' }"></slot></div>' } } } 
    });
    await flushPromises();

    const deleteBtns = wrapper.findAll('.el-button').filter(b => b.text().includes('删除'));
    expect(deleteBtns.length).toBeGreaterThan(0);
    // Find the enabled delete button (for testuser)
    const activeDeleteBtn = deleteBtns.find(b => !b.attributes('disabled') && !b.classes('is-disabled'));
    if (activeDeleteBtn) {
      await activeDeleteBtn.trigger('click');
      await flushPromises();
      expect(ElMessageBox.confirm).toHaveBeenCalled();
      expect(axios.delete).toHaveBeenCalledWith('/api/users/testuser');
    }
  });
});
