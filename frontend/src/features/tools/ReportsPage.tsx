import { useQuery } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import { ChevronDown, ChevronUp, ChevronsUpDown } from 'lucide-react';
import { AppTooltip } from '@/components/app-tooltip';
import { PermissionGate } from '@/components/permission-gate';
import { api } from '@/lib/auth';
import { PERM } from '@/lib/permissions';
import { canOpenRecordEditor } from '@/features/assets/record-links';
import { compareTableValues, getSortValue } from '@/features/assets/resource-helpers';

type Row = Record<string, unknown>;
export type ReportMeta = { name: string; description: string; chartType?: string };
type ReportResult = { meta: ReportMeta; rows: Row[]; chart: Array<{ x: string; y: number }> };

// 报告中文名（与原 itdb 项目一致的菜单文案）
const reportTitleMap: Record<string, string> = {
  itemperagent: '每个厂商的硬件数量(代理)',
  softwareperagent: '每个厂商已安装的软件数量(代理)',
  invoicesperagent: '每个供应商的单据数量(代理)',
  itemsperlocation: '每个地点的硬件数量',
  percsupitems: '在保的硬件数量',
  itemlistperlocation: '按地点显示硬件',
  itemsendwarranty: '今日前后授权过期的硬件',
  allips: '指定了 IPv4 的硬件清单',
  allip6: '指定了 IPv6 的硬件清单',
  noinvoice: '无单据的硬件',
  nolocation: '无地点的硬件',
  depreciation3: '硬件折旧价值 3 年',
  depreciation5: '硬件折旧价值 5 年',
};

// 报告标题行右侧的括号说明（仅展示在卡片内标题，下拉选项保持简洁）
const reportHintMap: Record<string, string> = {
  itemsendwarranty: '维保将在今日前后 360 天内到期，按剩余天数升序',
  depreciation3: '按 3 年直线折旧估算现值：每月扣减采购价的 1/36',
  depreciation5: '按 5 年直线折旧估算现值：每月扣减采购价的 1/60',
};

const reportColumnOrderMap: Record<string, string[]> = {
  itemperagent: ['totalcount', 'Agent', 'ID'],
  softwareperagent: ['totalcount', 'Agent', 'ID'],
  invoicesperagent: ['totalcount', 'Agent', 'ID'],
  itemsperlocation: ['totalcount', 'Location', 'Floor'],
  percsupitems: ['Type', 'Items'],
  itemlistperlocation: [
    'ID',
    'type',
    'manufacturer',
    'model',
    'dnsname',
    'Location',
    'Floor',
    'Area',
  ],
  itemsendwarranty: [
    'ID',
    'ipv4',
    'type',
    'manufacturer',
    'model',
    'dnsname',
    'label',
    'RemainingDays',
  ],
  allips: ['ID', 'ipv4', 'type', 'manufacturer', 'model', 'dnsname', 'label'],
  allip6: ['ID', 'ipv6', 'type', 'manufacturer', 'model', 'dnsname', 'label'],
  noinvoice: ['ID', 'type', 'manufacturer', 'model', 'PurchaseDate'],
  nolocation: ['ID', 'type', 'manufacturer', 'model'],
  depreciation3: [
    'ID',
    'type',
    'manufacturer',
    'model',
    'PurchaseDate',
    'PurchasePrice',
    'Months',
    'CurrentValue',
  ],
  depreciation5: [
    'ID',
    'type',
    'manufacturer',
    'model',
    'PurchaseDate',
    'PurchasePrice',
    'Months',
    'CurrentValue',
  ],
};

const reportColumnLabelMap: Record<string, string> = {
  ID: '编号',
  totalcount: '数量',
  Agent: '厂商',
  Location: '地点',
  Floor: '楼层',
  Area: '区域/办公室',
  Type: '类型',
  Items: '数量',
  type: '类型',
  manufacturer: '厂商',
  model: '型号',
  dnsname: '业务跳线',
  label: '标签',
  RemainingDays: '维保剩余天数',
  ipv4: 'IPv4',
  ipv6: 'IPv6',
  PurchaseDate: '采购日期',
  PurchasePrice: '采购价格',
  Months: '月数',
  CurrentValue: '当前价值',
};

// ID 列徽章的跳转目标：厂商类报告跳厂商编辑，资产类报告跳硬件编辑
const reportEditTargetMap: Record<string, { path: string; noun: string }> = {
  itemperagent: { path: '/assets/agents', noun: '厂商' },
  softwareperagent: { path: '/assets/agents', noun: '厂商' },
  invoicesperagent: { path: '/assets/agents', noun: '厂商' },
  itemlistperlocation: { path: '/assets/hardware', noun: '硬件' },
  itemsendwarranty: { path: '/assets/hardware', noun: '硬件' },
  allips: { path: '/assets/hardware', noun: '硬件' },
  allip6: { path: '/assets/hardware', noun: '硬件' },
  noinvoice: { path: '/assets/hardware', noun: '硬件' },
  nolocation: { path: '/assets/hardware', noun: '硬件' },
  depreciation3: { path: '/assets/hardware', noun: '硬件' },
  depreciation5: { path: '/assets/hardware', noun: '硬件' },
};

