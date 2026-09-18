const TOKEN_KEY = 'itdb.auth.token';
const USER_KEY = 'itdb.auth.user';
const EXPIRES_AT_KEY = 'itdb.auth.expires_at';
export const AUTH_SESSION_CHANGED_EVENT = 'itdb:auth-session-changed';

export type AuthUser = {
  id: string;
  username: string;
  displayName?: string;
  role: string;
  source: 'local' | 'ldap' | 'wecom';
  permissions: string[];
  wecomBound?: boolean;
};

export type AuthSession = {
  token: string;
  expiresAt: string;
  user: AuthUser;
};

export type PublicAuthProvider = {
  id: string;
  type: string;
  name: string;
  enabled: boolean;
};

type ApiErrorResponse = {
  error?: string;
  message?: string;
};

type ApiOptions = RequestInit & {
  auth?: boolean;
  redirectOn401?: boolean;
};

// WECOM_BIND_MESSAGE 绑定弹窗通过 postMessage 通知主窗口绑定结果的事件类型
export const WECOM_BIND_MESSAGE = 'itdb:wecom-bind';

type LogoutOptions = {
  waitForRemote?: boolean;
};

const CURRENT_USER_CACHE_TTL_MS = 1500;
const PUBLIC_AUTH_PROVIDERS_CACHE_TTL_MS = 1000;

export type PublicAuthConfiguration = {
  items: PublicAuthProvider[];
  total: number;
  passwordResetEnabled: boolean;
};

type AuthApiUser = {
  id: number;
  username: string;
  userType: number;
  userDesc?: string;
  displayName?: string;
  role?: string;
  source?: 'local' | 'ldap' | 'wecom';
  wecomBound?: boolean;
  permissions?: string[];
};

type AuthLoginResponse = {
  token: string;
  user: AuthApiUser;
};

let pendingCurrentUser: Promise<AuthUser> | null = null;
let cachedCurrentUser: { user: AuthUser; expiresAt: number } | null = null;
let pendingPublicAuthProviders: Promise<PublicAuthConfiguration> | null = null;
let cachedPublicAuthProviders: { value: PublicAuthConfiguration; expiresAt: number } | null = null;

function storage() {
  if (typeof window === 'undefined') return null;
  return window.localStorage;
}

function emitSessionChanged() {
  if (typeof window === 'undefined') return;
  window.dispatchEvent(new Event(AUTH_SESSION_CHANGED_EVENT));
}

export function getAuthToken() {
  return storage()?.getItem(TOKEN_KEY) ?? '';
}

export function getStoredUser(): AuthUser | null {
  const raw = storage()?.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    storage()?.removeItem(USER_KEY);
    return null;
  }
}

export function persistSession(session: AuthSession) {
  const store = storage();
  if (!store) return;
  store.setItem(TOKEN_KEY, session.token);
  store.setItem(USER_KEY, JSON.stringify(session.user));
  store.setItem(EXPIRES_AT_KEY, session.expiresAt);
  emitSessionChanged();
}

export function clearSession() {
  const store = storage();
  if (!store) return;
  store.removeItem(TOKEN_KEY);
  store.removeItem(USER_KEY);
  store.removeItem(EXPIRES_AT_KEY);
  emitSessionChanged();
}

export function userHasPermission(user: AuthUser | null, permission: string) {
  if (!user) return false;
  if (user.role === 'admin') return true;
  return user.permissions?.includes(permission) ?? false;
}

export function userHasAnyPermission(user: AuthUser | null, permissions: string[]) {
  return permissions.some(permission => userHasPermission(user, permission));
}

/* ApiError 携带后端 HTTP 状态码，供调用方区分权限不足（403）等场景做静默降级 */
export class ApiError extends Error {
  status: number;

  constructor(message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
  }
}

export function isApiStatus(error: unknown, status: number) {
  return error instanceof ApiError && error.status === status;
}

async function readApiError(response: Response) {
  try {
    const body = (await response.json()) as ApiErrorResponse;
    return normalizeApiErrorMessage(body.message || body.error || `请求失败：${response.status}`);
  } catch {
    return `请求失败：${response.status}`;
  }
}

