import { ChevronDown, ChevronUp, ChevronsUpDown, ExternalLink, Search } from 'lucide-react';
import { useMemo, useState } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { Input } from '@/components/ui/input';
import { apiBlob } from '@/lib/auth';
import { canOpenRecordEditor, canViewResource } from '../record-links';
import { type ResourceField } from '../resource-config';
import { compareTableValues, formatValue, parseInvoiceFileEntries } from '../resource-helpers';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

type RelationColumn = { key: string; label: string };
type RelationMeta = { title: string; noun: string; path?: string; columns: RelationColumn[] };

const relationMeta: Record<string, RelationMeta> = {
  itemLinks: {
    title: '内部硬件关联',
    noun: '硬件',
    path: '/assets/hardware',
    columns: [
      { key: 'itemtype', label: '类型' },
      { key: 'manufacturer', label: '厂商' },
      { key: 'model', label: '型号' },
      { key: 'label', label: '标签' },
      { key: 'dnsname', label: '管理跳线' },
      { key: 'principal', label: '负责人' },
      { key: 'sn', label: '设备序列号' },
    ],
  },
  invoiceLinks: {
    title: '单据关联',
    noun: '单据',
    path: '/assets/invoices',
    columns: [
      { key: 'vendor', label: '供应商' },
      { key: 'number', label: '订单编号' },
      { key: 'filedesc', label: '单据描述 / 文件' },
      { key: 'date', label: '日期' },
    ],
  },
  softwareLinks: {
    title: '软件关联',
    noun: '软件',
    path: '/assets/software',
    columns: [
      { key: 'manufacturer', label: '厂商' },
      { key: 'title', label: '标题/版本' },
    ],
  },
  contractLinks: {
    title: '合同关联',
    noun: '合同',
    path: '/assets/contracts',
    columns: [
      { key: 'contractor', label: '承包方' },
      { key: 'title', label: '标题' },
    ],
  },
  fileLinks: {
    title: '文件关联',
    noun: '文件',
    path: '/assets/files',
    columns: [
      { key: 'typeDesc', label: '类型' },
      { key: 'title', label: '标题' },
      { key: 'fname', label: '文件名' },
      { key: 'date', label: '签署日期' },
    ],
  },
};

const fallbackMeta: RelationMeta = { title: '关联信息', noun: '记录', columns: [] };

// 与后端 isInvoiceFileType 一致：内置编号 3、英文名 invoice 或中文名“发票”视为发票类型
export function resolveInvoiceFileTypeRow(lookups?: Lookups) {
  const rows = lookups?.filetypes ?? [];
  return (
    rows.find(row => /^invoice$/i.test(String(row.typedesc ?? '').trim())) ??
    rows.find(row => String(row.typedesc ?? '').trim() === '发票') ??
    rows.find(row => Number(row.id) === 3)
  );
}

export function resolveInvoiceFileTypeId(lookups?: Lookups) {
  const row = resolveInvoiceFileTypeRow(lookups);
  return row ? Number(row.id) : 3;
}

export function isInvoiceFileRow(row: Row, invoiceTypeId: number) {
  const desc = String(row.typeDesc ?? row.typedesc ?? '').trim();
  return Number(row.type) === invoiceTypeId || /^invoice$/i.test(desc) || desc === '发票';
}

export function resolveInvoiceFileTypeLabel(lookups?: Lookups) {
  const row = resolveInvoiceFileTypeRow(lookups);
  return String(row?.typedesc ?? '').trim() || 'invoice';
}

function relationTitle(tab: string, resourceKey: string) {
  if (tab === 'itemLinks' && resourceKey !== 'items') return '硬件关联';
  if (tab === 'invoiceLinks' && resourceKey !== 'items') return '单据关联';
  return (relationMeta[tab] ?? fallbackMeta).title;
}