export function reportDisplayName(report: ReportMeta) {
  return reportTitleMap[report.name] ?? report.description ?? report.name;
}

function parsePositiveID(value: unknown) {
  const id = Number.parseInt(String(value ?? '').trim(), 10);
  return Number.isFinite(id) && id > 0 ? id : 0;
}

export type ReportExportColumn = { key: string; label: string };

// useReportData 拉取指定报表数据并派生列顺序、导出列与关键词过滤结果，供报表页与导出弹窗共用
export function useReportData(active: string, keyword: string, enabled: boolean) {
  const result = useQuery({
    queryKey: ['itdb', 'report', active],
    enabled: enabled && Boolean(active),
    queryFn: () => api<ReportResult>(`/api/reports/${encodeURIComponent(active)}?limit=1000`),
  });
  const rows = Array.isArray(result.data?.rows) ? result.data!.rows : [];
  const chart = Array.isArray(result.data?.chart) ? result.data!.chart : [];
  const columns = useMemo(() => {
    const rowKeys = Object.keys(rows[0] ?? {});
    const preferred = reportColumnOrderMap[active] ?? rowKeys;
    const ordered = preferred.filter(key => rowKeys.includes(key));
    const remaining = rowKeys.filter(key => !ordered.includes(key));
    return [...ordered, ...remaining];
  }, [active, rows]);
  const exportColumns = useMemo<ReportExportColumn[]>(
    () => columns.map(key => ({ key, label: reportColumnLabelMap[key] ?? key })),
    [columns]
  );
  const filteredRows = useMemo(() => {
    const q = keyword.trim().toLowerCase();
    if (!q) return rows;
    return rows.filter(row =>
      Object.keys(row).some(key =>
        String(row[key] ?? '')
          .toLowerCase()
          .includes(q)
      )
    );
  }, [keyword, rows]);
  return {
    isLoading: result.isLoading,
    rows,
    chart,
    columns,
    exportColumns,
    filteredRows,
    hasChart: chart.length > 0,
  };
}

