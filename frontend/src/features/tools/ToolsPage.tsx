import { useQuery } from '@tanstack/react-query';
import { RotateCcw } from 'lucide-react';
import { useEffect, useState } from 'react';
import { toast } from 'sonner';
import { PermissionGate } from '@/components/permission-gate';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { api, getStoredUser, userHasAnyPermission } from '@/lib/auth';
import { PERM } from '@/lib/permissions';
import { BrowsePage } from './BrowsePage';
import { DictionaryManager, type DictionaryName } from './DictionaryManager';
import { LabelsPage } from './LabelsPage';
import { reportDisplayName, ReportsPage, type ReportMeta } from './ReportsPage';

type Row = Record<string, unknown>;

type ToolTab = 'dictionaries' | 'labels' | 'reports' | 'browse';

/* 各工具页签的整页访问权限，资料管理由内部页签与闸门自行控制 */
const toolAccess: Record<ToolTab, string[]> = {
  dictionaries: [],
  labels: [PERM.labelsPreview],
  reports: [PERM.reportsRead],
  browse: [PERM.browseRead],
};

export function ITDBToolsPage({
  initialTab = 'dictionaries',
  initialDictionary,
}: {
  initialTab?: ToolTab;
  initialDictionary?: DictionaryName;
}) {
  const [tab, setTab] = useState<ToolTab>(initialTab);
  const [browseResetKey, setBrowseResetKey] = useState(0);
  const [reportName, setReportName] = useState('');
  const [reportKeyword, setReportKeyword] = useState('');
  const canAccessReports = userHasAnyPermission(getStoredUser(), toolAccess.reports);
  const reportsQuery = useQuery({
    queryKey: ['itdb', 'reports'],
    queryFn: () => api<ReportMeta[]>('/api/reports'),
    enabled: canAccessReports,
  });
  const reportList = reportsQuery.data ?? [];
  const effectiveReport = reportName || reportList[0]?.name || '';

  useEffect(() => setTab(initialTab), [initialTab]);
  const pageMeta: Record<ToolTab, { title: string; description: string }> = {
    dictionaries: { title: '资料管理', description: '维护资产类型、部门、状态与标记等相关资料' },
    labels: { title: '打印标签', description: '选择并预览硬件资产标签' },
    reports: { title: '统计报表', description: '运行资产统计报表' },
    browse: { title: '资产导航', description: '按维度逐层定位资产' },
  };
  const meta = pageMeta[tab];
  const requiredAccess = toolAccess[tab];
  if (requiredAccess.length > 0 && !userHasAnyPermission(getStoredUser(), requiredAccess)) {
    return (
      <PermissionGate anyOf={requiredAccess}>
        <section className="h-full min-h-0" />
      </PermissionGate>
    );
  }
  return (
    <section
      className={`flex flex-col gap-5 ${tab === 'labels' ? 'min-h-full' : 'h-full min-h-0'}`}
    >
      <header className="flex shrink-0 flex-wrap items-center justify-between gap-4">
        <div className="min-w-0">
          <h1 className="text-lg font-semibold text-[var(--itdb-text)]">{meta.title}</h1>
          <p className="mt-1 text-sm text-[var(--itdb-text-muted)]">{meta.description}</p>
        </div>
        {tab === 'browse' ? (
          <Button
            type="button"
            variant="outline"
            className="itdb-action-button shrink-0"
            style={{
              borderColor: 'rgba(59,130,246,0.38)',
              background: 'rgba(59,130,246,0.1)',
              color: 'var(--itdb-accent-text)',
            }}
            onClick={() => setBrowseResetKey(value => value + 1)}
          >
            <RotateCcw size={16} />
            重置
          </Button>
        ) : null}
        {tab === 'reports' ? (
          <div className="flex shrink-0 flex-wrap items-center justify-end gap-3">
            <label className="flex items-center gap-2 text-sm text-[var(--itdb-text-muted)]">
              选择报表
              <Select
                value={effectiveReport || undefined}
                onValueChange={name => {
                  setReportName(name);
                  setReportKeyword('');
                }}
              >
                <SelectTrigger className="w-64" aria-label="选择报表">
                  <SelectValue placeholder="请选择报表" />
                </SelectTrigger>
                <SelectContent>
                  {reportList.map(report => (
                    <SelectItem key={report.name} value={report.name}>
                      {reportDisplayName(report)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </label>
            <label className="flex items-center gap-2 text-sm text-[var(--itdb-text-muted)]">
              查询
              <Input
                value={reportKeyword}
                onChange={event => setReportKeyword(event.target.value)}
                className="w-56"
                placeholder="输入关键词过滤当前报表"
              />
            </label>
          </div>
        ) : null}
      </header>
      {tab === 'dictionaries' && <DictionaryManager initialDictionary={initialDictionary} />}
      {tab === 'labels' && <LabelsPage />}
      {tab === 'reports' && <ReportsPage active={effectiveReport} keyword={reportKeyword} />}
      {tab === 'browse' && <BrowsePage key={browseResetKey} />}
    </section>
  );
}
