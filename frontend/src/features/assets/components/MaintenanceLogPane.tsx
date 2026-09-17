import { useQuery } from '@tanstack/react-query';
import { CheckCircle2, Download, Search, XCircle } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { trackAuditEvent } from '@/lib/audit-track';
import { api } from '@/lib/auth';
import { createExportBlob, downloadBlob, localTimestamp } from '@/lib/export-data';
import { compareTableValues, formatValue } from '../resource-helpers';
import { SortHeaderButton } from './HardwarePanes';

type Row = Record<string, unknown>;

export function MaintenanceLogPane({
  itemId,
  exportName,
}: {
  itemId?: unknown;
  exportName?: string;
}) {
  const [keyword, setKeyword] = useState('');
  const [sort, setSort] = useState<{ key: string; direction: 'asc' | 'desc' }>({
    key: 'id',
    direction: 'asc',
  });
  const logs = useQuery({
    queryKey: ['itdb', 'items', 'actions', itemId],
    enabled: Boolean(itemId),
    queryFn: () => api<Row[]>(`/api/items/${itemId}/actions`),
  });
  const allRows = Array.isArray(logs.data) ? logs.data : [];
  const dateText = (value: unknown) => (Number(value) > 0 ? formatValue(value, 'date') : '-');
  const descriptionText = (row: Row) => String(row.description ?? '').trim();
  const resultText = (row: Row) =>
    String(row.invoiceinfo ?? '').trim() === '失败' ? '失败' : '成功';
  const rowText = [
    (row: Row) => String(row.id ?? ''),
    (row: Row) => dateText(row.actiondate),
    (row: Row) => descriptionText(row),
    (row: Row) => resultText(row),
  ];
  const text = keyword.trim().toLowerCase();
  const rows = text
    ? allRows.filter(row => rowText.some(pick => pick(row).toLowerCase().includes(text)))
    : allRows;
  const sortValue = (row: Row, key: string) => {
    if (key === 'id' || key === 'actiondate') return Number(row[key] ?? 0);
    return String(row[key] ?? '').trim();
  };
  const sortedRows = [...rows].sort((left, right) => {
    const compared = compareTableValues(sortValue(left, sort.key), sortValue(right, sort.key));
    return sort.direction === 'asc' ? compared : -compared;
  });
  const toggleSort = (key: string) =>
    setSort(current =>
      current.key === key
        ? { key, direction: current.direction === 'asc' ? 'desc' : 'asc' }
        : { key, direction: 'asc' }
    );
  const exportLog = () => {
    if (sortedRows.length === 0) {
      toast.error('没有可导出的维护日志数据');
      return;
    }
    const matrix = [
      ['编号', '更新日期', '变更说明', '处理情况'],
      ...sortedRows.map(row => [
        Number(row.id ?? 0),
        dateText(row.actiondate),
        descriptionText(row),
        resultText(row),
      ]),
    ];
    const base = (exportName ?? '').trim();
    downloadBlob(
      createExportBlob(matrix, 'xlsx'),
      `${base ? `${base} ` : ''}维护日志-${localTimestamp().slice(0, 12)}.xlsx`
    );
    if (itemId) {
      void trackAuditEvent({
        type: 'export:items-maintenance',
        target: String(itemId),
        count: sortedRows.length,
      });
    }
    const assetName = (exportName ?? '').trim();
    toast.success(assetName ? `已导出 ${assetName} 维护日志` : '已导出维护日志');
  };
  const logColumns = [
    { key: 'id', label: '编号', className: 'w-24 px-3 py-2.5 text-center font-medium' },
    { key: 'actiondate', label: '更新日期' },
    { key: 'description', label: '变更说明' },
    { key: 'invoiceinfo', label: '处理情况' },
  ];
  return (
    <section className="mx-auto w-full max-w-6xl rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-5 pb-5 pt-3 flex min-h-0 flex-1 flex-col">
      <div className="flex shrink-0 flex-wrap items-center gap-3 border-b border-[var(--itdb-border)] pb-3">
        <h3 className="text-base font-semibold text-[var(--itdb-text)]">维护日志</h3>
        <label className="relative w-64">
          <Search
            className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--itdb-text-muted)]"
            size={15}
          />
          <Input
            value={keyword}
            onChange={event => setKeyword(event.target.value)}
            className="h-9 pl-8"
            placeholder="输入关键字筛选"
          />
        </label>
        <span className="text-sm text-[var(--itdb-text-muted)]">共 {rows.length} 条</span>
        <Button
          type="button"
          variant="outline"
          className="itdb-action-button itdb-maintenance-export-button ml-auto"
          onClick={exportLog}
        >
          <Download size={16} />
          导出
        </Button>
      </div>
      <div className="itdb-hidden-scrollbar itdb-resource-data-panel mt-4 min-h-0 flex-1 overflow-auto rounded-lg border border-[var(--itdb-border)]">
        <table className="min-w-full text-sm">
          <thead className="sticky top-0 z-10 bg-[var(--itdb-control-bg-soft)] text-[var(--itdb-text-muted)]">
            <tr>
              {logColumns.map(column => (
                <th
                  key={column.key}
                  className={`whitespace-nowrap px-3 py-2.5 text-center font-medium ${column.className ?? ''}`}
                >
                  <SortHeaderButton
                    label={column.label}
                    columnKey={column.key}
                    sort={sort}
                    onToggle={toggleSort}
                  />
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {logs.isLoading ? (
              <tr>
                <td
                  colSpan={4}
                  className="px-4 py-8 text-center text-sm text-[var(--itdb-text-muted)]"
                >
                  加载中
                </td>
              </tr>
            ) : (
              <>
                {sortedRows.map(row => {
                  const description = descriptionText(row) || '-';
                  const failed = resultText(row) === '失败';
                  const ResultIcon = failed ? XCircle : CheckCircle2;
                  return (
                    <tr
                      key={String(row.id)}
                      className="border-t border-[var(--itdb-border)]/70 hover:bg-[var(--itdb-control-bg-soft)]"
                    >
                      <td className="px-3 py-2.5 text-center">
                        <span className="itdb-relation-id">{String(row.id ?? '-')}</span>
                      </td>
                      <td className="whitespace-nowrap px-3 py-2.5 text-center text-[var(--itdb-text)]">
                        {dateText(row.actiondate)}
                      </td>
                      <td className="px-3 py-2.5 text-center text-[var(--itdb-text)]">
                        <AppTooltip
                          className="mx-auto block max-w-2xl"
                          label={description === '-' ? undefined : description}
                          placement="top"
                          wrap
                        >
                          <span className="block truncate">{description}</span>
                        </AppTooltip>
                      </td>
                      <td className="px-3 py-2.5 text-center">
                        <span
                          className={`inline-flex items-center justify-self-center gap-1 rounded-full border px-2 py-0.5 text-xs ${failed ? 'border-red-400/30 bg-red-400/10 text-red-600 dark:text-red-300' : 'border-emerald-400/30 bg-emerald-400/10 text-emerald-600 dark:text-emerald-300'}`}
                        >
                          <ResultIcon size={12} />
                          {failed ? '失败' : '成功'}
                        </span>
                      </td>
                    </tr>
                  );
                })}
                {rows.length === 0 ? (
                  <tr>
                    <td
                      colSpan={4}
                      className="px-4 py-8 text-center text-sm text-[var(--itdb-text-muted)]"
                    >
                      {itemId ? '暂无维护日志' : '新增记录后可查看维护日志。'}
                    </td>
                  </tr>
                ) : null}
              </>
            )}
          </tbody>
        </table>
      </div>
    </section>
  );
}
