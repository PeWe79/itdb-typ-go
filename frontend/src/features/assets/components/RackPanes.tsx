import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { canOpenRecordEditor } from '../record-links';
import type { CSSProperties, ReactNode } from 'react';
import { AppTooltip } from '@/components/app-tooltip';
import { type ResourceField } from '../resource-config';
import { hexToRgba, isHexColor, resolveFieldOptions } from '../resource-helpers';
import { FieldEditor } from './FieldEditor';
import { api } from '@/lib/auth';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

type RackDepthColumn = { key: 'F' | 'M' | 'B'; label: string; bit: number };

const rackDepthColumns: RackDepthColumn[] = [
  { key: 'F', label: '前侧', bit: 4 },
  { key: 'M', label: '中部', bit: 2 },
  { key: 'B', label: '后侧', bit: 1 },
];

function rackDepthMask(item: Row) {
  const value = Number(item.rackposdepth ?? 0);
  return Number.isFinite(value) && value > 0 ? value : 4;
}

function rackUnitSpan(item: Row, reverse: boolean) {
  const position = Number(item.rackposition ?? 0);
  const height = Math.max(1, Math.floor(Number(item.usize ?? 1) || 1));
  if (!Number.isFinite(position) || position <= 0) return null;
  return reverse
    ? { start: position, end: position + height - 1, height }
    : { start: Math.max(1, position - height + 1), end: position, height };
}

export function RackDataPane({
  fields,
  values,
  lookups,
  onChange,
  rackId,
}: {
  fields: ResourceField[];
  values: Row;
  lookups: Lookups | undefined;
  onChange: (field: ResourceField, value: unknown) => void;
  rackId?: unknown;
}) {
  const id = Number(rackId);
  const saved = Number.isFinite(id) && id > 0;
  const rackItems = useQuery({
    queryKey: ['itdb', 'racks', id, 'items'],
    enabled: saved,
    queryFn: () => api<Row[]>('/api/items?limit=-1&offset=0'),
  });
  const linkedItems = (Array.isArray(rackItems.data) ? rackItems.data : []).filter(
    item => Number(item.rackid) === id
  );
  const uSize = Math.max(0, Math.floor(Number(values.uSize ?? 0) || 0));
  const occupiedUnits = linkedItems.reduce(
    (total, item) => total + Math.max(0, Math.floor(Number(item.usize ?? 0) || 0)),
    0
  );
  const utilization = uSize > 0 ? Math.min(100, Math.round((occupiedUnits / uSize) * 100)) : 0;
  const renderField = (key: string) => {
    const current = fields.find(item => item.key === key);
    if (!current) return null;
    if (key === 'revNums' && saved) {
      return (
        <AppTooltip
          key={key}
          label="创建后不可修改"
          placement="top"
          align="end"
          className="block min-w-0"
        >
          <FieldEditor
            field={current}
            value={values[current.key]}
            options={resolveFieldOptions(current, lookups, values)}
            onChange={value => onChange(current, value)}
            compactHorizontal
            disabled
          />
        </AppTooltip>
      );
    }
    return (
      <FieldEditor
        key={key}
        field={current}
        value={values[current.key]}
        options={resolveFieldOptions(current, lookups, values)}
        onChange={value => onChange(current, value)}
        compactHorizontal
      />
    );
  };
  const readonlyRow = (label: string, labelTooltip: string | undefined, content: ReactNode) => (
    <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
      <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
        {labelTooltip ? (
          <AppTooltip label={labelTooltip} placement="top" align="end" wrap>
            <span className="cursor-help">{label}</span>
          </AppTooltip>
        ) : (
          label
        )}
      </span>
      <div className="flex min-h-10 items-center rounded-lg border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-3 text-sm text-[var(--itdb-text)]">
        {content}
      </div>
    </div>
  );

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-5 lg:flex-row">
      <section className="itdb-resource-form-section flex min-h-0 w-full flex-col rounded-xl border border-[var(--itdb-border)] px-5 pb-5 pt-3 lg:w-[22rem] lg:shrink-0">
        <h3 className="mb-4 shrink-0 border-b border-[var(--itdb-border)] pb-2 text-base font-semibold text-[var(--itdb-text)]">
          机架属性配置
        </h3>
        <div className="itdb-hidden-scrollbar min-h-0 flex-1 space-y-3 overflow-y-auto px-1 pt-1">
          {renderField('uSize')}
          {renderField('revNums')}
          {renderField('label')}
          {renderField('depth')}
          {renderField('model')}
          {renderField('locationId')}
          {renderField('locAreaId')}
          {readonlyRow('硬件', '多少硬件被分配到这个机架', `${linkedItems.length} 台`)}
          <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
            <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
              在用
            </span>
            <AppTooltip label={`${occupiedUnits}U / ${uSize}U`}>
              <div
                className="h-3.5 w-full max-w-56 cursor-help overflow-hidden rounded-full shadow-[inset_0_1px_3px_rgba(15,23,42,0.22)]"
                style={{
                  background: 'color-mix(in srgb, var(--itdb-text-muted) 26%, transparent)',
                }}
              >
                <div
                  className="h-full rounded-full bg-gradient-to-r from-[rgba(59,130,246,0.7)] via-[var(--itdb-accent)] to-[rgba(34,211,238,0.95)] shadow-[inset_0_1px_0_rgba(255,255,255,0.45)] transition-[width] duration-300"
                  style={{ width: `${utilization}%` }}
                />
              </div>
            </AppTooltip>
          </div>
          {renderField('comments')}
        </div>
      </section>
      <section className="itdb-resource-form-section flex min-h-0 min-w-0 flex-1 flex-col rounded-xl border border-[var(--itdb-border)] px-5 pb-5 pt-3">
        <h3 className="mb-4 shrink-0 border-b border-[var(--itdb-border)] pb-2 text-base font-semibold text-[var(--itdb-text)]">
          机架晟图
        </h3>
        <div className="itdb-hidden-scrollbar min-h-0 flex-1 overflow-auto">
          {uSize > 0 ? (
            <RackElevation
              units={uSize}
              reverse={Number(values.revNums ?? 0) === 1}
              items={linkedItems}
            />
          ) : (
            <div className="itdb-rack-empty h-full">请先填写机架高度后查看晟图</div>
          )}
        </div>
      </section>
    </div>
  );
}

