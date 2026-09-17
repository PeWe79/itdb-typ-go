import { Link } from '@tanstack/react-router';
import { canOpenRecordEditor } from '../record-links';
import { Download, SquarePen, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { AppTooltip } from '@/components/app-tooltip';
import { downloadStoredFile } from '@/lib/file-download';
import { formatValue } from '../resource-helpers';

type Row = Record<string, unknown>;
type Lookups = Record<string, Row[]>;

export function HardwareFilesPane({
  values,
  lookups,
  resourceKey,
  onChangeFileLinks,
  onUnlinkFile,
}: {
  values: Row;
  lookups: Lookups | undefined;
  resourceKey: string;
  onChangeFileLinks: (next: number[]) => void;
  onUnlinkFile?: (fileId: number) => void;
}) {
  const files = Array.isArray(values.fileLinks)
    ? (values.fileLinks as unknown[])
        .map(Number)
        .filter(id => Number.isFinite(id) && id > 0)
        .sort((left, right) => left - right)
    : [];
  const fileRows = lookups?.files_ref ?? [];
  const noun =
    resourceKey === 'items'
      ? '硬件'
      : resourceKey === 'software'
        ? '软件'
        : resourceKey === 'invoices'
          ? '单据'
          : '合同';
  const iconButtonClass =
    'itdb-action-button flex h-7 w-7 items-center justify-center rounded-lg border';
  const linkButtonStyle = {
    borderColor: 'rgba(59,130,246,0.38)',
    background: 'rgba(59,130,246,0.1)',
    color: 'var(--itdb-accent-text)',
  };

  return (
    <section className="itdb-resource-form-section mx-auto flex min-h-0 w-full max-w-3xl flex-1 flex-col rounded-xl border border-[var(--itdb-border)] px-5 pb-5 pt-3">
      <h3 className="mb-4 border-b border-[var(--itdb-border)] pb-2 text-base font-semibold text-[var(--itdb-text)]">
        管理文件
      </h3>
      <div className="itdb-hidden-scrollbar -my-1 min-h-0 flex-1 space-y-2 overflow-y-auto py-1">
        {files.length ? (
          files.map(id => {
            const row = fileRows.find(item => Number(item.id) === id);
            const fileName = String(row?.fname ?? '').trim() || `文件 [ID:${id}]`;
            const typeDesc = String(row?.typedesc ?? row?.typeDesc ?? '').trim();
            const title = String(row?.title ?? '').trim();
            const dateText = row ? formatValue(row.date, 'date') : '';
            return (
              <div
                key={id}
                className="itdb-surface-3d itdb-file-card-3d itdb-file-card-flat rounded-lg px-3.5 py-2.5"
                style={{ background: 'var(--itdb-control-bg-soft)' }}
              >
                <div className="flex items-center gap-2">
                  <span className="min-w-0 flex-1 truncate text-sm font-medium text-[var(--itdb-text)]">
                    #{id} {fileName}
                  </span>
                  <div className="flex shrink-0 items-center gap-1.5">
                    {resourceKey === 'invoices' ? (
                      <AppTooltip
                        label={`解除关联，保存${noun}后生效。\n若文件是孤立的(没有其他内容与之相关联)，则会将其删除`}
                        placement="top"
                        wrap
                      >
                        <button
                          type="button"
                          className={iconButtonClass}
                          style={{
                            borderColor: 'rgba(239,68,68,0.34)',
                            background: 'rgba(239,68,68,0.08)',
                            color: '#ef4444',
                          }}
                          aria-label={`解除文件 ${id} 的关联`}
                          onClick={() => {
                            onChangeFileLinks(files.filter(value => value !== id));
                            onUnlinkFile?.(id);
                            if (resourceKey === 'invoices') {
                              toast.success('文件解除关联将在保存后生效');
                            }
                          }}
                        >
                          <Trash2 size={13} />
                        </button>
                      </AppTooltip>
                    ) : null}
                    {canOpenRecordEditor('/assets/files') ? (
                      <AppTooltip label={`在新窗口编辑文件 ${id}`}>
                        <Link
                          to="/assets/files"
                          search={{ edit: id }}
                          target="_blank"
                          rel="noopener"
                          aria-label={`在新窗口编辑文件 ${id}`}
                          className={iconButtonClass}
                          style={linkButtonStyle}
                        >
                          <SquarePen size={13} />
                        </Link>
                      </AppTooltip>
                    ) : null}
                    <AppTooltip label={`下载文件: ${fileName}`}>
                      <button
                        type="button"
                        className={iconButtonClass}
                        style={linkButtonStyle}
                        aria-label={`下载文件: ${fileName}`}
                        onClick={() =>
                          void downloadStoredFile(
                            `/api/files/${id}/download`,
                            fileName.includes('.') ? fileName : `${fileName}.bin`
                          )
                        }
                      >
                        <Download size={13} />
                      </button>
                    </AppTooltip>
                  </div>
                </div>
                <div className="mt-2 grid grid-cols-3 gap-2 border-t border-[var(--itdb-border)]/60 pt-2 text-xs">
                  <div className="min-w-0">
                    <div className="text-[var(--itdb-text-muted)]">类型</div>
                    <div className="mt-0.5 truncate font-medium text-[var(--itdb-text)]">
                      {typeDesc || '-'}
                    </div>
                  </div>
                  <div className="min-w-0">
                    <div className="text-[var(--itdb-text-muted)]">日期</div>
                    <div className="mt-0.5 truncate font-medium text-[var(--itdb-text)]">
                      {dateText || '-'}
                    </div>
                  </div>
                  <div className="min-w-0">
                    <div className="text-[var(--itdb-text-muted)]">标题</div>
                    <div className="mt-0.5 truncate font-medium text-[var(--itdb-text)]">
                      {title || '-'}
                    </div>
                  </div>
                </div>
              </div>
            );
          })
        ) : (
          <div className="border border-dashed border-[var(--itdb-border)] px-2.5 py-3 text-xs text-[var(--itdb-text-muted)]">
            暂无关联文件
          </div>
        )}
      </div>
      <p className="mt-4 text-xs leading-5 text-[var(--itdb-text-muted)]">
        上传文件请在“文件”菜单中维护。
      </p>
    </section>
  );
}
