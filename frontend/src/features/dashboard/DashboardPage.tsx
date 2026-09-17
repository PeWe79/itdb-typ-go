import { Link } from '@tanstack/react-router';
import type { LucideIcon } from 'lucide-react';
import {
  BarChart3,
  Boxes,
  FileText,
  FolderOpen,
  Handshake,
  MapPin,
  PackageOpen,
  Printer,
  Search,
  ServerCog,
  Settings,
  Tags,
  Users,
  Warehouse,
  Wrench,
  Plus,
} from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { api, getStoredUser, isApiStatus, userHasAnyPermission } from '@/lib/auth';
import { PERM, SETTINGS_READ_PERMISSIONS } from '@/lib/permissions';

type DashboardResponse = { counts: Record<string, number> };

type CardAction = { label: string; href: string; icon: LucideIcon; permissions: string[] };
type DashboardCard = {
  key: string;
  title: string;
  description: string;
  icon: LucideIcon;
  href: string;
  countKey?: string;
  permissions: string[];
  actions: CardAction[];
};

const cards: DashboardCard[] = [
  {
    key: 'items',
    title: '硬件',
    description: '管理硬件资产，包含服务器、交换机、电话等设备。',
    icon: ServerCog,
    href: '/assets/hardware',
    countKey: 'items',
    permissions: [PERM.assetsItemsRead],
    actions: [
      {
        label: '查找',
        href: '/assets/hardware',
        icon: Search,
        permissions: [PERM.assetsItemsRead],
      },
      {
        label: '新增',
        href: '/assets/hardware?create=1',
        icon: Plus,
        permissions: [PERM.assetsItemsManage],
      },
      {
        label: '硬件类型',
        href: '/dictionaries/itemtypes',
        icon: Wrench,
        permissions: [PERM.dictionariesItemtypesRead],
      },
      {
        label: '状态类型',
        href: '/dictionaries/statustypes',
        icon: Wrench,
        permissions: [PERM.dictionariesStatustypesRead],
      },
    ],
  },
  {
    key: 'software',
    title: '软件',
    description: '管理软件授权、版本和安装关系。',
    icon: PackageOpen,
    href: '/assets/software',
    countKey: 'software',
    permissions: [PERM.assetsSoftwareRead],
    actions: [
      {
        label: '查找',
        href: '/assets/software',
        icon: Search,
        permissions: [PERM.assetsSoftwareRead],
      },
      {
        label: '新增',
        href: '/assets/software?create=1',
        icon: Plus,
        permissions: [PERM.assetsSoftwareManage],
      },
      {
        label: '标记',
        href: '/dictionaries/tags',
        icon: Tags,
        permissions: [PERM.dictionariesTagsRead],
      },
    ],
  },
  {
    key: 'invoices',
    title: '单据',
    description: '管理采购单据及其关联信息。',
    icon: FileText,
    href: '/assets/invoices',
    countKey: 'invoices',
    permissions: [PERM.assetsInvoicesRead],
    actions: [
      {
        label: '查找',
        href: '/assets/invoices',
        icon: Search,
        permissions: [PERM.assetsInvoicesRead],
      },
      {
        label: '新增',
        href: '/assets/invoices?create=1',
        icon: Plus,
        permissions: [PERM.assetsInvoicesManage],
      },
    ],
  },
  {
    key: 'agents',
    title: '代理',
    description: '管理厂商、供应商、采购方与承包方。',
    icon: Users,
    href: '/assets/agents',
    countKey: 'agents',
    permissions: [PERM.assetsAgentsRead],
    actions: [
      { label: '查找', href: '/assets/agents', icon: Search, permissions: [PERM.assetsAgentsRead] },
      {
        label: '新增',
        href: '/assets/agents?create=1',
        icon: Plus,
        permissions: [PERM.assetsAgentsManage],
      },
    ],
  },
  {
    key: 'files',
    title: '文件',
    description: '维护文件及其与资产的关联关系。',
    icon: FolderOpen,
    href: '/assets/files',
    countKey: 'files',
    permissions: [PERM.assetsFilesRead],
    actions: [
      { label: '查找', href: '/assets/files', icon: Search, permissions: [PERM.assetsFilesRead] },
      {
        label: '新增',
        href: '/assets/files?create=1',
        icon: Plus,
        permissions: [PERM.assetsFilesManage],
      },
      {
        label: '文件类型',
        href: '/dictionaries/filetypes',
        icon: Wrench,
        permissions: [PERM.dictionariesFiletypesRead],
      },
    ],
  },
  {
    key: 'contracts',
    title: '合同',
    description: '管理支持、授权、租赁等合同。',
    icon: Handshake,
    href: '/assets/contracts',
    countKey: 'contracts',
    permissions: [PERM.assetsContractsRead],
    actions: [
      {
        label: '查找',
        href: '/assets/contracts',
        icon: Search,
        permissions: [PERM.assetsContractsRead],
      },
      {
        label: '新增',
        href: '/assets/contracts?create=1',
        icon: Plus,
        permissions: [PERM.assetsContractsManage],
      },
      {
        label: '合同类型',
        href: '/dictionaries/contracttypes',
        icon: Wrench,
        permissions: [PERM.dictionariesContracttypesRead],
      },
    ],
  },
  {
    key: 'locations',
    title: '地点',
    description: '管理地点、楼层及区域信息。',
    icon: MapPin,
    href: '/assets/locations',
    countKey: 'locations',
    permissions: [PERM.assetsLocationsRead],
    actions: [
      {
        label: '查找',
        href: '/assets/locations',
        icon: Search,
        permissions: [PERM.assetsLocationsRead],
      },
      {
        label: '新增',
        href: '/assets/locations?create=1',
        icon: Plus,
        permissions: [PERM.assetsLocationsManage],
      },
    ],
  },
  {
    key: 'racks',
    title: '机架',
    description: '新增与查看机架及占用状态。',
    icon: Warehouse,
    href: '/assets/racks',
    countKey: 'racks',
    permissions: [PERM.assetsRacksRead],
    actions: [
      { label: '查找', href: '/assets/racks', icon: Search, permissions: [PERM.assetsRacksRead] },
      {
        label: '新增',
        href: '/assets/racks?create=1',
        icon: Plus,
        permissions: [PERM.assetsRacksManage],
      },
    ],
  },
  {
    key: 'labels',
    title: '打印标签',
    description: '选择并打印资产标签。',
    icon: Printer,
    href: '/labels',
    permissions: [PERM.labelsPreview],
    actions: [
      { label: '进入打印', href: '/labels', icon: Printer, permissions: [PERM.labelsPreview] },
    ],
  },
  {
    key: 'reports',
    title: '统计报表',
    description: '查看资产统计报表与清单。',
    icon: BarChart3,
    href: '/reports',
    permissions: [PERM.reportsRead],
    actions: [
      { label: '查看报表', href: '/reports', icon: BarChart3, permissions: [PERM.reportsRead] },
    ],
  },
  {
    key: 'browse',
    title: '资产导航',
    description: '按类型、用户、代理等维度树形导航定位资产。',
    icon: Boxes,
    href: '/browse',
    permissions: [PERM.browseRead],
    actions: [{ label: '浏览', href: '/browse', icon: Search, permissions: [PERM.browseRead] }],
  },
  {
    key: 'settings',
    title: '系统配置',
    description: '维护用户权限、身份认证与邮件配置。',
    icon: Settings,
    href: '/settings',
    permissions: SETTINGS_READ_PERMISSIONS,
    actions: [
      {
        label: '进入配置',
        href: '/settings',
        icon: Settings,
        permissions: SETTINGS_READ_PERMISSIONS,
      },
    ],
  },
];

