import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import {
  HeadContent,
  Link,
  Outlet,
  Scripts,
  createRootRouteWithContext,
  useRouter,
} from '@tanstack/react-router';
import {
  AlertTriangle,
  ArrowLeft,
  CheckCircle2,
  Home,
  Info,
  Loader2,
  RefreshCcw,
  SearchX,
  ShieldAlert,
  XCircle,
} from 'lucide-react';
import { type ReactNode, useEffect, useState } from 'react';
import { BootScreen } from '@/components/boot-screen';
import { type BrandSettings, setBrandSettings } from '@/lib/branding';
import { serverEnv } from '@/lib/server-env';
import { Toaster } from '@/components/ui/sonner';

type RootLoaderData = { brand: Partial<BrandSettings> | null };

// fetchServerBrand 仅在服务端渲染时拉取品牌配置，供首帧 HTML 直出站点名与图标，失败时回退默认
async function fetchServerBrand(): Promise<Partial<BrandSettings> | null> {
  if (typeof window !== 'undefined') return null;
  try {
    const origin = (await serverEnv('ITDB_SSR_API_ORIGIN')) || 'http://127.0.0.1:8080';
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), 1500);
    try {
      const response = await fetch(`${origin}/api/public/base`, { signal: controller.signal });
      if (!response.ok) {
        console.warn(
          `[itdb-ssr] fetch brand from ${origin}/api/public/base failed with status ${response.status}`,
        );
        return null;
      }
      return (await response.json()) as Partial<BrandSettings>;
    } finally {
      clearTimeout(timer);
    }
  } catch (error) {
    console.warn('[itdb-ssr] fetch brand failed:', error instanceof Error ? error.message : error);
    return null;
  }
}

/* 提示文本拖动选择桥：弹窗打开时 body 为 pointer-events:none，浏览器原生拖动选区会
   错误落入弹窗；这里拦截提示上的按下与拖动，用命中测试计算文本位置，支持从按下位置
   拖动选择部分文本、双击选中词组，选中后即可复制。提示按钮（关闭等）不受影响 */
type CaretCapableDocument = Document & {
  caretRangeFromPoint?: (x: number, y: number) => Range | null;
  caretPositionFromPoint?: (x: number, y: number) => { offsetNode: Node; offset: number } | null;
};
type WordCapableSelection = Selection & {
  modify?: (alter: string, direction: string, granularity: string) => void;
};

function ToastTextSelectionBridge() {
  useEffect(() => {
    let dragging = false;
    let dragRoot: HTMLElement | null = null;
    let anchorRange: Range | null = null;
    let lastDownAt = 0;
    let lastDownX = 0;
    let lastDownY = 0;

    const contentOf = (target: EventTarget | null): HTMLElement | null => {
      if (!(target instanceof HTMLElement)) return null;
      if (target.closest('button')) return null;
      const toast = target.closest<HTMLElement>('[data-sonner-toast]');
      if (!toast) return null;
      const content = toast.querySelector<HTMLElement>('[data-content]');
      return content ?? toast;
    };

    const caretAt = (x: number, y: number, root: HTMLElement): Range | null => {
      const doc = document as CaretCapableDocument;
      let range: Range | null = null;
      if (doc.caretRangeFromPoint) {
        range = doc.caretRangeFromPoint(x, y);
      } else if (doc.caretPositionFromPoint) {
        const position = doc.caretPositionFromPoint(x, y);
        if (position) {
          range = document.createRange();
          range.setStart(position.offsetNode, position.offset);
          range.collapse(true);
        }
      }
      if (!range || !root.contains(range.startContainer)) return null;
      return range;
    };

    const applyRange = (range: Range) => {
      const selection = window.getSelection();
      if (!selection) return;
      selection.removeAllRanges();
      selection.addRange(range);
    };

    const selectBetween = (from: Range, to: Range) => {
      const range = document.createRange();
      if (from.compareBoundaryPoints(Range.START_TO_START, to) <= 0) {
        range.setStart(from.startContainer, from.startOffset);
        range.setEnd(to.startContainer, to.startOffset);
      } else {
        range.setStart(to.startContainer, to.startOffset);
        range.setEnd(from.startContainer, from.startOffset);
      }
      applyRange(range);
    };

    const onPointerDown = (event: PointerEvent) => {
      const content = contentOf(event.target);
      if (!content) return;
      const caret = caretAt(event.clientX, event.clientY, content);
      if (!caret) return;
      event.preventDefault();
      const isWordPick =
        Date.now() - lastDownAt < 350 &&
        Math.hypot(event.clientX - lastDownX, event.clientY - lastDownY) < 8;
      lastDownAt = Date.now();
      lastDownX = event.clientX;
      lastDownY = event.clientY;
      if (isWordPick) {
        const selection = window.getSelection() as WordCapableSelection | null;
        applyRange(caret);
        selection?.modify?.('extend', 'forward', 'word');
        selection?.modify?.('extend', 'backward', 'word');
        dragging = false;
        anchorRange = null;
        return;
      }
      dragging = true;
      dragRoot = content;
      anchorRange = caret.cloneRange();
      applyRange(caret);
    };

    const onPointerMove = (event: PointerEvent) => {
      if (!dragging || !anchorRange || !dragRoot) return;
      const caret = caretAt(event.clientX, event.clientY, dragRoot);
      if (!caret) return;
      selectBetween(anchorRange, caret);
    };

    const onPointerEnd = () => {
      dragging = false;
      anchorRange = null;
      dragRoot = null;
    };

    document.addEventListener('pointerdown', onPointerDown, true);
    document.addEventListener('pointermove', onPointerMove, true);
    document.addEventListener('pointerup', onPointerEnd, true);
    document.addEventListener('pointercancel', onPointerEnd, true);
    return () => {
      document.removeEventListener('pointerdown', onPointerDown, true);
      document.removeEventListener('pointermove', onPointerMove, true);
      document.removeEventListener('pointerup', onPointerEnd, true);
      document.removeEventListener('pointercancel', onPointerEnd, true);
    };
  }, []);
  return null;
}

