export interface UserInfo {
  user_id: number;
  username: string;
  role: string;
  exp: number;
}

export const parseToken = (token: string): UserInfo | null => {
  if (!token) return null;
  const parts = token.split('.');
  if (parts.length !== 3) return null;
  try {
    const payload = parts[1];
    const base64 = payload.replace(/-/g, '+').replace(/_/g, '/');
    const jsonPayload = decodeURIComponent(
      window.atob(base64).split('').map(function (c) {
        return '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2);
      }).join('')
    );
    return JSON.parse(jsonPayload) as UserInfo;
  } catch (e) {
    console.error('Failed to parse JWT token', e);
    return null;
  }
};

export const getUserInfo = (): UserInfo | null => {
  const token = localStorage.getItem('token');
  if (!token) return null;
  return parseToken(token);
};

export const isAdmin = (): boolean => {
  const userInfo = getUserInfo();
  return userInfo !== null && userInfo.role === 'admin';
};