export function ITDBDashboardPage() {
  const summary = useQuery({
    queryKey: ['itdb', 'dashboard-summary'],
    queryFn: () => api<DashboardResponse>('/api/dashboard/summary'),
    retry: (failureCount, error) => !isApiStatus(error, 403) && failureCount < 3,
  });
  const summaryForbidden = isApiStatus(summary.error, 403);
  const counts = summary.data?.counts ?? {};
  const user = getStoredUser();
  const visibleCards = cards.map(card => ({
    ...card,
    actions: card.actions.filter(action => userHasAnyPermission(user, action.permissions)),
  }));

  return (
    <section className="flex h-full min-h-0 flex-col gap-4 overflow-visible">
      <header>
        <div>
          <h1 className="text-lg font-semibold text-[var(--itdb-text)]">首页</h1>
          <p className="mt-1 text-sm text-[var(--itdb-text-muted)]">
            资产、授权、合同与基础配置的统一工作台
          </p>
        </div>
      </header>
      {summary.isError ? (
        summaryForbidden ? (
          <p className="rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-4 py-3 text-sm text-[var(--itdb-text-muted)]">
            暂无统计数据查看权限，如需查看请联系管理员分配
          </p>
        ) : (
          <p className="rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-500">
            首页数据加载失败，请稍后重试
          </p>
        )
      ) : null}
      <div className="itdb-dashboard-grid grid min-h-0 flex-1 auto-rows-fr grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 lg:grid-rows-4 2xl:gap-6">
        {visibleCards.map(card => {
          const Icon = card.icon;
          const canViewCount = userHasAnyPermission(user, card.permissions);
          const count = card.countKey && canViewCount ? Number(counts[card.countKey] ?? 0) : null;
          return (
            <article
              key={card.key}
              className="itdb-dashboard-card itdb-surface-3d @container group flex min-h-0 items-center gap-3 overflow-hidden rounded-xl border p-3 transition-transform hover:-translate-y-0.5 xl:gap-4 xl:p-4 min-[2200px]:gap-5 min-[2200px]:p-6"
            >
              <Link
                to={card.href}
                className="grid h-12 w-12 shrink-0 place-items-center rounded-lg border border-[rgba(59,130,246,0.18)] bg-[rgba(59,130,246,0.08)] text-[var(--itdb-accent)] transition-colors group-hover:bg-[rgba(59,130,246,0.14)] @lg:h-16 @lg:w-16 @lg:rounded-xl @2xl:h-[72px] @2xl:w-[72px] @2xl:rounded-2xl"
                aria-label={`进入${card.title}`}
              >
                <Icon className="size-[25px] @lg:size-10 @2xl:size-12" aria-hidden="true" />
              </Link>
              <div className="itdb-hidden-scrollbar flex max-h-full min-w-0 flex-1 flex-col overflow-x-hidden overflow-y-auto">
                <div className="flex shrink-0 items-start justify-between gap-3">
                  <Link
                    to={card.href}
                    className="text-base font-bold text-[var(--itdb-text)] hover:text-[var(--itdb-accent)] @lg:text-2xl @2xl:text-3xl"
                  >
                    {card.title}
                  </Link>
                  <span className="shrink-0 text-sm font-bold text-[var(--itdb-accent)] @lg:text-lg @2xl:text-2xl">
                    {count === null ? '功能入口' : summary.isLoading ? '总数：…' : `总数：${count}`}
                  </span>
                </div>
                <p className="mt-2 line-clamp-2 shrink-0 text-xs leading-4 text-[var(--itdb-text-muted)] @lg:mt-3 @lg:text-[15px] @lg:leading-6 @2xl:mt-3.5 @2xl:text-lg @2xl:leading-7">
                  {card.description}
                </p>
                <div className="mt-3 flex shrink-0 flex-wrap gap-1.5 @lg:gap-2 @2xl:gap-2.5">
                  {card.actions.map(action => {
                    const ActionIcon = action.icon;
                    return (
                      <Link
                        key={action.label}
                        to={action.href}
                        className="itdb-action-button inline-flex h-7 items-center gap-1 rounded-md border px-2 text-[11px] @lg:h-9 @lg:gap-1.5 @lg:rounded-lg @lg:px-3 @lg:text-sm @2xl:h-10 @2xl:px-3.5 @2xl:text-base"
                      >
                        <ActionIcon
                          className="size-3 @lg:size-4 @2xl:size-[18px]"
                          aria-hidden="true"
                        />
                        {action.label}
                      </Link>
                    );
                  })}
                </div>
              </div>
            </article>
          );
        })}
      </div>
    </section>
  );
}