import appCss from '../styles.css?url';

const themeScript = `
(function () {
  var root = document.documentElement;
  try {
    var theme = localStorage.getItem('itdb.theme') === 'light' ? 'light' : 'dark';
    root.dataset.itdbTheme = theme;
    root.style.colorScheme = theme;
  } catch (_) {
    root.dataset.itdbTheme = 'dark';
    root.style.colorScheme = 'dark';
  } finally {
    root.style.visibility = 'visible';
  }
})();
`;

function NotFoundComponent() {
  return (
    <main className="relative grid min-h-screen place-items-center overflow-hidden bg-[var(--itdb-bg)] px-4 py-10 sm:px-6">
      <div className="pointer-events-none absolute inset-x-0 top-1/2 h-px -translate-y-1/2 bg-gradient-to-r from-transparent via-[rgba(96,165,250,0.28)] to-transparent" />
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(148,163,184,0.055)_1px,transparent_1px),linear-gradient(90deg,rgba(148,163,184,0.045)_1px,transparent_1px)] bg-[size:46px_46px] [mask-image:radial-gradient(circle_at_50%_50%,#000_0%,transparent_70%)]" />

      <section className="itdb-surface-3d w-full max-w-4xl rounded-2xl px-6 py-8 text-center sm:px-10 sm:py-10 lg:px-12">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl border border-[rgba(96,165,250,0.34)] bg-[rgba(59,130,246,0.12)] text-[var(--itdb-accent-hover)] shadow-[0_16px_36px_rgba(37,99,235,0.18)]">
          <SearchX size={28} aria-hidden="true" />
        </div>
        <div className="itdb-gradient-text mt-7 text-[clamp(5.25rem,15vw,9rem)] font-black leading-none tracking-normal drop-shadow-[0_14px_32px_rgba(37,99,235,0.18)]">
          404
        </div>
        <h1 className="itdb-gradient-text mt-4 text-2xl font-semibold tracking-normal sm:text-3xl">
          页面不存在
        </h1>
        <p className="mx-auto mt-3 max-w-xl text-sm leading-6 text-muted-foreground sm:text-base">
          当前访问的页面不存在，或已经被移动。请返回首页后重新选择需要访问的功能。
        </p>
        <div className="mt-8 flex flex-wrap justify-center gap-3">
          <Link
            to="/"
            className="itdb-login-submit inline-flex h-10 items-center justify-center gap-2 rounded-lg px-5 text-sm font-medium text-white"
          >
            <Home size={16} aria-hidden="true" />
            返回首页
          </Link>
          <button
            type="button"
            onClick={() => history.back()}
            className="itdb-action-button inline-flex h-10 items-center justify-center gap-2 rounded-lg border px-5 text-sm font-medium"
          >
            <ArrowLeft size={16} aria-hidden="true" />
            返回上一页
          </button>
        </div>
      </section>
    </main>
  );
}

