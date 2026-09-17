import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import {
  Boxes,
  ChevronDown,
  ChevronUp,
  Download,
  Edit3,
  FileDown,
  FileText,
  FolderOpen,
  Network,
  Plus,
  Search,
  Tag,
  Trash2,
  Upload,
  Users,
} from 'lucide-react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';

import { showErrorToast } from '@/lib/toast-errors';
import { AppTooltip } from '@/components/app-tooltip';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Pagination } from '@/components/pagination';
import { api, getStoredUser, userHasPermission } from '@/lib/auth';
import { trackAuditEvent } from '@/lib/audit-track';
import { downloadXlsxTemplate } from '@/lib/export-data';
import { dictionaryPermission } from '@/lib/permissions';
import { isHexColor } from '@/features/assets/resource-helpers';
import { PermissionGate } from '@/components/permission-gate';
import { ContractSubtypeDialog } from './ContractSubtypeDialog';
import { DictionaryEditorDialog, type DictionaryEditorValue } from './DictionaryEditorDialog';
import { DictionaryExportDialog } from './DictionaryExportDialog';
import { dictionaryRowSearchText } from './dictionary-helpers';
import {
  dictionaryImportHeaders,
  dictionaryImportPayloads,
  dictionaryImportTemplateSpecs,
  parseXlsxMatrix,
} from './dictionary-import';
import { TagRelatedDialog } from './TagRelatedDialog';

type Row = Record<string, unknown>;

export type DictionaryName =
  'itemtypes' | 'contracttypes' | 'statustypes' | 'filetypes' | 'dpttypes' | 'tags';
const dictionaries: { key: DictionaryName; label: string; field: string }[] = [
  { key: 'itemtypes', label: '硬件类型', field: 'typedesc' },
  { key: 'contracttypes', label: '合同类型', field: 'name' },
  { key: 'statustypes', label: '状态类型', field: 'statusdesc' },
  { key: 'filetypes', label: '文件类型', field: 'typedesc' },
  { key: 'dpttypes', label: '所属部门', field: 'dptname' },
  { key: 'tags', label: '标记', field: 'name' },
];
const dictionaryIcons = {
  itemtypes: Boxes,
  contracttypes: FileText,
  statustypes: Tag,
  filetypes: FolderOpen,
  dpttypes: Users,
  tags: Tag,
};
const protectedStatusDescriptions = new Set(['使用中', '库存', '有故障', '报废']);

/* builtinDictionaryNames 内置名称清单与后端判定保持一致：编号错位（如旧库迁移）时仍按名称保护 */
const builtinDictionaryNames: Partial<Record<DictionaryName, string[]>> = {
  itemtypes: ['服务器', '存储', '交换机', '电话', '安防'],
  filetypes: ['照片', '手册', '发票', '报价', '订单', '服务', '报告', '许可证', '合同', '其他'],
  contracttypes: ['支持 & 维护'],
};

function isProtectedDictionaryRow(name: DictionaryName, row: Row) {
  const id = Number(row.id ?? 0);
  if (name === 'itemtypes') {
    if (id >= 0 && id <= 5) return true;
  } else if (name === 'contracttypes') {
    if (id >= 0 && id <= 1) return true;
  } else if (name === 'filetypes') {
    if (id >= 0 && id <= 10) return true;
  } else if (name === 'statustypes') {
    if (id >= 0 && id <= 3) return true;
    return protectedStatusDescriptions.has(String(row.statusdesc ?? '').trim());
  }
  const names = builtinDictionaryNames[name];
  if (!names) return false;
  const field = name === 'contracttypes' ? 'name' : 'typedesc';
  const text = String(row[field] ?? '')
    .trim()
    .toLowerCase();
  return names.some(builtin => builtin.toLowerCase() === text);
}

