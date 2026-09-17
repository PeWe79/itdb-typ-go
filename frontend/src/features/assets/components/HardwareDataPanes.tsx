import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { Check, X, ExternalLink, Pencil, Trash2 } from 'lucide-react';
import { canOpenRecordEditor } from '../record-links';
import type { ReactNode } from 'react';
import { useEffect, useRef, useMemo, useState } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { type ResourceField } from '../resource-config';
import {
  getAgentServiceUrl,
  hardwareMainGroups,
  overviewEntryText,
  resolveFieldOptions,
} from '../resource-helpers';
import { FieldEditor } from './FieldEditor';
import { HardwareFilesPane } from './HardwareFilesPane';
import { api } from '@/lib/auth';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

export type OverviewTabConfig = {
  key: string;
  label: string;
  field: string;
  lookup: string;
  path: string;
};

const hardwareOverviewTabs: OverviewTabConfig[] = [
  {
    key: 'items',
    label: '硬件',
    field: 'itemLinks',
    lookup: 'items_ref',
    path: '/assets/hardware',
  },
  {
    key: 'software',
    label: '软件',
    field: 'softwareLinks',
    lookup: 'software_ref',
    path: '/assets/software',
  },
  {
    key: 'invoices',
    label: '单据',
    field: 'invoiceLinks',
    lookup: 'invoices_ref',
    path: '/assets/invoices',
  },
  {
    key: 'contracts',
    label: '合同',
    field: 'contractLinks',
    lookup: 'contracts_ref',
    path: '/assets/contracts',
  },
];

export function HardwareDataPane({
  fields,
  values,
  lookups,
  activeSection,
  onSectionChange,
  onChange,
  itemId,
  onPendingTagsChange,
  onUnlinkFile,
}: {
  fields: ResourceField[];
  values: Row;
  lookups: Lookups | undefined;
  activeSection: string;
  onSectionChange: (section: string) => void;
  onChange: (field: ResourceField, value: unknown) => void;
  itemId?: unknown;
  onPendingTagsChange?: (pending: { added: string[]; removed: string[] }) => void;
  onUnlinkFile?: (fileId: number) => void;
}) {
  const sections = [
    { value: 'basic', label: '基础信息配置' },
    { value: 'usage', label: '使用信息配置' },
    { value: 'accounting', label: '账目信息配置' },
    { value: 'warranty', label: '维保信息配置' },
    { value: 'hardware', label: '硬件信息配置' },
    { value: 'network', label: '网络信息配置' },
    { value: 'overview', label: '关联概览' },
    { value: 'tags', label: 'Tags' },
    { value: 'files', label: '管理文件' },
  ];
  const group = hardwareMainGroups(fields, activeSection)[0];
  const manufacturerId = Number(values.manufacturerId);
  const manufacturerServiceUrl =
    Number.isFinite(manufacturerId) && manufacturerId > 0
      ? getAgentServiceUrl(
          (lookups?.agents ?? []).find(agent => Number(agent.id) === manufacturerId)
        )
      : '';
  const manufacturerServiceAction = manufacturerServiceUrl ? (
    <AppTooltip label="打开该厂商的服务链接（URLs 中描述包含“服务”或“service”）">
      <a
        href={manufacturerServiceUrl}
        target="_blank"
        rel="noopener"
        aria-label="打开厂商服务链接"
        className="flex h-5 w-5 items-center justify-center rounded-md border border-[rgba(59,130,246,0.38)] bg-[rgba(59,130,246,0.1)] text-[var(--itdb-accent-text)] transition-colors hover:bg-[rgba(59,130,246,0.18)]"
      >
        <ExternalLink size={12} />
      </a>
    </AppTooltip>
  ) : null;

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-5">
      <nav
        className="itdb-hidden-scrollbar flex shrink-0 overflow-x-auto rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] p-1.5"
        aria-label="硬件数据配置导航"
      >
        <div className="flex min-w-max gap-1.5">
          {sections.map(section => (
            <button
              key={section.value}
              type="button"
              onClick={() => onSectionChange(section.value)}
              className="itdb-hardware-subtab h-9 rounded-lg px-3 text-sm font-medium"
              data-active={activeSection === section.value}
            >
              {section.label}
            </button>
          ))}
        </div>
      </nav>

      {group ? (
        <section className="itdb-resource-form-section mx-auto flex min-h-0 w-full max-w-3xl flex-1 flex-col rounded-xl border border-[var(--itdb-border)] px-5 pb-5 pt-3">
          <h3 className="mb-4 border-b border-[var(--itdb-border)] pb-2 text-base font-semibold text-[var(--itdb-text)]">
            {group.title}
          </h3>
          <div
            key={activeSection}
            className="itdb-hidden-scrollbar min-h-0 flex-1 space-y-3 overflow-y-auto px-1 pt-1"
            aria-label={`${group.title}字段列表`}
          >
            {group.fields.map(field => (
              <FieldEditor
                key={field.key}
                field={field}
                value={values[field.key]}
                options={resolveFieldOptions(field, lookups, values, 'items')}
                onChange={value => onChange(field, value)}
                compactHorizontal
                labelAction={field.key === 'manufacturerId' ? manufacturerServiceAction : null}
                rackViewHighlightId={field.key === 'rackId' ? itemId : undefined}
              />
            ))}
          </div>
        </section>
      ) : activeSection === 'overview' ? (
        <HardwareOverview values={values} lookups={lookups} />
      ) : activeSection === 'tags' ? (
        <HardwareTagsPane
          initialTags={values.tags}
          lookups={lookups}
          onPendingTagsChange={onPendingTagsChange}
        />
      ) : activeSection === 'files' ? (
        <HardwareFilesPane
          values={values}
          lookups={lookups}
          resourceKey="items"
          onChangeFileLinks={next => {
            const field = fields.find(item => item.key === 'fileLinks');
            if (field) onChange(field, next);
          }}
          onUnlinkFile={onUnlinkFile}
        />
      ) : null}
    </div>
  );
}

