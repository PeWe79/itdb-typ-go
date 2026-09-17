import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useMemo, useRef, useState } from 'react';
import { toast } from 'sonner';

import { ConfirmDialog } from '@/components/confirm-dialog';
import { PermissionGate } from '@/components/permission-gate';
import { api, getStoredUser, userHasPermission } from '@/lib/auth';
import { PERM } from '@/lib/permissions';
import { showErrorToast } from '@/lib/toast-errors';
import { compareTableValues, getSortValue } from '@/features/assets/resource-helpers';
import {
  DEFAULT_LABEL_CONFIG,
  labelConfigFromPreset,
  labelPresetPayload,
  paperSizeMM,
  type LabelConfig,
} from './labels-config';
import {
  buildLabelBodyLines,
  buildPrintDocumentHtml,
  qrImageDataUrl,
  type LabelCell,
  type LabelRow,
} from './labels-print';
import { HardwarePanel } from './labels-hardware-panel';
import { BUILTIN_PRESET_NAMES, PropertyPanel } from './labels-panels';
import { PreviewPanel } from './labels-preview-panel';
import { trackAuditEvent } from '@/lib/audit-track';

type Row = Record<string, unknown>;

/* 状态排序权重：与后端 SQL 的固定状态顺序一致，任何列排序时状态优先 */
function statusRank(row: Row) {
  const status = String(row.statusdesc ?? '').trim();
  if (status === '使用中') return 0;
  if (status === '库存') return 1;
  if (status === '有故障') return 2;
  if (status === '报废') return 3;
  return 4;
}