export function DictionaryManager({
  initialDictionary = 'itemtypes',
}: {
  initialDictionary?: DictionaryName;
}) {
  const client = useQueryClient();
  const user = getStoredUser();
  const permittedDictionaries = dictionaries.filter(item =>
    userHasPermission(user, dictionaryPermission(item.key, 'read'))
  );
  const [active, setActive] = useState<DictionaryName>(initialDictionary);
  const canManage = userHasPermission(user, dictionaryPermission(active, 'manage'));
  const [keyword, setKeyword] = useState('');
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState<number>(18);
  const [sortDesc, setSortDesc] = useState(false);
  const [editor, setEditor] = useState<DictionaryEditorValue | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Row | null>(null);
  const [subtypesFor, setSubtypesFor] = useState<Row | null>(null);
  const [exportOpen, setExportOpen] = useState(false);
  const [relatedView, setRelatedView] = useState<{
    tag: Row;
    target: 'items' | 'software';
  } | null>(null);
  useEffect(() => {
    setActive(initialDictionary);
    setKeyword('');
    setPage(1);
    setEditor(null);
    setDeleteTarget(null);
  }, [initialDictionary]);
  const query = useQuery({
    queryKey: ['itdb', 'dictionaries'],
    queryFn: () => api<Record<string, Row[]>>('/api/dictionaries'),
  });
  const config = dictionaries.find(item => item.key === active)!;
  const rows = useMemo(() => query.data?.[active] ?? [], [active, query.data]);
  const keywordText = keyword.trim().toLowerCase();
  const editParamAppliedRef = useRef(false);
  useEffect(() => {
    if (!canManage || editParamAppliedRef.current || !rows.length) return;
    const params = new URLSearchParams(window.location.search);
    const editId = Number(params.get('edit'));
    if (!Number.isFinite(editId) || editId <= 0) return;
    const row = rows.find(item => Number(item.id) === editId);
    if (!row) return;
    editParamAppliedRef.current = true;
    window.history.replaceState({}, '', window.location.pathname);
    if (isProtectedDictionaryRow(active, row)) return;
    setEditor({
      id: editId,
      name: String(row[config.field] ?? ''),
      originalName: String(row[config.field] ?? ''),
      hasSoftware: Number(row.hassoftware ?? 0) === 1,
      color: isHexColor(String(row.color ?? '').trim()) ? String(row.color).trim() : '#2f7fba',
    });
  }, [rows, active, config.field, canManage]);
  const filteredRows = useMemo(
    () =>
      rows.filter(row =>
        dictionaryRowSearchText(active, row, config.field).some(text => text.includes(keywordText))
      ),
    [active, config.field, keywordText, rows]
  );
  const sortedRows = useMemo(
    () =>
      [...filteredRows].sort((left, right) =>
        sortDesc ? Number(right.id) - Number(left.id) : Number(left.id) - Number(right.id)
      ),
    [filteredRows, sortDesc]
  );
  const effectivePageSize = pageSize === -1 ? Math.max(sortedRows.length, 1) : pageSize;
  const pageCount = Math.max(1, Math.ceil(sortedRows.length / effectivePageSize));
  const currentPage = Math.min(page, pageCount);
  const visibleRows = sortedRows.slice(
    (currentPage - 1) * effectivePageSize,
    currentPage * effectivePageSize
  );
  useEffect(() => setPage(1), [keyword, pageSize]);
  const save = useMutation({
    mutationFn: async () => {
      if (!editor) throw new Error('缺少编辑数据');
      const payload: Row = { [config.field]: editor.name.trim() };
      if (active === 'itemtypes') payload.hassoftware = editor.hasSoftware ? 1 : 0;
      if (active === 'statustypes') payload.color = editor.color;
      await api(`/api/dictionaries/${active}${editor.id ? `/${editor.id}` : ''}`, {
        method: editor.id ? 'PUT' : 'POST',
        body: JSON.stringify(payload),
      });
      return { id: editor.id, name: editor.name.trim(), originalName: editor.originalName };
    },
    onSuccess: info => {
      toast.success(
        info.id
          ? `${config.label} ${info.originalName || info.name} 已更新`
          : `${config.label} ${info.name} 已创建`
      );
      setEditor(null);
      void client.invalidateQueries({ queryKey: ['itdb', 'dictionaries'] });
      void client.invalidateQueries({ queryKey: ['itdb', 'bootstrap'] });
    },
    onError: error => showErrorToast(error.message),
  });
  const remove = useMutation({
    mutationFn: async (row: Row) => {
      await api(`/api/dictionaries/${active}/${row.id}`, { method: 'DELETE' });
      return row;
    },
    onSuccess: row => {
      toast.success(`${config.label} ${String(row[config.field] ?? '')} 已删除`);
      setDeleteTarget(null);
      void client.invalidateQueries({ queryKey: ['itdb', 'dictionaries'] });
      void client.invalidateQueries({ queryKey: ['itdb', 'bootstrap'] });
    },
    onError: error => showErrorToast(error.message),
  });
  const openCreate = () =>
    setEditor({
      name: '',
      originalName: '',
      hasSoftware: false,
      color: active === 'statustypes' ? '#2f7fba' : '',
    });
  const openEdit = (row: Row) => {
    if (isProtectedDictionaryRow(active, row)) {
      toast.error(`内置${config.label}不可编辑`);
      return;
    }
    setEditor({
      id: Number(row.id),
      name: String(row[config.field] ?? ''),
      originalName: String(row[config.field] ?? ''),
      hasSoftware: Number(row.hassoftware ?? 0) === 1,
      color: isHexColor(String(row.color ?? '').trim()) ? String(row.color).trim() : '#2f7fba',
    });
  };
  const importInputRef = useRef<HTMLInputElement | null>(null);
  const [importing, setImporting] = useState(false);
  async function handleImportFile(file: File) {
    if (importing) return;
    setImporting(true);
    try {
      const matrix = await parseXlsxMatrix(file);
      const headers = (matrix[0] ?? []).map(cell => cell.trim());
      if (headers.join('|') !== dictionaryImportHeaders(active, config.label).join('|')) {
        toast.error('表格字段与模板不匹配，请下载模板填写');
        return;
      }
      const dataRows = matrix.slice(1).filter(cells => cells.some(cell => cell.trim() !== ''));
      if (dataRows.length === 0) {
        toast.error('表格数据不能为空');
        return;
      }
      let success = 0;
      let subtypeSuccess = 0;
      let skipped = 0;
      const importedNames: string[] = [];
      const existingNames: string[] = [];
      const failures: string[] = [];
      for (const cells of dataRows) {
        const request = dictionaryImportPayloads(active, cells);
        const rowName = String(request.body[config.field] ?? '').trim();
        if (!rowName) {
          skipped += 1;
          continue;
        }
        try {
          const created = await api<{ id: number }>(`/api/dictionaries/${request.path}?audit=0`, {
            method: 'POST',
            body: JSON.stringify(request.body),
          });
          success += 1;
          const createdSubtypes: string[] = [];
          for (const subtype of request.subtypes ?? []) {
            try {
              await api('/api/dictionaries/contractsubtypes?audit=0', {
                method: 'POST',
                body: JSON.stringify({ name: subtype, contypeid: created.id }),
              });
              subtypeSuccess += 1;
              createdSubtypes.push(subtype);
            } catch (error) {
              const message = error instanceof Error ? error.message : String(error);
              if (message.includes('已存在')) existingNames.push(`${rowName}(${subtype})`);
              failures.push(
                message.startsWith('合同子类型') ? `合同类型 ${rowName} 中的${message}` : message
              );
            }
          }
          importedNames.push(
            createdSubtypes.length > 0 ? `${rowName}(${createdSubtypes.join('、')})` : rowName
          );
        } catch (error) {
          const message = error instanceof Error ? error.message : String(error);
          if (message.includes('已存在')) existingNames.push(rowName);
          failures.push(message);
        }
      }
      if (success === 0 && existingNames.length > 0) {
        void trackAuditEvent({
          type: `import:${active}`,
          names: existingNames,
          result: 'failure',
        });
      } else if (success > 0 || existingNames.length > 0) {
        void trackAuditEvent({
          type: `import:${active}`,
          names: importedNames,
          existing: existingNames,
        });
      }
      if (success > 0) {
        if (active === 'contracttypes') {
          toast.success(
            subtypeSuccess > 0
              ? `成功导入 ${success} 条合同类型和 ${subtypeSuccess} 条合同子类型`
              : `成功导入 ${success} 条合同类型`
          );
        } else {
          toast.success(`成功导入 ${success} 条${config.label}`);
        }
        void client.invalidateQueries({ queryKey: ['itdb', 'dictionaries'] });
        void client.invalidateQueries({ queryKey: ['itdb', 'bootstrap'] });
      }
      if (skipped > 0) toast.warning(`${skipped} 行名称为空已跳过`);
      failures.forEach(message => showErrorToast(message));
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : '导入失败，请检查文件格式');
    } finally {
      setImporting(false);
      if (importInputRef.current) importInputRef.current.value = '';
    }
  }
  return (
    <PermissionGate anyOf={[dictionaryPermission(active, 'read')]}>
      <>
        <nav
          className="itdb-card-hover itdb-hidden-scrollbar flex shrink-0 overflow-x-auto rounded-xl p-3"
          aria-label="基础资料分类"
          style={{
            background: 'var(--itdb-card)',
            border: '1px solid var(--itdb-border)',
            boxShadow: 'var(--shadow-card)',
          }}
        >
          <div className="flex min-w-max flex-wrap gap-2">
            {permittedDictionaries.map(item => (
              <DictionarySwitch
                key={item.key}
                to={`/dictionaries/${item.key}`}
                active={active === item.key}
                item={item}
              />
            ))}
          </div>
        </nav>
        <section
          className="itdb-card-hover flex min-h-0 flex-1 flex-col overflow-hidden rounded-xl p-5"
          style={{
            background: 'var(--itdb-card)',
            border: '1px solid var(--itdb-border)',
            boxShadow: 'var(--shadow-card)',
          }}
        >
          <header className="flex shrink-0 flex-wrap items-center justify-between gap-3">
            <p className="text-sm text-[var(--itdb-text-muted)]">
              维护{config.label}，为资源录入和筛选提供统一资料
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
                  className="itdb-action-button itdb-dict-template-button"
                  onClick={() => {
                    downloadXlsxTemplate(
                      dictionaryImportHeaders(active, config.label),
                      `${config.label}导入模板`,
                      dictionaryImportTemplateSpecs(active)
                    );
                    toast.success(`已下载 ${config.label} 导入模板`);
                  }}
                >
                  <FileDown size={16} />
                  下载 Excel 模板
                </Button>
              )}
              {canManage && (
                <Button
                  type="button"
                  variant="outline"
                  className="itdb-action-button itdb-dict-import-button"
                  disabled={importing}
                  onClick={() => importInputRef.current?.click()}
                >
                  <Upload size={16} />
                  {importing ? '导入中' : '导入'}
                </Button>
              )}
              {canManage && (
                <Button
                  type="button"
                  variant="outline"
                  className="itdb-action-button itdb-dict-export-button"
                  onClick={() => {
                    if (filteredRows.length === 0) {
                      toast.error(`没有可导出的${config.label}数据`);
                      return;
                    }
                    setExportOpen(true);
                  }}
                >
                  <Download size={16} />
                  导出
                </Button>
              )}
              {canManage && (
                <Button
                  type="button"
                  variant="outline"
                  className="itdb-action-button"
                  style={{
                    borderColor: 'rgba(59,130,246,0.38)',
                    background: 'rgba(59,130,246,0.1)',
                    color: 'var(--itdb-accent-text)',
                  }}
                  onClick={openCreate}
                >
                  <Plus size={16} />
                  新增
                </Button>
              )}
              <input
                ref={importInputRef}
                type="file"
                accept=".xlsx"
                className="hidden"
                onChange={event => {
                  const file = event.target.files?.[0];
                  if (file) void handleImportFile(file);
                }}
              />
            </div>
          </header>
          <div className="itdb-resource-data-panel mt-4 flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-[var(--itdb-border)]">
            <div className="itdb-hidden-scrollbar min-h-0 flex-1 overflow-auto">
              <table className="min-w-full text-center text-sm">
                <thead className="sticky top-0 z-30">
                  <tr>
                    <th className="whitespace-nowrap px-4 py-3 text-center font-medium">
                      <button
                        type="button"
                        className="inline-flex items-center gap-1"
                        onClick={() => setSortDesc(current => !current)}
                      >
                        编号
                        {sortDesc ? <ChevronDown size={14} /> : <ChevronUp size={14} />}
                      </button>
                    </th>
                    <th className="whitespace-nowrap px-4 py-3 text-center font-medium">
                      {config.label}
                    </th>
                    {active === 'itemtypes' ? (
                      <th className="whitespace-nowrap px-4 py-3 text-center font-medium">
                        支持软件
                      </th>
                    ) : null}
                    {active === 'tags' ? (
                      <>
                        <th className="whitespace-nowrap px-4 py-3 text-center font-medium">
                          关联硬件
                        </th>
                        <th className="whitespace-nowrap px-4 py-3 text-center font-medium">
                          关联软件
                        </th>
                      </>
                    ) : null}
                    {(canManage || active === 'contracttypes') && (
                      <th className="itdb-resource-operation-cell sticky right-0 z-40 px-3 py-3 text-center font-medium">
                        操作
                      </th>
                    )}
                  </tr>
                </thead>
                <tbody>
                  {visibleRows.map(row => (
                    <tr
                      key={String(row.id)}
                      className="border-b border-[var(--itdb-border)]/70 hover:bg-[var(--itdb-control-bg-soft)]"
                    >
                      <td className="whitespace-nowrap px-4 py-3 text-center text-[var(--itdb-text)]">
                        <span className="itdb-relation-id">{String(row.id)}</span>
                      </td>
                      <td className="px-4 py-3 text-center text-[var(--itdb-text)]">
                        {active === 'statustypes' && isHexColor(String(row.color ?? '').trim()) ? (
                          <span className="inline-flex items-center justify-center gap-2">
                            <span
                              className="h-2.5 w-2.5 shrink-0 rounded-full"
                              style={{ background: String(row.color).trim() }}
                            />
                            {String(row[config.field] ?? '-')}
                          </span>
                        ) : (
                          String(row[config.field] ?? '-')
                        )}
                      </td>
                      {active === 'itemtypes' ? (
                        <td className="px-4 py-3 text-center">
                          {Number(row.hassoftware ?? 0) === 1 ? (
                            <span className="rounded-full border border-[rgba(59,130,246,0.3)] bg-[rgba(59,130,246,0.1)] px-2.5 py-0.5 text-xs font-medium text-[var(--itdb-accent-text)]">
                              是
                            </span>
                          ) : (
                            <span className="rounded-full border border-[var(--itdb-border)] bg-[color-mix(in_srgb,var(--itdb-text-muted)_10%,transparent)] px-2.5 py-0.5 text-xs text-[var(--itdb-text-muted)]">
                              否
                            </span>
                          )}
                        </td>
                      ) : null}
                      {active === 'tags' ? (
                        <>
                          <td className="px-4 py-3 text-center">
                            {Number(row.itemCount ?? 0) > 0 ? (
                              <AppTooltip label={`查看关联硬件 ${Number(row.itemCount ?? 0)} 条`}>
                                <button
                                  type="button"
                                  className="itdb-action-button inline-flex h-7 min-w-9 items-center justify-center rounded-full border px-2.5 text-xs font-medium"
                                  style={{
                                    borderColor: 'rgba(59,130,246,0.38)',
                                    background: 'rgba(59,130,246,0.1)',
                                    color: 'var(--itdb-accent-text)',
                                  }}
                                  onClick={() => setRelatedView({ tag: row, target: 'items' })}
                                  aria-label={`查看标记 ${String(row.name ?? '')} 的关联硬件`}
                                >
                                  {Number(row.itemCount ?? 0)}
                                </button>
                              </AppTooltip>
                            ) : (
                              <span className="inline-flex h-7 min-w-9 items-center justify-center rounded-full border border-[var(--itdb-border)] bg-[color-mix(in_srgb,var(--itdb-text-muted)_10%,transparent)] px-2.5 text-xs text-[var(--itdb-text-muted)]">
                                0
                              </span>
                            )}
                          </td>
                          <td className="px-4 py-3 text-center">
                            {Number(row.softwareCount ?? 0) > 0 ? (
                              <AppTooltip
                                label={`查看关联软件 ${Number(row.softwareCount ?? 0)} 条`}
                              >
                                <button
                                  type="button"
                                  className="itdb-action-button inline-flex h-7 min-w-9 items-center justify-center rounded-full border px-2.5 text-xs font-medium"
                                  style={{
                                    borderColor: 'rgba(59,130,246,0.38)',
                                    background: 'rgba(59,130,246,0.1)',
                                    color: 'var(--itdb-accent-text)',
                                  }}
                                  onClick={() => setRelatedView({ tag: row, target: 'software' })}
                                  aria-label={`查看标记 ${String(row.name ?? '')} 的关联软件`}
                                >
                                  {Number(row.softwareCount ?? 0)}
                                </button>
                              </AppTooltip>
                            ) : (
                              <span className="inline-flex h-7 min-w-9 items-center justify-center rounded-full border border-[var(--itdb-border)] bg-[color-mix(in_srgb,var(--itdb-text-muted)_10%,transparent)] px-2.5 text-xs text-[var(--itdb-text-muted)]">
                                0
                              </span>
                            )}
                          </td>
                        </>
                      ) : null}
                      {(canManage || active === 'contracttypes') && (
                        <td className="itdb-resource-operation-cell sticky right-0 z-20 px-3 py-2">
                          <div className="flex justify-center gap-2.5">
                            {canManage && (
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
                                  onClick={() => openEdit(row)}
                                  aria-label={`编辑${config.label}`}
                                >
                                  <Edit3 size={14} />
                                </Button>
                              </AppTooltip>
                            )}
                            {canManage && (
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
                                  onClick={() => {
                                    if (isProtectedDictionaryRow(active, row)) {
                                      toast.error(`内置${config.label}不可删除`);
                                      return;
                                    }
                                    setDeleteTarget(row);
                                  }}
                                  aria-label={`删除${config.label}`}
                                >
                                  <Trash2 size={14} />
                                </Button>
                              </AppTooltip>
                            )}
                            {active === 'contracttypes' ? (
                              <AppTooltip label="查看合同子类型">
                                <Button
                                  size="icon"
                                  variant="ghost"
                                  className="itdb-action-button h-7 w-7"
                                  style={{
                                    borderColor: 'rgba(34,211,238,0.4)',
                                    background: 'rgba(34,211,238,0.1)',
                                    color: 'var(--itdb-accent-text)',
                                  }}
                                  onClick={() => setSubtypesFor(row)}
                                  aria-label={`查看/编辑合同子类型 ${String(row.name ?? '')}`}
                                >
                                  <Network size={14} />
                                </Button>
                              </AppTooltip>
                            ) : null}
                          </div>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
              {!query.isLoading && visibleRows.length === 0 ? (
                <div className="grid min-h-52 place-items-center px-6 text-sm text-[var(--itdb-text-muted)]">
                  {query.isError ? `${config.label}数据加载失败` : `暂无${config.label}记录`}
                </div>
              ) : null}
              {query.isLoading ? (
                <div className="grid min-h-52 place-items-center text-sm text-[var(--itdb-text-muted)]">
                  数据加载中...
                </div>
              ) : null}
            </div>
            <Pagination
              page={currentPage}
              pageCount={pageCount}
              pageSize={pageSize}
              total={sortedRows.length}
              onPageChange={setPage}
              onPageSizeChange={setPageSize}
            />
          </div>
        </section>
        <DictionaryEditorDialog
          editor={editor}
          dictionaryKey={active}
          label={config.label}
          busy={save.isPending}
          onChange={setEditor}
          onClose={() => setEditor(null)}
          onSave={() => save.mutate()}
        />
        <ContractSubtypeDialog
          contractType={subtypesFor}
          subtypes={query.data?.contractsubtypes ?? []}
          canManage={canManage}
          onClose={() => setSubtypesFor(null)}
        />
        <TagRelatedDialog
          tag={relatedView?.tag ?? null}
          target={relatedView?.target ?? null}
          onClose={() => setRelatedView(null)}
        />
        <DictionaryExportDialog
          open={exportOpen}
          label={config.label}
          dictionaryKey={active}
          field={config.field}
          rows={sortedRows}
          subtypes={query.data?.contractsubtypes ?? []}
          onOpenChange={setExportOpen}
        />
        <ConfirmDialog
          open={Boolean(deleteTarget)}
          title={`删除${config.label}`}
          description="删除后不可恢复，正在使用中或内置的记录无法删除"
          detail={`确认删除 编号=${String(deleteTarget?.id ?? '-')} 的记录吗？`}
          horizontalHeader
          confirmText="确认删除"
          busy={remove.isPending}
          softDestructive
          onOpenChange={open => !open && setDeleteTarget(null)}
          onConfirm={() => {
            if (deleteTarget) remove.mutate(deleteTarget);
          }}
        />
      </>
    </PermissionGate>
  );
}

function DictionarySwitch({
  item,
  active,
  to,
}: {
  item: (typeof dictionaries)[number];
  active: boolean;
  to: string;
}) {
  const Icon = dictionaryIcons[item.key];
  return (
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
      {item.label}
    </Link>
  );
}
