import { useRef } from 'react';
import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import type { DictionaryKey } from './dictionary-helpers';

const editorFieldLabels: Record<DictionaryKey, string> = {
  itemtypes: '名称',
  contracttypes: '名称',
  statustypes: '名称',
  filetypes: '名称',
  dpttypes: '名称',
  tags: '名称',
};

export type DictionaryEditorValue = {
  id?: number;
  name: string;
  originalName: string;
  hasSoftware: boolean;
  color: string;
};

export function DictionaryEditorDialog({
  editor,
  dictionaryKey,
  label,
  busy,
  onChange,
  onClose,
  onSave,
}: {
  editor: DictionaryEditorValue | null;
  dictionaryKey: DictionaryKey;
  label: string;
  busy: boolean;
  onChange: (value: DictionaryEditorValue) => void;
  onClose: () => void;
  onSave: () => void;
}) {
  const nameInputRef = useRef<HTMLInputElement | null>(null);
  return (
    <Dialog open={editor !== null} onOpenChange={open => !open && onClose()}>
      <DialogContent
        className="max-w-md"
        onOpenAutoFocus={event => {
          event.preventDefault();
          const input = nameInputRef.current;
          if (!input) return;
          input.focus();
          const length = input.value.length;
          input.setSelectionRange(length, length);
        }}
      >
        <DialogHeader>
          <DialogTitle>{editor?.id ? `修改${label}` : `新增${label}`}</DialogTitle>
        </DialogHeader>
        {editor ? (
          <div className="space-y-4 pr-20">
            <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
              <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
                {editorFieldLabels[dictionaryKey]}
                <span className="ml-1 text-red-500">*</span>
              </span>
              <Input
                ref={nameInputRef}
                value={editor.name}
                onChange={event => onChange({ ...editor, name: event.target.value })}
                placeholder={`请输入${label}名称`}
                onKeyDown={event => {
                  if (event.key === 'Enter') {
                    event.preventDefault();
                    if (editor.name.trim()) onSave();
                  }
                }}
              />
            </div>
            {dictionaryKey === 'itemtypes' ? (
              <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
                <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
                  支持软件
                </span>
                <label className="flex cursor-pointer items-center gap-2 text-sm text-[var(--itdb-text)]">
                  <input
                    type="checkbox"
                    className="itdb-check-box"
                    checked={editor.hasSoftware}
                    onChange={event => onChange({ ...editor, hasSoftware: event.target.checked })}
                  />
                  该类型硬件可关联软件
                </label>
              </div>
            ) : null}
            {dictionaryKey === 'statustypes' ? (
              <div className="grid grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-x-2.5">
                <span className="text-right text-xs font-medium leading-5 text-[var(--itdb-text-muted)]">
                  状态颜色
                </span>
                <div className="flex items-center gap-2">
                  <input
                    type="color"
                    value={editor.color}
                    onChange={event => onChange({ ...editor, color: event.target.value })}
                    className="h-9 w-14 cursor-pointer rounded-lg border border-[var(--itdb-border)] bg-[var(--itdb-control-bg-soft)] p-1"
                    aria-label="状态颜色"
                  />
                  <span
                    className="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-medium"
                    style={{
                      borderColor: `${editor.color}55`,
                      background: `${editor.color}1f`,
                      color: editor.color,
                    }}
                  >
                    <span className="h-2 w-2 rounded-full" style={{ background: editor.color }} />
                    {editor.name.trim() || '状态预览'}
                  </span>
                </div>
              </div>
            ) : null}
          </div>
        ) : null}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            取消
          </Button>
          <Button
            type="button"
            className="itdb-action-button disabled:pointer-events-auto disabled:cursor-not-allowed"
            style={{
              borderColor: 'rgba(59,130,246,0.38)',
              background: 'rgba(59,130,246,0.1)',
              color: 'var(--itdb-accent-text)',
            }}
            disabled={!editor?.name.trim() || busy}
            onClick={onSave}
          >
            {busy ? '保存中...' : editor?.id ? '保存修改' : '创建'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