export function RackElevation({
  units,
  reverse,
  items,
  highlightId,
}: {
  units: number;
  reverse: boolean;
  items: Row[];
  highlightId?: number;
}) {
  const total = Math.min(Math.max(Math.floor(units), 1), 99);
  const unitList = reverse
    ? Array.from({ length: total }, (_, index) => index + 1)
    : Array.from({ length: total }, (_, index) => total - index);
  const mounted = items
    .map(item => ({
      item,
      span: rackUnitSpan(item, reverse),
      mask: rackDepthMask(item),
      anchor: Math.floor(Number(item.rackposition ?? 0)),
    }))
    .filter(entry => entry.span !== null && entry.anchor > 0);
  const coverageOwners = new Map<string, number[]>();
  mounted.forEach(entry => {
    for (let unit = entry.span!.start; unit <= entry.span!.end; unit++) {
      rackDepthColumns.forEach(column => {
        if ((entry.mask & column.bit) !== column.bit) return;
        const key = `${unit}-${column.key}`;
        coverageOwners.set(key, [...(coverageOwners.get(key) ?? []), Number(entry.item.id)]);
      });
    }
  });
  const warnings: string[] = [];
  coverageOwners.forEach((owners, key) => {
    if (owners.length <= 1) return;
    const [unit, depth] = key.split('-');
    const column = rackDepthColumns.find(item => item.key === depth);
    const names = [...new Set(owners)].map(id => `硬件 ${id}`).join(' 与 ');
    warnings.push(`第 ${unit}U ${column?.label ?? ''}位置冲突：${names}`);
  });
  const cellStyle = (item: Row) => {
    const color = String(item.statuscolor ?? '').trim();
    if (!isHexColor(color)) return undefined;
    const unitCount = Math.max(1, Math.floor(Number(item.usize ?? 1) || 1));
    return {
      backgroundColor: hexToRgba(color, 0.25),
      backgroundImage: `linear-gradient(to bottom, rgba(148, 163, 184, 0.16) 1px, transparent 1px)`,
      backgroundSize: `100% ${100 / unitCount}%`,
      backgroundRepeat: 'repeat',
    } as CSSProperties;
  };
  if (total <= 0) {
    return <div className="itdb-rack-empty h-full">请先填写机架高度后查看晟图</div>;
  }
  return (
    <div className="itdb-rack-view">
      {warnings.length > 0 ? (
        <div className="itdb-hidden-scrollbar mb-2 max-h-24 space-y-1 overflow-y-auto">
          {warnings.map(warning => (
            <p key={warning} className="text-xs text-amber-600 dark:text-amber-400">
              {warning}
            </p>
          ))}
        </div>
      ) : null}
      <table className="itdb-rack-view-table">
        <thead>
          <tr>
            <th className="itdb-rack-view-ru">RU</th>
            {rackDepthColumns.map(column => (
              <th key={column.key}>{column.label}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {unitList.map(unit => (
            <tr key={unit}>
              <td className="itdb-rack-view-ru">{unit}</td>
              {rackDepthColumns.map((column, columnIndex) => {
                const entry = mounted.find(
                  candidate =>
                    (candidate.mask & column.bit) === column.bit &&
                    unit >= candidate.span!.start &&
                    unit <= candidate.span!.end
                );
                if (!entry) return <td key={column.key} className="itdb-rack-view-empty" />;
                if (unit !== entry.anchor) return null;
                const previous = rackDepthColumns[columnIndex - 1];
                if (previous && (entry.mask & previous.bit) === previous.bit) return null;
                let colSpan = 1;
                for (let next = columnIndex + 1; next < rackDepthColumns.length; next++) {
                  if ((entry.mask & rackDepthColumns[next].bit) !== rackDepthColumns[next].bit)
                    break;
                  colSpan++;
                }
                const item = entry.item;
                const highlighted = highlightId !== undefined && Number(item.id) === highlightId;
                const text = (value: unknown) => String(value ?? '').trim();
                const title = `${text(item.manufacturer) || '-'} ${text(item.model) || '-'} [ID:${item.id}]`;
                const subtitleParts = [
                  text(item.label),
                  text(item.ipv4) ? `[ip:${text(item.ipv4).split(',')[0]}]` : '',
                ].filter(Boolean);
                const tip = `编号: ${item.id} / 状态: ${text(item.status) || '使用中'} / 位置: ${text(item.rackposition)}U / 高度: ${Math.max(1, Math.floor(Number(item.usize ?? 1) || 1))}U`;
                return (
                  <td
                    key={column.key}
                    rowSpan={entry.span!.height}
                    colSpan={colSpan}
                    className="itdb-rack-view-item"
                    style={{
                      ...cellStyle(item),
                      ...(highlighted
                        ? { outline: '2px solid #f97316', outlineOffset: '-2px' }
                        : null),
                    }}
                  >
                    <AppTooltip label={tip}>
                      {canOpenRecordEditor('/assets/hardware') ? (
                        <Link to="/assets/hardware" search={{ edit: Number(item.id) }}>
                          <span className="itdb-rack-view-title">{title}</span>
                          {subtitleParts.length ? (
                            <span className="itdb-rack-view-subtitle">
                              {subtitleParts.join(' ')}
                            </span>
                          ) : null}
                        </Link>
                      ) : (
                        <span>
                          <span className="itdb-rack-view-title">{title}</span>
                          {subtitleParts.length ? (
                            <span className="itdb-rack-view-subtitle">
                              {subtitleParts.join(' ')}
                            </span>
                          ) : null}
                        </span>
                      )}
                    </AppTooltip>
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr>
            <td colSpan={4} className="itdb-rack-view-base" />
          </tr>
        </tfoot>
      </table>
      <div className="itdb-rack-view-wheels">
        {[0, 1].map(wheel => (
          <span key={wheel} className="itdb-rack-view-wheel" aria-hidden="true">
            <svg
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
            >
              <circle cx="12" cy="12" r="9" />
              <circle cx="12" cy="12" r="3" />
              <line x1="12" y1="3" x2="12" y2="7" />
              <line x1="12" y1="17" x2="12" y2="21" />
              <line x1="3" y1="12" x2="7" y2="12" />
              <line x1="17" y1="12" x2="21" y2="12" />
            </svg>
          </span>
        ))}
      </div>
    </div>
  );
}
