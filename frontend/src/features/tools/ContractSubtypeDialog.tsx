import { useMutation, useQueryClient } from '@tanstack/react-query';
import { Edit3, FileText, Trash2 } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { Button } from '@/components/ui/button';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { api } from '@/lib/auth';

type Row = Record<string, unknown>;

export function ContractSubtypeDialog({
  contractType,
  subtypes,
  canManage,
  onClose,
}: {
  contractType: Row | null;
  subtypes: Row[];
  canManage: boolean;
  onClose: () => void;
}) {
  const client = useQueryClient();
  const inputRef = useRef<HTMLInputElement | null>(null);
  const [subtypeName, setSubtypeName] = useState('');
  const [editingSubtype, setEditingSubtype] = useState<Row | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Row | null>(null);
  const contractTypeId = Number(contractType?.id ?? 0);
  useEffect(() => {
    setSubtypeName('');
    setEditingSubtype(null);
    setDeleteTarget(null);
  }, [contractType]);
  const visibleSubtypes = subtypes.filter(
    subtype => Number(subtype.contypeid ?? 0) === contractTypeId
  );
  const refresh = async () => {
    await client.invalidateQueries({ queryKey: ['itdb', 'dictionaries'] });
    await client.invalidateQueries({ queryKey: ['itdb', 'bootstrap'] });
  };
  const saveSubtype = useMutation({
    mutationFn: async () => {
      const name = subtypeName.trim();
      const editing = editingSubtype;
      const body = JSON.stringify({ name, contypeid: contractTypeId });
      await (editing
        ? api(`/api/dictionaries/contractsubtypes/${editing.id}`, { method: 'PUT', body })
        : api('/api/dictionaries/contractsubtypes', { method: 'POST', body }));
      return { editing, name };
    },
    onSuccess: async ({ editing, name }) => {
      setSubtypeName('');
      setEditingSubtype(null);
      toast.success(
        editing ? `合同子类型 ${String(editing.name ?? name)} 已更新` : `合同子类型 ${name} 已创建`
      );
      await refresh();
    },
    onError: error => toast.error(error.message),
  });
  const removeSubtype = useMutation({
    mutationFn: async (subtype: Row) => {
      await api(`/api/dictionaries/contractsubtypes/${subtype.id}`, { method: 'DELETE' });
      return subtype;
    },
    onSuccess: async subtype => {
      setDeleteTarget(null);
      toast.success(`合同子类型 ${String(subtype.name ?? '')} 已删除`);
      await refresh();
    },
    onError: error => toast.error(error.message),
  });
  return (
    <Dialog open={contractType !== null} onOpenChange={open => !open && onClose()}>
      <DialogContent
        className="max-w-lg"
        onOpenAutoFocus={event => {
          event.preventDefault();
          const input = inputRef.current;
          if (!input) return;
          input.focus();
          const length = input.value.length;
          input.setSelectionRange(length, length);
        }}
      >
        <DialogHeader>
          <DialogTitle className="flex flex-wrap items-center gap-2.5">
            合同子类型
            <span className="itdb-dialog-name-badge">
              <FileText size={14} className="shrink-0" />
              <span className="min-w-0 truncate">{String(contractType?.name ?? '-')}</span>
            </span>
          </DialogTitle>
        </DialogHeader>
        <div className="itdb-resource-data-panel max-h-72 overflow-auto rounded-lg border border-[var(--itdb-border)]">
          <table className="min-w-full text-center text-sm">
            <thead>
              <tr>
                <th className="whitespace-nowrap px-4 py-2.5 text-center font-medium">编号</th>
                <th className="whitespace-nowrap px-4 py-2.5 text-center font-medium">
                  子类型名称
                </th>
                {canManage && (
                  <th className="whitespace-nowrap px-4 py-2.5 text-center font-medium">操作</th>
                )}
              </tr>
            </thead>
            <tbody>
              {visibleSubtypes.map(subtype => (
                <tr key={String(subtype.id)} className="border-b border-[var(--itdb-border)]/70">
                  <td className="whitespace-nowrap px-4 py-2.5 text-center">
                    <span className="itdb-relation-id">{String(subtype.id)}</span>
                  </td>
                  <td className="px-4 py-2.5 text-center text-[var(--itdb-text)]">
                    {String(subtype.name ?? '-')}
                  </td>
                  {canManage && (
                    <td className="px-3 py-2 text-center">
                      <div className="flex justify-center gap-2.5">
                        <AppTooltip label="编辑">
                          <Button
                            size="icon"
                            variant="ghost"
                            className="itdb-action-button h-7 w-7"
                            style={{
                              borderColor: 'rgba(59,130,246,0.38)',
                              background: 'rgba(59,130,246,0.1)',
                              color: 'var(--itdb-accent-text)',
                            }}
                            onClick={() => {
                              setEditingSubtype(subtype);
                              setSubtypeName(String(subtype.name ?? ''));
                            }}
                            aria-label={`编辑合同子类型 ${String(subtype.name ?? '')}`}
                          >
                            <Edit3 size={14} />
                          </Button>
                        </AppTooltip>
                        <AppTooltip label="删除">
                          <Button
                            size="icon"
                            variant="ghost"
                            className="itdb-action-button h-7 w-7"
                            style={{
                              borderColor: 'rgba(239,68,68,0.34)',
                              background: 'rgba(239,68,68,0.08)',
                              color: '#ef4444',
                            }}
                            onClick={() => setDeleteTarget(subtype)}
                            aria-label={`删除合同子类型 ${String(subtype.name ?? '')}`}
                          >
                            <Trash2 size={14} />
                          </Button>
                        </AppTooltip>
                      </div>
                    </td>
                  )}
                </tr>
              ))}
              {visibleSubtypes.length === 0 ? (
                <tr>
                  <td
                    colSpan={canManage ? 3 : 2}
                    className="px-4 py-6 text-center text-sm text-[var(--itdb-text-muted)]"
                  >
                    暂无合同子类型
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
        {canManage && (
          <div className="flex gap-2">
            <Input
              ref={inputRef}
              value={subtypeName}
              onChange={event => setSubtypeName(event.target.value)}
              onKeyDown={event => {
                if (event.key === 'Enter') {
                  event.preventDefault();
                  if (subtypeName.trim()) saveSubtype.mutate();
                }
              }}
              placeholder="请输入合同子类型名称"
            />
            {editingSubtype ? (
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setEditingSubtype(null);
                  setSubtypeName('');
                }}
              >
                取消编辑
              </Button>
            ) : null}
            <Button
              type="button"
              className="itdb-action-button disabled:pointer-events-auto disabled:cursor-not-allowed"
              style={{
                borderColor: 'rgba(59,130,246,0.38)',
                background: 'rgba(59,130,246,0.1)',
                color: 'var(--itdb-accent-text)',
              }}
              disabled={!subtypeName.trim() || saveSubtype.isPending}
              onClick={() => saveSubtype.mutate()}
            >
              {editingSubtype ? '保存修改' : '新增'}
            </Button>
          </div>
        )}
      </DialogContent>
      <ConfirmDialog
        open={Boolean(deleteTarget)}
        title="删除合同子类型"
        description="删除后不可恢复，请确认该子类型未被合同记录使用"
        detail={`确认删除 编号=${String(deleteTarget?.id ?? '-')} 的记录吗？`}
        horizontalHeader
        confirmText="确认删除"
        busy={removeSubtype.isPending}
        softDestructive
        onOpenChange={open => !open && setDeleteTarget(null)}
        onConfirm={() => {
          if (deleteTarget) removeSubtype.mutate(deleteTarget);
        }}
      />
    </Dialog>
  );
}
