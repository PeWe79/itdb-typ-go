import { Download } from 'lucide-react';
import { useState, type ReactNode } from 'react';
import { AppTooltip } from '@/components/app-tooltip';
import { downloadStoredFile } from '@/lib/file-download';
import { type ResourceField } from '../resource-config';
import { lookupOptions } from '../resource-helpers';
import { FieldEditor } from './FieldEditor';
import { resolveInvoiceFileTypeId, resolveInvoiceFileTypeLabel } from './HardwarePanes';
import { INVOICE_FILE_ACCEPT } from './resource-editor-rules';
import { HardwareOverview } from './HardwareDataPanes';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

function formatUploadDateTime(value: unknown) {
  const timestamp = Number(value);
  if (!Number.isFinite(timestamp) || timestamp <= 0) return '-';
  const date = new Date(timestamp * 1000);
  const pad = (num: number) => String(num).padStart(2, '0');
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`;
}

export function FileDataPane({
  fields,
  values,
  lookups,
  onChange,
  itemId,
  detail,
}: {
  fields: ResourceField[];
  values: Row;
  lookups: Lookups | undefined;
  onChange: (field: ResourceField, value: unknown) => void;
  itemId?: unknown;
  detail?: Row;
}) {
  const [activeSection, setActiveSection] = useState('attribute');
  const sections = [
    { value: 'attribute', label: '文件属性配置' },
    { value: 'overview', label: '关联概览' },
  ];
  const id = Number(itemId ?? 0);
  const saved = Number.isFinite(id) && id > 0;
  const editableFields = fields.filter(field => !field.key.endsWith('Links'));
  const fileFieldRaw = fields.find(field => field.key === 'file');
  const fileName = String(detail?.fname ?? '').trim();
  const uploader = String(detail?.uploader ?? '').trim();
  const uploadDate = formatUploadDateTime(detail?.uploaddate);
  const uploaderText =
    uploader && uploadDate !== '-'
      ? `${uploader} / ${uploadDate}`
      : uploader || (uploadDate !== '-' ? uploadDate : '-');
  const associationCount = ['itemLinks', 'softwareLinks', 'invoiceLinks', 'contractLinks'].reduce(
    (total, key) => total + (Array.isArray(values[key]) ? (values[key] as unknown[]).length : 0),
    0
  );
  const invoiceTypeId = resolveInvoiceFileTypeId(lookups);
  const invoiceTypeLabel = resolveInvoiceFileTypeLabel(lookups);
  const invoiceSelected = invoiceTypeId > 0 && Number(values.typeId) === invoiceTypeId;
  const typeIsLockedInvoice = saved && invoiceSelected;
  const fileField = fileFieldRaw
    ? {
        ...fileFieldRaw,
        required: !saved,
        accept: invoiceSelected ? INVOICE_FILE_ACCEPT : fileFieldRaw.accept,
      }
    : undefined;
  const typeOptionsFor = (field: ResourceField) => {
    const all = field.options ?? lookupOptions(lookups, field.optionsKey);
    if (!saved) return all;
    return all.filter(option => Number(option.value) !== invoiceTypeId);
  };
  const readonlyRow = (label: string, content: ReactNode, tooltip?: string) => (
    <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
      <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
        {tooltip ? (
          <AppTooltip label={tooltip} placement="top" align="end">
            <span className="cursor-help">{label}</span>
          </AppTooltip>
        ) : (
          label
        )}
      </span>
      <div className="itdb-location-readonly">{content}</div>
    </div>
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-5">
      <nav
        className="itdb-hidden-scrollbar flex shrink-0 overflow-x-auto rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] p-1.5"
        aria-label="文件数据配置导航"
      >
        <div className="flex min-w-max gap-1.5">
          {sections.map(section => (
            <button
              key={section.value}
              type="button"
              onClick={() => setActiveSection(section.value)}
              className="itdb-hardware-subtab h-9 rounded-lg px-3 text-sm font-medium"
              data-active={activeSection === section.value}
            >
              {section.label}
            </button>
          ))}
        </div>
      </nav>

      {activeSection === 'attribute' ? (
        <section className="itdb-resource-form-section mx-auto flex min-h-0 w-full max-w-3xl flex-1 flex-col rounded-xl border border-[var(--itdb-border)] px-5 pb-5 pt-3">
          <h3 className="mb-4 border-b border-[var(--itdb-border)] pb-2 text-base font-semibold text-[var(--itdb-text)]">
            文件属性配置
          </h3>
          <div className="itdb-hidden-scrollbar min-h-0 flex-1 space-y-3 overflow-y-auto px-1 pt-1">
            {editableFields.map(field =>
              field.key === 'file' && fileField ? (
                <FieldEditor
                  key={field.key}
                  field={fileField}
                  value={values[fileField.key]}
                  options={[]}
                  onChange={value => onChange(fileField, value)}
                  compactHorizontal
                />
              ) : field.key === 'typeId' && typeIsLockedInvoice ? (
                <FieldEditor
                  key={field.key}
                  field={field}
                  value={values[field.key]}
                  options={[{ label: invoiceTypeLabel, value: Number(values.typeId) }]}
                  onChange={() => {}}
                  disabled
                  compactHorizontal
                />
              ) : (
                <FieldEditor
                  key={field.key}
                  field={field}
                  value={values[field.key]}
                  options={
                    field.key === 'typeId'
                      ? typeOptionsFor(field)
                      : (field.options ?? lookupOptions(lookups, field.optionsKey))
                  }
                  onChange={value => onChange(field, value)}
                  compactHorizontal
                />
              )
            )}
            {readonlyRow(
              '文件名称',
              saved && fileName ? (
                <AppTooltip label="下载当前文件">
                  <button
                    type="button"
                    className="itdb-invoice-file-pill max-w-full"
                    onClick={() =>
                      void downloadStoredFile(
                        `/api/files/${id}/download`,
                        fileName.includes('.') ? fileName : `${fileName}.bin`
                      )
                    }
                  >
                    <Download size={12} className="shrink-0" />
                    <span className="min-w-0 truncate">{fileName}</span>
                  </button>
                </AppTooltip>
              ) : (
                '-'
              )
            )}
            {readonlyRow('关联数', associationCount, '引用此文件的硬件、软件、合同总数')}
            {readonlyRow('上传人', uploaderText)}
          </div>
        </section>
      ) : (
        <HardwareOverview
          values={values}
          lookups={lookups}
          tabs={[
            {
              key: 'items',
              label: '硬件',
              field: 'itemLinks',
              lookup: 'items_ref',
              path: '/assets/hardware',
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
            {
              key: 'software',
              label: '软件',
              field: 'softwareLinks',
              lookup: 'software_ref',
              path: '/assets/software',
            },
          ]}
        />
      )}
    </div>
  );
}