export function ReportsPage({ active, keyword }: { active: string; keyword: string }) {
  const [sort, setSort] = useState<{ key: string; direction: 'asc' | 'desc' }>({
    key: '#',
    direction: 'asc',
  });
  const { isLoading, chart, columns, filteredRows, hasChart } = useReportData(
    active,
    keyword,
    true
  );
  const activeTitle = active
    ? reportHintMap[active]
      ? `${reportTitleMap[active] ?? active}（${reportHintMap[active]}）`
      : (reportTitleMap[active] ?? active)
    : '';

  const sortedRows = useMemo(() => {
    if (sort.key === '#') {
      return sort.direction === 'asc' ? filteredRows : [...filteredRows].reverse();
    }
    return [...filteredRows].sort(
      (left, right) =>
        compareTableValues(getSortValue(left, sort.key), getSortValue(right, sort.key)) *
        (sort.direction === 'asc' ? 1 : -1)
    );
  }, [filteredRows, sort]);

  const toggleSort = (key: string) =>
    setSort(previous => {
      if (previous.key !== key) return { key, direction: 'asc' };
      return {
        key,
        direction: previous.direction === 'asc' ? 'desc' : 'asc',
      };
    });

  const sortIcon = (key: string) =>
    sort.key === key ? (
      sort.direction === 'asc' ? (
        <ChevronUp size={14} />
      ) : (
        <ChevronDown size={14} />
      )
    ) : (
      <ChevronsUpDown size={14} className="opacity-55" />
    );

  const sortHeader = (key: string, label: string) => (
    <button
      type="button"
      className="relative inline-flex items-center"
      onClick={() => toggleSort(key)}
    >
      {label}
      <span className="pointer-events-none absolute -right-4 top-1/2 -translate-y-1/2">
        {sortIcon(key)}
      </span>
    </button>
  );

  const openRowEditor = (key: string, value: unknown) => {
    if (key.trim().toLowerCase() !== 'id') return;
    const target = reportEditTargetMap[active];
    const id = parsePositiveID(value);
    if (!target || !id) return;
    window.open(`${target.path}?edit=${id}`, '_blank', 'noopener');
  };

  const table = (
    <table className="w-full min-w-[720px] table-fixed text-center text-sm">
      <thead className="sticky top-0 z-30">
        <tr>
          <th className="whitespace-nowrap border-b border-[var(--itdb-border)] px-4 py-3 text-center font-medium">
            {sortHeader('#', '#')}
          </th>
          {columns.map(key => (
            <th
              key={key}
              className="whitespace-nowrap border-b border-[var(--itdb-border)] px-4 py-3 text-center font-medium"
            >
              {sortHeader(key, reportColumnLabelMap[key] ?? key)}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {sortedRows.map((row, index) => (
          <tr
            key={index}
            className="border-b border-[var(--itdb-border)]/70 hover:bg-[var(--itdb-control-bg-soft)]"
          >
            <td className="px-3 py-2.5 text-center">
              <span
                className="inline-flex min-w-10 justify-center rounded-full border px-2 py-0.5 text-xs font-semibold"
                style={{
                  borderColor: 'color-mix(in srgb, var(--itdb-accent2, #0891b2) 38%, transparent)',
                  background: 'color-mix(in srgb, var(--itdb-accent2, #0891b2) 12%, transparent)',
                  color: 'var(--itdb-accent2, #0891b2)',
                }}
              >
                {index + 1}
              </span>
            </td>
            {columns.map(key => {
              const text = String(row[key] ?? '');
              const target = reportEditTargetMap[active];
              const clickable =
                key.trim().toLowerCase() === 'id' &&
                target &&
                parsePositiveID(text) > 0 &&
                canOpenRecordEditor(target.path);
              return (
                <td key={key} className="truncate px-3 py-2.5 text-center text-[var(--itdb-text)]">
                  {clickable ? (
                    <AppTooltip label={`在新窗口编辑${target!.noun} ${text}`}>
                      <button
                        type="button"
                        className="itdb-invoice-file-pill"
                        style={{ minWidth: '2.5rem', justifyContent: 'center' }}
                        onClick={() => openRowEditor(key, text)}
                      >
                        {text}
                      </button>
                    </AppTooltip>
                  ) : key.trim().toLowerCase() === 'id' && target && parsePositiveID(text) > 0 ? (
                    <span
                      className="itdb-invoice-file-pill is-plain"
                      style={{ minWidth: '2.5rem', justifyContent: 'center' }}
                    >
                      {text}
                    </span>
                  ) : (
                    text || '-'
                  )}
                </td>
              );
            })}
          </tr>
        ))}
        {sortedRows.length === 0 ? (
          <tr>
            <td
              colSpan={columns.length + 1}
              className="px-4 py-8 text-center text-sm text-[var(--itdb-text-muted)]"
            >
              暂无报表数据
            </td>
          </tr>
        ) : null}
      </tbody>
    </table>
  );

  const tablePanel = hasChart ? (
    <div className="itdb-resource-data-panel itdb-hidden-scrollbar flex-1 overflow-x-auto rounded-lg border border-[var(--itdb-border)]">
      {table}
    </div>
  ) : (
    <div className="itdb-resource-data-panel itdb-hidden-scrollbar min-h-0 flex-1 overflow-auto rounded-lg border border-[var(--itdb-border)]">
      {table}
    </div>
  );

  return (
    <PermissionGate anyOf={[PERM.reportsRead]}>
      {hasChart ? (
        <div className="flex flex-1 flex-col gap-4">
          <div className="itdb-surface-3d flex flex-col gap-4 rounded-xl p-5">
            {activeTitle ? (
              <p className="text-sm text-[var(--itdb-text-muted)]">{activeTitle}</p>
            ) : null}
            <div className="grid grid-cols-[repeat(auto-fill,minmax(180px,1fr))] gap-3">
              {chart.map((item, index) => (
                <div
                  key={`${item.x}-${index}`}
                  className="itdb-surface-3d rounded-xl px-4 py-3"
                  style={{ background: 'var(--itdb-control-bg-soft)' }}
                >
                  <div className="truncate text-sm text-[var(--itdb-text-muted)]">{item.x}</div>
                  <div className="mt-1 text-2xl font-semibold text-[var(--itdb-text)]">
                    {item.y}
                  </div>
                </div>
              ))}
            </div>
          </div>
          <div className="itdb-surface-3d flex flex-1 flex-col rounded-xl p-5">{tablePanel}</div>
        </div>
      ) : (
        <div className="itdb-surface-3d flex min-h-0 flex-1 flex-col gap-4 rounded-xl p-5">
          {activeTitle ? (
            <p className="shrink-0 text-sm text-[var(--itdb-text-muted)]">{activeTitle}</p>
          ) : null}
          {isLoading ? (
            <div className="grid flex-1 place-items-center text-sm text-[var(--itdb-text-muted)]">
              加载中
            </div>
          ) : (
            tablePanel
          )}
        </div>
      )}
    </PermissionGate>
  );
}
