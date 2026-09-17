import { PenLine, Eye, Upload } from 'lucide-react';
import { ReactNode } from 'react';
import { AppTooltip } from '@/components/app-tooltip';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { DatePicker } from './DatePicker';
import { ResourceField } from '../resource-config';
import { canOpenRecordEditor } from '../record-links';
import { getStoredUser, userHasPermission } from '@/lib/auth';
import { PERM } from '@/lib/permissions';

type Row = Record<string, unknown>;

export function FieldEditor({
  field,
  value,
  options,
  onChange,
  compactHorizontal = false,
  labelAction = null,
  disabled = false,
  rackViewHighlightId,
}: {
  field: ResourceField;
  value: unknown;
  options: { label: string; value: string | number; tooltip?: string }[];
  onChange: (value: unknown) => void;
  compactHorizontal?: boolean;
  labelAction?: ReactNode;
  disabled?: boolean;
  rackViewHighlightId?: unknown;
}) {
  const id = `field-${field.key}`;
  if (field.type === 'contacts' || field.type === 'urls') return null;
  const labelContent = (
    <label
      htmlFor={compactHorizontal ? undefined : id}
      className={
        compactHorizontal
          ? 'cursor-default text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]'
          : 'cursor-default text-sm font-medium text-[var(--itdb-text)]'
      }
    >
      {field.tooltip ? (
        <AppTooltip
          label={field.tooltip}
          placement="top"
          align="end"
          wrap={field.tooltipWrap ?? true}
        >
          <span className="cursor-help">{field.label}</span>
        </AppTooltip>
      ) : (
        field.label
      )}
      {field.required && <span className="ml-1 text-red-500">*</span>}
    </label>
  );
  const rackLinkActions =
    compactHorizontal && field.key === 'rackId' ? (
      <span className="flex shrink-0 gap-1">
        {userHasPermission(getStoredUser(), PERM.assetsItemsRead) ? (
          <AppTooltip label="查看机架晟图">
            {value ? (
              <a
                href={`/rack-view/${Number(value)}${
                  rackViewHighlightId && Number(rackViewHighlightId) > 0
                    ? `?highlight=${Number(rackViewHighlightId)}`
                    : ''
                }`}
                target="_blank"
                rel="noopener"
                aria-label="查看机架晟图"
                className="flex h-5 w-5 items-center justify-center rounded-md border border-[rgba(59,130,246,0.38)] bg-[rgba(59,130,246,0.1)] text-[var(--itdb-accent-text)] transition-colors hover:bg-[rgba(59,130,246,0.18)]"
              >
                <Eye size={12} />
              </a>
            ) : (
              <span
                aria-disabled="true"
                className="flex h-5 w-5 cursor-not-allowed items-center justify-center rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] text-[var(--itdb-text-muted)] opacity-60"
              >
                <Eye size={12} />
              </span>
            )}
          </AppTooltip>
        ) : null}
        {canOpenRecordEditor('/assets/racks') ? (
          <AppTooltip label="在新窗口编辑机架">
            {value ? (
              <a
                href={`/assets/racks?edit=${Number(value)}`}
                target="_blank"
                rel="noopener"
                aria-label="在新窗口编辑机架"
                className="flex h-5 w-5 items-center justify-center rounded-md border border-[rgba(59,130,246,0.38)] bg-[rgba(59,130,246,0.1)] text-[var(--itdb-accent-text)] transition-colors hover:bg-[rgba(59,130,246,0.18)]"
              >
                <PenLine size={12} />
              </a>
            ) : (
              <span
                aria-disabled="true"
                className="flex h-5 w-5 cursor-not-allowed items-center justify-center rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] text-[var(--itdb-text-muted)] opacity-60"
              >
                <PenLine size={12} />
              </span>
            )}
          </AppTooltip>
        ) : null}
      </span>
    ) : null;
  const label = compactHorizontal ? (
    <div className="flex items-center justify-end gap-1.5">
      {labelAction}
      {rackLinkActions}
      {labelContent}
    </div>
  ) : (
    labelContent
  );
  const inlineRadioOptions =
    field.key === 'isPart' || field.key === 'rackMountable'
      ? [
          { value: 1, label: '是' },
          { value: 0, label: '否' },
        ]
      : field.options;
  if (
    compactHorizontal &&
    inlineRadioOptions &&
    (field.key === 'isPart' || field.key === 'rackMountable' || field.key === 'licenseType')
  )
    return (
      <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
        {label}
        <div className="flex h-10 items-center gap-1.5 rounded-lg border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-1.5 text-sm">
          {inlineRadioOptions.map((option, index) => (
            <label key={String(option.value)} className="cursor-pointer">
              <input
                id={index === 0 ? id : undefined}
                type="radio"
                name={id}
                className="peer sr-only"
                checked={
                  value !== '' &&
                  value !== null &&
                  value !== undefined &&
                  Number(value) === Number(option.value)
                }
                onChange={() => onChange(option.value)}
              />
              <span className="inline-flex items-center justify-center rounded-full border border-[var(--itdb-border)] bg-[var(--itdb-control-bg)] px-3.5 py-1 text-sm text-[var(--itdb-text)] transition-colors hover:border-[rgba(59,130,246,0.35)] hover:bg-[var(--itdb-control-bg-soft)] peer-checked:border-[rgba(59,130,246,0.45)] peer-checked:bg-[rgba(59,130,246,0.12)] peer-checked:font-medium peer-checked:text-[var(--itdb-accent-text)] peer-focus-visible:ring-2 peer-focus-visible:ring-[var(--itdb-ring)]">
                {option.label}
              </span>
            </label>
          ))}
        </div>
      </div>
    );
  if (field.type === 'file')
    return (
      <div
        className={
          compactHorizontal
            ? 'grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5'
            : 'space-y-2.5'
        }
      >
        {label}
        <div className="flex min-w-0 items-center gap-3">
          <label className="w-fit cursor-pointer">
            <input
              type="file"
              className="hidden"
              aria-label={field.label}
              accept={field.accept}
              onChange={event => onChange(event.target.files?.[0])}
            />
            <span className="flex h-10 shrink-0 items-center gap-2 rounded-lg border border-[rgba(59,130,246,0.35)] bg-[rgba(59,130,246,0.08)] px-3 text-sm font-medium text-[var(--itdb-accent-text)] transition-all hover:-translate-y-0.5 hover:border-[rgba(59,130,246,0.55)] hover:bg-[rgba(59,130,246,0.16)] hover:shadow-md">
              <Upload size={15} className="shrink-0" />
              选择文件
            </span>
          </label>
          <span
            className={`min-w-0 truncate text-sm ${value instanceof File ? 'text-[var(--itdb-text)]' : 'text-[var(--itdb-text-muted)]'}`}
          >
            {value instanceof File ? value.name : '未选择文件'}
          </span>
        </div>
      </div>
    );
  if (field.type === 'textarea')
    return (
      <div
        className={
          compactHorizontal
            ? 'grid grid-cols-[4.75rem_minmax(0,1fr)] items-start gap-x-2.5'
            : 'space-y-2.5 md:col-span-2'
        }
      >
        {label}
        <textarea
          id={id}
          value={String(value ?? '')}
          onChange={event => onChange(event.target.value)}
          aria-label={field.label}
          className="itdb-form-control min-h-24 w-full rounded-lg px-3 py-2 text-sm"
        />
      </div>
    );
  if (field.type === 'date')
    return (
      <div
        className={
          compactHorizontal
            ? 'grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5'
            : 'space-y-2.5'
        }
      >
        {label}
        <DatePicker value={String(value ?? '')} onChange={onChange} />
      </div>
    );
  if (field.type === 'select')
    return (
      <div
        className={
          compactHorizontal
            ? 'grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5'
            : 'space-y-2.5'
        }
      >
        {label}
        <div className="flex min-w-0 items-center gap-2">
          <span className="min-w-0 flex-1">
            <Select
              value={
                value === '' || value === null || value === undefined
                  ? '__placeholder__'
                  : String(value)
              }
              onValueChange={next => {
                if (next === '__placeholder__') {
                  onChange('');
                  return;
                }
                const matched = options.find(o => String(o.value) === next);
                if (!matched) return;
                onChange(matched.value);
              }}
            >
              <SelectTrigger
                id={id}
                aria-label={field.label}
                aria-required={field.required}
                disabled={disabled}
              >
                <SelectValue placeholder="请选择">
                  {options.find(option => String(option.value) === String(value))?.label}
                </SelectValue>
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="__placeholder__">请选择</SelectItem>
                {options.length ? (
                  options.map(option => (
                    <SelectItem
                      key={option.value}
                      value={String(option.value)}
                      title={option.tooltip}
                    >
                      {option.label}
                    </SelectItem>
                  ))
                ) : (
                  <SelectItem value="__empty_options__" disabled>
                    暂无{field.label}数据
                  </SelectItem>
                )}
              </SelectContent>
            </Select>
          </span>
        </div>
      </div>
    );
  if (field.type === 'multiselect')
    return (
      <div
        className={
          compactHorizontal
            ? 'grid grid-cols-[4.75rem_minmax(0,1fr)] items-start gap-x-2.5'
            : 'space-y-2.5 md:col-span-2'
        }
      >
        {label}
        <div
          id={id}
          className="itdb-hidden-scrollbar itdb-multi-select-box max-h-56 space-y-0.5 overflow-y-auto rounded-xl border p-1.5"
        >
          {options.map(option => {
            const selected = (Array.isArray(value) ? value : []).some(
              selectedValue => Number(selectedValue) === Number(option.value)
            );
            return (
              <label
                key={option.value}
                className={`flex cursor-pointer items-center gap-2.5 rounded-lg border px-2.5 py-1.5 text-sm transition-colors ${
                  selected
                    ? 'border-[rgba(59,130,246,0.32)] bg-[rgba(59,130,246,0.1)] text-[var(--itdb-accent-text)]'
                    : 'border-transparent text-[var(--itdb-text)] hover:bg-[rgba(59,130,246,0.08)]'
                }`}
              >
                <input
                  type="checkbox"
                  className="itdb-check-box"
                  checked={selected}
                  onChange={() => {
                    const current = (Array.isArray(value) ? value : []).map(Number);
                    const next = Number(option.value);
                    onChange(selected ? current.filter(item => item !== next) : [...current, next]);
                  }}
                />
                <span>{option.label}</span>
              </label>
            );
          })}
        </div>
      </div>
    );
  return (
    <div
      className={
        compactHorizontal
          ? 'grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5'
          : 'space-y-2.5'
      }
    >
      {label}
      <Input
        id={id}
        type={field.type}
        min={field.min}
        className={
          field.noSpinner
            ? '[appearance:textfield] [&::-webkit-inner-spin-button]:appearance-none [&::-webkit-outer-spin-button]:appearance-none'
            : undefined
        }
        aria-required={field.required}
        value={String(value ?? '')}
        aria-label={field.label}
        onChange={event =>
          onChange(
            field.type === 'number'
              ? event.target.value === ''
                ? ''
                : Number(event.target.value)
              : event.target.value
          )
        }
      />
    </div>
  );
}
