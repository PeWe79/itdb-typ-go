import { MessageCircle, Save, Trash2 } from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import { toast } from 'sonner';
import { showErrorToast } from '@/lib/toast-errors';
import { fetchWeComProvider, saveWeComProvider } from '@/features/settings/api';
import {
  ActionButton,
  ConfigField,
  EnableToggle,
  SettingsDetailHeader,
  SettingsDetailPanel,
  type SettingsField,
} from './settings-primitives';

type WeComForm = Record<string, unknown>;

const defaultForm: WeComForm = {
  corpid: '',
  agentid: '',
  secret: '',
  redirectPrefix: '',
};

const fields: SettingsField[] = [
  { key: 'corpid', label: '企业 ID（corpid）', placeholder: 'ww1234567890', required: true },
  {
    key: 'agentid',
    label: '应用 AgentID',
    placeholder: '1000002',
    required: true,
    inputMode: 'numeric',
  },
  {
    key: 'secret',
    label: '应用 Secret',
    type: 'password',
    placeholder: '请输入应用 Secret',
    required: true,
  },
  {
    key: 'redirectPrefix',
    label: '回调地址前缀',
    placeholder: 'https://itdb.example.com',
    required: true,
  },
];

// WeComSettingsPanel 企业微信认证配置详情面板
export function WeComSettingsPanel({
  canManage,
  onChanged,
}: {
  canManage: boolean;
  onChanged?: () => void;
}) {
  const [name, setName] = useState('企业微信');
  const [enabled, setEnabled] = useState(false);
  const [savedEnabled, setSavedEnabled] = useState(false);
  const [form, setForm] = useState<WeComForm>(defaultForm);
  const [busy, setBusy] = useState('');
  const [error, setError] = useState('');
  const [clearRequested, setClearRequested] = useState(false);
  const loadPromiseRef = useRef<Promise<void> | null>(null);

  const load = useCallback(() => {
    if (loadPromiseRef.current) return loadPromiseRef.current;
    const promise = (async () => {
      setError('');
      try {
        const setting = await fetchWeComProvider();
        setName(setting.name || '企业微信');
        setEnabled(setting.enabled);
        setSavedEnabled(setting.enabled);
        setForm({ ...defaultForm, ...setting.config });
      } catch (err) {
        const message = err instanceof Error ? err.message : '读取企业微信认证配置失败';
        setError(message);
        toast.error(message);
      }
    })().finally(() => {
      loadPromiseRef.current = null;
    });
    loadPromiseRef.current = promise;
    return promise;
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  function updateField(field: SettingsField, value: unknown) {
    setClearRequested(false);
    setForm(current => ({ ...current, [field.key]: value }));
  }

  async function save() {
    const displayName = name.trim();
    if (!displayName) {
      toast.error('显示名称不能为空');
      return;
    }
    const result = prepareConfig(form, enabled);
    if (result.error) {
      toast.error(result.error);
      return;
    }
    setBusy('save');
    try {
      const saved = await saveWeComProvider({
        name: displayName,
        enabled,
        clearConfig: clearRequested,
        config: result.config,
      });
      setName(saved.name);
      setEnabled(saved.enabled);
      setSavedEnabled(saved.enabled);
      setForm({ ...defaultForm, ...saved.config });
      setClearRequested(false);
      toast.success('企业微信认证配置已保存');
      onChanged?.();
    } catch (err) {
      showErrorToast(err instanceof Error ? err.message : '企业微信认证配置保存失败');
    } finally {
      setBusy('');
    }
  }

  function clearConfig() {
    setEnabled(false);
    setForm(defaultForm);
    setClearRequested(true);
  }

  return (
    <SettingsDetailPanel
      header={
        <SettingsDetailHeader
          icon={MessageCircle}
          color="#22d3ee"
          title={name}
          subtitle={enabled ? '已启用' : '未启用'}
          active={enabled}
        />
      }
      actions={
        canManage ? (
          <>
            <ActionButton
              icon={<Trash2 size={14} />}
              label="清空配置"
              tone="danger"
              disabled={busy !== ''}
              onClick={clearConfig}
            />
            <ActionButton
              icon={<Save size={14} />}
              label="保存"
              busy={busy === 'save'}
              disabled={busy !== '' && busy !== 'save'}
              onClick={save}
            />
          </>
        ) : null
      }
    >
      {error ? (
        <p className="mb-4 rounded-lg border border-amber-400/25 bg-amber-500/10 p-3 text-sm text-amber-400">
          {error}
        </p>
      ) : null}
      <p className="mb-4 text-sm leading-6 text-[var(--itdb-text-muted)]">
        启用后登录页提供企业微信扫码登录，用户可在右上角菜单绑定企微账号。
      </p>
      <div className="space-y-3">
        <EnableToggle
          enabled={enabled}
          disabled={!canManage}
          onChange={value => {
            setEnabled(value);
            setClearRequested(false);
          }}
          label="启用企业微信认证"
          enabledText="登录页将显示企业微信扫码登录"
          disabledText="关闭后不会显示在登录页"
        />
        <ConfigField
          field={{ key: 'name', label: '显示名称', required: true }}
          value={name}
          disabled={!canManage}
          onChange={value => {
            setName(String(value ?? ''));
            setClearRequested(false);
          }}
        />
        <div className="space-y-3">
          <SectionTitle title="应用配置" />
          {fields.map(field => (
            <div key={field.key}>
              <ConfigField
                field={field}
                value={form[field.key]}
                secretConfigured={field.key === 'secret' && Boolean(form.hasSecret)}
                disabled={!canManage}
                onChange={value => updateField(field, value)}
              />
            </div>
          ))}
        </div>
        <p className="rounded-lg border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] p-3 text-xs leading-5 text-[var(--itdb-text-muted)]">
          回调地址前缀为企业外部可访问的站点地址（不含 /login
          路径），并需在企业微信管理后台将该域名配置为应用的可信回调域名，扫码后浏览器将携带授权码跳转回登录页完成登录或绑定
        </p>
      </div>
    </SettingsDetailPanel>
  );
}

function SectionTitle({ title }: { title: string }) {
  return <div className="text-xs font-semibold leading-5 text-[var(--itdb-text)]">{title}</div>;
}

function prepareConfig(form: WeComForm, enabled: boolean) {
  const config = Object.fromEntries(
    Object.entries(form).filter(([key, value]) => key !== 'hasSecret' && value !== '')
  );
  if (!enabled) return { config, error: '' };
  for (const field of [
    { key: 'corpid', label: '企业 ID（corpid）' },
    { key: 'agentid', label: '应用 AgentID' },
    { key: 'redirectPrefix', label: '回调地址前缀' },
  ] as const) {
    if (!String(config[field.key] || '').trim()) {
      return { config: {}, error: `${field.label}不能为空` };
    }
  }
  if (!config.secret && !form.hasSecret) {
    return { config: {}, error: '应用 Secret 不能为空' };
  }
  return { config, error: '' };
}