export function LabelsPage() {
  const client = useQueryClient();
  const user = getStoredUser();
  const canPrint = userHasPermission(user, PERM.labelsPrint);
  const canEditProps = userHasPermission(user, PERM.labelsManage);
  const [search, setSearch] = useState('');
  const [sort, setSort] = useState<{ key: string; direction: 'asc' | 'desc' }>({
    key: 'id',
    direction: 'asc',
  });
  const [selectedIds, setSelectedIds] = useState<number[]>([]);
  const [config, setConfig] = useState<LabelConfig>(DEFAULT_LABEL_CONFIG);
  const [selectedPresetId, setSelectedPresetId] = useState('');
  const [presetName, setPresetName] = useState('');
  const [deleteTarget, setDeleteTarget] = useState<Row | null>(null);
  const [previewRows, setPreviewRows] = useState<LabelRow[]>([]);
  const [qrImages, setQrImages] = useState<Map<number, string>>(new Map());
  const previewRef = useRef<HTMLDivElement | null>(null);

  const items = useQuery({
    queryKey: ['itdb', 'labels', 'items', search],
    queryFn: () => api<Row[]>(`/api/labels/items?search=${encodeURIComponent(search)}&limit=1500`),
  });
  const itemList = items.data ?? [];
  useEffect(() => {
    const alive = new Set(itemList.map(item => Number(item.id)));
    setSelectedIds(current => current.filter(id => alive.has(id)));
  }, [itemList]);

  const presets = useQuery({
    queryKey: ['itdb', 'labels', 'presets'],
    queryFn: () => api<Row[]>('/api/labels/presets'),
  });
  const defaultPreset = (presets.data ?? []).find(
    row => String(row.name ?? '').trim() === 'Avery6106'
  );
  const defaultPresetApplied = useRef(false);
  useEffect(() => {
    if (defaultPresetApplied.current || !defaultPreset) return;
    defaultPresetApplied.current = true;
    setConfig(labelConfigFromPreset(defaultPreset));
    setSelectedPresetId(String(defaultPreset.id));
  }, [defaultPreset]);

  const sortedItems = useMemo(
    () =>
      [...itemList].sort((left, right) => {
        const rankDiff = statusRank(left) - statusRank(right);
        if (rankDiff !== 0) return rankDiff;
        return (
          compareTableValues(getSortValue(left, sort.key), getSortValue(right, sort.key)) *
          (sort.direction === 'asc' ? 1 : -1)
        );
      }),
    [itemList, sort]
  );
  const toggleSort = (key: string) =>
    setSort(previous => {
      if (previous.key !== key) return { key, direction: 'asc' };
      return {
        key,
        direction: previous.direction === 'asc' ? 'desc' : 'asc',
      };
    });

  useEffect(() => {
    if (!config.wantbarcode || previewRows.length === 0) return;
    const missing = previewRows.filter(row => !qrImages.get(row.id));
    if (missing.length === 0) return;
    let cancelled = false;
    void (async () => {
      const images = new Map(qrImages);
      await Promise.all(
        missing.map(async row => {
          images.set(row.id, await qrImageDataUrl(row.qrText));
        })
      );
      if (!cancelled) setQrImages(images);
    })();
    return () => {
      cancelled = true;
    };
  }, [config.wantbarcode, previewRows, qrImages]);

  useEffect(() => {
    if (previewRows.length === 0) return;
    const prefix = config.qrtext.trim();
    const expectedText = (row: LabelRow) =>
      prefix ? prefix + row.id : buildLabelBodyLines(row, false).join('\n');
    const updates = previewRows
      .filter(row => row.qrText !== expectedText(row))
      .map(row => ({ ...row, qrText: expectedText(row) }));
    if (updates.length === 0) return;
    const staleIds = new Set(updates.map(row => row.id));
    let cancelled = false;
    void (async () => {
      const images = new Map(qrImages);
      await Promise.all(
        updates.map(async row => {
          images.set(row.id, await qrImageDataUrl(row.qrText));
        })
      );
      if (cancelled) return;
      setPreviewRows(current =>
        current.map(row => (staleIds.has(row.id) ? { ...row, qrText: expectedText(row) } : row))
      );
      setQrImages(images);
    })();
    return () => {
      cancelled = true;
    };
  }, [config.qrtext, previewRows, qrImages]);

  const update = (key: keyof LabelConfig, value: string | number | boolean) =>
    setConfig(current => ({ ...current, [key]: value }));

  const preview = useMutation({
    mutationFn: async () => {
      const rows = await api<LabelRow[]>('/api/labels/preview', {
        method: 'POST',
        body: JSON.stringify({
          itemIds: selectedIds,
          qrPrefix: config.qrtext,
          headerText: config.headertext,
          presetName: (config.name || presetName).trim(),
        }),
      });
      const normalized = config.qrtext.trim()
        ? rows
        : rows.map(row => ({ ...row, qrText: buildLabelBodyLines(row, false).join('\n') }));
      const images = new Map<number, string>();
      if (config.wantbarcode) {
        await Promise.all(
          normalized.map(async row => {
            images.set(row.id, await qrImageDataUrl(row.qrText));
          })
        );
      }
      return { rows: normalized, images };
    },
    onSuccess: result => {
      setPreviewRows(result.rows);
      setQrImages(result.images);
      toast.success(`已生成 ${result.rows.length} 个标签预览`);
      previewRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    },
    onError: error => showErrorToast(error.message),
  });

  const savePreset = useMutation({
    mutationFn: () =>
      api<{ id: number }>('/api/labels/presets', {
        method: 'POST',
        body: JSON.stringify(
          labelPresetPayload({ ...config, name: (config.name || presetName).trim() })
        ),
      }),
    onSuccess: data => {
      setPresetName('');
      setSelectedPresetId(String(data.id));
      void client.invalidateQueries({ queryKey: ['itdb', 'labels', 'presets'] });
    },
    onError: error => showErrorToast(error.message),
  });
  const deletePreset = useMutation({
    mutationFn: (id: number) => api(`/api/labels/presets/${id}`, { method: 'DELETE' }),
    onSuccess: (_data, id) => {
      const name = String(
        (presets.data ?? []).find(row => Number(row.id) === Number(id))?.name ?? id
      );
      toast.success(`标签预设 ${name} 已删除`);
      setSelectedPresetId(current => (current === String(id) ? '' : current));
      setDeleteTarget(null);
      void client.invalidateQueries({ queryKey: ['itdb', 'labels', 'presets'] });
    },
    onError: error => showErrorToast(error.message),
  });

  const print = async () => {
    if (previewRows.length === 0) {
      toast.error('请先生成标签预览');
      return;
    }
    const popup = window.open('', '_blank');
    if (!popup) {
      toast.error('浏览器拦截了弹出窗口，请允许弹窗后重试');
      return;
    }
    const images = new Map(qrImages);
    const missing = config.wantbarcode ? previewRows.filter(row => !images.get(row.id)) : [];
    await Promise.all(
      missing.map(async row => {
        images.set(row.id, await qrImageDataUrl(row.qrText));
      })
    );
    if (missing.length > 0) setQrImages(images);
    popup.document.write(
      buildPrintDocumentHtml(
        computePages(previewRows, config),
        images,
        config,
        paperSizeMM(config.papersize)
      )
    );
    popup.document.close();
    void trackAuditEvent({
      type: 'print:labels',
      target: (config.name || presetName).trim(),
      count: previewRows.length,
    });
  };

  return (
    <PermissionGate anyOf={[PERM.labelsPreview]}>
      <>
        <div className="flex flex-col gap-4">
          <HardwarePanel
            search={search}
            setSearch={setSearch}
            items={sortedItems}
            sort={sort}
            onSortChange={toggleSort}
            selectedIds={selectedIds}
            allChecked={
              itemList.length > 0 && itemList.every(item => selectedIds.includes(Number(item.id)))
            }
            toggleAll={() =>
              setSelectedIds(
                itemList.every(item => selectedIds.includes(Number(item.id)))
                  ? []
                  : itemList.map(item => Number(item.id))
              )
            }
            toggleOne={(id: number) =>
              setSelectedIds(current =>
                current.includes(id) ? current.filter(value => value !== id) : [...current, id]
              )
            }
            loading={items.isPending}
            onPreview={() => {
              if (selectedIds.length === 0) {
                toast.error('请先勾选需要打印标签的硬件');
                return;
              }
              preview.mutate();
            }}
            previewing={preview.isPending}
            onPrint={canPrint ? () => void print() : undefined}
          />
          <PropertyPanel
            config={config}
            update={update}
            presets={presets.data ?? []}
            selectedPresetId={selectedPresetId}
            presetName={presetName}
            setPresetName={setPresetName}
            onSavePreset={() => {
              const name = (config.name || presetName).trim();
              if (!name) {
                toast.error('请填写预设名称');
                return;
              }
              if (BUILTIN_PRESET_NAMES.has(name)) {
                toast.error('内置标签预设无法更新');
                return;
              }
              const existed = (presets.data ?? []).some(
                row => String(row.name ?? '').trim() === name
              );
              savePreset.mutate(undefined, {
                onSuccess: () =>
                  toast.success(existed ? `标签预设 ${name} 已更新` : `标签预设 ${name} 已保存`),
              });
            }}
            saving={savePreset.isPending}
            onApply={row => {
              setConfig(labelConfigFromPreset(row));
              setSelectedPresetId(String(row.id));
              setPresetName('');
              toast.success(`已载入标签预设 ${String(row.name ?? '')}`);
            }}
            onDelete={row => setDeleteTarget(row)}
            onReset={() => {
              const selected = (presets.data ?? []).find(
                row => String(row.id) === selectedPresetId
              );
              if (selected) {
                setConfig(labelConfigFromPreset(selected));
                setPresetName('');
                toast.success(`已恢复标签预设 ${String(selected.name ?? '')}`);
              } else {
                setConfig(DEFAULT_LABEL_CONFIG);
                setPresetName('');
                toast.success('已恢复默认标签属性');
              }
            }}
            canEdit={canEditProps}
          />
          <PreviewPanel
            ref={previewRef}
            rows={previewRows}
            qrImages={qrImages}
            config={config}
            onPrint={canPrint ? () => void print() : undefined}
            onClear={() => {
              setPreviewRows([]);
              setQrImages(new Map());
            }}
          />
        </div>
        <ConfirmDialog
          open={Boolean(deleteTarget)}
          title="删除标签预设"
          description="删除后不可恢复。"
          detail={`确认删除标签预设 ${String(deleteTarget?.name ?? '')} 吗？`}
          confirmText="确认删除"
          horizontalHeader
          softDestructive
          onConfirm={() => deleteTarget && deletePreset.mutate(Number(deleteTarget.id))}
          onOpenChange={open => !open && setDeleteTarget(null)}
        />
      </>
    </PermissionGate>
  );
}

function computePages(rows: LabelRow[], config: LabelConfig): LabelCell[][] {
  const capacity = Math.max(1, config.cols * config.rows);
  const skip = Math.max(0, Math.min(capacity - 1, Number(config.labelskip) || 0));
  const cells: LabelCell[] = [
    ...Array.from({ length: skip }, () => ({ kind: 'skip' }) as LabelCell),
    ...rows.map(row => ({ kind: 'label', row }) as LabelCell),
  ];
  const pages: LabelCell[][] = [];
  for (let index = 0; index < cells.length; index += capacity) {
    const chunk = cells.slice(index, index + capacity);
    while (chunk.length < capacity) chunk.push({ kind: 'blank' });
    pages.push(chunk);
  }
  return pages.length > 0 ? pages : [[]];
}