function normalizeApiErrorMessage(message: string) {
  return message;
}

export function persistUser(user: AuthUser) {
  storage()?.setItem(USER_KEY, JSON.stringify(user));
}

export function setCurrentUserSnapshot(user: AuthUser) {
  cachedCurrentUser = { user, expiresAt: Date.now() + CURRENT_USER_CACHE_TTL_MS };
  persistUser(user);
}

export async function api<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const headers = new Headers(options.headers);
  if (options.body && !(options.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json; charset=utf-8');
  }
  const token = getAuthToken();
  if (options.auth !== false && token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  let response: Response;
  try {
    response = await fetch(path, { ...options, headers });
  } catch {
    throw new Error('无法连接后端服务');
  }
  if (!response.ok) {
    const message = await readApiError(response);
    if (options.auth !== false && options.redirectOn401 !== false && response.status === 401) {
      clearSession();
      if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
        window.location.assign('/login');
      }
    }
    throw new ApiError(message, response.status);
  }
  if (response.status === 204) {
    return undefined as T;
  }
  return response.json() as Promise<T>;
}

export async function apiBlob(path: string): Promise<Blob> {
  const headers = new Headers();
  const token = getAuthToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);
  let response: Response;
  try {
    response = await fetch(path, { headers });
  } catch {
    throw new Error('无法连接后端服务');
  }
  if (!response.ok) {
    const message = await readApiError(response);
    if (response.status === 401) {
      clearSession();
      if (typeof window !== 'undefined' && window.location.pathname !== '/login') {
        window.location.assign('/login');
      }
    }
    throw new Error(message);
  }
  return response.blob();
}

export async function login(username: string, password: string, provider = 'local') {
  const response = await api<AuthLoginResponse>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({
      username,
      password,
      mode: provider === 'local' ? 'local' : 'ldap',
    }),
  });
  const session: AuthSession = {
    token: response.token,
    expiresAt: new Date(Date.now() + 48 * 60 * 60 * 1000).toISOString(),
    user: normalizeAuthUser(response.user),
  };
  persistSession(session);
  setCurrentUserSnapshot(session.user);
  return session;
}

// loginWithWecomCallback 企业微信扫码回调换取会话并持久化
export async function loginWithWecomCallback(code: string, state: string) {
  const response = await api<AuthLoginResponse>('/api/auth/wecom/callback', {
    method: 'POST',
    body: JSON.stringify({ code, state }),
  });
  const session: AuthSession = {
    token: response.token,
    expiresAt: new Date(Date.now() + 48 * 60 * 60 * 1000).toISOString(),
    user: normalizeAuthUser(response.user),
  };
  persistSession(session);
  setCurrentUserSnapshot(session.user);
  return session;
}

// fetchWecomLoginUrl 获取企业微信扫码登录页地址（公开接口）
export async function fetchWecomLoginUrl() {
  const response = await api<{ url: string }>('/api/auth/wecom/authorize');
  return response.url;
}

// fetchWecomBindUrl 获取当前用户的企业微信绑定扫码地址
export async function fetchWecomBindUrl() {
  const response = await api<{ url: string }>('/api/auth/wecom/bind-url');
  return response.url;
}

// bindWecomCallback 完成当前登录用户的企微绑定（绑定弹窗内调用；401 不触发会话清理跳转）
export async function bindWecomCallback(code: string, state: string) {
  return api<{ ok: boolean; wecomUserid: string }>('/api/auth/wecom/bind', {
    method: 'POST',
    body: JSON.stringify({ code, state }),
    redirectOn401: false,
  });
}

// unbindWecom 解除当前登录用户的企微绑定
export async function unbindWecom() {
  return api<{ ok: boolean }>('/api/auth/wecom/bind', { method: 'DELETE' });
}

// WECOM_PROVIDERS_CHANGED_EVENT 企业微信认证启用状态变更事件，通知布局刷新绑定入口
export const WECOM_PROVIDERS_CHANGED_EVENT = 'itdb:wecom-providers-changed';

