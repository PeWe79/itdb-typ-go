import {
  DatabaseBackup,
  FileUp,
  Image,
  KeyRound,
  RotateCcw,
  Save,
  UploadCloud,
} from 'lucide-react';
import { useCallback, useEffect, useRef, useState, type DragEvent } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import { fetchSystemBaseConfig, updateSystemBaseConfig } from '@/features/settings/api';
import type { SystemBaseConfig } from '@/features/settings/types';
import { api, clearSession, getAuthToken } from '@/lib/auth';
import { setBrandSettings } from '@/lib/branding';
import {
  ActionButton,
  ConfigField,
  EnableToggle,
  NumberControl,
  SettingsDetailHeader,
  SettingsDetailPanel,
  SettingsSplitLayout,
} from './settings-primitives';
import { ConfirmDialog } from '@/components/confirm-dialog';
import { cardStyle } from './settings-style';

type BaseSection = 'brand' | 'security' | 'backup';

const sections: Array<{
  id: BaseSection;
  title: string;
  description: string;
  icon: typeof Image;
  color: string;
}> = [
  {
    id: 'brand',
    title: '品牌标识',
    description: '网站名称、登录展示和图标',
    icon: Image,
    color: '#38bdf8',
  },
  {
    id: 'security',
    title: '安全时效',
    description: '找回密码验证码、发送冷却、限流窗口与企业微信扫码有效期',
    icon: KeyRound,
    color: '#22c55e',
  },
  {
    id: 'backup',
    title: '数据备份',
    description: '数据库备份、导入与定时计划',
    icon: DatabaseBackup,
    color: '#f59e0b',
  },
];

