import { Download, ExternalLink } from 'lucide-react';
import { AppTooltip } from '@/components/app-tooltip';
import { canOpenRecordEditor } from '../record-links';
import { downloadStoredFile, previewStoredFile } from '@/lib/file-download';

/** 新窗口打开对应资源的编辑弹窗 */
function openRecordEditor(path: string, id: number) {
  window.open(`${path}?edit=${id}`, '_blank', 'noopener');
}
import {
  getAgentContactEntries,
  parseAgentTypeBadges,
  parseInvoiceFileEntries,
  parseSoftwareInstalledEntries,
  getSoftwareQuantity,
  getInstalledItemBadgeStyle,
  getItemIDBadgeStyle,
  getWarrantyRemain,
  formatValue,
} from '../resource-helpers';

type Row = Record<string, unknown>;

export function TableCellValue({
  row,
  columnKey,
  statusColors,
  installedItemColors,
  resourceKey,
}: {
  row: Row;
  columnKey: string;
  statusColors: Map<string, string>;
  installedItemColors: Map<number, string>;
  resourceKey?: string;
}) {
  if (columnKey === 'id') {
    const value = formatValue(row[columnKey], columnKey);
    return value === '-' ? (
      value
    ) : (
      <span className="itdb-resource-id-badge" style={getItemIDBadgeStyle(row, statusColors)}>
        {value}
      </span>
    );
  }
  if (columnKey === 'warrantyRemain') {
    const warranty = getWarrantyRemain(row);
    if (warranty.text === '-') return warranty.text;
    return (
      <AppTooltip label={`维保截止：${warranty.endDate}`}>
        <span
          className={`itdb-resource-warranty-badge ${warranty.expired ? 'is-expired' : 'is-active'}`}
        >
          {warranty.text}
        </span>
      </AppTooltip>
    );
  }
  if (columnKey === 'purchasedate' || columnKey === 'purchdate') {
    const date = formatValue(row[columnKey], columnKey);
    return date === '-' ? (
      date
    ) : (
      <AppTooltip label={`采购日期：${date}`}>
        <span className="itdb-resource-date">{date}</span>
      </AppTooltip>
    );
  }
  if (columnKey === 'qty') {
    const quantity = getSoftwareQuantity(row);
    return quantity.text === '-' ? (
      quantity.text
    ) : (
      <span className={`itdb-software-quantity ${quantity.tone}`}>{quantity.text}</span>
    );
  }
  if (columnKey === 'maintend') {
    const value = Number(row.maintend ?? 0);
    if (!Number.isFinite(value) || value <= 0) return '-';
    return (
      <AppTooltip label="维护截止日期">
        <span className="itdb-resource-date">{formatValue(value, 'date')}</span>
      </AppTooltip>
    );
  }
  if (columnKey === 'software') {
    const entries = String(row[columnKey] ?? '')
      .split(',')
      .map(entry => entry.trim())
      .filter(Boolean)
      .map(entry => {
        const separator = entry.indexOf('#');
        if (separator <= 0) return { id: null, text: entry };
        const id = Number(entry.slice(0, separator));
        return {
          id: Number.isFinite(id) && id > 0 ? id : null,
          text: entry.slice(separator + 1).trim(),
        };
      });
    if (entries.length === 0) return '-';
    return (
      <div className="itdb-invoice-file-list">
        {entries.map((entry, index) =>
          entry.id ? (
            canOpenRecordEditor('/assets/software') ? (
              <AppTooltip key={`${entry.id}-${index}`} label={`在新窗口编辑软件 ${entry.id}`}>
                <button
                  type="button"
                  className="itdb-software-pill"
                  onClick={() => void openRecordEditor('/assets/software', entry.id!)}
                >
                  <ExternalLink size={12} className="shrink-0" />
                  <span className="min-w-0 truncate">{entry.text}</span>
                </button>
              </AppTooltip>
            ) : (
              <span key={`${entry.id}-${index}`} className="itdb-software-pill">
                {entry.text}
              </span>
            )
          ) : (
            <span key={`na-${index}`} className="itdb-software-pill">
              {entry.text}
            </span>
          )
        )}
      </div>
    );
  }
  if (columnKey === 'tags') {
    const names = String(row.tags ?? '')
      .split(',')
      .map(name => name.trim())
      .filter(Boolean);
    const ids = String(row.tagids ?? '')
      .split(',')
      .map(id => Number(id.trim()))
      .filter(id => Number.isFinite(id) && id > 0);
    if (names.length === 0) return '-';
    return (
      <div className="itdb-invoice-file-list">
        {names.map((name, index) => {
          const id = ids[index] ?? null;
          if (!id) {
            return (
              <span key={`${name}-${index}`} className="itdb-tag-pill">
                {name}
              </span>
            );
          }
          return canOpenRecordEditor('/dictionaries/tags') ? (
            <AppTooltip key={`${name}-${index}`} label={`在新窗口编辑标记 ${id}`}>
              <button
                type="button"
                className="itdb-tag-pill"
                onClick={() => void openRecordEditor('/dictionaries/tags', id)}
              >
                <ExternalLink size={12} className="shrink-0" />
                <span className="min-w-0 truncate">{name}</span>
              </button>
            </AppTooltip>
          ) : (
            <span key={`${name}-${index}`} className="itdb-tag-pill">
              {name}
            </span>
          );
        })}
      </div>
    );
  }
  if (columnKey === 'vendor') {
    const names = String(row.vendor ?? '')
      .split(',')
      .map(name => name.trim())
      .filter(Boolean);
    const ids = String(row.vendorids ?? '')
      .split(',')
      .map(id => Number(id.trim()))
      .filter(id => Number.isFinite(id) && id > 0);
    if (names.length === 0) return '-';
    return (
      <div className="itdb-invoice-file-list">
        {names.map((name, index) => {
          const id = ids[index] ?? null;
          if (!id) {
            return (
              <span key={`${name}-${index}`} className="itdb-vendor-pill">
                {name}
              </span>
            );
          }
          return canOpenRecordEditor('/assets/agents') ? (
            <AppTooltip key={`${name}-${index}`} label={`在新窗口编辑代理 ${id}`}>
              <button
                type="button"
                className="itdb-vendor-pill"
                onClick={() => void openRecordEditor('/assets/agents', id)}
              >
                <ExternalLink size={12} className="shrink-0" />
                <span className="min-w-0 truncate">{name}</span>
              </button>
            </AppTooltip>
          ) : (
            <span key={`${name}-${index}`} className="itdb-vendor-pill">
              {name}
            </span>
          );
        })}
      </div>
    );
  }
  if (columnKey === 'invoice') {
    const entries = String(row.invoice ?? '')
      .split(',')
      .map(part => part.trim())
      .filter(Boolean)
      .map(part => {
        const sep = part.indexOf(':');
        const id = Number(sep === -1 ? part : part.slice(0, sep));
        const number = sep === -1 ? part : part.slice(sep + 1);
        return { id, number: number || String(id) };
      })
      .filter(entry => Number.isFinite(entry.id) && entry.id > 0);
    if (entries.length === 0) return '-';
    return (
      <div className="itdb-invoice-file-list">
        {entries.map(entry =>
          canOpenRecordEditor('/assets/invoices') ? (
            <AppTooltip key={entry.id} label={`在新窗口编辑单据 ${entry.id}`}>
              <button
                type="button"
                className="itdb-invoice-file-pill"
                onClick={() => void openRecordEditor('/assets/invoices', entry.id)}
              >
                <ExternalLink size={12} className="shrink-0" />
                <span className="min-w-0 truncate">{entry.number}</span>
              </button>
            </AppTooltip>
          ) : (
            <span key={entry.id} className="itdb-invoice-file-pill">
              <span className="min-w-0 truncate">{entry.number}</span>
            </span>
          )
        )}
      </div>
    );
  }
  if (columnKey === 'population') {
    const count = Number(row.population ?? 0);
    return (
      <span className={`itdb-rack-population-badge ${count > 0 ? 'is-linked' : 'is-empty'}`}>
        {count}
      </span>
    );
  }
  if (columnKey === 'usize' || columnKey === 'depth') {
    const value = formatValue(row[columnKey], columnKey);
    if (!value || value === '-') return '-';
    return columnKey === 'usize' ? `${value}U` : `${value}mm`;
  }
  if (columnKey === 'occupation') {
    const usedRaw = Number(row.occupation ?? 0);
    const totalRaw = Number(row.usize ?? 0);
    const used = Number.isFinite(usedRaw) && usedRaw > 0 ? usedRaw : 0;
    const total = Number.isFinite(totalRaw) && totalRaw > 0 ? totalRaw : 0;
    if (!total) return '-';
    const percent = Math.min(100, Math.max(0, (used / total) * 100));
    return (
      <AppTooltip label={`${used}U 在用 / 总计 ${total}U`}>
        <span className="itdb-rack-usage">
          <span>
            <i style={{ width: `${percent}%` }} />
          </span>
        </span>
      </AppTooltip>
    );
  }
  if (columnKey === 'installedon') {
    const entries = parseSoftwareInstalledEntries(row[columnKey]);
    if (entries.length === 0)
      return (
        <div className="w-full text-center" style={{ textAlign: 'center' }}>
          -
        </div>
      );
    return (
      <div className="itdb-invoice-file-list">
        {entries.map(entry => (
          <span key={`${entry.index}-${entry.id ?? 'na'}`} className="itdb-invoice-file-row">
            {entry.id ? (
              <AppTooltip label={`硬件编号 ${entry.id}`}>
                <span className="itdb-invoice-file-index">{entry.id}</span>
              </AppTooltip>
            ) : null}
            {entry.id ? (
              canOpenRecordEditor('/assets/hardware') ? (
                <AppTooltip label={`在新窗口编辑硬件 ${entry.id}`}>
                  <button
                    type="button"
                    className="itdb-installed-pill"
                    onClick={() => void openRecordEditor('/assets/hardware', Number(entry.id))}
                  >
                    <ExternalLink size={12} className="shrink-0" />
                    <span className="min-w-0 truncate">{entry.text}</span>
                  </button>
                </AppTooltip>
              ) : (
                <span className="itdb-installed-pill">
                  <span className="min-w-0 truncate">{entry.text}</span>
                </span>
              )
            ) : (
              <span className="itdb-invoice-file-plain">{entry.text}</span>
            )}
          </span>
        ))}
      </div>
    );
  }
  if (columnKey === 'type' && resourceKey === 'agents') {
    const badges = parseAgentTypeBadges(row[columnKey]);
    if (badges.length === 0) return '-';
    return (
      <div className="itdb-agent-type-list">
        {badges.map(badge => (
          <span key={badge.key} className={`itdb-agent-type-badge is-${badge.tone}`}>
            {badge.label}
          </span>
        ))}
      </div>
    );
  }
  if (columnKey === 'contacts') {
    const contacts = getAgentContactEntries(row[columnKey]);
    if (contacts.length === 0) return '-';
    return (
      <div className="itdb-agent-contact-list">
        {contacts.map((fields, index) => (
          <div
            key={`${index}-${fields.map(field => field.value).join('/')}`}
            className="itdb-agent-contact-row"
          >
            {fields.map(field => (
              <AppTooltip key={field.key} label={field.label}>
                <span className="itdb-agent-contact-field">{field.value}</span>
              </AppTooltip>
            ))}
          </div>
        ))}
      </div>
    );
  }
  if (columnKey === 'fname') {
    const name = String(row.fname ?? '').trim();
    const fileId = Number(row.id ?? 0);
    if (!name || !fileId) return '-';
    const ext = name.includes('.') ? name.slice(name.lastIndexOf('.')) : '';
    const title = String(row.title ?? '').trim();
    return (
      <AppTooltip label={`下载当前文件 ${name}`}>
        <button
          type="button"
          className="itdb-invoice-file-pill"
          onClick={() =>
            void downloadStoredFile(`/api/files/${fileId}/download`, `${title || name}${ext}`)
          }
        >
          <Download size={12} className="shrink-0" />
          <span className="min-w-0 truncate">{name}</span>
        </button>
      </AppTooltip>
    );
  }
  if (columnKey === 'links') {
    const count = Number(row.links ?? 0);
    if (!Number.isFinite(count)) return '-';
    return (
      <span className={`itdb-file-links-badge ${count > 0 ? 'is-linked' : 'is-empty'}`}>
        {count}
      </span>
    );
  }
  if (columnKey === 'floorplanfn') {
    const name = String(row.floorplanfn ?? '').trim();
    const locationId = Number(row.id ?? 0);
    if (!name || !locationId) return '-';
    return (
      <AppTooltip label="在新窗口预览平面图">
        <button
          type="button"
          className="itdb-invoice-file-pill"
          onClick={() => void previewStoredFile(`/api/locations/${locationId}/floorplan`)}
        >
          <ExternalLink size={12} className="shrink-0" />
          <span className="min-w-0 truncate">{name}</span>
        </button>
      </AppTooltip>
    );
  }
  if (columnKey === 'areaname') {
    const areas = String(row[columnKey] ?? '')
      .split(',')
      .map(value => value.trim())
      .filter(Boolean);
    if (areas.length === 0) return '-';
    return (
      <div className="itdb-location-area-list">
        {areas.map((area, index) => (
          <span key={`${index}-${area}`} className="itdb-location-area-pill">
            {area}
          </span>
        ))}
      </div>
    );
  }
  if (columnKey === 'parentid') {
    const value = formatValue(row[columnKey], columnKey);
    if (!value || value === '-') return '-';
    return <span className="itdb-resource-parent-badge">{value}</span>;
  }
  if (columnKey === 'files') {
    const entries = parseInvoiceFileEntries(row[columnKey]);
    if (entries.length === 0) return '-';
    return (
      <div className="itdb-invoice-file-list">
        {entries.map(entry => (
          <span key={`${entry.index}-${entry.id ?? 'na'}`} className="itdb-invoice-file-row">
            <AppTooltip
              label={
                entry.id
                  ? `文件编号 ${entry.id} · 点击新窗口预览`
                  : `第 ${entry.index} 个文件（无编号）`
              }
            >
              <span className="itdb-invoice-file-index">{entry.id ?? entry.index}</span>
            </AppTooltip>
            {entry.id ? (
              <AppTooltip label={entry.title || entry.text}>
                <button
                  type="button"
                  className="itdb-invoice-file-pill"
                  onClick={() => void previewStoredFile(`/api/files/${entry.id}/download`)}
                >
                  <ExternalLink size={12} className="shrink-0" />
                  <span className="min-w-0 truncate">{entry.text}</span>
                </button>
              </AppTooltip>
            ) : (
              <span className="itdb-invoice-file-plain">{entry.text}</span>
            )}
          </span>
        ))}
      </div>
    );
  }
  return formatValue(row[columnKey], columnKey);
}
