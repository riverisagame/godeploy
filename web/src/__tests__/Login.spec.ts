import { mount } from '@vue/test-utils';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import Login from '../views/Login.vue';
import axios from 'axios';
import { useRouter } from 'vue-router';
import ElementPlus from 'element-plus';

// 严格 Mock，实现零污染
vi.mock('axios');
vi.mock('vue-router', () => ({
  useRouter: vi.fn()
}));

describe('Login.vue Component UI Test', () => {
  const mockPush = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
    (useRouter as any).mockReturnValue({
      push: mockPush
    });
  });

  it('1. 阻止空表单提交: 点击登录按钮但不输入时，不应当调用 API', async () => {
    const wrapper = mount(Login, {
      global: { plugins: [ElementPlus] }
    });
    // 查找包含“登录”文字的按钮并触发点击
    const btn = wrapper.find('button');
    await btn.trigger('click');
    
    // axios.post 不应该被调用，因为表单验证拦截了
    expect(axios.post).not.toHaveBeenCalled();
  });

  it('2. 正确填写后点击能够触发后端接口跳转', async () => {
    (axios.post as any).mockResolvedValueOnce({ data: { token: 'mock-token' } });
    const wrapper = mount(Login, {
      global: { plugins: [ElementPlus] }
    });

    // 设置合法账号密码
    await wrapper.find('input[type="text"]').setValue('admin');
    await wrapper.find('input[type="password"]').setValue('admin123');

    // 触发登录
    const btn = wrapper.find('button');
    await btn.trigger('click');

    // 由于 element-plus form validate 是异步的，需要等待 nextTick 或 flushPromises
    await new Promise(resolve => setTimeout(resolve, 0));

    // 断言 axios 成功触发
    expect(axios.post).toHaveBeenCalledWith('/api/login', {
      username: 'admin',
      password: 'admin123'
    });

    // 验证 router 是否发生了跳转
    expect(mockPush).toHaveBeenCalledWith('/');
  });
});