function ErrorComponent({ error, reset }: { error: unknown; reset: () => void }) {
  console.error(error);
  const router = useRouter();

  return (
    <main className="relative grid min-h-screen place-items-center overflow-hidden bg-[var(--itdb-bg)] px-4 py-10 sm:px-6">
      <div className="pointer-events-none absolute inset-0 bg-[linear-gradient(rgba(148,163,184,0.052)_1px,transparent_1px),linear-gradient(90deg,rgba(148,163,184,0.04)_1px,transparent_1px)] bg-[size:44px_44px] [mask-image:radial-gradient(circle_at_50%_46%,#000_0%,transparent_72%)]" />

      <section className="itdb-surface-3d w-full max-w-4xl rounded-2xl px-6 py-8 text-center sm:px-10 sm:py-10">
        <div className="mx-auto flex h-14 w-14 items-center justify-center rounded-2xl border border-[rgba(239,68,68,0.34)] bg-[rgba(239,68,68,0.11)] text-[var(--itdb-status-red-text)] shadow-[0_16px_36px_rgba(239,68,68,0.14)]">
          <ShieldAlert size={28} aria-hidden="true" />
        </div>
        <h1 className="mt-6 text-2xl font-semibold tracking-tight text-foreground sm:text-3xl">
          页面加载失败
        </h1>
        <p className="mx-auto mt-3 max-w-xl text-sm leading-6 text-muted-foreground sm:text-base">
          页面加载过程中出现异常，可以重试或返回首页。
        </p>
        {import.meta.env.DEV ? (
          <pre className="itdb-error-detail-pre itdb-hidden-scrollbar mx-auto mt-6 max-h-80 w-full max-w-3xl overflow-auto rounded-xl border p-5 text-left text-xs leading-5 sm:text-[0.8125rem]">
            {(error as Error)?.stack ?? (error as Error)?.message}
          </pre>
        ) : null}
        <div className="mt-8 flex flex-wrap justify-center gap-3">
          <button
            type="button"
            onClick={() => {
              router.invalidate();
              reset();
            }}
            className="itdb-login-submit inline-flex h-10 items-center justify-center gap-2 rounded-lg px-5 text-sm font-medium text-white"
          >
            <RefreshCcw size={16} aria-hidden="true" />
            重试
          </button>
          <a
            href="/"
            className="itdb-action-button inline-flex h-10 items-center justify-center gap-2 rounded-lg border px-5 text-sm font-medium"
          >
            <Home size={16} aria-hidden="true" />
            返回首页
          </a>
        </div>
      </section>
    </main>
  );
}

export const Route = createRootRouteWithContext<{ queryClient: QueryClient }>()({
  loader: async (): Promise<RootLoaderData> => ({ brand: await fetchServerBrand() }),
  head: ({ loaderData }) => {
    const siteName = loaderData?.brand?.siteName?.trim() || 'ITDB';
    const iconHref = loaderData?.brand?.iconData?.trim() || '/favicon.svg';
    return {
      meta: [
        { charSet: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { title: siteName },
        { name: 'description', content: '企业 IT 资产、合同与基础设施管理平台' },
        { name: 'author', content: 'ITDB' },
        { property: 'og:title', content: 'ITDB 控制台' },
        { property: 'og:description', content: '企业 IT 资产、合同与基础设施管理平台' },
        { property: 'og:type', content: 'website' },
        { name: 'twitter:card', content: 'summary' },
        { name: 'twitter:title', content: 'ITDB 控制台' },
        { name: 'twitter:description', content: '企业 IT 资产、合同与基础设施管理平台' },
      ],
      links: [
        {
          rel: 'stylesheet',
          href: appCss,
        },
        iconHref.startsWith('data:')
          ? { rel: 'icon', href: iconHref }
          : { rel: 'icon', href: iconHref, type: 'image/svg+xml' },
      ],
    };
  },
  shellComponent: RootShell,
  component: RootComponent,
  notFoundComponent: NotFoundComponent,
  errorComponent: ErrorComponent,
});

function RootShell({ children }: { children: ReactNode }) {
  const initialStyle =
    typeof document === 'undefined' ? { visibility: 'hidden' as const } : undefined;

  return (
    <html lang="zh-CN" data-itdb-theme="dark" style={initialStyle} suppressHydrationWarning>
      <head>
        <script dangerouslySetInnerHTML={{ __html: themeScript }} />
        <HeadContent />
      </head>
      <body>
        {children}
        <Scripts />
      </body>
    </html>
  );
}

function RootComponent() {
  const { queryClient } = Route.useRouteContext();
  const { brand } = Route.useLoaderData();
  const [booting, setBooting] = useState(true);

  useEffect(() => {
    if (brand) setBrandSettings(brand);
  }, [brand]);

  useEffect(() => {
    const timer = window.setTimeout(() => setBooting(false), 520);
    return () => window.clearTimeout(timer);
  }, []);

  return (
    <QueryClientProvider client={queryClient}>
      <ToastTextSelectionBridge />
      {booting ? <BootScreen initialBrand={brand} /> : <Outlet />}
      <Toaster
        position="top-right"
        theme="system"
        closeButton
        expand
        visibleToasts={8}
        gap={10}
        duration={3000}
        offset={{ top: 28, right: 20 }}
        mobileOffset={{ top: 16, right: 12, left: 12 }}
        swipeDirections={[]}
        icons={{
          success: <CheckCircle2 size={18} />,
          error: <XCircle size={18} />,
          warning: <AlertTriangle size={18} />,
          info: <Info size={18} />,
          loading: <Loader2 className="itdb-toast-spin" size={18} />,
        }}
      />
    </QueryClientProvider>
  );
}
