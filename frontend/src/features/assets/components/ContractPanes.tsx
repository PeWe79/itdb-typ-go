import { useQuery } from '@tanstack/react-query';
import { Edit3, Search, Trash2 } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { api } from '@/lib/auth';
import { DatePicker } from './DatePicker';
import { lookupOptions, formatValue } from '../resource-helpers';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

export function RenewalEditor({
  rows,
  lookups,
  onChange,
}: {
  rows: Row[];
  lookups: Lookups | undefined;
  onChange: (rows: Row[]) => void;
}) {
  const actualRows = rows.length ? rows : [{}];
  const update = (index: number, key: string, value: string) =>
    onChange(
      actualRows.map((row, rowIndex) => (rowIndex === index ? { ...row, [key]: value } : row))
    );
  const userOptions = lookupOptions(lookups, 'users');
  return (
    <section className="mx-auto flex min-h-0 w-full max-w-6xl flex-1 flex-col rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-5 pb-5 pt-3">
      <h3 className="mb-4 shrink-0 border-b border-[var(--itdb-border)] pb-2 text-base font-semibold text-[var(--itdb-text)]">
        备件
      </h3>
      <div className="itdb-hidden-scrollbar itdb-resource-data-panel min-h-0 flex-1 overflow-auto rounded-lg border border-[var(--itdb-border)]">
        <table className="w-full table-fixed text-sm">
          <thead className="sticky top-0 z-10 text-[var(--itdb-text-muted)]">
            <tr>
              <th className="whitespace-nowrap px-3 py-2.5 text-center font-medium">到期前</th>
              <th className="whitespace-nowrap px-3 py-2.5 text-center font-medium">到期后</th>
              <th className="whitespace-nowrap px-3 py-2.5 text-center font-medium">生效日期</th>
              <th className="whitespace-nowrap px-3 py-2.5 text-center font-medium">备注</th>
              <th className="whitespace-nowrap px-3 py-2.5 text-center font-medium">录入日期</th>
              <th className="whitespace-nowrap px-3 py-2.5 text-center font-medium">录入人</th>
              <th className="w-20 px-3 py-2.5 text-center font-medium">操作</th>
            </tr>
          </thead>
          <tbody>
            {actualRows.map((row, index) => (
              <tr
                key={index}
                className="border-t border-[var(--itdb-border)]/70 hover:bg-[var(--itdb-control-bg-soft)]"
              >
                <td className="px-2 py-2">
                  <DatePicker
                    value={String(row.endDateBefore ?? '')}
                    onChange={value => update(index, 'endDateBefore', value)}
                    panelMinWidth={320}
                  />
                </td>
                <td className="px-2 py-2">
                  <DatePicker
                    value={String(row.endDateAfter ?? '')}
                    onChange={value => update(index, 'endDateAfter', value)}
                    panelMinWidth={320}
                  />
                </td>
                <td className="px-2 py-2">
                  <DatePicker
                    value={String(row.effectiveDate ?? '')}
                    onChange={value => update(index, 'effectiveDate', value)}
                    panelMinWidth={320}
                  />
                </td>
                <td className="px-2 py-2">
                  <Input
                    value={String(row.notes ?? '')}
                    onChange={event => update(index, 'notes', event.target.value)}
                  />
                </td>
                <td className="px-2 py-2">
                  <DatePicker
                    value={String(row.enteredDate ?? '')}
                    onChange={value => update(index, 'enteredDate', value)}
                    panelMinWidth={320}
                  />
                </td>
                <td className="px-2 py-2">
                  <Select
                    value={String(row.enteredBy ?? '') || '__placeholder__'}
                    onValueChange={next =>
                      update(index, 'enteredBy', next === '__placeholder__' ? '' : next)
                    }
                  >
                    <SelectTrigger aria-label="录入人">
                      <SelectValue placeholder="请选择" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="__placeholder__">请选择</SelectItem>
                      {userOptions.map(option => (
                        <SelectItem key={option.value} value={String(option.value)}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </td>
                <td className="px-3 py-2.5 text-center">
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    className="itdb-danger-sm-btn h-8 text-xs"
                    onClick={() => onChange(actualRows.filter((_, rowIndex) => rowIndex !== index))}
                  >
                    删除
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Button
        type="button"
        variant="outline"
        className="mt-3 w-full shrink-0"
        onClick={() => onChange([...actualRows, {}])}
      >
        新增备件
      </Button>
    </section>
  );
}

// 事件待定变更：页签内的增删改先记录在本地，点“保存修改”后由编辑器统一提交
export type ContractEventOp = {
  action: 'create' | 'update' | 'delete';
  key: number;
  eventId?: number;
  data: { siblingId: number; startDate: string; endDate: string; description: string };
};

type ContractEventDisplayRow = {
  key: number;
  id: number;
  siblingId: number;
  startDate: string;
  endDate: string;
  description: string;
};

export function ContractEventsPane({
  contractId,
  onPendingEventsChange,
}: {
  contractId?: unknown;
  onPendingEventsChange?: (ops: ContractEventOp[]) => void;
}) {
  const [keyword, setKeyword] = useState('');
  const [siblingId, setSiblingId] = useState('');
  const [startDate, setStartDate] = useState('');
  const [endDate, setEndDate] = useState('');
  const [description, setDescription] = useState('');
  const [editingKey, setEditingKey] = useState<number | null>(null);
  const [pendingOps, setPendingOps] = useState<ContractEventOp[]>([]);
  const tempKeyRef = useRef(-1);
  const id = Number(contractId);
  const saved = Number.isFinite(id) && id > 0;
  const events = useQuery({
    queryKey: ['itdb', 'contracts', id, 'events'],
    enabled: saved,
    queryFn: () => api<Row[]>(`/api/contracts/${id}/events`),
  });
  const nextEventIdQuery = useQuery({
    queryKey: ['itdb', 'contracts', 'next-event-id'],
    enabled: saved,
    staleTime: 0,
    queryFn: () => api<{ nextId: number }>('/api/contracts/next-event-id'),
  });
  const nextEventBaseId = Number(nextEventIdQuery.data?.nextId) || 1;
  useEffect(() => {
    onPendingEventsChange?.(pendingOps);
  }, [onPendingEventsChange, pendingOps]);
  const resetForm = () => {
    setSiblingId('');
    setStartDate('');
    setEndDate('');
    setDescription('');
    setEditingKey(null);
  };
  const toInputDate = (value: unknown) => {
    const display = formatValue(value, 'startdate');
    return display === '-' ? '' : display;
  };
  const displayRows: ContractEventDisplayRow[] = (() => {
    const baseRows = Array.isArray(events.data) ? events.data : [];
    const opByKey = new Map(pendingOps.map(op => [op.key, op]));
    const rows: ContractEventDisplayRow[] = [];
    for (const row of baseRows) {
      const eventId = Number(row.id);
      const op = opByKey.get(eventId);
      if (op?.action === 'delete') continue;
      if (op?.action === 'update') {
        rows.push({ key: eventId, id: eventId, ...op.data });
        continue;
      }
      rows.push({
        key: eventId,
        id: eventId,
        siblingId: Number(row.siblingid) > 0 ? Number(row.siblingid) : 0,
        startDate: toInputDate(row.startdate),
        endDate: toInputDate(row.enddate),
        description: String(row.description ?? ''),
      });
    }
    let createIndex = 0;
    for (const op of pendingOps) {
      if (op.action === 'create') {
        rows.push({ key: op.key, id: nextEventBaseId + createIndex, ...op.data });
        createIndex += 1;
      }
    }
    return rows;
  })();
  const text = keyword.trim().toLowerCase();
  const filteredRows = text
    ? displayRows.filter(row =>
        [String(row.id), String(row.siblingId), row.startDate, row.endDate, row.description]
          .join(' ')
          .toLowerCase()
          .includes(text)
      )
    : displayRows;
  const startEdit = (row: ContractEventDisplayRow) => {
    setEditingKey(row.key);
    setSiblingId(row.siblingId > 0 ? String(row.siblingId) : '');
    setStartDate(row.startDate);
    setEndDate(row.endDate);
    setDescription(row.description);
  };
  const submitPending = () => {
    const missing: string[] = [];
    if (!startDate.trim() || !endDate.trim()) missing.push('开始日期、结束日期');
    if (!description.trim()) missing.push('事件描述');
    if (missing.length > 0) {
      toast.error(`请完善必填项：${missing.join('、')}`);
      return;
    }
    if (endDate < startDate) {
      toast.error('合同事件中，结束日期不能早于开始日期');
      return;
    }
    const data = {
      siblingId: Number(siblingId) > 0 ? Number(siblingId) : 0,
      startDate,
      endDate,
      description,
    };
    setPendingOps(ops => {
      if (editingKey !== null) {
        const existing = ops.find(op => op.key === editingKey);
        if (existing?.action === 'create') {
          return ops.map(op => (op.key === editingKey ? { ...op, data } : op));
        }
        const rest = ops.filter(op => op.key !== editingKey);
        return [...rest, { action: 'update', key: editingKey, eventId: editingKey, data }];
      }
      const key = tempKeyRef.current;
      tempKeyRef.current -= 1;
      return [...ops, { action: 'create', key, data }];
    });
    toast.success('事件变更将在保存后生效');
    resetForm();
  };
  const removeRow = (row: ContractEventDisplayRow) => {
    if (row.key < 0) {
      setPendingOps(ops => ops.filter(op => op.key !== row.key));
    } else {
      setPendingOps(ops => [
        ...ops.filter(op => op.key !== row.key),
        {
          action: 'delete',
          key: row.key,
          eventId: row.key,
          data: { siblingId: 0, startDate: '', endDate: '', description: '' },
        },
      ]);
    }
    if (editingKey === row.key) resetForm();
    toast.success('事件变更将在保存后生效');
  };
  return (
    <section className="mx-auto flex min-h-0 w-full max-w-6xl flex-1 flex-col rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-5 pb-5 pt-3">
      <div className="flex shrink-0 flex-wrap items-center gap-3 border-b border-[var(--itdb-border)] pb-3">
        <h3 className="text-base font-semibold text-[var(--itdb-text)]">事件历史</h3>
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
        <span className="text-sm text-[var(--itdb-text-muted)]">共 {filteredRows.length} 条</span>
      </div>
      <div className="itdb-hidden-scrollbar itdb-resource-data-panel mt-4 min-h-0 flex-1 overflow-auto rounded-lg border border-[var(--itdb-border)]">
        <table className="min-w-full text-sm">
          <thead className="sticky top-0 z-10 bg-[var(--itdb-control-bg-soft)] text-[var(--itdb-text-muted)]">
            <tr>
              <th className="w-24 px-3 py-2.5 text-center font-medium">编号</th>
              <th className="whitespace-nowrap px-3 py-2.5 text-center font-medium">同组编号</th>
              <th className="whitespace-nowrap px-3 py-2.5 text-center font-medium">开始日期</th>
              <th className="whitespace-nowrap px-3 py-2.5 text-center font-medium">结束日期</th>
              <th className="whitespace-nowrap px-3 py-2.5 text-center font-medium">事件描述</th>
              {saved ? <th className="w-28 px-3 py-2.5 text-center font-medium">操作</th> : null}
            </tr>
          </thead>
          <tbody>
            {events.isLoading ? (
              <tr>
                <td
                  colSpan={6}
                  className="px-4 py-8 text-center text-sm text-[var(--itdb-text-muted)]"
                >
                  加载中
                </td>
              </tr>
            ) : (
              <>
                {filteredRows.map(row => (
                  <tr
                    key={row.key}
                    className="border-t border-[var(--itdb-border)]/70 hover:bg-[var(--itdb-control-bg-soft)]"
                  >
                    <td className="px-3 py-2.5 text-center">
                      <span className="itdb-relation-id">{row.id}</span>
                    </td>
                    <td className="px-3 py-2.5 text-center">
                      {row.siblingId > 0 ? (
                        <span className="itdb-sibling-badge">{row.siblingId}</span>
                      ) : (
                        '-'
                      )}
                    </td>
                    <td className="whitespace-nowrap px-3 py-2.5 text-center text-[var(--itdb-text)]">
                      {row.startDate || '-'}
                    </td>
                    <td className="whitespace-nowrap px-3 py-2.5 text-center text-[var(--itdb-text)]">
                      {row.endDate || '-'}
                    </td>
                    <td className="px-3 py-2.5 text-center text-[var(--itdb-text)]">
                      {row.description.trim() || '-'}
                    </td>
                    {saved ? (
                      <td className="px-3 py-2.5 text-center">
                        <div className="flex justify-center gap-1.5">
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
                            aria-label={`编辑事件 ${String(row.id ?? '新')}`}
                          >
                            <Edit3 size={13} />
                          </Button>
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
                            aria-label={`删除事件 ${String(row.id ?? '新')}`}
                          >
                            <Trash2 size={13} />
                          </Button>
                        </div>
                      </td>
                    ) : null}
                  </tr>
                ))}
                {filteredRows.length === 0 ? (
                  <tr>
                    <td
                      colSpan={6}
                      className="px-4 py-8 text-center text-sm text-[var(--itdb-text-muted)]"
                    >
                      {saved ? '暂无事件记录' : '新增合同并保存后，可维护事件历史。'}
                    </td>
                  </tr>
                ) : null}
              </>
            )}
          </tbody>
        </table>
      </div>
      {saved ? (
        <div className="mt-4 grid shrink-0 gap-x-6 gap-y-3 border-t border-[var(--itdb-border)] pt-4 md:grid-cols-2">
          <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
            <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
              <AppTooltip
                label="填写相同编号可把相关事件归为一组，便于按组筛选查看"
                placement="top"
                align="end"
                wrap
              >
                <span className="cursor-help">同组编号</span>
              </AppTooltip>
            </span>
            <Input
              type="number"
              min="0"
              value={siblingId}
              onChange={event => setSiblingId(event.target.value)}
              aria-label="同组编号"
            />
          </div>
          <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
            <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
              事件描述<span className="ml-1 text-red-500">*</span>
            </span>
            <Input
              value={description}
              onChange={event => setDescription(event.target.value)}
              aria-label="事件描述"
            />
          </div>
          <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
            <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
              开始日期<span className="ml-1 text-red-500">*</span>
            </span>
            <DatePicker value={startDate} onChange={setStartDate} />
          </div>
          <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
            <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
              结束日期<span className="ml-1 text-red-500">*</span>
            </span>
            <DatePicker value={endDate} onChange={setEndDate} />
          </div>
          <div className="flex flex-wrap items-center gap-2 md:col-span-2">
            <Button
              type="button"
              className="itdb-action-button disabled:pointer-events-auto disabled:cursor-not-allowed"
              onClick={submitPending}
            >
              {editingKey !== null ? '保存事件修改' : '新增事件'}
            </Button>
            {editingKey !== null ? (
              <Button type="button" variant="outline" onClick={resetForm}>
                取消编辑
              </Button>
            ) : null}
            <span className="text-sm text-[var(--itdb-text-muted)]">
              事件用于记录合同生命周期中的重要节点，同组编号可将相关事件分为一组，变更在保存合同后生效
            </span>
          </div>
        </div>
      ) : null}
    </section>
  );
}
