import { Link, Outlet, useLocation, useNavigate } from '@tanstack/react-router';
import type { CSSProperties, ReactNode } from 'react';
import {
  Boxes,
  ChevronDown,
  ChevronRight,
  GitBranch,
  KeyRound,
  LayoutDashboard,
  Link2,
  LogOut,
  MonitorCog,
  Moon,
  PanelLeftClose,
  PanelLeftOpen,
  RefreshCw,
  ScrollText,
  Settings,
  Sun,
  User,
  Printer,
  BarChart3,
  type LucideIcon,
} from 'lucide-react';
import { Suspense, useEffect, useMemo, useRef, useState } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { BrandIcon } from '@/components/brand-mark';
import { PasswordDialog } from '@/components/password-dialog';
import { useBrandSettings } from '@/lib/branding';
import {
  AUTH_SESSION_CHANGED_EVENT,
  clearSession,
  fetchCurrentUser,
  fetchPublicAuthProviders,
  fetchWecomBindUrl,
  getAuthToken,
  getStoredUser,
  logout,
  setCurrentUserSnapshot,
  unbindWecom,
  userHasAnyPermission,
  WECOM_BIND_MESSAGE,
  WECOM_PROVIDERS_CHANGED_EVENT,
} from '@/lib/auth';
import { showErrorToast } from '@/lib/toast-errors';
import {
  ANY_READ_PERMISSIONS,
  ASSETS_READ_PERMISSIONS,
  DICTIONARIES_READ_PERMISSIONS,
  PERM,
  SETTINGS_READ_PERMISSIONS,
} from '@/lib/permissions';
import {
  applyTheme,
  cn,
  getInitialTheme,
  persistTheme,
  toggleTheme,
  type ItdbTheme,
} from '@/lib/utils';

type NavItem = {
  to: string;
  label: string;
  description: string;
  permissions: string[];
  icon: LucideIcon;
};

const navigation: NavItem[] = [
  {
    to: '/',
    label: '首页',
    description: '资产、软件与合同实时总览',
    permissions: ANY_READ_PERMISSIONS,
    icon: LayoutDashboard,
  },
  {
    to: '/assets',
    label: '资产管理',
    description: '硬件、软件、单据及相关资产',
    permissions: ASSETS_READ_PERMISSIONS,
    icon: MonitorCog,
  },
  {
    to: '/dictionaries',
    label: '资料管理',
    description: '资产类型、部门、状态与标记等相关资料',
    permissions: DICTIONARIES_READ_PERMISSIONS,
    icon: Boxes,
  },
  {
    to: '/labels',
    label: '打印标签',
    description: '选择并预览资产标签',
    permissions: [PERM.labelsPreview],
    icon: Printer,
  },
  {
    to: '/reports',
    label: '统计报表',
    description: '运行资产统计报表',
    permissions: [PERM.reportsRead],
    icon: BarChart3,
  },
  {
    to: '/browse',
    label: '资产导航',
    description: '按维度逐层定位资产',
    permissions: [PERM.browseRead],
    icon: GitBranch,
  },
  {
    to: '/history',
    label: '审计日志',
    description: '追踪身份认证、资源与配置安全事件',
    permissions: [PERM.auditRead],
    icon: ScrollText,
  },
  {
    to: '/settings',
    label: '系统配置',
    description: '用户权限、身份认证与邮件配置',
    permissions: SETTINGS_READ_PERMISSIONS,
    icon: Settings,
  },
];

const sidebarItemHeight = 46;
const sidebarItemGap = 4;
// 选中背景块统一 1 秒缓入缓出平移，不随菜单距离缩放
const sidebarTransitionMs = 1000;

function matchesPath(to: string, path: string) {
  return to === '/' ? path === '/' : path === to || path.startsWith(`${to}/`);
}

