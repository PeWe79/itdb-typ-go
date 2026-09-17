import { Link } from '@tanstack/react-router';
import { canOpenRecordEditor } from '../record-links';
import { Check, ChevronDown, ChevronUp, ExternalLink, Pencil, Trash2, X } from 'lucide-react';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import { AppTooltip } from '@/components/app-tooltip';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { type ResourceField } from '../resource-config';
import { overviewEntryText } from '../resource-helpers';
import { FieldEditor } from './FieldEditor';
import { api, apiBlob } from '@/lib/auth';
import { previewStoredFile } from '@/lib/file-download';
import { useQuery } from '@tanstack/react-query';

type Row = Record<string, unknown>;

export type LocationAreaOp = {
  action: 'create' | 'update' | 'delete';
  areaId?: number;
  areaName: string;
};

export function LocationDataPane({
  fields,
  values,
  lookups,
  onChange,
  locationId,
  floorplanName,
  onPendingAreasChange,
  onAreasSnapshot,
}: {
  fields: ResourceField[];
  values: Row;
  lookups?: Record<string, Row[]>;
  onChange: (field: ResourceField, value: unknown) => void;
  locationId?: unknown;
  floorplanName: string;
  onPendingAreasChange?: (ops: LocationAreaOp[]) => void;
  onAreasSnapshot?: (names: string[]) => void;
}) {
  const [activeSection, setActiveSection] = useState('attribute');
  const sections = [
    { value: 'attribute', label: '地点属性配置' },
    { value: 'areas', label: '区域' },
    { value: 'overview', label: '关联概览' },
  ];
  const [areaName, setAreaName] = useState('');
  const [editingAreaId, setEditingAreaId] = useState<number | null>(null);
  const [editingAreaName, setEditingAreaName] = useState('');
  const [editingAreaOriginal, setEditingAreaOriginal] = useState('');
  const [areaList, setAreaList] = useState<Row[]>([]);
  const [pendingAreas, setPendingAreas] = useState<LocationAreaOp[]>([]);
  const [previewUrl, setPreviewUrl] = useState('');
  const [overviewTab, setOverviewTab] = useState<'items' | 'racks'>('items');
  const tempIdRef = useRef(-1);
  const id = Number(locationId);
  const saved = Number.isFinite(id) && id > 0;
  const areas = useQuery({
    queryKey: ['itdb', 'locations', id, 'areas'],
    enabled: saved,
    queryFn: () => api<Row[]>(`/api/locations/${id}/areas`),
  });
  const locationItems = useQuery({
    queryKey: ['itdb', 'locations', id, 'items'],
    enabled: saved,
    queryFn: () => api<Row[]>('/api/items?limit=-1&offset=0'),
  });
  const locationRacks = useQuery({
    queryKey: ['itdb', 'locations', id, 'racks'],
    enabled: saved,
    queryFn: () => api<Row[]>('/api/racks'),
  });
  useEffect(() => {
    setAreaList(Array.isArray(areas.data) ? areas.data : []);
  }, [areas.data]);
  useEffect(() => {
    onPendingAreasChange?.(pendingAreas);
  }, [onPendingAreasChange, pendingAreas]);
  useEffect(() => {
    onAreasSnapshot?.(areaList.map(row => String(row.areaname ?? '').trim()).filter(Boolean));
  }, [areaList, onAreasSnapshot]);
  const commitAreaEdit = () => {
    if (editingAreaId === null) return;
    const name = editingAreaName.trim();
    if (!name) return;
    if (editingAreaId < 0) {
      setAreaList(list =>
        list.map(row => (Number(row.id) === editingAreaId ? { ...row, areaname: name } : row))
      );
      setPendingAreas(ops =>
        ops.map(op =>
          op.action === 'create' && op.areaId === editingAreaId ? { ...op, areaName: name } : op
        )
      );
    } else {
      setAreaList(list =>
        list.map(row => (Number(row.id) === editingAreaId ? { ...row, areaname: name } : row))
      );
      setPendingAreas(ops => [
        ...ops.filter(op => !(op.areaId === editingAreaId && op.action !== 'delete')),
        { action: 'update', areaId: editingAreaId, areaName: name },
      ]);
    }
    toast.success('区域变更将在保存后生效');
    setEditingAreaId(null);
    setEditingAreaName('');
    setEditingAreaOriginal('');
  };
  const cancelAreaEdit = () => {
    if (editingAreaId === null) return;
    setAreaList(list =>
      list.map(row =>
        Number(row.id) === editingAreaId ? { ...row, areaname: editingAreaOriginal } : row
      )
    );
    setEditingAreaId(null);
    setEditingAreaName('');
    setEditingAreaOriginal('');
  };
  const submitArea = () => {
    const name = areaName.trim();
    if (!name) return;
    const tempId = tempIdRef.current;
    tempIdRef.current -= 1;
    setAreaList(list => [...list, { id: tempId, areaname: name }]);
    setPendingAreas(ops => [...ops, { action: 'create', areaId: tempId, areaName: name }]);
    toast.success('区域变更将在保存后生效');
    setAreaName('');
  };
  const removeArea = (row: Row) => {
    const areaId = Number(row.id);
    if (areaId > 0) {
      const messages: string[] = [];
      const rackCount = racks.filter(rack => Number(rack.locareaid) === areaId).length;
      if (rackCount > 0) {
        messages.push(`该区域已被 ${rackCount} 条机架记录使用，无法删除`);
      }
      const itemCount = items.filter(item => Number(item.locareaid) === areaId).length;
      if (itemCount > 0) {
        messages.push(`该区域已被 ${itemCount} 条硬件记录使用，无法删除`);
      }
      if (messages.length > 0) {
        showErrorToast(messages.join('\n'));
        return;
      }
    }
    setAreaList(list => list.filter(item => Number(item.id) !== areaId));
    setPendingAreas(ops => {
      const rest = ops.filter(op => op.areaId !== areaId);
      if (areaId > 0) return [...rest, { action: 'delete', areaId, areaName: '' }];
      return rest;
    });
    if (editingAreaId === areaId) {
      setEditingAreaId(null);
      setEditingAreaName('');
      setEditingAreaOriginal('');
    }
    toast.success('区域变更将在保存后生效');
  };
  const selectedFile = values.file instanceof File ? values.file : undefined;
  useEffect(() => {
    if (!selectedFile) {
      setPreviewUrl('');
      return undefined;
    }
    const nextUrl = URL.createObjectURL(selectedFile);
    setPreviewUrl(nextUrl);
    return () => URL.revokeObjectURL(nextUrl);
  }, [selectedFile]);
  const [savedFloorplanUrl, setSavedFloorplanUrl] = useState('');
  useEffect(() => {
    if (!saved || !floorplanName) {
      setSavedFloorplanUrl('');
      return undefined;
    }
    let objectUrl = '';
    let cancelled = false;
    apiBlob(`/api/locations/${id}/floorplan`)
      .then(blob => {
        if (cancelled || !blob.size) return;
        objectUrl = URL.createObjectURL(blob);
        setSavedFloorplanUrl(objectUrl);
      })
      .catch(() => setSavedFloorplanUrl(''));
    return () => {
      cancelled = true;
      if (objectUrl) URL.revokeObjectURL(objectUrl);
    };
  }, [floorplanName, id, saved]);
  const mainFields = fields.filter(field => field.key !== 'file');
  const fileField = fields.find(field => field.key === 'file');
  const items = Array.isArray(locationItems.data) ? locationItems.data : [];
  const racks = Array.isArray(locationRacks.data) ? locationRacks.data : [];
  const linkedItems = saved ? items.filter(item => Number(item.locationid) === id) : [];
  const linkedRacks = saved ? racks.filter(rack => Number(rack.locationid) === id) : [];
  const itemsRefById = new Map(
    (lookups?.items_ref ?? []).map(row => [Number(row.id), row] as const)
  );
  const nextAreaIdQuery = useQuery({
    queryKey: ['itdb', 'locations', 'next-area-id'],
    enabled: activeSection === 'areas',
    staleTime: 0,
    queryFn: () => api<{ nextId: number }>('/api/locations/next-area-id'),
  });
  const nextAreaBaseId =
    Number(nextAreaIdQuery.data?.nextId) ||
    Math.max(0, ...(lookups?.locareas ?? []).map(row => Number(row.id) || 0)) + 1;
  const pendingDisplayIds = new Map<number, number>();
  {
    let pendingIndex = 0;
    for (const row of areaList) {
      if (Number(row.id) < 0) {
        pendingDisplayIds.set(Number(row.id), nextAreaBaseId + pendingIndex);
        pendingIndex += 1;
      }
    }
  }
  const areaDisplayId = (area: Row) => {
    const areaId = Number(area.id);
    return areaId < 0 ? (pendingDisplayIds.get(areaId) ?? nextAreaBaseId) : areaId;
  };
  const [areaSortDesc, setAreaSortDesc] = useState(false);
  const sortedAreaList = [...areaList].sort((left, right) =>
    areaSortDesc
      ? areaDisplayId(right) - areaDisplayId(left)
      : areaDisplayId(left) - areaDisplayId(right)
  );
  const previewSource = previewUrl || savedFloorplanUrl;
  const readonlyRow = (label: string, content: ReactNode) => (
    <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
      <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
        {label}
      </span>
      <div className="itdb-location-readonly">{content}</div>
    </div>
  );
  const overviewEntries =
    overviewTab === 'items'
      ? linkedItems.map(item => ({
          id: Number(item.id),
          label: overviewEntryText('items_ref', itemsRefById.get(Number(item.id)), Number(item.id)),
          path: '/assets/hardware' as const,
          tip: `编辑硬件 ${Number(item.id)}`,
        }))
      : linkedRacks.map(rack => ({
          id: Number(rack.id),
          label: String(rack.label ?? '').trim() || '-',
          path: '/assets/racks' as const,
          tip: `编辑机架 ${Number(rack.id)}`,
        }));

  return (
    <div className="flex min-h-0 flex-1 flex-col gap-5">
      <nav
        className="itdb-hidden-scrollbar flex shrink-0 overflow-x-auto rounded-xl border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] p-1.5"
        aria-label="地点数据配置导航"
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
            地点属性配置
          </h3>
          <div className="itdb-hidden-scrollbar min-h-0 flex-1 space-y-3 overflow-y-auto px-1 pt-1">
            {mainFields.map(field => (
              <FieldEditor
                key={field.key}
                field={field}
                value={values[field.key]}
                options={field.options ?? []}
                onChange={value => onChange(field, value)}
                compactHorizontal
              />
            ))}
            {readonlyRow(
              '文件名称',
              saved && floorplanName ? (
                <AppTooltip label="在新窗口预览平面图">
                  <button
                    type="button"
                    className="itdb-invoice-file-pill max-w-full"
                    onClick={() => void previewStoredFile(`/api/locations/${id}/floorplan`)}
                  >
                    <ExternalLink size={12} className="shrink-0" />
                    <span className="min-w-0 truncate">{floorplanName}</span>
                  </button>
                </AppTooltip>
              ) : (
                '-'
              )
            )}
            {readonlyRow(
              '关联',
              <div className="flex flex-wrap items-center gap-2">
                <span className="itdb-location-link-badge is-item">
                  硬件&nbsp;<b>{linkedItems.length}</b>
                </span>
                <span className="itdb-location-link-badge is-rack">
                  机架&nbsp;<b>{linkedRacks.length}</b>
                </span>
              </div>
            )}
            {fileField ? (
              <FieldEditor
                field={fileField}
                value={values[fileField.key]}
                options={[]}
                onChange={value => onChange(fileField, value)}
                compactHorizontal
              />
            ) : null}
            <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-start gap-x-2.5">
              <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
                平面图预览
              </span>
              <div className="itdb-location-preview">
                {previewSource ? (
                  <img src={previewSource} alt="建筑平面图预览" />
                ) : (
                  <span>暂无平面图</span>
                )}
              </div>
            </div>
          </div>
        </section>
      ) : activeSection === 'areas' ? (
        <section className="itdb-resource-form-section mx-auto flex min-h-0 w-full max-w-3xl flex-1 flex-col rounded-xl border border-[var(--itdb-border)] px-5 pb-5 pt-3">
          <h3 className="mb-4 border-b border-[var(--itdb-border)] pb-2 text-base font-semibold text-[var(--itdb-text)]">
            区域：房间，办公室
          </h3>
          <div className="itdb-resource-data-panel itdb-hidden-scrollbar min-h-0 flex-1 overflow-auto rounded-lg border border-[var(--itdb-border)]">
            <table className="min-w-full text-sm">
              <thead className="sticky top-0 z-10 bg-[var(--itdb-control-bg-soft)]">
                <tr>
                  <th className="w-24 px-3 py-2.5 text-center font-medium">
                    <button
                      type="button"
                      className="inline-flex items-center gap-1"
                      onClick={() => setAreaSortDesc(current => !current)}
                    >
                      编号
                      {areaSortDesc ? <ChevronDown size={14} /> : <ChevronUp size={14} />}
                    </button>
                  </th>
                  <th className="px-3 py-2.5 text-center font-medium">区域名称</th>
                  <th className="w-28 px-3 py-2.5 text-center font-medium">操作</th>
                </tr>
              </thead>
              <tbody>
                {sortedAreaList.map(area => (
                  <tr key={String(area.id)} className="border-t border-[var(--itdb-border)]/70">
                    <td className="px-3 py-2.5 text-center">
                      <span className="itdb-relation-id">{areaDisplayId(area)}</span>
                    </td>
                    <td className="px-3 py-2 text-center">
                      {Number(area.id) === editingAreaId ? (
                        <Input
                          value={editingAreaName}
                          onChange={event => setEditingAreaName(event.target.value)}
                          onKeyDown={event => {
                            if (event.key === 'Enter') {
                              event.preventDefault();
                              commitAreaEdit();
                            }
                          }}
                          aria-label="区域名称"
                        />
                      ) : (
                        <span className="text-[var(--itdb-text)]">
                          {String(area.areaname ?? '-')}
                        </span>
                      )}
                    </td>
                    <td className="px-3 py-2.5 text-center">
                      <div className="flex justify-center gap-1.5">
                        {Number(area.id) === editingAreaId ? (
                          <>
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
                                onClick={commitAreaEdit}
                                aria-label={`提交区域修改 ${String(area.areaname ?? '')}`}
                              >
                                <Check size={13} />
                              </Button>
                            </AppTooltip>
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
                                onClick={cancelAreaEdit}
                                aria-label={`取消编辑区域 ${String(area.areaname ?? '')}`}
                              >
                                <X size={13} />
                              </Button>
                            </AppTooltip>
                          </>
                        ) : (
                          <>
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
                                onClick={() => {
                                  setEditingAreaId(Number(area.id));
                                  setEditingAreaName(String(area.areaname ?? ''));
                                  setEditingAreaOriginal(String(area.areaname ?? ''));
                                }}
                                aria-label={`编辑区域 ${String(area.areaname ?? '')}`}
                              >
                                <Pencil size={13} />
                              </Button>
                            </AppTooltip>
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
                                onClick={() => removeArea(area)}
                                aria-label={`删除区域 ${String(area.areaname ?? '')}`}
                              >
                                <Trash2 size={13} />
                              </Button>
                            </AppTooltip>
                          </>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
                {areaList.length === 0 ? (
                  <tr>
                    <td
                      colSpan={3}
                      className="px-4 py-8 text-center text-sm text-[var(--itdb-text-muted)]"
                    >
                      暂无区域数据
                    </td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
          <div className="mt-4 flex shrink-0 gap-2">
            <Input
              value={areaName}
              onChange={event => setAreaName(event.target.value)}
              onKeyDown={event => {
                if (event.key === 'Enter') {
                  event.preventDefault();
                  submitArea();
                }
              }}
              placeholder="请输入区域名称"
            />
            <Button
              type="button"
              className="disabled:pointer-events-auto disabled:cursor-not-allowed"
              disabled={!areaName.trim()}
              onClick={submitArea}
            >
              新增区域
            </Button>
          </div>
        </section>
      ) : (
        <section className="itdb-resource-form-section mx-auto flex min-h-0 w-full max-w-3xl flex-1 flex-col rounded-xl border border-[var(--itdb-border)] px-5 pb-5 pt-3">
          <h3 className="mb-4 border-b border-[var(--itdb-border)] pb-2 text-base font-semibold text-[var(--itdb-text)]">
            关联概览
          </h3>
          <div className="mb-3 flex shrink-0 flex-wrap gap-2">
            {[
              { key: 'items' as const, label: '硬件', count: linkedItems.length },
              { key: 'racks' as const, label: '机架', count: linkedRacks.length },
            ].map(tab => (
              <button
                key={tab.key}
                type="button"
                onClick={() => setOverviewTab(tab.key)}
                className="rounded-lg border px-3 py-1.5 text-sm font-medium transition-colors"
                data-active={overviewTab === tab.key}
                style={{
                  borderColor:
                    overviewTab === tab.key ? 'rgba(59,130,246,0.42)' : 'var(--itdb-border)',
                  background:
                    overviewTab === tab.key
                      ? 'rgba(59,130,246,0.14)'
                      : 'var(--itdb-control-bg-soft)',
                  color:
                    overviewTab === tab.key ? 'var(--itdb-accent-text)' : 'var(--itdb-text-muted)',
                }}
              >
                {tab.label} <span className="ml-1">{tab.count}</span>
              </button>
            ))}
          </div>
          <div className="itdb-hidden-scrollbar min-h-0 flex-1 space-y-1.5 overflow-y-auto px-1 pt-1">
            {overviewEntries.length ? (
              overviewEntries.map((entry, index) =>
                canOpenRecordEditor(entry.path) ? (
                  <Link
                    key={`${overviewTab}-${entry.id}`}
                    to={entry.path}
                    search={{ edit: Number(entry.id) }}
                    className="flex items-start gap-2 rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-2.5 py-2 text-xs text-[var(--itdb-accent-text)] transition-colors hover:border-[rgba(59,130,246,0.42)] hover:bg-[rgba(59,130,246,0.1)]"
                  >
                    <span className="shrink-0 font-mono text-[var(--itdb-text-muted)]">
                      {index + 1}:
                    </span>
                    <AppTooltip label={entry.tip}>
                      <span className="min-w-0 break-words">{entry.label}</span>
                    </AppTooltip>
                  </Link>
                ) : (
                  <div
                    key={`${overviewTab}-${entry.id}`}
                    className="flex items-start gap-2 rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] px-2.5 py-2 text-xs text-[var(--itdb-text-muted)]"
                  >
                    <span className="shrink-0 font-mono text-[var(--itdb-text-muted)]">
                      {index + 1}:
                    </span>
                    <AppTooltip label={entry.tip}>
                      <span className="min-w-0 break-words">{entry.label}</span>
                    </AppTooltip>
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
      )}
    </div>
  );
}