export function HardwareOverview({
  values,
  lookups,
  tabs = hardwareOverviewTabs,
  children,
}: {
  values: Row;
  lookups: Lookups | undefined;
  tabs?: OverviewTabConfig[];
  children?: ReactNode;
}) {
  const [activeTab, setActiveTab] = useState(tabs[0]?.key ?? '');
  const active = tabs.find(tab => tab.key === activeTab) ?? tabs[0];
  const ids = Array.isArray(values[active.field])
    ? (values[active.field] as unknown[]).map(Number).filter(id => Number.isFinite(id) && id > 0)
    : [];
  const rowsById = new Map(
    (lookups?.[active.lookup] ?? []).map(row => [Number(row.id), row] as const)
  );

  return (
    <section className="itdb-resource-form-section mx-auto flex min-h-0 w-full max-w-3xl flex-1 flex-col rounded-xl border border-[var(--itdb-border)] px-5 pb-5 pt-3">
      <h3 className="mb-4 border-b border-[var(--itdb-border)] pb-2 text-base font-semibold text-[var(--itdb-text)]">
        关联概览
      </h3>
      <div className="mb-3 flex flex-wrap gap-2">
        {tabs.map(tab => {
          const count = Array.isArray(values[tab.field])
            ? (values[tab.field] as unknown[]).length
            : 0;
          return (
            <button
              key={tab.key}
              type="button"
              onClick={() => setActiveTab(tab.key)}
              className="rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors"
              data-active={activeTab === tab.key}
              style={{
                borderColor: activeTab === tab.key ? 'rgba(59,130,246,0.42)' : 'var(--itdb-border)',
                background:
                  activeTab === tab.key ? 'rgba(59,130,246,0.14)' : 'var(--itdb-control-bg-soft)',
                color: activeTab === tab.key ? 'var(--itdb-accent-text)' : 'var(--itdb-text-muted)',
              }}
            >
              {tab.label} <span className="ml-1">{count}</span>
            </button>
          );
        })}
      </div>
      <div className="itdb-hidden-scrollbar min-h-0 flex-1 space-y-1.5 overflow-y-auto px-1 pt-1">
        {ids.length ? (
          ids.map((id, index) =>
            canOpenRecordEditor(active.path) ? (
              <Link
                key={`${active.key}-${id}`}
                to={active.path}
                search={{ edit: Number(id) }}
                target="_blank"
                rel="noopener"
                aria-label={`在新窗口编辑${active.label} ${id}`}
                className="flex items-start gap-2 rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-2.5 py-2 text-xs text-[var(--itdb-accent-text)] transition-colors hover:border-[rgba(59,130,246,0.42)] hover:bg-[rgba(59,130,246,0.1)]"
              >
                <span className="shrink-0 font-mono text-[var(--itdb-text-muted)]">
                  {index + 1}:
                </span>
                <span className="min-w-0 break-words">
                  {overviewEntryText(active.lookup, rowsById.get(id), id)}
                </span>
              </Link>
            ) : (
              <div
                key={`${active.key}-${id}`}
                className="flex items-start gap-2 rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-2.5 py-2 text-xs text-[var(--itdb-text-muted)]"
              >
                <span className="shrink-0 font-mono text-[var(--itdb-text-muted)]">
                  {index + 1}:
                </span>
                <span className="min-w-0 break-words">
                  {overviewEntryText(active.lookup, rowsById.get(id), id)}
                </span>
              </div>
            )
          )
        ) : (
          <div className="border border-dashed border-[var(--itdb-border)] px-2.5 py-3 text-xs text-[var(--itdb-text-muted)]">
            暂无关联记录
          </div>
        )}
      </div>
      {children}
    </section>
  );
}

