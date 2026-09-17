import { Link } from '@tanstack/react-router';
import { canOpenRecordEditor } from '../record-links';
import { PenLine } from 'lucide-react';
import { type ResourceField } from '../resource-config';
import { contractMainGroup, resolveFieldOptions } from '../resource-helpers';
import { AppTooltip } from '@/components/app-tooltip';
import { FieldEditor } from './FieldEditor';
import { HardwareOverview, type OverviewTabConfig } from './HardwareDataPanes';
import { HardwareFilesPane } from './HardwareFilesPane';
import { RenewalEditor } from './ContractPanes';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

const contractOverviewTabs: OverviewTabConfig[] = [
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
];

function agentJumpAction(value: unknown, label: string) {
  const id = Number(value ?? 0);
  const enabled = Number.isFinite(id) && id > 0 && canOpenRecordEditor('/assets/agents');
  return (
    <AppTooltip label={`在新窗口编辑${label}（代理）`}>
      {enabled ? (
        <Link
          to="/assets/agents"
          search={{ edit: Number(id) }}
          target="_blank"
          rel="noopener"
          aria-label={`在新窗口编辑${label}（代理）`}
          className="flex h-5 w-5 items-center justify-center rounded-md border border-[rgba(59,130,246,0.38)] bg-[rgba(59,130,246,0.1)] text-[var(--itdb-accent-text)] transition-colors hover:bg-[rgba(59,130,246,0.18)]"
        >
          <PenLine size={12} />
        </Link>
      ) : (
        <span
          aria-disabled="true"
          className="flex h-5 w-5 cursor-not-allowed items-center justify-center rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] text-[var(--itdb-text-muted)] opacity-60"
        >
          <PenLine size={12} />
        </span>
      )}
    </AppTooltip>
  );
}

function contractJumpAction(value: unknown) {
  const id = Number(value ?? 0);
  const enabled = Number.isFinite(id) && id > 0 && canOpenRecordEditor('/assets/contracts');
  return (
    <AppTooltip label="在新窗口编辑上级合同">
      {enabled ? (
        <Link
          to="/assets/contracts"
          search={{ edit: Number(id) }}
          target="_blank"
          rel="noopener"
          aria-label="在新窗口编辑上级合同"
          className="flex h-5 w-5 items-center justify-center rounded-md border border-[rgba(59,130,246,0.38)] bg-[rgba(59,130,246,0.1)] text-[var(--itdb-accent-text)] transition-colors hover:bg-[rgba(59,130,246,0.18)]"
        >
          <PenLine size={12} />
        </Link>
      ) : (
        <span
          aria-disabled="true"
          className="flex h-5 w-5 cursor-not-allowed items-center justify-center rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] text-[var(--itdb-text-muted)] opacity-60"
        >
          <PenLine size={12} />
        </span>
      )}
    </AppTooltip>
  );
}

export function ContractDataPane({
  fields,
  values,
  lookups,
  activeSection,
  onSectionChange,
  onChange,
  renewals,
  onRenewalsChange,
  onUnlinkFile,
}: {
  fields: ResourceField[];
  values: Row;
  lookups: Lookups | undefined;
  activeSection: string;
  onSectionChange: (section: string) => void;
  onChange: (field: ResourceField, value: unknown) => void;
  renewals: Row[];
  onRenewalsChange: (value: Row[]) => void;
  onUnlinkFile?: (fileId: number) => void;
}) {
  const sections = [
    { value: 'attribute', label: '合同属性配置' },
    { value: 'overview', label: '关联概览' },
    { value: 'renewals', label: '备件' },
    { value: 'files', label: '管理文件' },
  ];
  const group = contractMainGroup(fields, activeSection)[0];

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-5">
      <nav
        className="itdb-hidden-scrollbar flex shrink-0 overflow-x-auto rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] p-1.5"
        aria-label="合同数据配置导航"
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
                options={resolveFieldOptions(field, lookups, values, 'contracts')}
                onChange={value => onChange(field, value)}
                compactHorizontal
                labelAction={
                  field.key === 'contractorId'
                    ? agentJumpAction(values.contractorId, '承包方')
                    : field.key === 'parentId'
                      ? contractJumpAction(values.parentId)
                      : undefined
                }
              />
            ))}
          </div>
        </section>
      ) : activeSection === 'overview' ? (
        <HardwareOverview values={values} lookups={lookups} tabs={contractOverviewTabs} />
      ) : activeSection === 'renewals' ? (
        <RenewalEditor rows={renewals} lookups={lookups} onChange={onRenewalsChange} />
      ) : activeSection === 'files' ? (
        <HardwareFilesPane
          values={values}
          lookups={lookups}
          resourceKey="contracts"
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
