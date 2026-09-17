import { toast } from 'sonner';

/**
 * 展示后端错误消息：含多行（如代理多个类型同时被引用的聚合拦截）时
 * 拆分为多条提示，单行时行为与 toast.error 一致
 */
export function showErrorToast(message: string) {
  const lines = String(message ?? '')
    .split('\n')
    .map(line => line.trim())
    .filter(Boolean);
  if (lines.length <= 1) {
    toast.error(lines[0] ?? String(message ?? '操作失败'));
    return;
  }
  lines.forEach(line => toast.error(line));
}