function relationCellText(tab: string, key: string, row: Row) {
  const text = (v: unknown) => String(v ?? '').trim();
  if (tab === 'itemLinks' && key === 'sn')
    return [row.sn, row.sn2, row.sn3].map(text).filter(Boolean).join(' ');
  if (tab === 'invoiceLinks') {
    if (key === 'filedesc')
      return [text(row.description), text(row.files)].filter(Boolean).join(' · ');
    if (key === 'date') return formatValue(row.date, 'date');
  }
  if (tab === 'softwareLinks' && key === 'title')
    return [text(row.stitle), text(row.sversion)].filter(Boolean).join(' ');
  if (tab === 'fileLinks' && key === 'date') return formatValue(row.date, 'date');
  if (tab === 'fileLinks' && key === 'typeDesc') return text(row.typeDesc ?? row.typedesc);
  return text(row[key]);
}

async function openFilePreview(id: number) {
  try {
    const blob = await apiBlob(`/api/files/${id}/download`);
    if (!blob.size) {
      toast.error('文件预览失败');
      return;
    }
    const url = URL.createObjectURL(blob);
    window.open(url, '_blank');
    window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
  } catch (error) {
    toast.error(error instanceof Error ? error.message : '文件预览失败');
  }
}

function InvoiceFileDescCell({ raw }: { raw: Row }) {
  const description = String(raw.description ?? '').trim();
  const files = parseInvoiceFileEntries(raw.files);
  if (!description && files.length === 0) return <span>-</span>;
  return (
    <div className="flex flex-col items-center gap-1.5">
      {description ? <span className="whitespace-pre-wrap break-words">{description}</span> : null}
      {files.length ? (
        <div className="flex flex-wrap justify-center gap-1.5">
          {files.map(file =>
            file.id ? (
              <AppTooltip key={`${file.index}-${file.id}`} label={`在新窗口预览文件 ${file.id}`}>
                <button
                  type="button"
                  className="itdb-invoice-file-pill"
                  onClick={() => void openFilePreview(file.id as number)}
                >
                  <ExternalLink size={12} className="shrink-0" />
                  <span className="min-w-0 truncate">{file.text}</span>
                </button>
              </AppTooltip>
            ) : (
              <span key={`${file.index}-na`} className="text-xs text-[var(--itdb-text-muted)]">
                {file.text}
              </span>
            )
          )}
        </div>
      ) : null}
    </div>
  );
}

export function SortHeaderButton({
  label,
  columnKey,
  sort,
  onToggle,
}: {
  label: string;
  columnKey: string;
  sort: { key: string; direction: 'asc' | 'desc' };
  onToggle: (key: string) => void;
}) {
  return (
    <button
      type="button"
      className="inline-flex items-center gap-1"
      onClick={() => onToggle(columnKey)}
    >
      {label}
      {sort.key === columnKey ? (
        sort.direction === 'asc' ? (
          <ChevronUp size={14} />
        ) : (
          <ChevronDown size={14} />
        )
      ) : (
        <ChevronsUpDown size={14} className="opacity-55" />
      )}
    </button>
  );
}