export function RequireAuth({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const [token, setToken] = useState(getAuthToken);
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    const sync = () => setToken(getAuthToken());
    window.addEventListener(AUTH_SESSION_CHANGED_EVENT, sync);
    window.addEventListener('storage', sync);
    return () => {
      window.removeEventListener(AUTH_SESSION_CHANGED_EVENT, sync);
      window.removeEventListener('storage', sync);
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    if (!token) {
      setChecking(false);
      void navigate({ to: '/login', replace: true });
      return;
    }
    void fetchCurrentUser()
      .then(() => {
        if (!cancelled) setChecking(false);
      })
      .catch(() => {
        if (cancelled) return;
        clearSession();
        setChecking(false);
        void navigate({ to: '/login', replace: true });
      });
    return () => {
      cancelled = true;
    };
  }, [navigate, token]);

  if (checking || !token) return null;
  return <>{children}</>;
}

export function AppLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [theme, setTheme] = useState<ItdbTheme>(getInitialTheme);
  const [user, setUser] = useState(getStoredUser);
  const brand = useBrandSettings();
  const [userMenuOpen, setUserMenuOpen] = useState(false);
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [wecomEnabled, setWecomEnabled] = useState(false);
  const userMenuRef = useRef<HTMLDivElement | null>(null);
  const mainContentRef = useRef<HTMLElement | null>(null);
  const pageContentRef = useRef<HTMLDivElement | null>(null);
  const pageEndSpacerRef = useRef<HTMLDivElement | null>(null);
  const [pageEndSpacing, setPageEndSpacing] = useState(0);
  useEffect(() => {
    applyTheme(theme);
    persistTheme(theme);
  }, [theme]);

  useEffect(() => {
    const sync = () => setUser(getStoredUser());
    window.addEventListener(AUTH_SESSION_CHANGED_EVENT, sync);
    return () => window.removeEventListener(AUTH_SESSION_CHANGED_EVENT, sync);
  }, []);

  useEffect(() => {
    const refreshWecomEnabled = () => {
      fetchPublicAuthProviders({ force: true })
        .then(response =>
          setWecomEnabled(response.items.some(item => item.enabled && item.type === 'wecom'))
        )
        .catch(() => undefined);
    };
    let cancelled = false;
    fetchPublicAuthProviders()
      .then(response => {
        if (cancelled) return;
        setWecomEnabled(response.items.some(item => item.enabled && item.type === 'wecom'));
      })
      .catch(() => undefined);
    window.addEventListener(WECOM_PROVIDERS_CHANGED_EVENT, refreshWecomEnabled);
    return () => {
      cancelled = true;
      window.removeEventListener(WECOM_PROVIDERS_CHANGED_EVENT, refreshWecomEnabled);
    };
  }, []);

  useEffect(() => {
    const onMessage = (event: MessageEvent) => {
      if (event.origin !== window.location.origin) return;
      const data = event.data as { type?: string; ok?: boolean; message?: string } | null;
      if (data?.type !== WECOM_BIND_MESSAGE) return;
      if (data.ok) {
        toast.success('企业微信绑定成功');
      } else {
        showErrorToast(data.message || '企业微信绑定失败');
      }
      void fetchCurrentUser({ force: true })
        .then(fresh => {
          setCurrentUserSnapshot(fresh);
          setUser(fresh);
        })
        .catch(() => undefined);
    };
    window.addEventListener('message', onMessage);
    return () => window.removeEventListener('message', onMessage);
  }, []);

  async function startWecomBind() {
    setUserMenuOpen(false);
    try {
      const url = await fetchWecomBindUrl();
      const popup = window.open(url, 'itdb-wecom-bind', 'width=680,height=680');
      if (!popup) {
        toast.error('浏览器拦截了绑定窗口，请允许弹窗后重试');
      }
    } catch (err) {
      showErrorToast(err instanceof Error ? err.message : '获取企业微信绑定地址失败');
    }
  }

  async function handleUnbindWecom() {
    setUserMenuOpen(false);
    try {
      await unbindWecom();
      toast.success('已解绑企业微信');
      const fresh = await fetchCurrentUser({ force: true });
      setCurrentUserSnapshot(fresh);
      setUser(fresh);
    } catch (err) {
      showErrorToast(err instanceof Error ? err.message : '解绑企业微信失败');
    }
  }

  useEffect(() => {
    const close = (event: MouseEvent) => {
      const target = event.target as Node;
      if (userMenuRef.current && !userMenuRef.current.contains(target)) {
        setUserMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', close);
    return () => document.removeEventListener('mousedown', close);
  }, []);

  const visibleItems = navigation.filter(
    item => item.to === '/' || userHasAnyPermission(user, item.permissions)
  );
  const current = location.pathname.startsWith('/rack-view/')
    ? navigation.find(item => item.to === '/assets')
    : navigation.find(item => matchesPath(item.to, location.pathname));
  const activeIndex = Math.max(
    0,
    visibleItems.findIndex(item => item.to === current?.to)
  );

  useEffect(() => {
    const main = mainContentRef.current;
    const page = pageContentRef.current;
    if (!main || !page) return;

    let frame = 0;
    const measure = () => {
      window.cancelAnimationFrame(frame);
      frame = window.requestAnimationFrame(() => {
        const spacerHeight = pageEndSpacerRef.current?.offsetHeight ?? 0;
        const mainRect = main.getBoundingClientRect();
        const surfaces = Array.from(
          page.querySelectorAll<HTMLElement>('.itdb-surface-3d, .itdb-card-hover')
        ).filter(element => element.offsetParent !== null);
        const lastSurface = surfaces.at(-1);
        const contentBottom = main.scrollHeight - spacerHeight;
        const trailingSpace = lastSurface
          ? contentBottom -
            (lastSurface.getBoundingClientRect().bottom - mainRect.top + main.scrollTop)
          : 0;
        const nextSpacing = Math.max(0, Math.ceil(24 - trailingSpace));
        setPageEndSpacing(current => (current === nextSpacing ? current : nextSpacing));
      });
    };
    const resizeObserver = new ResizeObserver(measure);
    const mutationObserver = new MutationObserver(measure);
    resizeObserver.observe(main);
    mutationObserver.observe(page, { childList: true, subtree: true });
    window.addEventListener('resize', measure);
    measure();
    return () => {
      window.cancelAnimationFrame(frame);
      resizeObserver.disconnect();
      mutationObserver.disconnect();
      window.removeEventListener('resize', measure);
    };
  }, [location.pathname]);

  async function handleLogout() {
    setUserMenuOpen(false);
    setUser(null);
    void logout({ waitForRemote: false });
    toast.success('已退出登录');
    await navigate({ to: '/login', replace: true });
  }

  async function refreshCurrentPage(showToast = false) {
    await queryClient.invalidateQueries({ queryKey: ['itdb'] });
    window.dispatchEvent(new Event('itdb:refresh'));
    if (showToast) {
      toast.success('页面数据已刷新');
    }
  }

  const shellVars = useMemo(
    () =>
      ({
        '--itdb-sidebar-width': sidebarOpen ? '240px' : '64px',
      }) as CSSProperties,
    [sidebarOpen]
  );

  return (
    <div
      data-cmp="ITDBLayout"
      className="flex h-screen w-full overflow-hidden max-md:flex-col"
      style={{ ...shellVars, background: 'var(--itdb-bg)', color: 'var(--itdb-text)' }}
    >
      <aside
        className={cn(
          'z-30 flex shrink-0 flex-col overflow-hidden border-r transition-all duration-300 max-md:h-auto max-md:w-full max-md:flex-row max-md:overflow-visible max-md:border-b max-md:border-r-0',
          sidebarOpen ? 'w-[240px]' : 'w-16'
        )}
        style={{
          background: 'var(--itdb-sidebar)',
          borderColor: 'var(--itdb-border)',
          boxShadow: 'var(--itdb-shell-shadow)',
        }}
      >
        <div
          className="flex min-h-16 items-center gap-3 border-b px-3 py-4 max-md:w-16 max-md:shrink-0 max-md:justify-center max-md:border-b-0 max-md:border-r max-md:px-2"
          style={{ borderColor: 'var(--itdb-border)' }}
        >
          <BrandIcon
            iconData={brand.iconData}
            alt={brand.appName}
            className="h-9 w-9 shrink-0 rounded-lg object-contain shadow-[0_10px_28px_rgba(37,99,235,0.22)]"
          />
          {sidebarOpen ? (
            <span className="min-w-0 max-md:hidden">
              <span className="itdb-gradient-text block truncate text-base font-bold">
                {brand.appName}
              </span>
              <span className="block truncate whitespace-nowrap text-xs tracking-[0.16em] text-[var(--itdb-text-muted)] max-md:hidden">
                {brand.appSubtitle}
              </span>
            </span>
          ) : null}
        </div>

        <nav className="itdb-hidden-scrollbar relative flex-1 overflow-y-auto px-2 py-4 max-md:flex max-md:overflow-x-auto max-md:overflow-y-visible max-md:py-2">
          {visibleItems.length > 0 ? (
            <span
              className="itdb-sidebar-active-indicator"
              aria-hidden="true"
              style={
                {
                  top: `calc(1rem + ${activeIndex * (sidebarItemHeight + sidebarItemGap)}px)`,
                  transition: `top ${sidebarTransitionMs}ms cubic-bezier(0.45, 0, 0.25, 1), opacity 0.2s ease`,
                } as CSSProperties
              }
            />
          ) : null}
          <div className="max-md:flex">
            {visibleItems.map(item => {
              const active = matchesPath(item.to, location.pathname);
              const Icon = item.icon;
              const link = (
                <Link
                  key={item.to}
                  to={item.to}
                  preload="intent"
                  onClick={event => {
                    if (active) {
                      event.preventDefault();
                      return;
                    }
                  }}
                  aria-current={active ? 'page' : undefined}
                  className={cn(
                    'itdb-sidebar-button relative z-10 mb-1 flex h-[46px] items-center rounded-lg text-sm transition-[color,transform] duration-300 max-md:mb-0 max-md:min-w-11',
                    sidebarOpen
                      ? 'justify-start px-3 max-md:justify-center max-md:px-2'
                      : 'justify-center px-2',
                    active && 'itdb-sidebar-item-active'
                  )}
                  style={{ color: active ? 'var(--itdb-accent)' : 'var(--itdb-text-muted)' }}
                >
                  <Icon size={17} aria-hidden="true" />
                  {sidebarOpen ? (
                    <span className="ml-3 min-w-0 flex-1 truncate font-medium max-md:hidden">
                      {item.label}
                    </span>
                  ) : null}
                </Link>
              );
              return sidebarOpen ? (
                link
              ) : (
                <AppTooltip key={item.to} label={item.label}>
                  {link}
                </AppTooltip>
              );
            })}
          </div>
        </nav>

        <div
          className="flex items-center justify-center border-t py-4 max-md:hidden"
          style={{ borderColor: 'var(--itdb-border)' }}
        >
          <AppTooltip label={sidebarOpen ? '收起侧边栏' : '展开侧边栏'} placement="top">
            <button
              type="button"
              onClick={() => setSidebarOpen(value => !value)}
              className="itdb-action-button flex h-9 w-9 items-center justify-center rounded-lg border"
              aria-label={sidebarOpen ? '收起侧边栏' : '展开侧边栏'}
            >
              {sidebarOpen ? <PanelLeftClose size={16} /> : <PanelLeftOpen size={16} />}
            </button>
          </AppTooltip>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header
          className="relative z-20 flex h-16 shrink-0 items-center justify-between border-b px-6 backdrop-blur max-md:px-3"
          style={{
            background: 'var(--itdb-header)',
            borderColor: 'var(--itdb-border)',
            boxShadow: 'var(--itdb-header-shadow)',
          }}
        >
          <div className="flex min-w-0 items-center gap-2">
            <span className="text-sm text-[var(--itdb-text-muted)] max-sm:hidden">
              {brand.appName} 控制台
            </span>
            <ChevronRight size={14} className="text-[var(--itdb-text-muted)] max-sm:hidden" />
            <span className="truncate text-sm font-medium">{current?.label ?? '无可用页面'}</span>
          </div>
          <div className="flex items-center gap-3">
            <AppTooltip label="刷新当前页面数据" placement="bottom">
              <button
                type="button"
                onClick={() => void refreshCurrentPage(true)}
                className="itdb-action-button flex h-[42px] w-[42px] items-center justify-center rounded-lg border"
                aria-label="刷新当前页面数据"
              >
                <RefreshCw size={17} />
              </button>
            </AppTooltip>
            <AppTooltip
              label={theme === 'dark' ? '切换浅色背景' : '切换深色背景'}
              placement="bottom"
            >
              <button
                type="button"
                onClick={() => setTheme(toggleTheme)}
                className="itdb-action-button flex h-[42px] w-[42px] items-center justify-center rounded-lg border"
                aria-label={theme === 'dark' ? '切换浅色背景' : '切换深色背景'}
              >
                {theme === 'dark' ? <Sun size={17} /> : <Moon size={17} />}
              </button>
            </AppTooltip>
            <div ref={userMenuRef} className="relative">
              <AppTooltip
                label={user?.displayName || user?.username || 'admin'}
                placement="bottom"
                align="end"
              >
                <button
                  type="button"
                  onClick={() => setUserMenuOpen(value => !value)}
                  className="itdb-action-button grid h-[42px] w-[150px] grid-cols-[26px_minmax(0,1fr)_14px] items-center gap-2 rounded-lg border px-3 py-1.5 text-sm"
                  aria-haspopup="menu"
                  aria-expanded={userMenuOpen}
                  aria-label="用户菜单"
                >
                  <span className="grid h-[26px] w-[26px] place-items-center rounded-full bg-[linear-gradient(135deg,var(--itdb-accent),var(--itdb-accent2))] text-white">
                    <User size={14} />
                  </span>
                  <span className="min-w-0 truncate">
                    {user?.displayName || user?.username || 'admin'}
                  </span>
                  <ChevronDown
                    size={14}
                    className={
                      userMenuOpen ? 'rotate-180 transition-transform' : 'transition-transform'
                    }
                  />
                </button>
              </AppTooltip>
              {userMenuOpen ? (
                <div
                  role="menu"
                  className="absolute right-0 top-[calc(100%+8px)] z-[1300] w-full rounded-lg border p-2"
                  style={{
                    background: 'var(--itdb-menu-bg)',
                    borderColor: 'var(--itdb-border)',
                    boxShadow: 'var(--itdb-menu-shadow)',
                  }}
                >
                  <button
                    type="button"
                    role="menuitem"
                    className="itdb-menu-action-item flex h-10 w-full items-center gap-2 rounded-lg px-3 text-sm"
                    onClick={() => {
                      setUserMenuOpen(false);
                      setPasswordOpen(true);
                    }}
                  >
                    <KeyRound size={16} />
                    修改密码
                  </button>
                  {wecomEnabled ? (
                    <button
                      type="button"
                      role="menuitem"
                      className="itdb-menu-action-item flex h-10 w-full items-center gap-2 rounded-lg px-3 text-sm"
                      onClick={() => {
                        if (user?.wecomBound) {
                          void handleUnbindWecom();
                          return;
                        }
                        void startWecomBind();
                      }}
                    >
                      <Link2 size={16} />
                      {user?.wecomBound ? '解绑企微' : '绑定企微'}
                    </button>
                  ) : null}
                  <button
                    type="button"
                    role="menuitem"
                    className="itdb-menu-action-item itdb-danger-button flex h-10 w-full items-center gap-2 rounded-lg px-3 text-sm"
                    onClick={() => void handleLogout()}
                  >
                    <LogOut size={16} />
                    退出系统
                  </button>
                </div>
              ) : null}
            </div>
          </div>
        </header>

        <main
          ref={mainContentRef}
          className="itdb-hidden-scrollbar min-h-0 flex-1 overflow-y-auto bg-[var(--itdb-bg)]"
        >
          <div className="flex h-full min-h-full flex-col p-6 max-md:p-4">
            <div
              ref={pageContentRef}
              key={location.pathname}
              className="itdb-page-enter min-h-0 flex-1"
            >
              <Suspense fallback={<div aria-hidden="true" className="min-h-[60vh]" />}>
                <Outlet />
                {pageEndSpacing > 0 ? (
                  <div
                    ref={pageEndSpacerRef}
                    aria-hidden="true"
                    style={{ height: pageEndSpacing }}
                  />
                ) : null}
              </Suspense>
            </div>
          </div>
        </main>
      </div>

      <PasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} />
    </div>
  );
}
