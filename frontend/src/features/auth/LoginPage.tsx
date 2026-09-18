import { Link, useNavigate } from '@tanstack/react-router';
import { Eye, EyeOff, Loader2, Lock, Moon, QrCode, Sun, User } from 'lucide-react';
import { type FormEvent, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { BrandIcon } from '@/components/brand-mark';
import { useBrandSettings } from '@/lib/branding';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  bindWecomCallback,
  fetchCurrentUser,
  fetchPublicAuthProviders,
  fetchWecomLoginUrl,
  getAuthToken,
  login,
  loginWithWecomCallback,
  persistUser,
  WECOM_BIND_MESSAGE,
  type PublicAuthProvider,
} from '@/lib/auth';
import {
  applyTheme,
  getInitialTheme,
  persistTheme,
  toggleTheme,
  type ItdbTheme,
} from '@/lib/utils';

export function LoginPage() {
  const navigate = useNavigate();
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [provider, setProvider] = useState('local');
  const [providers, setProviders] = useState<PublicAuthProvider[]>([]);
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [theme, setTheme] = useState<ItdbTheme>(getInitialTheme);
  const brand = useBrandSettings();
  const toggleLabel = theme === 'dark' ? '切换浅色背景' : '切换深色背景';
  const callbackParams = new URLSearchParams(
    typeof window === 'undefined' ? '' : window.location.search
  );
  const callbackCode = callbackParams.get('code') ?? '';
  const callbackState = callbackParams.get('state') ?? '';
  const isWecomProvider = provider === 'wecom';

  useEffect(() => {
    applyTheme(theme);
    persistTheme(theme);
  }, [theme]);

  useEffect(() => {
    if (callbackCode && callbackState) return;
    let cancelled = false;
    if (getAuthToken()) {
      void fetchCurrentUser()
        .then(user => {
          if (cancelled) return;
          persistUser(user);
          void navigate({ to: '/', replace: true });
        })
        .catch(() => undefined);
    }
    return () => {
      cancelled = true;
    };
  }, [callbackCode, callbackState, navigate]);

  useEffect(() => {
    let cancelled = false;
    fetchPublicAuthProviders()
      .then(response => {
        if (cancelled) return;
        setProviders(
          response.items.filter(
            item => item.enabled && (item.type === 'ldap' || item.type === 'wecom')
          )
        );
      })
      .catch(() => {
        if (!cancelled) {
          setProviders([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    if (!callbackCode || !callbackState) return;
    let cancelled = false;
    setLoading(true);
    (async () => {
      try {
        if (getAuthToken()) {
          await bindWecomCallback(callbackCode, callbackState);
          window.opener?.postMessage(
            { type: WECOM_BIND_MESSAGE, ok: true },
            window.location.origin
          );
          toast.success('企业微信绑定成功');
          window.history.replaceState({}, '', window.location.pathname);
          window.setTimeout(() => window.close(), 300);
          return;
        }
        const session = await loginWithWecomCallback(callbackCode, callbackState);
        if (cancelled) return;
        toast.success(`欢迎回来，${session.user.displayName || session.user.username}`);
        window.history.replaceState({}, '', window.location.pathname);
        void navigate({ to: '/', replace: true });
        return;
      } catch (err) {
        const message = err instanceof Error ? err.message : '企业微信登录失败，请稍后重试';
        if (!cancelled) setError(message);
        if (getAuthToken()) {
          window.opener?.postMessage(
            { type: WECOM_BIND_MESSAGE, ok: false, message },
            window.location.origin
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [callbackCode, callbackState, navigate]);

  useEffect(() => {
    const handlePageShow = (event: PageTransitionEvent) => {
      if (!event.persisted) return;
      if (callbackCode && callbackState) {
        window.location.replace('/login');
        return;
      }
      setLoading(false);
    };
    window.addEventListener('pageshow', handlePageShow);
    return () => window.removeEventListener('pageshow', handlePageShow);
  }, [callbackCode, callbackState]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (isWecomProvider) {
      await startWecomLogin();
      return;
    }
    const normalizedUsername = username.trim();
    if (!normalizedUsername || !password) {
      setError('用户名或密码不能为空');
      return;
    }
    setLoading(true);
    setError('');
    try {
      const session = await login(normalizedUsername, password, provider);
      toast.success(`欢迎回来，${session.user.displayName || session.user.username}`);
      void navigate({ to: '/', replace: true });
    } catch (err) {
      setError(err instanceof Error ? err.message : '登录失败，请稍后重试');
    } finally {
      setLoading(false);
    }
  }

  async function startWecomLogin() {
    setLoading(true);
    setError('');
    try {
      const url = await fetchWecomLoginUrl();
      window.location.assign(url);
      window.setTimeout(() => setLoading(false), 4000);
    } catch (err) {
      setError(err instanceof Error ? err.message : '获取企业微信登录地址失败');
      setLoading(false);
    }
  }

  if (callbackCode && callbackState) {
    return (
      <main
        data-cmp="Login"
        className="relative flex min-h-dvh items-center justify-center overflow-hidden px-4 py-8 sm:px-6"
        style={{
          background:
            'radial-gradient(circle at 50% 0%, rgba(59,130,246,0.24), transparent 30%), var(--itdb-login-bg)',
          color: 'var(--itdb-text)',
        }}
      >
        <div className="itdb-login-grid absolute inset-0" aria-hidden="true" />
        <section
          className="relative z-10 flex w-full max-w-[420px] flex-col items-center gap-4 rounded-[24px] p-8 text-center"
          style={{
            background: 'var(--itdb-login-panel-bg)',
            border: '1px solid var(--itdb-border)',
            backdropFilter: 'blur(18px)',
            boxShadow: 'var(--itdb-login-panel-shadow)',
          }}
        >
          {error ? (
            <>
              <p className="text-sm leading-6" style={{ color: '#fca5a5' }} role="alert">
                {error}
              </p>
              <button
                type="button"
                onClick={() => window.location.replace('/login')}
                className="itdb-action-button rounded-xl px-4 py-2 text-sm font-medium"
                style={{
                  borderColor: 'var(--itdb-border)',
                  background: 'var(--itdb-control-bg)',
                  color: 'var(--itdb-accent-text)',
                }}
              >
                返回登录
              </button>
            </>
          ) : (
            <>
              <Loader2 size={28} className="itdb-spinner" />
              <p className="text-sm font-medium" style={{ color: 'var(--itdb-text)' }}>
                正在处理企业微信授权，请稍候…
              </p>
            </>
          )}
        </section>
      </main>
    );
  }

  return (
    <main
      data-cmp="Login"
      className="relative flex min-h-dvh items-center justify-center overflow-hidden px-4 py-8 sm:px-6"
      style={{
        background:
          'radial-gradient(circle at 50% 0%, rgba(59,130,246,0.24), transparent 30%), radial-gradient(circle at 12% 22%, rgba(6,182,212,0.18), transparent 28%), radial-gradient(circle at 88% 82%, rgba(16,185,129,0.14), transparent 30%), var(--itdb-login-bg)',
        color: 'var(--itdb-text)',
      }}
    >
      <AppTooltip
        label={toggleLabel}
        placement="bottom"
        className="absolute right-5 top-5 z-20 sm:right-6 sm:top-6"
      >
        <button
          type="button"
          onClick={() => setTheme(toggleTheme)}
          className="itdb-action-button grid h-11 w-11 place-items-center rounded-lg border transition-transform hover:-translate-y-0.5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--itdb-accent-text)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--itdb-login-bg)]"
          style={{
            background: 'var(--itdb-control-bg)',
            borderColor: 'var(--itdb-border)',
            color: 'var(--itdb-text-muted)',
            boxShadow: 'var(--itdb-menu-shadow)',
          }}
          aria-label={toggleLabel}
        >
          {theme === 'dark' ? <Sun size={18} /> : <Moon size={18} />}
        </button>
      </AppTooltip>
      <div className="itdb-login-grid absolute inset-0" aria-hidden="true" />
      <div
        className="itdb-login-orb absolute left-[-6rem] top-20 h-64 w-64 rounded-full"
        aria-hidden="true"
      />
      <div
        className="itdb-login-orb absolute bottom-[-4rem] right-[-5rem] h-80 w-80 rounded-full"
        aria-hidden="true"
      />

      <section className="relative z-10 flex w-full max-w-[460px] flex-col items-center gap-5">
        <ITDBBrand brand={brand} />
        <section
          className="itdb-login-frame itdb-login-reveal w-full rounded-[28px] p-1"
          style={{ animationDelay: '80ms' }}
        >
          <div
            className="rounded-[24px] p-6 sm:p-8"
            style={{
              background: 'var(--itdb-login-panel-bg)',
              border: '1px solid var(--itdb-border)',
              backdropFilter: 'blur(18px)',
              boxShadow: 'var(--itdb-login-panel-shadow)',
            }}
          >
            <div className="mb-7 text-center">
              <h1
                className="text-2xl font-bold"
                style={{ color: 'var(--itdb-text)', textShadow: '0 0 22px rgba(6,182,212,0.16)' }}
              >
                欢迎回来
              </h1>
              <p className="mt-2 text-sm" style={{ color: 'var(--itdb-text-muted)' }}>
                登录您的账户以继续
              </p>
            </div>

            <form className="space-y-5" onSubmit={handleSubmit} noValidate>
              {providers.length > 0 ? (
                <LoginProviderSelect
                  value={provider}
                  providers={providers}
                  onChange={setProvider}
                />
              ) : null}
              {isWecomProvider ? (
                <div
                  className="flex flex-col items-center gap-2 rounded-2xl px-4 py-5 text-center"
                  style={inputStyle}
                >
                  <span
                    className="grid h-14 w-14 place-items-center rounded-full"
                    style={{ background: 'rgba(59,130,246,0.14)' }}
                  >
                    <QrCode size={26} style={{ color: 'var(--itdb-accent-text)' }} />
                  </span>
                  <p className="text-sm font-medium" style={{ color: 'var(--itdb-text)' }}>
                    企业微信扫码登录
                  </p>
                  <p className="text-xs leading-5" style={{ color: 'var(--itdb-text-muted)' }}>
                    点击下方按钮跳转至企业微信授权页，
                    <br />
                    使用企业微信 App 扫码确认后自动登录
                  </p>
                </div>
              ) : (
                <>
                  <AuthInput id="username" label="用户名" icon={<User size={17} />}>
                    <input
                      id="username"
                      autoComplete="username"
                      value={username}
                      onChange={event => setUsername(event.target.value)}
                      className="h-12 w-full rounded-2xl py-3 pl-12 pr-4 text-sm outline-none transition-all"
                      style={inputStyle}
                      placeholder="请输入用户名"
                    />
                  </AuthInput>
                  <div>
                    <div className="mb-2 flex items-center justify-between">
                      <label className="block text-sm font-medium" htmlFor="password">
                        密码
                      </label>
                      <Link
                        to="/forgot-password"
                        className="text-xs font-semibold"
                        style={{ color: 'var(--itdb-accent-text)' }}
                      >
                        忘记密码?
                      </Link>
                    </div>
                    <div className="relative">
                      <Lock
                        className="absolute left-4 top-1/2 -translate-y-1/2"
                        size={17}
                        style={{ color: 'var(--itdb-text-muted)' }}
                      />
                      <input
                        id="password"
                        autoComplete="current-password"
                        type={showPassword ? 'text' : 'password'}
                        value={password}
                        onChange={event => setPassword(event.target.value)}
                        className="h-12 w-full rounded-2xl py-3 pl-12 pr-12 text-sm outline-none transition-all"
                        style={inputStyle}
                        placeholder="请输入密码"
                      />
                      <button
                        type="button"
                        onClick={() => setShowPassword(value => !value)}
                        className="absolute right-2 top-1/2 grid h-9 w-9 -translate-y-1/2 place-items-center rounded-xl"
                        style={{ color: 'var(--itdb-text-muted)' }}
                        aria-label={showPassword ? '隐藏密码' : '显示密码'}
                      >
                        {showPassword ? <EyeOff size={17} /> : <Eye size={17} />}
                      </button>
                    </div>
                  </div>
                </>
              )}
              <div aria-live="polite" className="min-h-6">
                {error ? (
                  <p
                    role="alert"
                    className="rounded-xl px-3 py-2 text-xs"
                    style={{
                      background: 'rgba(239,68,68,0.12)',
                      border: '1px solid rgba(239,68,68,0.28)',
                      color: '#fca5a5',
                    }}
                  >
                    {error}
                  </p>
                ) : null}
              </div>
              <button
                type="submit"
                disabled={loading}
                className="itdb-login-submit flex h-12 w-full items-center justify-center gap-2 rounded-2xl text-sm font-semibold transition-all disabled:cursor-not-allowed disabled:opacity-60"
                style={{
                  background: 'linear-gradient(135deg, #2563eb, #06b6d4)',
                  color: '#fff',
                  boxShadow: '0 18px 48px rgba(37,99,235,0.35)',
                }}
              >
                {loading ? <Loader2 size={17} className="itdb-spinner" /> : null}
                {loading ? '处理中...' : isWecomProvider ? '企业微信扫码登录' : '登录'}
              </button>
            </form>
          </div>
        </section>
        <p className="text-xs" style={{ color: 'var(--itdb-text-muted)' }}>
          © 2026 {brand.siteName}. Secure asset management operations console.
        </p>
      </section>
    </main>
  );
}

const inputStyle = {
  background: 'var(--itdb-control-bg)',
  border: '1px solid var(--itdb-border)',
  color: 'var(--itdb-text)',
};

function ITDBBrand({ brand }: { brand: ReturnType<typeof useBrandSettings> }) {
  return (
    <div className="itdb-login-reveal flex flex-col items-center text-center">
      <BrandIcon
        iconData={brand.iconData}
        alt={brand.loginName}
        className="itdb-brand-float mb-4 h-20 w-20 drop-shadow-[0_18px_42px_rgba(6,182,212,0.18)]"
      />
      <p className="itdb-gradient-text text-2xl font-bold tracking-wide">{brand.loginName}</p>
      <p
        className="mt-1 text-xs uppercase tracking-[0.28em]"
        style={{ color: 'var(--itdb-text-muted)' }}
      >
        {brand.appSubtitle}
      </p>
    </div>
  );
}

function AuthInput({
  id,
  label,
  icon,
  children,
}: {
  id: string;
  label: string;
  icon: ReactNode;
  children: ReactNode;
}) {
  return (
    <div>
      <label className="mb-2 block text-sm font-medium" htmlFor={id}>
        {label}
      </label>
      <div className="relative">
        <span
          className="absolute left-4 top-1/2 -translate-y-1/2"
          style={{ color: 'var(--itdb-text-muted)' }}
        >
          {icon}
        </span>
        {children}
      </div>
    </div>
  );
}

function LoginProviderSelect({
  value,
  providers,
  onChange,
}: {
  value: string;
  providers: PublicAuthProvider[];
  onChange: (value: string) => void;
}) {
  const options = [
    { id: 'local', name: '本地账号' },
    ...providers.map(item => ({
      id: item.id,
      name: item.name || (item.type === 'wecom' ? '企业微信' : 'AD/LDAP'),
    })),
  ];
  return (
    <div>
      <label className="mb-2 block text-sm font-medium" htmlFor="login-provider-select">
        登录方式
      </label>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger id="login-provider-select" className="h-12 rounded-2xl px-4 font-normal">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map(item => (
            <SelectItem key={item.id} value={item.id} className="font-normal">
              {item.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
