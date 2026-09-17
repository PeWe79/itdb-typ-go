import { RotateCcw, Save, Trash2 } from 'lucide-react';
import { type ReactNode, useState } from 'react';
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
import { type LabelConfig } from './labels-config';

type Row = Record<string, unknown>;

type PropertyPanelProps = {
  config: LabelConfig;
  update: (key: keyof LabelConfig, value: string | number | boolean) => void;
  presets: Row[];
  selectedPresetId: string;
  presetName: string;
  setPresetName: (value: string) => void;
  onSavePreset: () => void;
  saving: boolean;
  onApply: (row: Row) => void;
  onDelete: (row: Row) => void;
  onReset: () => void;
  canEdit: boolean;
};

const TEXT_FIELDS: Array<{ key: keyof LabelConfig; label: string; placeholder?: string }> = [
  { key: 'lwidth', label: '标签宽度(mm)', placeholder: '66' },
  { key: 'lheight', label: '标签高度(mm)', placeholder: '35' },
  { key: 'hpitch', label: '水平间距(mm)', placeholder: '70' },
  { key: 'vpitch', label: '垂直间距(mm)', placeholder: '35' },
  { key: 'tmargin', label: '上边距(mm)', placeholder: '12' },
  { key: 'bmargin', label: '下边距(mm)', placeholder: '12' },
  { key: 'lmargin', label: '左边距(mm)', placeholder: '6' },
  { key: 'rmargin', label: '右边距(mm)', placeholder: '6' },
  { key: 'border', label: '边框颜色(0-255)', placeholder: '200' },
  { key: 'padding', label: '文本填充(mm)', placeholder: '1' },
  { key: 'fontsize', label: '正文字号(pt)', placeholder: '6' },
  { key: 'idfontsize', label: '编号字号(pt)', placeholder: '7' },
  { key: 'headerfontsize', label: '标题字号(pt)', placeholder: '6' },
  { key: 'barcodesize', label: '二维码边长(mm)', placeholder: '20' },
  { key: 'imagewidth', label: '图片宽(mm)', placeholder: '5' },
  { key: 'imageheight', label: '图片高(mm)', placeholder: '5' },
];

const SWITCH_FIELDS: Array<{ key: keyof LabelConfig; label: string; tooltip?: string }> = [
  { key: 'wantbarcode', label: '打印二维码' },
  { key: 'wantheadertext', label: '打印页头文字' },
  { key: 'wantheaderimage', label: '打印页头图片' },
  { key: 'wantnotext', label: '仅打印条码', tooltip: '仅打印条码，不显示正文文本' },
  { key: 'wantraligntext', label: '条码右侧显示文字', tooltip: '将正文文本显示在二维码右侧' },
];

/* 内置标签预设名单：同名预设不允许删除或覆盖保存 */
export const BUILTIN_PRESET_NAMES = new Set(['Avery6106', 'AveryL6009', 'AveryL6009-6110']);