export function BaseSettingsPanel({ canManage }: { canManage: boolean }) {
  const [active, setActive] = useState<BaseSection>('brand');
  const [form, setForm] = useState<SystemBaseConfig | null>(null);
  const [saved, setSaved] = useState<SystemBaseConfig | null>(null);
  const [busy, setBusy] = useState(false);
  const [dragging, setDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const load = useCallback(async () => {
    try {
      const config = await fetchSystemBaseConfig();
      setForm(config);
      setSaved(config);
      setBrandSettings(config);
    } catch (err) {
      showErrorToast(err instanceof Error ? err.message : '读取基础配置失败');
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  if (!form || !saved) {
    return (
      <div className="grid min-h-40 place-items-center text-sm text-[var(--itdb-text-muted)]">
        正在加载基础配置
      </div>
    );
  }

  const selected = sections.find(section => section.id === active) ?? sections[0];

  async function save() {
    if (!form || !saved) return;
    const next = { ...saved, ...pickSectionPatch(form, active) };
    const error = validateSection(active, next);
    if (error) {
      showErrorToast(error);
      return;
    }
    setBusy(true);
    try {
      const updated = await updateSystemBaseConfig({ ...next, section: active });
      setForm(updated);
      setSaved(updated);
      setBrandSettings(updated);
      toast.success('基础配置已保存');
    } catch (err) {
      showErrorToast(err instanceof Error ? err.message : '保存基础配置失败');
    } finally {
      setBusy(false);
    }
  }

  function update(patch: Partial<SystemBaseConfig>) {
    setForm(current => (current ? { ...current, ...patch } : current));
  }

  function selectSection(section: BaseSection) {
    setActive(section);
    if (saved) setForm(saved);
  }

  async function updateFile(file: File | undefined) {
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      toast.error('请选择图片文件');
      return;
    }
    if (file.size > 256 * 1024) {
      toast.error('图标文件不能超过 256KB');
      return;
    }
    update({ iconData: await readFileAsDataURL(file) });
  }

  function onDrop(event: DragEvent<HTMLButtonElement>) {
    event.preventDefault();
    setDragging(false);
    void updateFile(event.dataTransfer.files[0]);
  }

  return (
    <SettingsSplitLayout
      sidebarLabel="基础配置"
      sidebar={sections.map(section => (
        <BaseSectionCard
          key={section.id}
          section={section}
          active={active === section.id}
          onClick={() => selectSection(section.id)}
        />
      ))}
    >
      <SettingsDetailPanel
        header={
          <SettingsDetailHeader
            icon={selected.icon}
            color={selected.color}
            title={selected.title}
            subtitle={selected.description}
          />
        }
        actions={
          canManage ? (
            <>
              <ActionButton
                icon={<RotateCcw size={14} />}
                label="恢复默认"
                tone="muted"
                disabled={busy}
                onClick={() => update(pickSectionPatch(defaultBaseConfig, active))}
              />
              <ActionButton icon={<Save size={14} />} label="保存" busy={busy} onClick={save} />
            </>
          ) : null
        }
      >
        {active === 'brand' ? (
          <BrandPanel
            form={form}
            dragging={dragging}
            fileInputRef={fileInputRef}
            setDragging={setDragging}
            onDrop={onDrop}
            onFile={updateFile}
            onUpdate={update}
            disabled={!canManage}
          />
        ) : null}
        {active === 'security' ? (
          <SecurityPanel form={form} onUpdate={update} disabled={!canManage} />
        ) : null}
        {active === 'backup' ? (
          <BackupSettingsPanel form={form} onUpdate={update} disabled={!canManage} />
        ) : null}
      </SettingsDetailPanel>
    </SettingsSplitLayout>
  );
}

const defaultBaseConfig: SystemBaseConfig = {
  siteName: 'ITDB',
  loginName: 'ITDB',
  appName: 'ITDB',
  appSubtitle: 'IT Asset Management',
  iconData: '/favicon.svg',
  resetCodeTtlMinutes: 10,
  resetCaptchaTtlMinutes: 1,
  passwordResetSendCooldownMinutes: 0.5,
  passwordResetRateLimitMinutes: 5,
  wecomStateTtlMinutes: 5,
  backupEnabled: false,
  backupCron: '0 0 * * *',
  backupRetentionDays: 30,
};

function BaseSectionCard({
  section,
  active,
  onClick,
}: {
  section: (typeof sections)[number];
  active: boolean;
  onClick: () => void;
}) {
  const Icon = section.icon;
  return (
    <button
      type="button"
      onClick={onClick}
      className="itdb-config-side-card flex w-full items-start gap-3 rounded-lg p-3 text-left transition-all duration-200"
      data-active={active}
      style={cardStyle(active)}
    >
      <span
        className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg"
        style={{
          color: section.color,
          background: 'rgba(255,255,255,0.05)',
          border: '1px solid rgba(255,255,255,0.08)',
        }}
      >
        <Icon size={19} />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-semibold">{section.title}</span>
        <span className="mt-1 block text-xs leading-5" style={{ color: 'var(--itdb-text-muted)' }}>
          {section.description}
        </span>
      </span>
    </button>
  );
}

function BrandPanel({
  form,
  dragging,
  fileInputRef,
  setDragging,
  onDrop,
  onFile,
  onUpdate,
  disabled,
}: {
  form: SystemBaseConfig;
  dragging: boolean;
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  setDragging: (value: boolean) => void;
  onDrop: (event: DragEvent<HTMLButtonElement>) => void;
  onFile: (file: File | undefined) => void;
  onUpdate: (patch: Partial<SystemBaseConfig>) => void;
  disabled?: boolean;
}) {
  return (
    <div className="space-y-5">
      <div className="grid gap-4 lg:grid-cols-2">
        <ConfigField
          field={{ key: 'siteName', label: '网站名称', placeholder: 'ITDB' }}
          value={form.siteName}
          disabled={disabled}
          onChange={value => onUpdate({ siteName: String(value) })}
        />
        <ConfigField
          field={{ key: 'loginName', label: '认证页品牌名称', placeholder: 'ITDB' }}
          value={form.loginName}
          disabled={disabled}
          onChange={value => onUpdate({ loginName: String(value) })}
        />
        <ConfigField
          field={{ key: 'appName', label: '控制台品牌名称', placeholder: 'ITDB' }}
          value={form.appName}
          disabled={disabled}
          onChange={value => onUpdate({ appName: String(value) })}
        />
        <ConfigField
          field={{
            key: 'appSubtitle',
            label: '控制台品牌副标题',
            placeholder: 'IT Asset Management',
          }}
          value={form.appSubtitle}
          disabled={disabled}
          onChange={value => onUpdate({ appSubtitle: String(value) })}
        />
      </div>
      <div className="grid gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
        <button
          type="button"
          disabled={disabled}
          onClick={() => {
            if (!disabled) fileInputRef.current?.click();
          }}
          onDragOver={event => {
            event.preventDefault();
            setDragging(true);
          }}
          onDragLeave={() => setDragging(false)}
          onDrop={onDrop}
          className="itdb-action-button flex min-h-44 flex-col items-center justify-center gap-3 rounded-xl border border-dashed p-4"
          style={{
            borderColor: dragging ? 'rgba(96,165,250,0.78)' : 'var(--itdb-border)',
            background: dragging ? 'rgba(59,130,246,0.12)' : 'rgba(255,255,255,0.026)',
            color: 'var(--itdb-text-muted)',
          }}
        >
          <img
            src={form.iconData || '/favicon.svg'}
            alt="系统图标预览"
            className="h-16 w-16 rounded-xl object-contain"
          />
          <span className="flex items-center gap-2 text-sm">
            <UploadCloud size={15} />
            拖动或点击上传
          </span>
          <span className="text-xs">支持图片，建议 256KB 内</span>
        </button>
        <BrandPreview config={form} />
      </div>
      <input
        ref={fileInputRef}
        type="file"
        accept="image/*"
        disabled={disabled}
        className="hidden"
        onChange={event => void onFile(event.target.files?.[0])}
      />
    </div>
  );
}

function BrandPreview({ config }: { config: SystemBaseConfig }) {
  return (
    <div
      className="flex min-h-44 flex-col rounded-xl border p-5"
      style={{ borderColor: 'var(--itdb-border)', background: 'rgba(255,255,255,0.026)' }}
    >
      <div className="text-sm font-semibold" style={{ color: 'var(--itdb-text-muted)' }}>
        实时预览
      </div>
      <div className="flex flex-1 items-center">
        <div className="flex min-w-0 items-center gap-4">
          <img
            src={config.iconData || '/favicon.svg'}
            alt={config.appName}
            className="h-16 w-16 shrink-0 rounded-xl object-contain"
          />
          <div className="min-w-0">
            <div className="itdb-gradient-text truncate text-xl font-bold">{config.appName}</div>
            <div
              className="mt-1 truncate text-sm tracking-widest"
              style={{ color: 'var(--itdb-text-muted)' }}
            >
              {config.appSubtitle}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function SecurityPanel({
  form,
  onUpdate,
  disabled,
}: {
  form: SystemBaseConfig;
  onUpdate: (patch: Partial<SystemBaseConfig>) => void;
  disabled?: boolean;
}) {
  return (
    <div className="grid gap-3 lg:grid-cols-2">
      <NumberControl
        label="找回密码验证码有效期"
        unit="分钟"
        value={form.resetCodeTtlMinutes}
        min={1}
        max={60}
        disabled={disabled}
        onChange={value => onUpdate({ resetCodeTtlMinutes: value })}
      />
      <NumberControl
        label="图形验证码有效期"
        unit="分钟"
        value={form.resetCaptchaTtlMinutes}
        min={1}
        max={10}
        disabled={disabled}
        onChange={value => onUpdate({ resetCaptchaTtlMinutes: value })}
      />
      <NumberControl
        label="发送冷却时间"
        description={`验证码发送后 ${form.passwordResetSendCooldownMinutes} 分钟内不可重复请求`}
        unit="分钟"
        value={form.passwordResetSendCooldownMinutes}
        min={0.5}
        max={10}
        step={0.5}
        disabled={disabled}
        onChange={value => onUpdate({ passwordResetSendCooldownMinutes: value })}
      />
      <NumberControl
        label="频率限制统计窗口"
        description={`验证码在 ${form.passwordResetRateLimitMinutes} 分钟内最多请求 5 次`}
        unit="分钟"
        value={form.passwordResetRateLimitMinutes}
        min={5}
        max={10}
        disabled={disabled}
        onChange={value => onUpdate({ passwordResetRateLimitMinutes: value })}
      />
      <NumberControl
        label="企业微信扫码有效期"
        description="企微授权 state 的有效窗口，超时需重新扫码登录或绑定"
        unit="分钟"
        value={form.wecomStateTtlMinutes}
        min={1}
        max={60}
        disabled={disabled}
        onChange={value => onUpdate({ wecomStateTtlMinutes: value })}
      />
    </div>
  );
}

function BackupSettingsPanel({
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

  /* filenameFromDisposition 优先取服务端返回的下载文件名，缺失时回退本地命名 */
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

function pickSectionPatch(
  config: SystemBaseConfig,
  section: BaseSection
): Partial<SystemBaseConfig> {
  if (section === 'backup') {
    return {
      backupEnabled: Boolean(config.backupEnabled),
      backupCron: config.backupCron?.trim() || defaultBaseConfig.backupCron,
      backupRetentionDays: config.backupRetentionDays,
    };
  }
  if (section === 'brand') {
    return {
      siteName: withDefault(config.siteName, defaultBaseConfig.siteName),
      loginName: withDefault(config.loginName, defaultBaseConfig.loginName),
      appName: withDefault(config.appName, defaultBaseConfig.appName),
      appSubtitle: withDefault(config.appSubtitle, defaultBaseConfig.appSubtitle),
      iconData: withDefault(config.iconData, defaultBaseConfig.iconData),
    };
  }
  return {
    resetCodeTtlMinutes: config.resetCodeTtlMinutes,
    resetCaptchaTtlMinutes: config.resetCaptchaTtlMinutes,
    passwordResetSendCooldownMinutes: config.passwordResetSendCooldownMinutes,
    passwordResetRateLimitMinutes: config.passwordResetRateLimitMinutes,
    wecomStateTtlMinutes: config.wecomStateTtlMinutes,
  };
}

function validateSection(section: BaseSection, config: SystemBaseConfig) {
  if (section === 'brand') {
    for (const [label, value] of [
      ['网站名称', config.siteName],
      ['认证页品牌名称', config.loginName],
      ['控制台品牌名称', config.appName],
      ['控制台品牌副标题', config.appSubtitle],
    ] as const) {
      if ([...value].length > 60) {
        return `${label}长度不能超过 60 个字符`;
      }
    }
  }
  if (section === 'backup') {
    if (config.backupCron.trim().split(/\s+/).length !== 5) {
      return '定时备份 Cron 需为五段表达式：分 时 日 月 周';
    }
  }
  return '';
}

function withDefault(value: string, fallback: string) {
  return value.trim() || fallback;
}

function readFileAsDataURL(file: File) {
  return new Promise<string>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(String(reader.result || ''));
    reader.onerror = () => reject(new Error('读取图标失败'));
    reader.readAsDataURL(file);
  });
}
