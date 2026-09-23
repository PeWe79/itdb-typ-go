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

// storage 登录态统一存取 localStorage，使同一浏览器内新建标签页与重启浏览器后仍保持登录，登录态仅随服务端会话过期失效
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

// sessionExpiresAt 解析 JWT 的 exp 声明作为本地会话过期时间，解析失败按后端默认 12 小时估算
function sessionExpiresAt(token: string): string {
  const fallback = new Date(Date.now() + 12 * 60 * 60 * 1000).toISOString();
  try {
    const payloadPart = token.split('.')[1] ?? '';
    const bytes = Uint8Array.from(atob(payloadPart.replace(/-/g, '+').replace(/_/g, '/')), char =>
      char.charCodeAt(0)
    );
    const claims = JSON.parse(new TextDecoder().decode(bytes)) as { exp?: number };
    if (typeof claims.exp === 'number' && claims.exp > 0) {
      return new Date(claims.exp * 1000).toISOString();
    }
  } catch {
    return fallback;
  }
  return fallback;
}

// persistLoginResponse 将登录类接口返回的令牌与用户信息持久化为本地会话
function persistLoginResponse(response: AuthLoginResponse): AuthSession {
  const session: AuthSession = {
    token: response.token,
    expiresAt: sessionExpiresAt(response.token),
    user: normalizeAuthUser(response.user),
  };
  persistSession(session);
  setCurrentUserSnapshot(session.user);
  return session;
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
  return persistLoginResponse(response);
}

// loginWithWecomCallback 企业微信直连扫码回调换取会话并持久化
export async function loginWithWecomCallback(code: string, state: string) {
  const response = await api<AuthLoginResponse>('/api/auth/wecom/callback', {
    method: 'POST',
    body: JSON.stringify({ code, state }),
  });
  return persistLoginResponse(response);
}

// loginWithWecomSSO 统一认证中心回跳 ticket 换取会话并持久化
export async function loginWithWecomSSO(ticket: string) {
  const response = await api<AuthLoginResponse>('/api/auth/wecom/sso/callback', {
    method: 'POST',
    body: JSON.stringify({ ticket }),
  });
  return persistLoginResponse(response);
}

// WecomAuthorizeEmbed 内嵌二维码登录参数：iframe 地址、回跳路径与直连模式签名 state
export type WecomAuthorizeEmbed = {
  auth_mode: 'direct' | 'sso';
  iframe_url: string;
  state?: string;
  callback_path: string;
};

// fetchWecomAuthorize 获取企业微信扫码登录跳转地址与内嵌二维码参数（公开接口）
export async function fetchWecomAuthorize() {
  return api<{ url: string; embed?: WecomAuthorizeEmbed }>('/api/auth/wecom/authorize');
}

// fetchWecomLoginUrl 获取企业微信扫码登录页地址（公开接口，整页跳转降级用）
export async function fetchWecomLoginUrl() {
  const response = await fetchWecomAuthorize();
  return response.url;
}

// fetchWecomBindUrl 获取当前用户的企业微信绑定扫码地址
export async function fetchWecomBindUrl() {
  const response = await api<{ url: string }>('/api/auth/wecom/bind-url');
  return response.url;
}

// bindWecomCallback 完成当前登录用户的企微绑定（直连模式，绑定弹窗内调用；401 不触发会话清理跳转）
export async function bindWecomCallback(code: string, state: string) {
  return api<{ ok: boolean; wecomUserid: string }>('/api/auth/wecom/bind', {
    method: 'POST',
    body: JSON.stringify({ code, state }),
    redirectOn401: false,
  });
}

// bindWecomSSO 完成当前登录用户的企微绑定（统一认证中心模式，绑定弹窗内调用；401 不触发会话清理跳转）
export function bindWecomSSO(ticket: string) {
  return api<{ ok: boolean; wecomUserid: string }>('/api/auth/wecom/sso/bind', {
    method: 'POST',
    body: JSON.stringify({ ticket }),
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

// fetchCurrentUser 获取当前用户（带 1.5 秒缓存；绑定状态变更等需要立即反映的场景传 force 绕过缓存）
export function fetchCurrentUser(options?: { force?: boolean }) {
  if (!options?.force) {
    if (cachedCurrentUser && cachedCurrentUser.expiresAt > Date.now()) {
      return Promise.resolve(cachedCurrentUser.user);
    }
    if (pendingCurrentUser) return pendingCurrentUser;
  }
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

// logout 先清理本地会话再通知后端：退出引发的组件重挂载若触发重新请求，令牌已不在本地，
// 不会携带"服务端已注销"的旧令牌而误报会话过期；注销请求失败静默忽略，不阻塞退出流程
export async function logout(options: LogoutOptions = {}) {
  const token = getAuthToken();
  pendingCurrentUser = null;
  cachedCurrentUser = null;
  clearSession();
  if (!token) return;

  const request = fetch('/api/auth/logout', {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
    keepalive: true,
  }).catch(() => undefined);

  if (options.waitForRemote === false) {
    void request;
    return;
  }
  await request.then(() => undefined);
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
