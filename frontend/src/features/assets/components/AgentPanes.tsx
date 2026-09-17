import { Link } from '@tanstack/react-router';
import { canOpenRecordEditor } from '../record-links';
import { useState } from 'react';
import { AppTooltip } from '@/components/app-tooltip';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { type ResourceField } from '../resource-config';
import { agentMainGroup, resolveFieldOptions } from '../resource-helpers';
import { FieldEditor } from './FieldEditor';
import { overviewEntryText } from '../resource-helpers';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

export function AgentDataPane({
  fields,
  values,
  lookups,
  activeSection,
  onSectionChange,
  onChange,
  itemId,
}: {
  fields: ResourceField[];
  values: Row;
  lookups: Lookups | undefined;
  activeSection: string;
  onSectionChange: (section: string) => void;
  onChange: (field: ResourceField, value: unknown) => void;
  itemId?: unknown;
}) {
  const [activeOverviewTab, setActiveOverviewTab] = useState('items');
  const sections = [
    { value: 'attribute', label: '代理属性配置' },
    { value: 'overview', label: '关联概览' },
    { value: 'contacts', label: '合同' },
    { value: 'urls', label: 'URLs' },
  ];
  const group = agentMainGroup(fields, activeSection)[0];
  const contactsField = fields.find(field => field.key === 'contacts');
  const urlsField = fields.find(field => field.key === 'urls');
  const agentId = Number(itemId ?? 0);
  const overviewTabs = [
    {
      key: 'items',
      label: '硬件',
      lookup: 'items_ref',
      path: '/assets/hardware',
      match: (row: Row) => Number(row.manufacturerid) === agentId,
    },
    {
      key: 'software',
      label: '软件',
      lookup: 'software_ref',
      path: '/assets/software',
      match: (row: Row) => Number(row.manufacturerid) === agentId,
    },
    {
      key: 'invoicesVendor',
      label: '单据(供应方)',
      lookup: 'invoices_ref',
      path: '/assets/invoices',
      match: (row: Row) => Number(row.vendorid) === agentId,
    },
    {
      key: 'invoicesBuyer',
      label: '单据(采购方)',
      lookup: 'invoices_ref',
      path: '/assets/invoices',
      match: (row: Row) => Number(row.buyerid) === agentId,
    },
  ];
  const activeTab = overviewTabs.find(tab => tab.key === activeOverviewTab) ?? overviewTabs[0];
  const overviewRows =
    agentId > 0
      ? (lookups?.[activeTab.lookup] ?? [])
          .filter(activeTab.match)
          .map(row => ({ id: Number(row.id), row }))
      : [];

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-5">
      <nav
        className="itdb-hidden-scrollbar flex shrink-0 overflow-x-auto rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] p-1.5"
        aria-label="代理数据配置导航"
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
                options={resolveFieldOptions(field, lookups, values)}
                onChange={value => onChange(field, value)}
                compactHorizontal
              />
            ))}
          </div>
        </section>
      ) : activeSection === 'overview' ? (
        <section className="itdb-resource-form-section mx-auto flex min-h-0 w-full max-w-3xl flex-1 flex-col rounded-xl border border-[var(--itdb-border)] px-5 pb-5 pt-3">
          <h3 className="mb-4 border-b border-[var(--itdb-border)] pb-2 text-base font-semibold text-[var(--itdb-text)]">
            关联概览
          </h3>
          <div className="mb-3 flex flex-wrap gap-2">
            {overviewTabs.map(tab => (
              <button
                key={tab.key}
                type="button"
                onClick={() => setActiveOverviewTab(tab.key)}
                className="rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors"
                data-active={activeOverviewTab === tab.key}
                style={{
                  borderColor:
                    activeOverviewTab === tab.key ? 'rgba(59,130,246,0.42)' : 'var(--itdb-border)',
                  background:
                    activeOverviewTab === tab.key
                      ? 'rgba(59,130,246,0.14)'
                      : 'var(--itdb-control-bg-soft)',
                  color:
                    activeOverviewTab === tab.key
                      ? 'var(--itdb-accent-text)'
                      : 'var(--itdb-text-muted)',
                }}
              >
                {tab.label}
              </button>
            ))}
          </div>
          <div className="itdb-hidden-scrollbar min-h-0 flex-1 space-y-1.5 overflow-y-auto px-1 pt-1">
            {overviewRows.length ? (
              overviewRows.map((entry, index) =>
                canOpenRecordEditor(activeTab.path) ? (
                  <Link
                    key={`${activeTab.key}-${entry.id}`}
                    to={activeTab.path}
                    search={{ edit: Number(entry.id) }}
                    className="flex items-start gap-2 rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-2.5 py-2 text-xs text-[var(--itdb-accent-text)] transition-colors hover:border-[rgba(59,130,246,0.42)] hover:bg-[rgba(59,130,246,0.1)]"
                  >
                    <span className="shrink-0 font-mono text-[var(--itdb-text-muted)]">
                      {index + 1}:
                    </span>
                    <span className="min-w-0 break-words">
                      {overviewEntryText(activeTab.lookup, entry.row, entry.id)}
                    </span>
                  </Link>
                ) : (
                  <div
                    key={`${activeTab.key}-${entry.id}`}
                    className="flex items-start gap-2 rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-2.5 py-2 text-xs text-[var(--itdb-text-muted)]"
                  >
                    <span className="shrink-0 font-mono text-[var(--itdb-text-muted)]">
                      {index + 1}:
                    </span>
                    <span className="min-w-0 break-words">
                      {overviewEntryText(activeTab.lookup, entry.row, entry.id)}
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
        </section>
      ) : activeSection === 'contacts' ? (
        contactsField ? (
          <AgentCollectionEditor
            title="合同"
            kind="contacts"
            value={values.contacts}
            onChange={value => onChange(contactsField, value)}
          />
        ) : null
      ) : activeSection === 'urls' ? (
        urlsField ? (
          <AgentCollectionEditor
            title="URLs"
            kind="urls"
            tip="提示：当描述包含“服务”或“service”时，硬件修改页面会显示此链接（多条命中时取第一条）。"
            value={values.urls}
            onChange={value => onChange(urlsField, value)}
          />
        ) : null
      ) : null}
    </div>
  );
}

export function AgentCollectionEditor({
  title,
  kind,
  value,
  onChange,
  tip,
}: {
  title: string;
  kind: 'contacts' | 'urls';
  value: unknown;
  onChange: (value: unknown) => void;
  tip?: string;
}) {
  const rows = Array.isArray(value) ? (value as Row[]) : [];
  const isContact = kind === 'contacts';
  const actualRows = rows.length ? rows : [isContact ? { name: '' } : { description: '' }];
  const add = () =>
    onChange([
      ...actualRows,
      isContact
        ? { name: '', phones: '', email: '', role: '', comments: '' }
        : { description: '', url: '' },
    ]);
  const update = (index: number, key: string, nextValue: string) =>
    onChange(
      actualRows.map((row, rowIndex) => (rowIndex === index ? { ...row, [key]: nextValue } : row))
    );
  const remove = (index: number) =>
    onChange(actualRows.filter((_, rowIndex) => rowIndex !== index));
  const columns: Array<{ key: string; label: string; tooltip?: string }> = isContact
    ? [
        { key: 'name', label: '姓名' },
        { key: 'phones', label: '电话' },
        { key: 'email', label: '邮箱' },
        { key: 'role', label: '角色' },
        { key: 'comments', label: '备注' },
      ]
    : [
        {
          key: 'description',
          label: '描述',
        },
        {
          key: 'url',
          label: 'URL',
          tooltip:
            '跳转规则：http(s):// 开头按实际地址打开；/ 开头从站点根拼接；其余从当前页面所在目录拼接',
        },
      ];
  return (
    <section className="mx-auto flex min-h-0 w-full max-w-6xl flex-1 flex-col rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-5 pb-5 pt-3">
      <div className="mb-4 flex shrink-0 flex-wrap items-center justify-between gap-3 border-b border-[var(--itdb-border)] pb-2">
        <h3 className="text-base font-semibold text-[var(--itdb-text)]">{title}</h3>
        {tip ? <span className="text-xs text-[var(--itdb-text-muted)]">{tip}</span> : null}
      </div>
      <div className="itdb-hidden-scrollbar itdb-resource-data-panel min-h-0 flex-1 overflow-auto rounded-lg border border-[var(--itdb-border)]">
        <table className="w-full table-fixed text-sm">
          <colgroup>
            {columns.map(column => (
              <col key={column.key} className={kind === 'urls' ? 'w-[40%]' : undefined} />
            ))}
            {kind === 'urls' ? <col className="w-20" /> : null}
            <col className="w-20" />
          </colgroup>
          <thead className="sticky top-0 z-10 text-[var(--itdb-text-muted)]">
            <tr>
              {columns.map(column => (
                <th key={column.key} className="px-3 py-2.5 text-center font-medium">
                  {column.tooltip ? (
                    <AppTooltip label={column.tooltip} placement="top">
                      <span className="cursor-help">{column.label}</span>
                    </AppTooltip>
                  ) : (
                    column.label
                  )}
                </th>
              ))}
              {kind === 'urls' ? (
                <th className="px-3 py-2.5 text-center font-medium">跳转</th>
              ) : null}
              <th className="px-3 py-2.5 text-center font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            {actualRows.map((row, index) => (
              <tr
                key={index}
                className="border-t border-[var(--itdb-border)]/70 hover:bg-[var(--itdb-control-bg-soft)]"
              >
                {columns.map(column => (
                  <td key={column.key} className="px-2 py-2">
                    <Input
                      className="h-8"
                      value={String(row[column.key] ?? '')}
                      onChange={event => update(index, column.key, event.target.value)}
                    />
                  </td>
                ))}
                {kind === 'urls' ? (
                  <td className="px-2 py-2 text-center text-[var(--itdb-text-muted)]">
                    {String(row.url ?? '').trim() ? (
                      <AppTooltip label="在新窗口打开该链接">
                        <a
                          href={String(row.url)}
                          target="_blank"
                          rel="noopener"
                          className="inline-flex items-center rounded-full border border-[rgba(59,130,246,0.32)] bg-[rgba(59,130,246,0.12)] px-2.5 py-0.5 text-xs font-semibold text-[var(--itdb-accent-text)] transition-colors hover:bg-[rgba(59,130,246,0.2)]"
                        >
                          GO
                        </a>
                      </AppTooltip>
                    ) : (
                      '-'
                    )}
                  </td>
                ) : null}
                <td className="px-3 py-2.5 text-center">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="itdb-danger-sm-btn h-8 text-xs"
                    onClick={() => remove(index)}
                  >
                    删除
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Button type="button" variant="outline" className="mt-3 w-full shrink-0" onClick={add}>
        新增{title === 'URLs' ? ' URL' : '联系人'}
      </Button>
    </section>
  );
}
