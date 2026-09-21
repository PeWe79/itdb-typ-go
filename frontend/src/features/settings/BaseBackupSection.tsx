import { FileUp } from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import type { SystemBaseConfig } from '@/features/settings/types';
import { api, clearSession, getAuthToken } from '@/lib/auth';
import { ActionButton, ConfigField, EnableToggle } from './settings-primitives';
import { ConfirmDialog } from '@/components/confirm-dialog';

// BackupSettingsPanel 渲染基础配置的数据备份区块：定时备份、手动备份与数据库导入
export function BackupSettingsPanel({
  form,
  onUpdate,
  disabled,
}: {
  form: SystemBaseConfig;
  onUpdate: (patch: Partial<SystemBaseConfig>) => void;
  disabled?: boolean;
}) {
  const [downloading, setDownloading] = useState('');
  const [includeFiles, setIncludeFiles] = useState(false);
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importing, setImporting] = useState(false);

  function pad(value: number) {
    return String(value).padStart(2, '0');
  }
  function localStamp() {
    const now = new Date();
    return `${now.getFullYear()}${pad(now.getMonth() + 1)}${pad(now.getDate())}`;
  }

  function filenameFromDisposition(response: Response, fallback: string) {
    const disposition = response.headers.get('Content-Disposition') ?? '';
    const match = disposition.match(/filename\s*=\s*"?([^";]+)"?/i);
    return match ? match[1] : fallback;
  }

  async function downloadBackup(include: boolean) {
    setDownloading(include ? 'zip' : 'db');
    try {
      const response = await fetch(`/api/backups/database?files=${include ? '1' : '0'}`, {
        headers: { Authorization: `Bearer ${getAuthToken()}` },
      });
      if (!response.ok) throw new Error('备份下载失败');
      const blob = await response.blob();
      const link = document.createElement('a');
      link.href = URL.createObjectURL(blob);
      link.download = filenameFromDisposition(
        response,
        include ? `itdb-${localStamp()}.zip` : `itdb-${localStamp()}.db`
      );
      link.click();
      URL.revokeObjectURL(link.href);
      toast.success('备份文件已下载');
    } catch (err) {
      showErrorToast(err instanceof Error ? err.message : '备份下载失败');
    } finally {
      setDownloading('');
    }
  }

  async function importDatabase() {
    if (!importFile) return;
    setImporting(true);
    try {
      const payload = new FormData();
      payload.append('file', importFile);
      await api('/api/import/database', { method: 'POST', body: payload });
      clearSession();
      setImportFile(null);
      toast.success('数据库导入成功，正在跳转登录页');
    } catch (err) {
      showErrorToast(err instanceof Error ? err.message : '数据库导入失败');
      setImporting(false);
    }
  }

  return (
    <div className="space-y-5">
      <section
        className="space-y-3 rounded-xl border p-4"
        style={{ borderColor: 'var(--itdb-border)', background: 'rgba(255,255,255,0.026)' }}
      >
        <EnableToggle
          enabled={Boolean(form.backupEnabled)}
          disabled={disabled}
          onChange={value => onUpdate({ backupEnabled: value })}
          label="启用定时备份"
          enabledText="按下方 Cron 计划自动备份数据库"
          disabledText="关闭后不执行定时备份"
        />
        <div className="grid gap-3 md:grid-cols-2">
          <ConfigField
            field={{
              key: 'backupCron',
              label: '定时备份计划',
              placeholder: '0 0 * * *',
              helper: '使用五段 Cron：分 时 日 月 周，例如 0 0 * * * 表示每天 0 点备份',
            }}
            value={form.backupCron}
            disabled={disabled}
            onChange={value => onUpdate({ backupCron: String(value ?? '') })}
          />
          <ConfigField
            field={{
              key: 'backupRetentionDays',
              label: '保留备份天数',
              type: 'number',
              placeholder: '30',
              helper: '定时备份后自动清理超过该天数的备份，0 表示永久保留',
            }}
            value={form.backupRetentionDays}
            disabled={disabled}
            onChange={value => onUpdate({ backupRetentionDays: Number(value ?? 0) })}
          />
        </div>
      </section>

      <div className="grid gap-5 lg:grid-cols-2">
        <section
          className="flex flex-col space-y-3 rounded-xl border p-4"
          style={{ borderColor: 'var(--itdb-border)', background: 'rgba(255,255,255,0.026)' }}
        >
          <h3 className="text-sm font-semibold text-[var(--itdb-text)]">手动备份</h3>
          <p className="text-xs leading-5 text-[var(--itdb-text-muted)]">
            备份数据库文件；勾选「包含上传的文件」时，数据库与上传文件会一并打包。
          </p>
          <div className="mt-auto flex items-center justify-between gap-3">
            <label
              className={`flex items-center gap-2 text-sm transition-colors ${
                disabled || downloading !== ''
                  ? 'cursor-not-allowed opacity-70'
                  : 'cursor-pointer hover:text-[var(--itdb-text)]'
              }`}
            >
              <input
                type="checkbox"
                checked={includeFiles}
                disabled={disabled || downloading !== ''}
                onChange={event => setIncludeFiles(event.target.checked)}
                className="itdb-check-box"
              />
              包含上传的文件
            </label>
            <ActionButton
              icon={<FileUp size={14} />}
              label="立即备份"
              busy={downloading !== ''}
              disabled={disabled}
              onClick={() => void downloadBackup(includeFiles)}
            />
          </div>
        </section>

        <section
          className="flex flex-col space-y-3 rounded-xl border p-4"
          style={{ borderColor: 'var(--itdb-border)', background: 'rgba(255,255,255,0.026)' }}
        >
          <h3 className="text-sm font-semibold text-[var(--itdb-text)]">数据库导入</h3>
          <p className="text-xs leading-5 text-[var(--itdb-text-muted)]">
            支持 .db 数据库文件，或包含数据库与上传文件的 .zip
            压缩包（文件统一恢复到上传目录）。导入将覆盖当前全部数据，导入前会先预备份。
          </p>
          <ConfirmDialog
            open={Boolean(importFile)}
            title="导入数据库"
            description="导入会替换当前系统数据库，并使所有已登录用户退出"
            detail={importFile ? `待导入文件：${importFile.name}` : undefined}
            horizontalHeader
            confirmText="确认导入"
            busy={importing}
            onOpenChange={open => !open && !importing && setImportFile(null)}
            onConfirm={() => void importDatabase()}
          />
          <div className="mt-auto flex justify-end">
            <ActionButton
              icon={<FileUp size={14} />}
              label="选择文件并导入"
              tone="danger"
              disabled={disabled || importing}
              onClick={() => {
                const input = document.createElement('input');
                input.type = 'file';
                input.accept = '.db,.zip';
                input.onchange = () => {
                  const file = input.files?.[0];
                  if (file) setImportFile(file);
                };
                input.click();
              }}
            />
          </div>
        </section>
      </div>
    </div>
  );
}