export function PropertyPanel(props: PropertyPanelProps) {
  const [imageDraft, setImageDraft] = useState<string | null>(null);
  const [qrDraft, setQrDraft] = useState<string | null>(null);
  const updateText = (key: keyof LabelConfig) => (event: { target: { value: string } }) =>
    props.update(key, event.target.value);
  const updateBorder = (event: { target: { value: string } }) => {
    const raw = event.target.value.trim();
    if (raw !== '' && (!/^\d+$/.test(raw) || Number(raw) > 255)) {
      toast.warning('边框颜色需为 0-255 范围内的整数');
      return;
    }
    props.update('border', raw);
  };
  const selectedPreset = props.presets.find(item => String(item.id) === props.selectedPresetId);
  const effectiveName = (props.config.name || props.presetName).trim();
  const nameExists = props.presets.some(row => String(row.name ?? '').trim() === effectiveName);
  return (
    <section
      className="itdb-card-hover flex flex-col rounded-xl p-5"
      style={{
        background: 'var(--itdb-card)',
        border: '1px solid var(--itdb-border)',
        boxShadow: 'var(--shadow-card)',
      }}
    >
      <div className="flex shrink-0 items-center justify-between border-b border-[var(--itdb-border)] pb-3">
        <h3 className="text-base font-semibold text-[var(--itdb-text)]">标签属性</h3>
        <Button
          size="sm"
          variant="outline"
          className="itdb-action-button h-8 disabled:cursor-not-allowed disabled:opacity-50"
          disabled={!props.canEdit}
          onClick={props.onReset}
        >
          <RotateCcw size={14} />
          恢复默认
        </Button>
      </div>
      <div className="mt-3 grid items-start gap-4 md:grid-cols-2 xl:grid-cols-3">
        <ConfigGroup title="预设">
          <div className="space-y-1">
            <span className="text-xs text-[var(--itdb-text-muted)]">载入预设</span>
            {props.presets.length === 0 ? (
              <div className="itdb-form-control h-9 w-full rounded-lg px-2 text-sm leading-9 text-[var(--itdb-text-muted)]">
                选择预设载入...
              </div>
            ) : (
              <Select
                value={props.selectedPresetId}
                onValueChange={value => {
                  const preset = props.presets.find(item => String(item.id) === value);
                  if (preset) props.onApply(preset);
                }}
              >
                <SelectTrigger className="h-9 w-full" aria-label="载入预设">
                  <SelectValue placeholder="选择预设载入..." />
                </SelectTrigger>
                <SelectContent>
                  {props.presets.map(preset => (
                    <SelectItem key={String(preset.id)} value={String(preset.id)}>
                      {String(preset.name ?? preset.id)}（{String(preset.papersize ?? '')}）
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>
          <div className="space-y-1">
            <span className="text-xs text-[var(--itdb-text-muted)]">预设名称</span>
            <div className="flex gap-2">
              <Input
                value={props.presetName || props.config.name}
                onChange={event => {
                  props.setPresetName(event.target.value);
                  props.update('name', event.target.value);
                }}
                placeholder="预设名称"
                className="h-9 flex-1"
                disabled={!props.canEdit}
              />
              <AppTooltip label={nameExists ? '更新当前标签预设' : '保存为新标签预设'}>
                <Button
                  size="sm"
                  variant="outline"
                  className="itdb-action-button h-9 w-9 shrink-0 px-0 disabled:cursor-not-allowed disabled:opacity-50"
                  style={{
                    borderColor: 'rgba(59,130,246,0.5)',
                    background: 'rgba(59,130,246,0.12)',
                    color: 'var(--itdb-accent-text)',
                  }}
                  disabled={!props.canEdit || props.saving}
                  onClick={props.onSavePreset}
                >
                  <Save size={14} />
                </Button>
              </AppTooltip>
              <AppTooltip label="删除">
                <Button
                  size="sm"
                  variant="outline"
                  className="itdb-action-button itdb-danger-button h-9 w-9 shrink-0 px-0 disabled:cursor-not-allowed disabled:opacity-50"
                  style={{
                    borderColor: 'rgba(239,68,68,0.5)',
                    background: 'rgba(239,68,68,0.12)',
                    color: 'var(--itdb-status-red-text, #ef4444)',
                  }}
                  disabled={!props.canEdit}
                  onClick={() => {
                    if (!selectedPreset) {
                      toast.error('请先选择要删除的标签预设');
                      return;
                    }
                    if (BUILTIN_PRESET_NAMES.has(String(selectedPreset.name ?? '').trim())) {
                      toast.error('内置标签预设无法删除');
                      return;
                    }
                    props.onDelete(selectedPreset);
                  }}
                  aria-label="删除选中的标签预设"
                >
                  <Trash2 size={14} />
                </Button>
              </AppTooltip>
            </div>
          </div>
          <p className="text-xs text-[var(--itdb-text-muted)]">
            共 {props.presets.length}{' '}
            个预设。如需创建新标签预设，使用新的预设名称以及需修改的配置后保存即可
          </p>
        </ConfigGroup>
        <ConfigGroup title="纸张与布局">
          <label className="block space-y-1">
            <span className="text-xs text-[var(--itdb-text-muted)]">纸张规格</span>
            <Select
              value={props.config.papersize}
              disabled={!props.canEdit}
              onValueChange={value => props.update('papersize', value)}
            >
              <SelectTrigger className="h-9 w-full" aria-label="纸张规格">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {Object.keys(paperSizeCatalog()).map(name => (
                  <SelectItem key={name} value={name}>
                    {name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </label>
          <div className="grid grid-cols-2 gap-2">
            <NumberSelect
              label="行数"
              value={props.config.rows}
              min={1}
              max={39}
              disabled={!props.canEdit}
              onChange={value => props.update('rows', value)}
            />
            <NumberSelect
              label="列数"
              value={props.config.cols}
              min={1}
              max={9}
              disabled={!props.canEdit}
              onChange={value => props.update('cols', value)}
            />
          </div>
          <div className="space-y-1">
            <AppTooltip label="当顶部标签已经被打印过时使用" placement="top" align="end">
              <span className="cursor-help text-xs text-[var(--itdb-text-muted)]">
                顶部跳过标签数
              </span>
            </AppTooltip>
            <Input
              value={props.config.labelskip}
              onChange={updateText('labelskip')}
              placeholder="0"
              className="h-9"
              disabled={!props.canEdit}
            />
          </div>
        </ConfigGroup>
        <ConfigGroup title="字号与样式">
          <div className="grid grid-cols-2 gap-2">
            {TEXT_FIELDS.slice(10).map(field => (
              <TextField
                key={String(field.key)}
                label={field.label}
                value={String(props.config[field.key] ?? '')}
                onChange={updateText(field.key)}
                placeholder={field.placeholder}
                disabled={!props.canEdit}
              />
            ))}
          </div>
          <p className="text-xs text-[var(--itdb-text-muted)]">
            字号以打印 pt 生效（1pt=0.3527 mm），预览按比例近似渲染
          </p>
        </ConfigGroup>
        <ConfigGroup title="间距与边距">
          <div className="grid grid-cols-2 gap-2">
            {TEXT_FIELDS.slice(0, 10).map(field => (
              <TextField
                key={String(field.key)}
                label={field.label}
                value={String(props.config[field.key] ?? '')}
                onChange={field.key === 'border' ? updateBorder : updateText(field.key)}
                placeholder={field.placeholder}
                disabled={!props.canEdit}
              />
            ))}
          </div>
        </ConfigGroup>
        <ConfigGroup title="页头与二维码" className="md:col-span-2 xl:col-span-2">
          <label className="block space-y-1">
            <span className="text-xs text-[var(--itdb-text-muted)]">页头文字（回车换行）</span>
            <textarea
              value={props.config.headertext}
              onChange={updateText('headertext')}
              placeholder={'IT资产标签\n第二行文字'}
              rows={3}
              disabled={!props.canEdit}
              className="itdb-form-control min-h-24 w-full resize-y rounded-lg px-3 py-2 text-sm disabled:cursor-not-allowed disabled:opacity-60"
            />
          </label>
          <div className="grid gap-2 md:grid-cols-2">
            <label className="block space-y-1">
              <AppTooltip
                label="留空时扫码显示正文文本、下方显示编号；填写后扫码显示前缀+编号，如资产访问网址"
                placement="top"
              >
                <span className="cursor-help text-xs text-[var(--itdb-text-muted)]">
                  二维码 URL 前缀
                </span>
              </AppTooltip>
              <Input
                value={qrDraft ?? props.config.qrtext}
                onChange={event => setQrDraft(event.target.value)}
                onBlur={() => {
                  if (qrDraft !== null) {
                    props.update('qrtext', qrDraft);
                    setQrDraft(null);
                  }
                }}
                placeholder="http://服务器地址/assets/hardware?edit="
                className="h-9"
                disabled={!props.canEdit}
              />
            </label>
            <label className="block space-y-1">
              <AppTooltip label="支持站点根路径、完整网址或 Base64 图片" placement="top">
                <span className="cursor-help text-xs text-[var(--itdb-text-muted)]">
                  页头图片路径
                </span>
              </AppTooltip>
              <Input
                value={imageDraft ?? props.config.image}
                onChange={event => setImageDraft(event.target.value)}
                onBlur={() => {
                  if (imageDraft !== null) {
                    props.update('image', imageDraft);
                    setImageDraft(null);
                  }
                }}
                placeholder="images/itdb.png"
                className="h-9"
                disabled={!props.canEdit}
              />
            </label>
          </div>
          <div className="flex flex-wrap gap-x-6 gap-y-2 pt-3">
            {SWITCH_FIELDS.map(field => (
              <label
                key={String(field.key)}
                className={`flex items-center gap-2 text-sm text-[var(--itdb-text)] ${props.canEdit ? 'cursor-pointer' : 'cursor-not-allowed opacity-60'}`}
              >
                <input
                  type="checkbox"
                  className="itdb-check-box"
                  checked={Boolean(props.config[field.key])}
                  onChange={event => props.update(field.key, event.target.checked)}
                  disabled={!props.canEdit}
                />
                {field.tooltip ? (
                  <AppTooltip label={field.tooltip} placement="top">
                    <span className="cursor-help">{field.label}</span>
                  </AppTooltip>
                ) : (
                  field.label
                )}
              </label>
            ))}
          </div>
        </ConfigGroup>
      </div>
    </section>
  );
}

function paperSizeCatalog() {
  return {
    A3: 1,
    A4: 1,
    A5: 1,
    A6: 1,
    B4: 1,
    B5: 1,
    B6: 1,
    LETTER: 1,
    LEGAL: 1,
    EXECUTIVE: 1,
    TABLOID: 1,
  };
}

function ConfigGroup({
  title,
  children,
  className,
}: {
  title: string;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={`space-y-2 self-stretch rounded-lg border border-[var(--itdb-border)] p-3 ${className ?? ''}`}
    >
      <p className="text-xs font-semibold text-[var(--itdb-text)]">{title}</p>
      {children}
    </div>
  );
}

function TextField({
  label,
  value,
  onChange,
  placeholder,
  disabled,
}: {
  label: string;
  value: string;
  onChange: (event: { target: { value: string } }) => void;
  placeholder?: string;
  disabled?: boolean;
}) {
  return (
    <label className="block space-y-1">
      <span className="text-xs text-[var(--itdb-text-muted)]">{label}</span>
      <Input
        value={value}
        onChange={onChange as never}
        placeholder={placeholder}
        className="h-9"
        disabled={disabled}
      />
    </label>
  );
}

function NumberSelect({
  label,
  value,
  min,
  max,
  onChange,
  disabled,
}: {
  label: string;
  value: number;
  min: number;
  max: number;
  onChange: (value: number) => void;
  disabled?: boolean;
}) {
  return (
    <label className="block space-y-1">
      <span className="text-xs text-[var(--itdb-text-muted)]">{label}</span>
      <Select
        value={String(value)}
        onValueChange={next => onChange(Number(next))}
        disabled={disabled}
      >
        <SelectTrigger className="h-9 w-full">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {Array.from({ length: max - min + 1 }, (_, index) => min + index).map(number => (
            <SelectItem key={number} value={String(number)}>
              {number}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </label>
  );
}