export function HardwareRelationPane({
  resourceKey,
  tab,
  field,
  value,
  lookups,
  onChange,
  currentTypeId,
  currentId,
  lockNote = '',
  stretch = false,
}: {
  resourceKey: string;
  tab: string;
  field?: ResourceField;
  value: unknown;
  lookups: Lookups | undefined;
  onChange: (value: unknown) => void;
  currentTypeId?: number;
  currentId?: number;
  lockNote?: string;
  stretch?: boolean;
}) {
  const [keyword, setKeyword] = useState('');
  const [sort, setSort] = useState<{ key: string; direction: 'asc' | 'desc' }>({
    key: 'id',
    direction: 'asc',
  });
  const meta = useMemo(() => {
    const base = relationMeta[tab] ?? fallbackMeta;
    if (tab === 'invoiceLinks' && resourceKey === 'agents') {
      return { ...base, columns: base.columns.filter(column => column.key !== 'filedesc') };
    }
    return base;
  }, [tab, resourceKey]);
  const canViewRelation = meta.path ? canViewResource(meta.path) : true;
  const selectedValues = Array.isArray(value) ? value.map(item => Number(item)) : [];
  const selected = new Set(selectedValues);
  const rows = useMemo(() => {
    const source = field?.optionsKey ? (lookups?.[field.optionsKey] ?? []) : [];
    const isFileLinks = tab === 'fileLinks';
    const invoiceTypeId = isFileLinks ? resolveInvoiceFileTypeId(lookups) : 0;
    const invoiceOnly = isFileLinks && resourceKey === 'invoices';
    const excludeInvoice = isFileLinks && ['items', 'software', 'contracts'].includes(resourceKey);
    return source
      .filter(row =>
        invoiceOnly
          ? isInvoiceFileRow(row, invoiceTypeId)
          : !excludeInvoice || !isInvoiceFileRow(row, invoiceTypeId)
      )
      .filter(row =>
        tab === 'itemLinks' && resourceKey === 'software'
          ? Number(row.hassoftware ?? 1) !== 0
          : true
      )
      .map(row => ({
        id: Number(row.id),
        raw: row,
        cells: Object.fromEntries(
          meta.columns.map(column => [column.key, relationCellText(tab, column.key, row)])
        ),
      }))
      .filter(
        rowData =>
          Number.isFinite(rowData.id) &&
          rowData.id > 0 &&
          (currentId === undefined || rowData.id !== Number(currentId))
      );
  }, [field?.optionsKey, lookups, meta, resourceKey, tab]);
  const filteredRows = rows.filter(rowData => {
    const text = keyword.trim().toLowerCase();
    if (!text) return true;
    return (
      String(rowData.id).includes(text) ||
      meta.columns.some(column => rowData.cells[column.key]?.toLowerCase().includes(text))
    );
  });
  const sortedRows = [...filteredRows].sort((left, right) => {
    const linked = Number(selected.has(right.id)) - Number(selected.has(left.id));
    if (linked) return linked;
    const compared =
      sort.key === 'id'
        ? left.id - right.id
        : compareTableValues(left.cells[sort.key], right.cells[sort.key]);
    return sort.direction === 'asc' ? compared : -compared;
  });
  const toggleSort = (key: string) =>
    setSort(current =>
      current.key === key
        ? { key, direction: current.direction === 'asc' ? 'desc' : 'asc' }
        : { key, direction: 'asc' }
    );
  const toggleSelected = (optionValue: number) => {
    onChange(
      selected.has(optionValue)
        ? selectedValues.filter(value => value !== optionValue)
        : [...selectedValues, optionValue]
    );
  };
  const invoiceLockLabel =
    resourceKey === 'files' &&
    ['itemLinks', 'softwareLinks', 'contractLinks'].includes(tab) &&
    Number(currentTypeId) > 0 &&
    Number(currentTypeId) === resolveInvoiceFileTypeId(lookups)
      ? resolveInvoiceFileTypeLabel(lookups)
      : '';
  const invoiceLockNoun = tab === 'itemLinks' ? '硬件' : tab === 'softwareLinks' ? '软件' : '合同';
  return (
    <div className="flex min-h-0 flex-1 flex-col gap-5">
      <section
        className={`mx-auto w-full max-w-6xl rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-5 pb-5 pt-3${stretch ? ' flex min-h-0 flex-1 flex-col' : ''}`}
      >
        <div className="flex shrink-0 flex-wrap items-center gap-3 border-b border-[var(--itdb-border)] pb-3">
          <h3 className="text-base font-semibold text-[var(--itdb-text)]">
            {relationTitle(tab, resourceKey)}
          </h3>
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
          <span className="text-sm text-[var(--itdb-text-muted)]">
            共 {canViewRelation ? filteredRows.length : 0} 条
          </span>
          <span className="ml-auto rounded-full bg-[rgba(59,130,246,0.12)] px-2.5 py-1 text-xs font-medium text-[var(--itdb-accent)]">
            已关联 {selectedValues.length} 项
          </span>
        </div>
        {invoiceLockLabel ? (
          <div className="grid min-h-24 place-items-center rounded-lg border border-dashed border-[var(--itdb-border)] px-4 py-6 text-center text-sm text-[var(--itdb-text-muted)]">
            {`该文件类型为“${invoiceLockLabel}”，不能在此关联${invoiceLockNoun}。`}
          </div>
        ) : (
          <div
            className={`itdb-hidden-scrollbar itdb-resource-data-panel mt-4 overflow-auto rounded-lg border border-[var(--itdb-border)] ${stretch ? 'min-h-0 flex-1' : 'max-h-80'}`}
          >
            <table className="min-w-full text-sm">
              <thead className="sticky top-0 z-10 bg-[var(--itdb-control-bg-soft)] text-[var(--itdb-text-muted)]">
                <tr>
                  <th className="w-20 px-3 py-2.5 text-center font-medium">关联</th>
                  <th className="px-3 py-2.5 text-center font-medium">
                    <SortHeaderButton
                      label="编号"
                      columnKey="id"
                      sort={sort}
                      onToggle={toggleSort}
                    />
                  </th>
                  {meta.columns.map(column => (
                    <th
                      key={column.key}
                      className="whitespace-nowrap px-3 py-2.5 text-center font-medium"
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
                {lockNote ? (
                  <tr>
                    <td
                      colSpan={meta.columns.length + 2}
                      className="px-4 py-10 text-center text-sm text-[var(--itdb-text-muted)]"
                    >
                      {lockNote}
                    </td>
                  </tr>
                ) : !canViewRelation ? (
                  <tr>
                    <td
                      colSpan={meta.columns.length + 2}
                      className="px-4 py-10 text-center text-sm text-[var(--itdb-text-muted)]"
                    >
                      {`当前没有查看${meta.noun}权限，请先授权`}
                    </td>
                  </tr>
                ) : (
                  <>
                    {sortedRows.map(rowData => (
                      <tr
                        key={rowData.id}
                        className="border-t border-[var(--itdb-border)]/70 hover:bg-[var(--itdb-control-bg-soft)]"
                      >
                        <td className="px-3 py-2.5 text-center">
                          <input
                            aria-label={`关联 ${rowData.id}`}
                            type="checkbox"
                            className="itdb-check-box"
                            checked={selected.has(rowData.id)}
                            onChange={() => toggleSelected(rowData.id)}
                          />
                        </td>
                        <td className="px-3 py-2.5 text-center">
                          {meta.path && canOpenRecordEditor(meta.path) ? (
                            <AppTooltip label={`在新窗口编辑${meta.noun} ${rowData.id}`}>
                              <a
                                href={`${meta.path}?edit=${rowData.id}`}
                                target="_blank"
                                rel="noopener"
                                aria-label={`在新窗口编辑${meta.noun} ${rowData.id}`}
                                className="itdb-relation-id itdb-relation-id-link"
                              >
                                {rowData.id}
                              </a>
                            </AppTooltip>
                          ) : (
                            <span className="itdb-relation-id">{rowData.id}</span>
                          )}
                        </td>
                        {meta.columns.map(column => {
                          const cellText = rowData.cells[column.key];
                          if (tab === 'invoiceLinks' && column.key === 'filedesc') {
                            return (
                              <td
                                key={column.key}
                                className="max-w-64 px-3 py-2.5 text-center text-[var(--itdb-text)]"
                              >
                                <InvoiceFileDescCell raw={rowData.raw} />
                              </td>
                            );
                          }
                          return (
                            <td
                              key={column.key}
                              className="max-w-64 px-3 py-2.5 text-center text-[var(--itdb-text)]"
                            >
                              <AppTooltip label={cellText || undefined}>
                                <span className="block truncate">{cellText || '-'}</span>
                              </AppTooltip>
                            </td>
                          );
                        })}
                      </tr>
                    ))}
                    {sortedRows.length === 0 ? (
                      <tr>
                        <td
                          colSpan={meta.columns.length + 2}
                          className="px-4 py-8 text-center text-sm text-[var(--itdb-text-muted)]"
                        >
                          暂无可关联{meta.noun}
                        </td>
                      </tr>
                    ) : null}
                  </>
                )}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </div>
  );
}
