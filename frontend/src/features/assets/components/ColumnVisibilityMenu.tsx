import { Check, Columns3, GripVertical, RotateCcw } from 'lucide-react';
import { Fragment, useEffect, useRef, useState, type DragEvent } from 'react';
import { toast } from 'sonner';
import { Button } from '@/components/ui/button';

export type ResourceColumn = { key: string; label: string; tooltip?: string };

export type ColumnDisplayState = {
  order: string[];
  hidden: string[];
};

// DropIndicator 拖拽排序时的插入位置横线指示
function DropIndicator() {
  return (
    <li
      aria-hidden
      className="mx-2 my-0.5 list-none rounded-full bg-[var(--itdb-accent)] shadow-[0_0_6px_rgba(59,130,246,0.65)]"
      style={{ height: 2 }}
    />
  );
}

/* 列显示偏好存储于 localStorage，随资源维度独立保存 */
function storageKey(resourceKey: string) {
  return `itdb:columns:${resourceKey}`;
}

// loadColumnDisplayState 读取资源列显示偏好，缺失、损坏或与当前列集不匹配时回退为全部列按配置顺序
export function loadColumnDisplayState(
  resourceKey: string,
  columns: ResourceColumn[]
): ColumnDisplayState {
  const defaults: ColumnDisplayState = { order: columns.map(column => column.key), hidden: [] };
  if (typeof window === 'undefined') return defaults;
  try {
    const raw = window.localStorage.getItem(storageKey(resourceKey));
    if (!raw) return defaults;
    const parsed = JSON.parse(raw) as Partial<ColumnDisplayState>;
    const known = new Set(columns.map(column => column.key));
    const order = (parsed.order ?? []).filter(key => known.has(key));
    columns.forEach(column => {
      if (!order.includes(column.key)) order.push(column.key);
    });
    const hidden = [...new Set(parsed.hidden ?? [])].filter(key => known.has(key));
    return { order, hidden };
  } catch {
    return defaults;
  }
}

// saveColumnDisplayState 保存资源列显示偏好，存储不可用时静默降级为会话内生效
export function saveColumnDisplayState(resourceKey: string, state: ColumnDisplayState) {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(storageKey(resourceKey), JSON.stringify(state));
  } catch {
    return;
  }
}