// 标记关联的待定变更：页签内的增删改先记录在本地，点“保存修改”后由编辑器统一提交
export type PendingTagChanges = {
  added: string[];
  removed: string[];
};

type TagRow = { key: number; name: string };

export function HardwareTagsPane({
  initialTags,
  tagResource = 'items',
  lookups,
  onPendingTagsChange,
}: {
  initialTags: unknown;
  tagResource?: string;
  noun?: string;
  lookups?: Lookups;
  onPendingTagsChange?: (pending: PendingTagChanges) => void;
}) {
  const [rows, setRows] = useState<TagRow[]>(() =>
    (Array.isArray(initialTags) ? initialTags.map(String) : []).map((name, index) => ({
      key: index + 1,
      name,
    }))
  );
  const [value, setValue] = useState('');
  const [editingKey, setEditingKey] = useState<number | null>(null);
  const [editingOriginal, setEditingOriginal] = useState<string | null>(null);
  const tempKeyRef = useRef(-1);
  const nextIdQuery = useQuery({
    queryKey: ['itdb', 'tags', 'next-id'],
    staleTime: 0,
    queryFn: () => api<{ nextId: number }>('/api/tags/next-id'),
  });
  const nextBase = Number(nextIdQuery.data?.nextId) || 1;
  const dict = Array.isArray(lookups?.tags) ? lookups.tags : [];

  const baselineNames = useMemo<string[]>(
    () => (Array.isArray(initialTags) ? initialTags.map(String) : []).map(name => name.trim()),
    [initialTags]
  );
  useEffect(() => {
    if (!onPendingTagsChange) return;
    const current = rows.map(row => row.name);
    const added = current.filter((name: string) => !baselineNames.includes(name));
    const removed = baselineNames.filter((name: string) => !current.includes(name));
    onPendingTagsChange({ added, removed });
  }, [rows, baselineNames, onPendingTagsChange, tagResource]);

  const displayIds = new Map<number, number>();
  {
    let seq = 0;
    rows.forEach(row => {
      const dictTag = dict.find(tag => String(tag.name ?? '').trim() === row.name);
      if (dictTag) {
        displayIds.set(row.key, Number(dictTag.id));
      } else {
        displayIds.set(row.key, nextBase + seq);
        seq += 1;
      }
    });
  }
  const displayIdOf = (row: TagRow) => displayIds.get(row.key) ?? nextBase;

  const addPending = () => {
    const name = value.trim();
    if (!name) return;
    if (rows.some(row => row.name === name)) {
      toast.error(`标记 ${name} 已存在`);
      return;
    }
    const key = tempKeyRef.current;
    tempKeyRef.current -= 1;
    setRows(list => [...list, { key, name }]);
    setValue('');
    toast.success('标记变更将在保存后生效');
  };
  const startEdit = (row: TagRow) => {
    setEditingKey(row.key);
    setEditingOriginal(row.name);
  };
  const cancelEdit = (row: TagRow) => {
    const restored = editingOriginal ?? row.name;
    setRows(list => list.map(item => (item.key === row.key ? { ...item, name: restored } : item)));
    setEditingKey(null);
    setEditingOriginal(null);
  };
  const renamePending = (key: number, name: string) => {
    setRows(list => list.map(row => (row.key === key ? { ...row, name } : row)));
  };
  const removeRow = (row: TagRow) => {
    setRows(list => list.filter(item => item.key !== row.key));
    if (editingKey === row.key) setEditingKey(null);
    toast.success('标记变更将在保存后生效');
  };

  return (
    <section className="itdb-resource-form-section mx-auto flex min-h-0 w-full max-w-3xl flex-1 flex-col rounded-xl border border-[var(--itdb-border)] px-5 pb-5 pt-3">
      <div className="mb-4 flex shrink-0 items-center justify-between gap-3 border-b border-[var(--itdb-border)] pb-2">
        <h3 className="text-base font-semibold text-[var(--itdb-text)]">Tags</h3>
        <span className="text-xs text-[var(--itdb-text-muted)]">
          标记变更将在保存后生效；删除仅解除关联，不会删除标记本身
        </span>
      </div>
      <div className="itdb-resource-data-panel itdb-hidden-scrollbar min-h-0 flex-1 overflow-auto rounded-lg border border-[var(--itdb-border)]">
        <table className="min-w-full text-sm">
          <thead className="sticky top-0 z-10 bg-[var(--itdb-control-bg-soft)] text-[var(--itdb-text-muted)]">
            <tr>
              <th className="w-24 px-3 py-2.5 text-center font-medium">编号</th>
              <th className="px-3 py-2.5 text-center font-medium">标记名称</th>
              <th className="w-28 px-3 py-2.5 text-center font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            {rows.length ? (
              rows.map(row => {
                const editing = editingKey === row.key;
                return (
                  <tr
                    key={row.key}
                    className="border-t border-[var(--itdb-border)]/70 hover:bg-[var(--itdb-control-bg-soft)]"
                  >
                    <td className="px-3 py-2.5 text-center">
                      <span className="itdb-relation-id">{displayIdOf(row)}</span>
                    </td>
                    <td className="px-3 py-2 text-center">
                      {editing ? (
                        <Input
                          value={row.name}
                          onChange={event => renamePending(row.key, event.target.value)}
                          aria-label="标记名称"
                        />
                      ) : (
                        <span className="text-[var(--itdb-text)]">{row.name || '-'}</span>
                      )}
                    </td>
                    <td className="px-3 py-2.5 text-center">
                      <div className="flex justify-center gap-1.5">
                        {editing ? (
                          <AppTooltip label="提交">
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              className="itdb-action-button h-7 w-7"
                              style={{
                                borderColor: 'rgba(16,185,129,0.4)',
                                background: 'rgba(16,185,129,0.1)',
                                color: '#059669',
                              }}
                              onClick={() => {
                                setEditingKey(null);
                                setEditingOriginal(null);
                              }}
                              aria-label={`提交标记修改 ${row.name}`}
                            >
                              <Check size={13} />
                            </Button>
                          </AppTooltip>
                        ) : (
                          <AppTooltip label="编辑">
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              className="itdb-action-button h-7 w-7"
                              style={{
                                borderColor: 'rgba(59,130,246,0.38)',
                                background: 'rgba(59,130,246,0.1)',
                                color: 'var(--itdb-accent-text)',
                              }}
                              onClick={() => startEdit(row)}
                              aria-label={`编辑标记 ${row.name}`}
                            >
                              <Pencil size={13} />
                            </Button>
                          </AppTooltip>
                        )}
                        {editing ? (
                          <AppTooltip label="取消">
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              className="itdb-action-button h-7 w-7"
                              style={{
                                borderColor: 'rgba(239,68,68,0.34)',
                                background: 'rgba(239,68,68,0.08)',
                                color: '#ef4444',
                              }}
                              onClick={() => cancelEdit(row)}
                              aria-label={`取消编辑标记 ${row.name}`}
                            >
                              <X size={13} />
                            </Button>
                          </AppTooltip>
                        ) : (
                          <AppTooltip label="删除">
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              className="itdb-action-button h-7 w-7"
                              style={{
                                borderColor: 'rgba(239,68,68,0.34)',
                                background: 'rgba(239,68,68,0.08)',
                                color: '#ef4444',
                              }}
                              onClick={() => removeRow(row)}
                              aria-label={`删除标记 ${row.name}`}
                            >
                              <Trash2 size={13} />
                            </Button>
                          </AppTooltip>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })
            ) : (
              <tr>
                <td
                  colSpan={3}
                  className="px-4 py-8 text-center text-sm text-[var(--itdb-text-muted)]"
                >
                  暂无标记
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
      <div className="mt-4 flex shrink-0 gap-2">
        <Input
          value={value}
          onChange={event => setValue(event.target.value)}
          placeholder="请输入标记名称"
          onKeyDown={event => {
            if (event.key === 'Enter') {
              event.preventDefault();
              addPending();
            }
          }}
        />
        <Button
          type="button"
          className="disabled:pointer-events-auto disabled:cursor-not-allowed"
          disabled={!value.trim()}
          onClick={addPending}
        >
          新增标记
        </Button>
      </div>
    </section>
  );
}