export function fetchPublicAuthProviders(options?: { force?: boolean }) {
  if (!options?.force) {
    if (cachedPublicAuthProviders && cachedPublicAuthProviders.expiresAt > Date.now()) {
      return Promise.resolve(cachedPublicAuthProviders.value);
    }
    if (pendingPublicAuthProviders) return pendingPublicAuthProviders;
  }
  pendingPublicAuthProviders = api<PublicAuthConfiguration>('/api/auth/providers', { auth: false })
    .then(value => {
      cachedPublicAuthProviders = {
        value,
        expiresAt: Date.now() + PUBLIC_AUTH_PROVIDERS_CACHE_TTL_MS,
      };
      return value;
    })
    .finally(() => {
      pendingPublicAuthProviders = null;
    });
  return pendingPublicAuthProviders;
}

export function fetchCurrentUser() {
  if (cachedCurrentUser && cachedCurrentUser.expiresAt > Date.now()) {
    return Promise.resolve(cachedCurrentUser.user);
  }
  if (pendingCurrentUser) return pendingCurrentUser;
  pendingCurrentUser = api<AuthApiUser>('/api/auth/me')
    .then(response => {
      const user = normalizeAuthUser(response);
      cachedCurrentUser = { user, expiresAt: Date.now() + CURRENT_USER_CACHE_TTL_MS };
      persistUser(user);
      return user;
    })
    .finally(() => {
      pendingCurrentUser = null;
    });
  return pendingCurrentUser;
}

export async function logout(options: LogoutOptions = {}) {
  await notifyServerLogout(options.waitForRemote);
  pendingCurrentUser = null;
  cachedCurrentUser = null;
  clearSession();
}

/* notifyServerLogout 通知后端记录用户注销审计：须在清除本地会话前调用以携带有效令牌；
   请求失败静默忽略，注销流程不被网络问题阻塞 */
function notifyServerLogout(waitForRemote?: boolean): Promise<void> {
  const token = getAuthToken();
  if (!token) return Promise.resolve();
  const request = fetch('/api/auth/logout', {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
    keepalive: true,
  }).catch(() => undefined);
  if (!waitForRemote) return Promise.resolve();
  return request.then(() => undefined);
}

export type PasswordResetCaptcha = {
  token: string;
  question: string;
  expiresAt: string;
};

export type PasswordResetChannel = {
  id: string;
  name: string;
  maskedTo: string;
  requiresTo: boolean;
};

export const fetchPasswordResetCaptcha = () =>
  api<PasswordResetCaptcha>('/api/auth/password-reset/captcha', { auth: false });

export const verifyPasswordResetIdentity = (body: {
  username: string;
  captchaToken: string;
  captchaAnswer: string;
}) =>
  api<{ verificationToken: string; channels: PasswordResetChannel[] }>(
    '/api/auth/password-reset/verify',
    {
      method: 'POST',
      auth: false,
      body: JSON.stringify(body),
    }
  );

export const sendPasswordResetCode = (body: {
  username: string;
  verificationToken: string;
  channel: string;
  verifyEmail: string;
}) =>
  api<{ status: string; cooldownSeconds: number }>('/api/auth/password-reset/send', {
    method: 'POST',
    auth: false,
    body: JSON.stringify(body),
  });

export const confirmPasswordReset = (body: {
  username: string;
  verificationToken: string;
  code: string;
  newPassword: string;
  confirmPassword: string;
}) =>
  api<{ status: string }>('/api/auth/password-reset/confirm', {
    method: 'POST',
    auth: false,
    body: JSON.stringify(body),
  });

export function changePassword(body: {
  old_password: string;
  new_password: string;
  confirm_password: string;
}) {
  return api<{ status: string }>('/api/auth/change-password', {
    method: 'POST',
    body: JSON.stringify(body),
  });
}

function normalizeAuthUser(user: AuthApiUser): AuthUser {
  const isAdmin = user.userType === 0 || user.username.toLowerCase() === 'admin';
  return {
    id: String(user.id),
    username: user.username,
    displayName: user.displayName?.trim() || user.userDesc?.trim() || user.username,
    role: user.role || (isAdmin ? 'admin' : 'viewer'),
    source: user.source || 'local',
    permissions: user.permissions ?? [],
    wecomBound: user.wecomBound ?? false,
  };
}