// ColumnVisibilityMenu 列显示选择面板：勾选控制列显隐，拖拽调整列顺序，重置恢复默认
export function ColumnVisibilityMenu({
  resourceKey,
  columns,
  state,
  onChange,
}: {
  resourceKey: string;
  columns: ResourceColumn[];
  state: ColumnDisplayState;
  onChange: (next: ColumnDisplayState) => void;
}) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const dragIndexRef = useRef<number | null>(null);
  const [dragIndex, setDragIndex] = useState<number | null>(null);
  const [dropIndex, setDropIndex] = useState<number | null>(null);

  useEffect(() => {
    if (!open) return;
    const handlePointerDown = (event: MouseEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setOpen(false);
    };
    window.addEventListener('mousedown', handlePointerDown);
    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('mousedown', handlePointerDown);
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [open]);

  const visibleCount = columns.length - state.hidden.length;
  const isDefault =
    state.hidden.length === 0 &&
    state.order.length === columns.length &&
    state.order.every((key, index) => key === columns[index]?.key);

  function toggleColumn(key: string) {
    const isHidden = state.hidden.includes(key);
    if (!isHidden && visibleCount <= 1) {
      toast.error('至少保留一列显示');
      return;
    }
    onChange({
      ...state,
      hidden: isHidden ? state.hidden.filter(item => item !== key) : [...state.hidden, key],
    });
  }

  function handleDragOver(event: DragEvent<HTMLLIElement>, index: number) {
    event.preventDefault();
    const rect = event.currentTarget.getBoundingClientRect();
    const after = event.clientY > rect.top + rect.height / 2;
    setDropIndex(after ? index + 1 : index);
  }

  function handleDrop() {
    const from = dragIndexRef.current;
    const to = dropIndex;
    clearDragState();
    if (from === null || to === null) return;
    const target = from < to ? to - 1 : to;
    if (target === from) return;
    const order = [...state.order];
    const [moved] = order.splice(from, 1);
    order.splice(target, 0, moved);
    onChange({ ...state, order });
  }

  function clearDragState() {
    dragIndexRef.current = null;
    setDragIndex(null);
    setDropIndex(null);
  }

  return (
    <div ref={containerRef} className="relative">
      <Button
        type="button"
        variant="outline"
        className="itdb-action-button"
        style={{
          borderColor: 'var(--itdb-border)',
          background: 'var(--itdb-control-bg)',
          color: 'var(--itdb-text-muted)',
        }}
        onClick={() => setOpen(previous => !previous)}
        aria-label="列显示设置"
      >
        <Columns3 size={16} />
        列显示 {visibleCount}/{columns.length}
      </Button>
      {open && (
        <div
          className="absolute right-0 top-11 z-50 w-64 overflow-hidden rounded-xl border border-[var(--itdb-border)]"
          style={{ background: 'var(--itdb-card)', boxShadow: 'var(--shadow-card)' }}
        >
          <div className="flex items-center justify-between border-b border-[var(--itdb-border)] px-3 py-2.5">
            <span className="text-xs font-semibold text-[var(--itdb-text)]">列显示与顺序</span>
            <button
              type="button"
              className="flex items-center gap-1 text-xs text-[var(--itdb-text-muted)] transition-colors hover:text-[var(--itdb-text)] disabled:cursor-not-allowed disabled:opacity-50"
              onClick={() => onChange({ order: columns.map(column => column.key), hidden: [] })}
              disabled={isDefault}
            >
              <RotateCcw size={12} />
              重置
            </button>
          </div>
          <ul
            className="itdb-hidden-scrollbar max-h-72 overflow-y-auto p-1.5"
            onDragLeave={event => {
              if (!event.currentTarget.contains(event.relatedTarget as Node)) clearDragState();
            }}
          >
            {state.order.map((key, index) => {
              const column = columns.find(item => item.key === key);
              if (!column) return null;
              const visible = !state.hidden.includes(key);
              return (
                <Fragment key={key}>
                  {dropIndex === index && <DropIndicator />}
                  <li
                    draggable
                    onDragStart={() => {
                      dragIndexRef.current = index;
                      setDragIndex(index);
                    }}
                    onDragOver={event => handleDragOver(event, index)}
                    onDrop={handleDrop}
                    onDragEnd={clearDragState}
                    className={`flex items-center gap-1.5 rounded-lg px-1.5 py-1 transition-colors hover:bg-[var(--itdb-control-bg-soft)] ${dragIndex === index ? 'opacity-40' : ''} ${visible ? '' : 'opacity-55'}`}
                  >
                    <span
                      className="flex h-6 w-5 shrink-0 cursor-grab items-center justify-center text-[var(--itdb-text-muted)] active:cursor-grabbing"
                      title="拖动调整列顺序"
                    >
                      <GripVertical size={14} />
                    </span>
                    <button
                      type="button"
                      className="flex min-w-0 flex-1 items-center gap-2 py-0.5 text-left text-sm text-[var(--itdb-text)]"
                      onClick={() => toggleColumn(key)}
                    >
                      <span
                        className={`flex h-4 w-4 shrink-0 items-center justify-center rounded border ${visible ? 'border-teal-400/70 bg-teal-400/20' : 'border-[var(--itdb-border)]'}`}
                      >
                        {visible ? <Check size={12} /> : null}
                      </span>
                      <span className="min-w-0 truncate">{column.label}</span>
                    </button>
                    <span className="shrink-0 text-xs text-[var(--itdb-text-muted)]">
                      {index + 1}
                    </span>
                  </li>
                </Fragment>
              );
            })}
            {dropIndex === state.order.length && <DropIndicator />}
          </ul>
        </div>
      )}
    </div>
  );
}
