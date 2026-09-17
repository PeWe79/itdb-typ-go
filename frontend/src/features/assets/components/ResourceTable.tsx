import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Download,
  Edit3,
  Trash2,
  ChevronDown,
  ChevronUp,
  ChevronsUpDown,
  Plus,
  Search,
  Copy,
} from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import { AppTooltip } from '@/components/app-tooltip';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { Pagination } from '@/components/pagination';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { api, getStoredUser, userHasPermission } from '@/lib/auth';
import { assetPermission } from '@/lib/permissions';
import { type ResourceConfig } from '../resource-config';
import {
  getRowSearchText,
  compareTableValues,
  getSortValue,
  isHexColor,
} from '../resource-helpers';
import { ResourceEditor } from './ResourceEditor';
import { AssetExportDialog } from './AssetExportDialog';
import { TableCellValue } from './TableCellValue';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;
const pageableResourceKeys = new Set(['items', 'software', 'invoices']);

export function ResourceTable({ resource }: { resource: ResourceConfig }) {
  const queryClient = useQueryClient();
  const canManage = userHasPermission(getStoredUser(), assetPermission(resource.key, 'manage'));
  const [keyword, setKeyword] = useState('');
  const [sort, setSort] = useState<{ key: string; direction: 'asc' | 'desc' }>({
    key: 'id',
    direction: resource.key === 'items' ? 'desc' : 'asc',
  });
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(18);
  const [editor, setEditor] = useState<{ row?: Row; prefillId?: number } | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Row | null>(null);
  const [exportOpen, setExportOpen] = useState(false);
  const query = useQuery({
    queryKey: ['itdb', resource.key],
    queryFn: () => {
      const queryString = pageableResourceKeys.has(resource.key) ? '?limit=-1&offset=0' : '';
      return api<Row[]>(`/api${resource.endpoint}${queryString}`);
    },
  });
  const bootstrap = useQuery({
    queryKey: ['itdb', 'bootstrap'],
    queryFn: () => api<Lookups>('/api/bootstrap'),
    enabled: resource.key === 'items' || resource.key === 'software',
  });
  const statusColors = useMemo(
    () =>
      new Map(
        (bootstrap.data?.statustypes ?? []).map(status => [
          String(status.statusdesc ?? '').trim(),
          String(status.color ?? '').trim(),
        ])
      ),
    [bootstrap.data]
  );
  const installedItemColors = useMemo(() => {
    const colors = new Map<number, string>();
    (bootstrap.data?.items_ref ?? []).forEach(item => {
      const direct = String(item.statuscolor ?? item.statusColor ?? '').trim();
      const color = isHexColor(direct)
        ? direct
        : statusColors.get(String(item.status ?? item.statusdesc ?? '').trim());
      const id = Number(item.id);
      if (id > 0 && isHexColor(color)) colors.set(id, color);
    });
    return colors;
  }, [bootstrap.data, statusColors]);
  const filteredRows = useMemo(() => {
    const rows = Array.isArray(query.data) ? query.data : [];
    const normalized = keyword.trim().toLowerCase();
    const matched = !normalized
      ? rows
      : rows.filter(row =>
          getRowSearchText(
            row,
            resource.columns.map(column => column.key)
          ).includes(normalized)
        );
    return [...matched].sort(
      (left, right) =>
        compareTableValues(getSortValue(left, sort.key), getSortValue(right, sort.key)) *
        (sort.direction === 'asc' ? 1 : -1)
    );
  }, [keyword, query.data, resource.columns, sort]);
  const effectivePageSize = pageSize === -1 ? Math.max(filteredRows.length, 1) : pageSize;
  const totalPages = Math.max(1, Math.ceil(filteredRows.length / effectivePageSize));
  const currentPage = Math.min(page, totalPages);
  const visibleRows = filteredRows.slice(
    (currentPage - 1) * effectivePageSize,
    currentPage * effectivePageSize
  );
  useEffect(() => setPage(1), [keyword, pageSize]);
  useEffect(() => {
    if (!canManage) return;
    const openFromParams = () => {
      const params = new URLSearchParams(window.location.search);
      const editId = Number(params.get('edit'));
      const copyId = Number(params.get('copy'));
      if (params.get('create') === '1') {
        setEditor({});
      } else if (Number.isFinite(editId) && editId > 0) {
        setEditor({ row: { id: editId } });
      } else if (Number.isFinite(copyId) && copyId > 0) {
        setEditor({ prefillId: copyId });
      } else {
        return;
      }
      window.history.replaceState({}, '', window.location.pathname);
    };
    openFromParams();
    window.addEventListener('itdb-open-editor', openFromParams);
    return () => window.removeEventListener('itdb-open-editor', openFromParams);
  }, [resource.key, canManage]);

  const invalidate = async () => {
    await queryClient.invalidateQueries({ queryKey: ['itdb', resource.key] });
  };
  const remove = useMutation({
    mutationFn: async (id: string) => {
      await api<void>(`/api${resource.endpoint}/${id}`, { method: 'DELETE' });
      return id;
    },
    onSuccess: id => {
      toast.success(`${resource.title}编号 ${id} 已删除`);
      void invalidate();
    },
    onError: error => showErrorToast(error.message),
  });
  const toggleSort = (key: string) =>
    setSort(previous => {
      if (previous.key !== key) return { key, direction: 'asc' };
      return {
        key,
        direction: previous.direction === 'asc' ? 'desc' : 'asc',
      };
    });
  return (
    <section
      className="itdb-card-hover flex min-h-0 flex-1 flex-col gap-4 overflow-hidden rounded-xl p-5"
      style={{
        background: 'var(--itdb-card)',
        border: '1px solid var(--itdb-border)',
        boxShadow: 'var(--shadow-card)',
      }}
    >
      <header className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-sm text-[var(--itdb-text-muted)]">
          搜索、维护和关联{resource.title}记录
        </p>
        <div className="flex w-full flex-wrap items-center justify-end gap-2 sm:w-auto">
          <label className="relative min-w-0 flex-1 sm:w-64">
            <Search
              className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--itdb-text-muted)]"
              size={16}
            />
            <Input
              value={keyword}
              onChange={event => setKeyword(event.target.value)}
              placeholder="输入关键字实时搜索"
              className="pl-9"
            />
          </label>
          {canManage && (
            <Button
              type="button"
              variant="outline"
              className="itdb-action-button itdb-dict-export-button"
              onClick={() => {
                if (filteredRows.length === 0) {
                  toast.error(`没有可导出的${resource.title}数据`);
                  return;
                }
                setExportOpen(true);
              }}
            >
              <Download size={16} />
              导出
            </Button>
          )}
          {!resource.readonly && canManage && (
            <Button
              type="button"
              variant="outline"
              className="itdb-action-button"
              style={{
                borderColor: 'rgba(59,130,246,0.38)',
                background: 'rgba(59,130,246,0.1)',
                color: 'var(--itdb-accent-text)',
              }}
              onClick={() => setEditor({})}
            >
              <Plus size={16} />
              新增
            </Button>
          )}
        </div>
      </header>
      <div className="itdb-resource-data-panel flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-[var(--itdb-border)]">
        <div className="itdb-hidden-scrollbar min-h-0 flex-1 overflow-auto">
          <table className="min-w-full text-center text-sm">
            <thead className="sticky top-0 z-30 text-center text-[var(--itdb-text-muted)]">
              <tr>
                {resource.columns.map(column => (
                  <th
                    key={column.key}
                    className="whitespace-nowrap border-b border-[var(--itdb-border)] px-4 py-3 text-center font-medium"
                  >
                    <AppTooltip label={column.tooltip ?? column.label}>
                      <button
                        type="button"
                        className="inline-flex items-center gap-1"
                        onClick={() => toggleSort(column.key)}
                      >
                        {column.label}
                        {sort.key === column.key ? (
                          sort.direction === 'asc' ? (
                            <ChevronUp size={14} />
                          ) : (
                            <ChevronDown size={14} />
                          )
                        ) : (
                          <ChevronsUpDown size={14} className="opacity-55" />
                        )}
                      </button>
                    </AppTooltip>
                  </th>
                ))}
                {canManage && (
                  <th className="itdb-resource-operation-cell sticky right-0 z-40 border-b border-[var(--itdb-border)] px-3 py-3 text-center font-medium">
                    操作
                  </th>
                )}
              </tr>
            </thead>
            <tbody>
              {visibleRows.map((row, index) => {
                const id = String(row.id ?? index);
                return (
                  <tr
                    key={id}
                    className="border-b border-[var(--itdb-border)]/70 hover:bg-[var(--itdb-control-bg-soft)]"
                  >
                    {resource.columns.map(column => (
                      <td
                        key={column.key}
                        className={
                          column.key === 'installedon'
                            ? 'min-w-[300px] whitespace-nowrap px-4 py-3 text-left text-[var(--itdb-text)]'
                            : 'max-w-64 whitespace-nowrap px-4 py-3 text-center text-[var(--itdb-text)]'
                        }
                      >
                        <TableCellValue
                          row={row}
                          columnKey={column.key}
                          statusColors={statusColors}
                          installedItemColors={installedItemColors}
                          resourceKey={resource.key}
                        />
                      </td>
                    ))}
                    {canManage && (
                      <td className="itdb-resource-operation-cell sticky right-0 z-20 px-3 py-2">
                        <div className="flex justify-center gap-2.5">
                          <AppTooltip label="编辑">
                            <Button
                              size="icon"
                              variant="ghost"
                              className="itdb-action-button h-7 w-7"
                              style={{
                                borderColor: 'rgba(59,130,246,0.38)',
                                background: 'rgba(59,130,246,0.1)',
                                color: 'var(--itdb-accent-text)',
                              }}
                              onClick={() => setEditor({ row })}
                              aria-label={`编辑${resource.title}`}
                            >
                              <Edit3 size={14} />
                            </Button>
                          </AppTooltip>
                          {!resource.readonly && (
                            <>
                              <AppTooltip label="复制">
                                <Button
                                  size="icon"
                                  variant="ghost"
                                  className="itdb-action-button h-7 w-7"
                                  style={{
                                    borderColor: 'rgba(59,130,246,0.38)',
                                    background: 'rgba(59,130,246,0.1)',
                                    color: 'var(--itdb-accent-text)',
                                  }}
                                  onClick={() => {
                                    const url = new URL(window.location.href);
                                    url.searchParams.set('copy', String(row.id));
                                    window.history.replaceState({}, '', url);
                                    window.dispatchEvent(new Event('itdb-open-editor'));
                                  }}
                                  aria-label={`复制${resource.title}`}
                                >
                                  <Copy size={14} />
                                </Button>
                              </AppTooltip>
                              <AppTooltip label="删除">
                                <Button
                                  size="icon"
                                  variant="ghost"
                                  className="itdb-action-button h-7 w-7"
                                  style={{
                                    borderColor: 'rgba(239,68,68,0.34)',
                                    background: 'rgba(239,68,68,0.08)',
                                    color: '#ef4444',
                                  }}
                                  onClick={() => setDeleteTarget(row)}
                                  aria-label={`删除${resource.title}`}
                                >
                                  <Trash2 size={14} />
                                </Button>
                              </AppTooltip>
                            </>
                          )}
                        </div>
                      </td>
                    )}
                  </tr>
                );
              })}
            </tbody>
          </table>
          {!query.isLoading && visibleRows.length === 0 && (
            <div className="grid min-h-52 place-items-center px-6 text-sm text-[var(--itdb-text-muted)]">
              {query.isError ? `${resource.title}数据加载失败` : `暂无${resource.title}记录`}
            </div>
          )}
          {query.isLoading && (
            <div className="grid min-h-52 place-items-center text-sm text-[var(--itdb-text-muted)]">
              数据加载中...
            </div>
          )}
        </div>
        <Pagination
          page={currentPage}
          pageCount={totalPages}
          pageSize={pageSize}
          total={filteredRows.length}
          onPageChange={setPage}
          onPageSizeChange={setPageSize}
        />
      </div>
      {editor && (
        <ResourceEditor
          resource={resource}
          row={editor.row}
          prefillId={editor.prefillId}
          onClose={() => setEditor(null)}
          onSaved={() => {
            setEditor(null);
            void invalidate();
          }}
        />
      )}
      <AssetExportDialog
        open={exportOpen}
        resource={resource}
        rows={filteredRows}
        onOpenChange={setExportOpen}
      />
      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title={`删除${resource.title}记录`}
        description="删除后不可恢复，关联关系也会一并移除"
        detail={`确认删除 编号=${String(deleteTarget?.id ?? '-')} 的记录吗？`}
        horizontalHeader
        confirmText="确认删除"
        busy={remove.isPending}
        softDestructive
        onOpenChange={open => !open && setDeleteTarget(null)}
        onConfirm={() => {
          if (deleteTarget?.id !== undefined) {
            remove.mutate(String(deleteTarget.id), { onSuccess: () => setDeleteTarget(null) });
          }
        }}
      />
    </section>
  );
}
