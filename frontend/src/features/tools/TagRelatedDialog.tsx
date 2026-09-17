import { useQuery } from '@tanstack/react-query';
import { Boxes, ChevronRight, Disc } from 'lucide-react';
import { AppTooltip } from '@/components/app-tooltip';
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { api } from '@/lib/auth';
import { canOpenRecordEditor } from '@/features/assets/record-links';

type Row = Record<string, unknown>;
type RelatedEntry = { id: number; txt: string };

/* 对照原 itdb 项目：点击关联硬件/关联软件数量，查看关联明细，条目可跳转编辑 */
export function TagRelatedDialog({
  tag,
  target,
  onClose,
}: {
  tag: Row | null;
  target: 'items' | 'software' | null;
  onClose: () => void;
}) {
  const tagId = Number(tag?.id ?? 0);
  const open = Boolean(tag && target);
  const query = useQuery({
    queryKey: ['itdb', 'tag-related', target, tagId],
    queryFn: () =>
      api<RelatedEntry[]>(`/api/tags/${tagId}/${target === 'items' ? 'items' : 'software'}`),
    enabled: open && tagId > 0,
  });
  const entries = Array.isArray(query.data) ? query.data : [];
  const isItems = target === 'items';
  const label = isItems ? '关联硬件' : '关联软件';
  return (
    <Dialog open={open} onOpenChange={value => !value && onClose()}>
      <DialogContent className="max-w-lg" onOpenAutoFocus={event => event.preventDefault()}>
        <DialogHeader>
          <DialogTitle className="flex flex-wrap items-center gap-2.5">
            {label}
            <span className="itdb-dialog-name-badge">
              {isItems ? (
                <Boxes size={14} className="shrink-0" />
              ) : (
                <Disc size={14} className="shrink-0" />
              )}
              <span className="min-w-0 truncate">{String(tag?.name ?? '-')}</span>
            </span>
          </DialogTitle>
        </DialogHeader>
        <div className="itdb-hidden-scrollbar max-h-80 min-h-24 overflow-y-auto rounded-lg border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)]/40 p-2">
          {query.isLoading ? (
            <p className="px-3 py-6 text-center text-sm text-[var(--itdb-text-muted)]">
              正在加载关联{isItems ? '硬件' : '软件'}...
            </p>
          ) : query.isError ? (
            <p className="px-3 py-6 text-center text-sm text-[var(--itdb-text-muted)]">
              关联数据加载失败
            </p>
          ) : entries.length === 0 ? (
            <p className="px-3 py-6 text-center text-sm text-[var(--itdb-text-muted)]">
              暂无关联记录
            </p>
          ) : (
            <div className="space-y-1.5">
              {entries.map((entry, index) => {
                const jumpPath = isItems ? '/assets/hardware' : '/assets/software';
                return canOpenRecordEditor(jumpPath) ? (
                  <AppTooltip
                    key={`${entry.id}-${index}`}
                    label={`在新窗口编辑${isItems ? '硬件' : '软件'} ${entry.id}`}
                  >
                    <button
                      type="button"
                      onClick={() =>
                        window.open(`${jumpPath}?edit=${entry.id}`, '_blank', 'noopener')
                      }
                      className="flex w-full items-center gap-2 rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-card)] px-2.5 py-2 text-left text-xs text-[var(--itdb-text)] transition-colors hover:border-[rgba(59,130,246,0.42)] hover:bg-[rgba(59,130,246,0.1)]"
                    >
                      <span className="itdb-relation-id shrink-0">{index + 1}</span>
                      <span className="min-w-0 flex-1 truncate font-medium text-[var(--itdb-accent-text)]">
                        {entry.txt}
                      </span>
                      <ChevronRight size={14} className="shrink-0 text-[var(--itdb-text-muted)]" />
                    </button>
                  </AppTooltip>
                ) : (
                  <div
                    key={`${entry.id}-${index}`}
                    className="flex w-full items-center gap-2 rounded-md border border-[var(--itdb-border)] bg-[var(--itdb-card)] px-2.5 py-2 text-left text-xs text-[var(--itdb-text-muted)]"
                  >
                    <span className="itdb-relation-id shrink-0">{index + 1}</span>
                    <span className="min-w-0 flex-1 truncate font-medium">{entry.txt}</span>
                  </div>
                );
              })}
            </div>
          )}
        </div>
        <p className="text-xs text-[var(--itdb-text-muted)]">点击条目将在新窗口编辑对应记录</p>
      </DialogContent>
    </Dialog>
  );
}
