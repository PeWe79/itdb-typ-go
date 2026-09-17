import { api } from '@/lib/auth';

export type AuditTrackPayload = {
  type: string;
  names?: string[];
  existing?: string[];
  target?: string;
  count?: number;
  result?: 'success' | 'failure';
};

/* trackAuditEvent 上报前端执行的审计事件（导出/导入/打印），失败仅告警不阻塞前端操作 */
export async function trackAuditEvent(payload: AuditTrackPayload) {
  try {
    await api('/api/history/events', { method: 'POST', body: JSON.stringify(payload) });
  } catch (error) {
    console.warn('审计事件上报失败', error);
  }
}
