import { ChevronDown, ChevronUp, ChevronsUpDown, Search } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { AppTooltip } from '@/components/app-tooltip';
import { canOpenRecordEditor } from '@/features/assets/record-links';
import { getItemIDBadgeStyle } from '@/features/assets/resource-helpers';

type Row = Record<string, unknown>;

const COLUMNS: Array<{ key: string; label: string }> = [
  { key: 'id', label: '编号' },
  { key: 'itemtype', label: '类型' },
  { key: 'manufacturer', label: '厂商' },
  { key: 'model', label: '型号' },
  { key: 'sn', label: '设备序列号' },
  { key: 'label', label: '标签' },
];

const NO_STATUS_LOOKUPS = new Map<string, string>();

type HardwarePanelProps = {
  search: string;
  setSearch: (value: string) => void;
  items: Row[];
  sort: { key: string; direction: 'asc' | 'desc' };
  onSortChange: (key: string) => void;
  selectedIds: number[];
  allChecked: boolean;
  toggleAll: () => void;
  toggleOne: (id: number) => void;
  loading: boolean;
  onPreview: () => void;
  previewing: boolean;
  onPrint?: () => void;
};

export function HardwarePanel(props: HardwarePanelProps) {
  const total = props.items.length;
  const cellText = (value: unknown) => String(value ?? '').trim() || '-';
  return (
    <section
      className="itdb-card-hover flex flex-col rounded-xl p-5"
      style={{
        background: 'var(--itdb-card)',
        border: '1px solid var(--itdb-border)',
        boxShadow: 'var(--shadow-card)',
      }}
    >
      <div className="flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-[var(--itdb-border)] pb-3">
        <h3 className="text-base font-semibold text-[var(--itdb-text)]">选择硬件</h3>
        <span className="text-xs text-[var(--itdb-text-muted)]">
          共 {total} 台 · 已选 {props.selectedIds.length} 台
        </span>
      </div>
      <label className="relative mt-3 shrink-0">
        <Search
          className="absolute left-3 top-1/2 -translate-y-1/2 text-[var(--itdb-text-muted)]"
          size={15}
        />
        <Input
          value={props.search}
          onChange={event => props.setSearch(event.target.value)}
          placeholder="搜索编号 / 类型 / 厂商 / 型号 / 设备序列号 / 标签"
          className="h-9 pl-8"
        />
      </label>
      <div className="itdb-resource-data-panel mt-3 h-96 overflow-hidden rounded-lg border border-[var(--itdb-border)]">
        <div className="itdb-hidden-scrollbar h-full overflow-y-auto">
          <table className="w-full text-center text-sm">
            <thead className="sticky top-0 z-10 text-center text-[var(--itdb-text-muted)]">
              <tr>
                <th className="w-12 px-3 py-3">
                  <input
                    type="checkbox"
                    className="itdb-check-box"
                    checked={props.allChecked}
                    onChange={props.toggleAll}
                    aria-label="全选硬件"
                  />
                </th>
                {COLUMNS.map(column => {
                  const active = props.sort.key === column.key;
                  return (
                    <th
                      key={column.key}
                      className="whitespace-nowrap px-3 py-3 text-center font-medium"
                    >
                      <AppTooltip label={column.label}>
                        <button
                          type="button"
                          className="inline-flex items-center gap-1"
                          onClick={() => props.onSortChange(column.key)}
                        >
                          {column.label}
                          {active ? (
                            props.sort.direction === 'asc' ? (
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
                  );
                })}
              </tr>
            </thead>
            <tbody>
              {props.loading ? (
                <tr>
                  <td
                    colSpan={7}
                    className="py-8 text-center text-sm text-[var(--itdb-text-muted)]"
                  >
                    正在加载硬件...
                  </td>
                </tr>
              ) : total === 0 ? (
                <tr>
                  <td
                    colSpan={7}
                    className="py-8 text-center text-sm text-[var(--itdb-text-muted)]"
                  >
                    没有匹配的硬件
                  </td>
                </tr>
              ) : (
                props.items.map(item => {
                  const id = Number(item.id);
                  const checked = props.selectedIds.includes(id);
                  const statusText = String(item.statusdesc ?? '').trim();
                  return (
                    <tr
                      key={id}
                      className="cursor-pointer border-b border-[var(--itdb-border)]/70 transition-colors hover:bg-[var(--itdb-control-bg-soft)]"
                      onClick={() => props.toggleOne(id)}
                    >
                      <td className="px-3 py-2" onClick={event => event.stopPropagation()}>
                        <input
                          type="checkbox"
                          className="itdb-check-box"
                          checked={checked}
                          onChange={() => props.toggleOne(id)}
                        />
                      </td>
                      <td className="whitespace-nowrap px-3 py-2">
                        {canOpenRecordEditor('/assets/hardware') ? (
                          <AppTooltip
                            label={
                              statusText
                                ? `${statusText}，点击后在新窗口编辑硬件 ${id}`
                                : `点击后在新窗口编辑硬件 ${id}`
                            }
                          >
                            <a
                              href={`/assets/hardware?edit=${id}`}
                              target="_blank"
                              rel="noopener"
                              onClick={event => {
                                event.preventDefault();
                                event.stopPropagation();
                                window.open(`/assets/hardware?edit=${id}`, '_blank', 'noopener');
                              }}
                              aria-label={`在新窗口编辑硬件 ${id}`}
                              className="inline-flex"
                            >
                              <span
                                className="itdb-resource-id-badge"
                                style={getItemIDBadgeStyle(item, NO_STATUS_LOOKUPS)}
                              >
                                {String(id).padStart(4, '0')}
                              </span>
                            </a>
                          </AppTooltip>
                        ) : (
                          <AppTooltip label={statusText || '-'}>
                            <span
                              className="itdb-resource-id-badge"
                              style={getItemIDBadgeStyle(item, NO_STATUS_LOOKUPS)}
                            >
                              {String(id).padStart(4, '0')}
                            </span>
                          </AppTooltip>
                        )}
                      </td>
                      <td className="whitespace-nowrap px-3 py-2 text-[var(--itdb-text)]">
                        {cellText(item.itemtype)}
                      </td>
                      <td className="max-w-40 truncate whitespace-nowrap px-3 py-2 text-[var(--itdb-text)]">
                        {cellText(item.manufacturer)}
                      </td>
                      <td className="max-w-44 truncate whitespace-nowrap px-3 py-2 text-[var(--itdb-text)]">
                        {cellText(item.model)}
                      </td>
                      <td
                        className="max-w-64 truncate whitespace-nowrap px-3 py-2 font-mono text-[var(--itdb-text)]"
                        title={cellText(item.sn)}
                      >
                        {cellText(item.sn)}
                      </td>
                      <td className="max-w-40 truncate whitespace-nowrap px-3 py-2 text-[var(--itdb-text)]">
                        {cellText(item.label)}
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </div>
      </div>
      <div className="mt-3 flex shrink-0 flex-wrap items-center justify-end gap-2">
        <Button
          className="itdb-action-button itdb-dict-export-button"
          disabled={props.previewing}
          onClick={props.onPreview}
        >
          {props.previewing ? '生成中...' : '生成标签预览'}
        </Button>
        {props.onPrint ? (
          <Button
            className="itdb-action-button"
            style={{
              borderColor: 'rgba(59,130,246,0.38)',
              background: 'rgba(59,130,246,0.1)',
              color: 'var(--itdb-accent-text)',
            }}
            onClick={props.onPrint}
          >
            打印 / 导出 PDF
          </Button>
        ) : null}
      </div>
      <ol className="mt-3 list-decimal space-y-1 border-t border-[var(--itdb-border)] pl-5 pt-3 text-xs text-[var(--itdb-text-muted)]">
        <li>从上方表格勾选需要打印标签的硬件</li>
        <li>在下方「标签属性」中设置参数（手工或套用预设）</li>
        <li>点击「生成标签预览」确认数据</li>
        <li>后续导出 PDF 时，打印设置建议关闭自动缩放</li>
      </ol>
    </section>
  );
}
export type { HardwarePanelProps };
