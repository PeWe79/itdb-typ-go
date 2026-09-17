import { Link } from '@tanstack/react-router';
import {
  Building2,
  FileText,
  FolderOpen,
  Handshake,
  MapPin,
  MonitorCog,
  PackageOpen,
  Warehouse,
} from 'lucide-react';
import { resourceMap, resources, type ResourceConfig } from './resource-config';
import { AppTooltip } from '@/components/app-tooltip';
import { PermissionGate } from '@/components/permission-gate';
import { getStoredUser, userHasPermission } from '@/lib/auth';
import { assetPermission } from '@/lib/permissions';
import { ResourceTable } from './components/ResourceTable';

const resourceIcons = {
  items: MonitorCog,
  software: PackageOpen,
  invoices: FileText,
  agents: Building2,
  files: FolderOpen,
  contracts: Handshake,
  locations: MapPin,
  racks: Warehouse,
};

// 菜单悬浮提示：六个菜单文案与原 itdb 项目侧边栏一致；地点、机架为当前项目补充
const resourceTooltips: Partial<Record<string, string>> = {
  items: '硬件清单',
  software: '软件清单',
  invoices: '单据清单',
  agents: '供应商/采购方/承包方/厂商',
  files: '文档, 手册, 认购书, 授权书, ...',
  contracts: '维保, 租赁, ...',
  locations: '建筑, 楼层, 区域/办公室, ...',
  racks: '机柜, U位, 硬件挂载, ...',
};

/* 菜单直达最终路由（items 走 hardware），配合 Link 渲染即预载对应路由分包，切换无加载间隙 */
const resourcePaths: Record<string, string> = {
  items: '/assets/hardware',
  software: '/assets/software',
  invoices: '/assets/invoices',
  agents: '/assets/agents',
  files: '/assets/files',
  contracts: '/assets/contracts',
  locations: '/assets/locations',
  racks: '/assets/racks',
};

export function ResourceListPage({ resourceKey }: { resourceKey: string }) {
  const resource = resourceMap[resourceKey];
  if (!resource) return <div className="text-sm text-red-500">未找到资产配置</div>;
  const user = getStoredUser();
  const permittedResources = resources.filter(item =>
    userHasPermission(user, assetPermission(item.key, 'read'))
  );
  return (
    <PermissionGate anyOf={[assetPermission(resource.key, 'read')]}>
      <div className="flex h-full min-h-0 flex-col gap-5">
        <header className="shrink-0">
          <h1 className="text-lg font-semibold text-[var(--itdb-text)]">资产管理</h1>
          <p className="mt-1 text-sm text-[var(--itdb-text-muted)]">
            集中维护硬件、软件、单据及相关资产
          </p>
        </header>
        <nav
          className="itdb-card-hover itdb-hidden-scrollbar flex shrink-0 overflow-x-auto rounded-xl p-3"
          aria-label="资产管理导航"
          style={{
            background: 'var(--itdb-card)',
            border: '1px solid var(--itdb-border)',
            boxShadow: 'var(--shadow-card)',
          }}
        >
          <div className="flex min-w-max flex-wrap gap-2">
            {permittedResources.map(item => (
              <ResourceSwitch
                key={item.key}
                to={resourcePaths[item.key] ?? `/assets/${item.key}`}
                active={item.key === resourceKey}
                item={item}
              />
            ))}
          </div>
        </nav>
        <ResourceTable key={resource.key} resource={resource} />
      </div>
    </PermissionGate>
  );
}

function ResourceSwitch({
  item,
  active,
  to,
}: {
  item: ResourceConfig;
  active: boolean;
  to: string;
}) {
  const Icon = resourceIcons[item.key as keyof typeof resourceIcons] ?? MonitorCog;
  return (
    <AppTooltip label={resourceTooltips[item.key]} placement="bottom">
      <Link
        to={to}
        onClick={event => {
          if (active) event.preventDefault();
        }}
        className="itdb-config-nav-card flex h-10 items-center gap-2 rounded-lg px-4 text-sm font-medium transition-all"
        data-active={active}
        style={{ color: active ? 'var(--itdb-accent-hover)' : 'var(--itdb-text-muted)' }}
      >
        <Icon size={16} />
        {item.title}
      </Link>
    </AppTooltip>
  );
}
